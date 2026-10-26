package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
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

func TestListenAddrHonoursPort(t *testing.T) {
	cases := map[string]string{"": ":8080", " ": ":8080", "8091": ":8091", " 8092\n": ":8092"}
	for in, want := range cases {
		if got := listenAddr(in); got != want {
			t.Fatalf("listenAddr(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHealthReportsEnvironmentLabel(t *testing.T) {
	t.Setenv("ORDER_PORTAL_ENV", "prod-demo")
	s := newServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	s.routes().ServeHTTP(res, req)
	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode healthz: %v", err)
	}
	if body["environment"] != "prod-demo" {
		t.Fatalf("healthz environment = %q, want prod-demo", body["environment"])
	}
	if environmentLabel("") != "local" {
		t.Fatalf("unset ORDER_PORTAL_ENV should report local")
	}
}

func TestHealthReportsRevisionAndBuildDigest(t *testing.T) {
	t.Setenv("ORDER_PORTAL_ENV", "staging")
	s := newServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	s.routes().ServeHTTP(res, req)
	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode healthz: %v", err)
	}
	if !strings.HasPrefix(body["revision"], "staging-") || body["revision"] != s.revision {
		t.Fatalf("revision must be reported and carry the environment label: %q", body["revision"])
	}
	if want := independentExecutableDigest(t); body["build_digest"] != want {
		t.Fatalf("build_digest must be the sha256 of the executable: got %q, want %q", body["build_digest"], want)
	}
	if body["started_at"] == "" {
		t.Fatalf("started_at must be reported")
	}
	// The same build answers with the same digest; a new start is a new revision.
	again := newServer()
	if again.buildDigest != s.buildDigest {
		t.Fatalf("build digest must be stable for one executable: %q vs %q", again.buildDigest, s.buildDigest)
	}
	if revisionName("prod-demo", time.Date(2026, 10, 26, 1, 2, 3, 0, time.UTC), 42) != "prod-demo-20261026t010203z-42" {
		t.Fatalf("unexpected revision format: %s", revisionName("prod-demo", time.Date(2026, 10, 26, 1, 2, 3, 0, time.UTC), 42))
	}
}

// independentExecutableDigest recomputes the digest without executableDigest so
// the test fails if that helper hashes the wrong file or a constant.
func independentExecutableDigest(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open executable: %v", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Fatalf("hash executable: %v", err)
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func TestRevisionLabelIsSanitised(t *testing.T) {
	cases := map[string]string{"staging": "staging", "Prod Demo": "prod-demo", "a/b\nc": "a-b-c", "--": "local", "": "local"}
	for in, want := range cases {
		if got := revisionLabel(in); got != want {
			t.Fatalf("revisionLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestRestartIsANewRevision starts this test binary twice as a real process
// (the helper below prints one /healthz body) and checks that the two starts
// report different revisions and the same build digest. The two starts usually
// land in the same second, so it is the pid part that separates them.
func TestRestartIsANewRevision(t *testing.T) {
	if os.Getenv("ORDER_PORTAL_HEALTH_HELPER") == "1" {
		t.Skip("helper process")
	}
	var bodies [2]map[string]string
	for i := range bodies {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHealthHelperProcess$")
		cmd.Env = append(os.Environ(), "ORDER_PORTAL_HEALTH_HELPER=1", "ORDER_PORTAL_ENV=staging")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("start %d: %v", i+1, err)
		}
		line, _, _ := strings.Cut(string(out), "\n")
		if err := json.Unmarshal([]byte(line), &bodies[i]); err != nil {
			t.Fatalf("start %d: decode %q: %v", i+1, line, err)
		}
	}
	if bodies[0]["revision"] == "" || bodies[0]["revision"] == bodies[1]["revision"] {
		t.Fatalf("two starts must report different revisions: %q vs %q", bodies[0]["revision"], bodies[1]["revision"])
	}
	if bodies[0]["build_digest"] == "" || bodies[0]["build_digest"] != bodies[1]["build_digest"] {
		t.Fatalf("two starts of one binary must report the same digest: %q vs %q", bodies[0]["build_digest"], bodies[1]["build_digest"])
	}
}

func TestHealthHelperProcess(t *testing.T) {
	if os.Getenv("ORDER_PORTAL_HEALTH_HELPER") != "1" {
		t.Skip("only runs as the helper process of TestRestartIsANewRevision")
	}
	res := httptest.NewRecorder()
	newServer().health(res, httptest.NewRequest(http.MethodGet, "/healthz", nil)) // health directly: the logging middleware would print first
	fmt.Print(res.Body.String())
}
