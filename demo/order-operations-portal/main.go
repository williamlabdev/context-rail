package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// evidenceReference is one supporting reference a reviewer attaches to an
// order exception (REQ-002): a label and an http(s) link, kept in memory next
// to the order. Nothing is uploaded or stored; only the link is remembered.
type evidenceReference struct {
	Label   string `json:"label"`
	URL     string `json:"url"`
	AddedAt string `json:"addedAt"`
}

type evidenceRequest struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// maxEvidencePerOrder caps the references on one order (AC-002).
const maxEvidencePerOrder = 3

// orderView is what GET /api/orders returns: the order plus its evidence
// references (AC-004). The references live outside `order` so that existing
// review behaviour and its tests stay untouched (AC-005).
type orderView struct {
	order
	Evidence []evidenceReference `json:"evidence"`
}

type server struct {
	mu          sync.RWMutex
	environment string
	revision    string
	buildDigest string
	startedAt   string
	orders      map[string]*order
	evidence    map[string][]evidenceReference
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
		},
		evidence: map[string][]evidenceReference{},
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.home)
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/orders", s.listOrders)
	mux.HandleFunc("POST /api/orders/", s.orderAction)
	return logging(mux)
}

// orderAction dispatches POST /api/orders/{id}/review and
// POST /api/orders/{id}/evidence; anything else under the prefix is 404.
func (s *server) orderAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "orders" {
		http.NotFound(w, r)
		return
	}
	switch parts[3] {
	case "review":
		s.review(w, r, parts[2])
	case "evidence":
		s.addEvidence(w, r, parts[2])
	default:
		http.NotFound(w, r)
	}
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

// orderStatuses are the only values an order's status can take: the initial
// PENDING_REVIEW and the two review decisions.
var orderStatuses = []string{"PENDING_REVIEW", "APPROVED", "REJECTED"}

// validOrderStatus reports whether raw is one of orderStatuses, compared
// exactly (no trimming or case folding) so the filter matches the stored value.
func validOrderStatus(raw string) bool {
	for _, status := range orderStatuses {
		if raw == status {
			return true
		}
	}
	return false
}

// listOrders returns every order, or only the orders in one status when the
// `status` query parameter is present (AWO-001). A status parameter that is
// not one of orderStatuses is a 400 with a JSON error; no parameter at all
// keeps the previous behaviour of listing everything.
func (s *server) listOrders(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := query.Get("status")
	if query.Has("status") && !validOrderStatus(filter) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be " + strings.Join(orderStatuses, ", ")})
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]orderView, 0, len(s.orders))
	for _, item := range s.orders {
		if filter != "" && item.Status != filter {
			continue
		}
		result = append(result, orderView{order: *item, Evidence: s.evidenceOf(item.ID)})
	}
	writeJSON(w, http.StatusOK, result)
}

// evidenceOf returns a copy of the order's references, never nil, so the JSON
// always carries an `evidence` array. Callers hold s.mu.
func (s *server) evidenceOf(orderID string) []evidenceReference {
	return append([]evidenceReference{}, s.evidence[orderID]...)
}

// decodeSingleJSON reads exactly one JSON object from the body; anything
// malformed or trailing is a 400 with a JSON error (same rule as review).
func decodeSingleJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request must be valid JSON"})
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request must contain one JSON object"})
		return false
	}
	return true
}

func (s *server) review(w http.ResponseWriter, r *http.Request, orderID string) {
	var req reviewRequest
	if !decodeSingleJSON(w, r, &req) {
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
	item, ok := s.orders[orderID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	item.Status = req.Decision
	item.Note = strings.TrimSpace(req.Note)
	item.ReviewedAt = time.Now().UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusOK, map[string]any{"order": item, "reviewedAt": item.ReviewedAt})
}

// addEvidence attaches one evidence reference to an order (REQ-002).
// AC-001: label and http(s) URL required → otherwise 400 JSON error.
// AC-002: the fourth reference on the same order → 409.
// AC-003: unknown order → 404.
func (s *server) addEvidence(w http.ResponseWriter, r *http.Request, orderID string) {
	var req evidenceRequest
	if !decodeSingleJSON(w, r, &req) {
		return
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "label is required"})
		return
	}
	link, err := validEvidenceURL(req.URL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orders[orderID]; !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
		return
	}
	if len(s.evidence[orderID]) >= maxEvidencePerOrder {
		writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("order already has %d evidence references", maxEvidencePerOrder)})
		return
	}
	reference := evidenceReference{Label: label, URL: link, AddedAt: time.Now().UTC().Format(time.RFC3339)}
	s.evidence[orderID] = append(s.evidence[orderID], reference)
	writeJSON(w, http.StatusCreated, map[string]any{"order": orderID, "evidence": s.evidenceOf(orderID)})
}

// validEvidenceURL accepts only absolute http or https links with a host; the
// link is remembered as given (trimmed), never fetched.
func validEvidenceURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("url must be an absolute http or https link")
	}
	return raw, nil
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
