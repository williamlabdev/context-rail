package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	mu     sync.RWMutex
	orders map[string]*order
}

func newServer() *server {
	return &server{orders: map[string]*order{
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "order-operations-portal"})
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
	_ = http.ListenAndServe(":8080", newServer().routes())
}
