package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthAndOrderList(t *testing.T) {
	s := newServer()
	for _, path := range []string{"/healthz", "/api/orders"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		s.routes().ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("GET %s: got %d", path, res.Code)
		}
	}
}

func TestReviewRequiresNote(t *testing.T) {
	s := newServer()
	body, _ := json.Marshal(reviewRequest{Decision: "APPROVED"})
	req := httptest.NewRequest(http.MethodPost, "/api/orders/ORD-1001/review", bytes.NewReader(body))
	res := httptest.NewRecorder()
	s.routes().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestReviewUpdatesOrder(t *testing.T) {
	s := newServer()
	body, _ := json.Marshal(reviewRequest{Decision: "APPROVED", Note: "Address confirmed with customer"})
	req := httptest.NewRequest(http.MethodPost, "/api/orders/ORD-1001/review", bytes.NewReader(body))
	res := httptest.NewRecorder()
	s.routes().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", res.Code)
	}
	if s.orders["ORD-1001"].Status != "APPROVED" {
		t.Fatalf("status was not updated")
	}
}

func TestReviewUnknownOrder(t *testing.T) {
	s := newServer()
	body, _ := json.Marshal(reviewRequest{Decision: "REJECTED", Note: "Insufficient information"})
	req := httptest.NewRequest(http.MethodPost, "/api/orders/ORD-9999/review", bytes.NewReader(body))
	res := httptest.NewRecorder()
	s.routes().ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", res.Code)
	}
}
