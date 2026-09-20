package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	registryhttp "context-rail/internal/http"
)

func builtWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><div id=\"root\">workspace</div>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestHealthReportsLivenessOnly(t *testing.T) {
	handler := registryhttp.NewHealthHandler("container")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" || payload["mode"] != "container" {
		t.Fatalf("unexpected health payload: %v", payload)
	}
	surfaces, _ := payload["surfaces"].(map[string]any)
	if surfaces["registry"] != "read-only" || surfaces["topology"] != "versioned-writes" {
		t.Fatalf("unexpected surfaces: %v", payload["surfaces"])
	}
	for _, forbidden := range []string{"readiness", "authorization", "deployed"} {
		if _, present := payload[forbidden]; present {
			t.Fatalf("health payload must not claim %q", forbidden)
		}
	}
}

func TestHealthRejectsNonGET(t *testing.T) {
	handler := registryhttp.NewHealthHandler("local")
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/healthz", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: expected 405, got %d", method, response.Code)
		}
	}
}

func TestStaticServesIndexAssetsAndSPAFallback(t *testing.T) {
	handler := registryhttp.NewStaticHandler(builtWorkspace(t))
	cases := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/", http.StatusOK, "workspace"},
		{"/index.html", http.StatusOK, "workspace"},
		{"/assets/app.js", http.StatusOK, "console.log"},
		{"/projects/order-operations-portal", http.StatusOK, "workspace"}, // client route → index
		{"/assets/missing.js", http.StatusNotFound, "ASSET_NOT_FOUND"},
	}
	for _, tc := range cases {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if response.Code != tc.wantStatus {
			t.Fatalf("%s: expected %d, got %d (%s)", tc.path, tc.wantStatus, response.Code, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), tc.wantBody) {
			t.Fatalf("%s: body %q does not contain %q", tc.path, response.Body.String(), tc.wantBody)
		}
	}
}

func TestStaticNeverEscapesWorkspaceRoot(t *testing.T) {
	handler := registryhttp.NewStaticHandler(builtWorkspace(t))
	for _, path := range []string{"/../go.mod", "/assets/../../go.mod", "/%2e%2e/go.mod"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if strings.Contains(response.Body.String(), "module context-rail") {
			t.Fatalf("%s: served a file outside the workspace root", path)
		}
		if response.Code != http.StatusNotFound && response.Code != http.StatusOK {
			t.Fatalf("%s: unexpected status %d", path, response.Code)
		}
	}
}

func TestStaticRejectsMutationMethods(t *testing.T) {
	handler := registryhttp.NewStaticHandler(builtWorkspace(t))
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: expected 405, got %d", method, response.Code)
		}
	}
}

func TestStaticReportsUnbuiltWorkspace(t *testing.T) {
	handler := registryhttp.NewStaticHandler(t.TempDir())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for missing bundle, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "WORKSPACE_NOT_BUILT") {
		t.Fatalf("expected WORKSPACE_NOT_BUILT, got %s", response.Body.String())
	}
}
