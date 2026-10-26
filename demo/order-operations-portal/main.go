package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed web/index.html
var indexHTML []byte

type order struct {
	ID         string `json:"id"`
	Customer   string `json:"customer"`
	Amount     int    `json:"amount"`
	Reason     string `json:"reason"`
	Status     string `json:"status"`
	Note       string `json:"note,omitempty"`
	ReviewedAt string `json:"reviewedAt,omitempty"`
}

type reviewRequest struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
}

type server struct {
	mu          sync.RWMutex
	environment string
	revision    string
	buildDigest string
	startedAt   string
	orders      map[string]*order
}

// newServer builds the in-memory portal. The environment label is read from
// ORDER_PORTAL_ENV so two local instances (DR-015: local staging and local
// prod-demo) can be told apart from their own /healthz read-back, not only from
// the port they were started on. The revision and build digest give a local
// instance the two identities a Cloud Run revision has: which build is
// running (digest of this executable) and which start of it answers
// (revision, new on every process start).
func newServer() *server {
	started := time.Now().UTC()
	environment := environmentLabel(os.Getenv("ORDER_PORTAL_ENV"))
	return &server{
		environment: environment,
		revision:    revisionName(environment, started, os.Getpid()),
		buildDigest: processDigest(),
		startedAt:   started.Format(time.RFC3339),
		orders: map[string]*order{
			"ORD-1001": {ID: "ORD-1001", Customer: "Acme North", Amount: 12800, Reason: "Shipping address changed after payment", Status: "PENDING_REVIEW"},
			"ORD-1002": {ID: "ORD-1002", Customer: "Beta Studio", Amount: 4200, Reason: "High-value order needs a second check", Status: "PENDING_REVIEW"},
		}}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.home)
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/orders", s.listOrders)
	mux.HandleFunc("POST /api/orders/", s.review)
	return logging(mux)
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok", "service": "order-operations-portal", "environment": s.environment,
		"revision": s.revision, "build_digest": s.buildDigest, "started_at": s.startedAt,
	})
}

// revisionName is unique per process start, like a Cloud Run revision is
// unique per deployment: <environment>-<start time>-<pid>. Restarting the
// same binary yields a new revision; the build digest is what stays equal.
// The label is reduced to [a-z0-9-] so an odd ORDER_PORTAL_ENV cannot put
// spaces, slashes or control characters into the revision.
func revisionName(environment string, started time.Time, pid int) string {
	return fmt.Sprintf("%s-%s-%d", revisionLabel(environment), started.Format("20060102t150405z"), pid)
}

func revisionLabel(environment string) string {
	label := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '-'
		}
	}, environment)
	if label = strings.Trim(label, "-"); label == "" {
		return "local"
	}
	return label
}

// processDigest hashes the executable once per process: every server in the
// process runs the same file, so later calls reuse the first result.
var processDigest = sync.OnceValue(executableDigest)

// executableDigest hashes the running executable so the read-back can tell
// which build answers, without a container registry. It is a binary digest,
// not an OCI image digest; an empty string means it could not be read. It
// hashes the file at os.Executable(), so a binary replaced on disk between
// exec and the first call would be misreported; there is no portable way to
// hash the mapped image instead (macOS has no /proc/self/exe).
func executableDigest() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

// environmentLabel normalises ORDER_PORTAL_ENV; an unset value reports "local"
// so a single-instance run is still distinguishable from a labelled one.
func environmentLabel(raw string) string {
	if label := strings.TrimSpace(raw); label != "" {
		return label
	}
	return "local"
}

// listenAddr honours PORT (Cloud Run convention, also what ContextRail itself
// reads) and falls back to :8080 so the documented `go run .` keeps working.
func listenAddr(port string) string {
	if port = strings.TrimSpace(port); port != "" {
		return ":" + port
	}
	return ":8080"
}

func (s *server) listOrders(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]order, 0, len(s.orders))
	for _, item := range s.orders {
		result = append(result, *item)
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *server) review(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "orders" || parts[3] != "review" {
		http.NotFound(w, r)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var req reviewRequest
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request must be valid JSON"})
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request must contain one JSON object"})
		return
	}
	if req.Decision != "APPROVED" && req.Decision != "REJECTED" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "decision must be APPROVED or REJECTED"})
		return
	}
	if strings.TrimSpace(req.Note) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "note is required"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.orders[parts[2]]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	item.Status = req.Decision
	item.Note = strings.TrimSpace(req.Note)
	item.ReviewedAt = time.Now().UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusOK, map[string]any{"order": item, "reviewedAt": item.ReviewedAt})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	addr := listenAddr(os.Getenv("PORT"))
	fmt.Printf("order-operations-portal listening on %s (environment=%s)\n", addr, environmentLabel(os.Getenv("ORDER_PORTAL_ENV")))
	_ = http.ListenAndServe(addr, newServer().routes())
}
