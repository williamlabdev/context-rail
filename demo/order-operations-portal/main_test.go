package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	for _, note := range []string{"", " \t\n ", "　"} {
		s := newServer()
		before := *s.orders["ORD-1001"]
		body, _ := json.Marshal(reviewRequest{Decision: "APPROVED", Note: note})
		req := httptest.NewRequest(http.MethodPost, "/api/orders/ORD-1001/review", bytes.NewReader(body))
		res := httptest.NewRecorder()
		s.routes().ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("note %q: got %d, want 400", note, res.Code)
		}
		if *s.orders["ORD-1001"] != before {
			t.Fatalf("note %q changed the order", note)
		}
	}
}

func TestReviewRejectsInvalidJSON(t *testing.T) {
	s := newServer()
	req := httptest.NewRequest(http.MethodPost, "/api/orders/ORD-1001/review", bytes.NewBufferString("{"))
	res := httptest.NewRecorder()
	s.routes().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.Code)
	}
}

func TestReviewRejectsTrailingJSON(t *testing.T) {
	for _, payload := range []string{
		`{"decision":"APPROVED","note":"valid prefix"}garbage`,
		`{"decision":"APPROVED","note":"first"}{"decision":"REJECTED","note":"second"}`,
	} {
		s := newServer()
		before := *s.orders["ORD-1001"]
		req := httptest.NewRequest(http.MethodPost, "/api/orders/ORD-1001/review", strings.NewReader(payload))
		res := httptest.NewRecorder()
		s.routes().ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("payload %q: got %d, want 400", payload, res.Code)
		}
		if *s.orders["ORD-1001"] != before {
			t.Fatalf("payload %q changed the order", payload)
		}
	}
}

func TestReviewRejectsUnsupportedDecision(t *testing.T) {

	s := newServer()
	body, _ := json.Marshal(reviewRequest{Decision: "PENDING", Note: "Needs another check"})
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
	if s.orders["ORD-1001"].ReviewedAt == "" {
		t.Fatalf("review timestamp was not recorded")
	}
	var bodyResponse struct {
		Order      order  `json:"order"`
		ReviewedAt string `json:"reviewedAt"`
	}
	if err := json.NewDecoder(res.Body).Decode(&bodyResponse); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if bodyResponse.ReviewedAt == "" || bodyResponse.Order.ReviewedAt != bodyResponse.ReviewedAt {
		t.Fatalf("response timestamp was not consistent: %+v", bodyResponse)
	}
}

func TestReviewCanBeRepeated(t *testing.T) {
	s := newServer()
	for _, review := range []reviewRequest{
		{Decision: "APPROVED", Note: "Address confirmed"},
		{Decision: "REJECTED", Note: "Customer requested cancellation"},
	} {
		body, _ := json.Marshal(review)
		req := httptest.NewRequest(http.MethodPost, "/api/orders/ORD-1001/review", bytes.NewReader(body))
		res := httptest.NewRecorder()
		s.routes().ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("got %d, want 200", res.Code)
		}
	}
	if s.orders["ORD-1001"].Status != "REJECTED" {
		t.Fatalf("status was not replaced by the latest review")
	}
	if s.orders["ORD-1001"].Note != "Customer requested cancellation" {
		t.Fatalf("note was not replaced by the latest review")
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
