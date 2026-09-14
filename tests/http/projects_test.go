package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	registryhttp "context-rail/internal/http"
	"context-rail/internal/projectregistry"
)

func workspaceRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func fixtureRoots(t *testing.T) []string {
	root := workspaceRoot(t)
	return []string{
		filepath.Join(root, "demo", "order-operations-portal"),
		filepath.Join(root, "examples", "support-insights"),
	}
}

func decodeSnapshot(t *testing.T, response *httptest.ResponseRecorder) projectregistry.Snapshot {
	t.Helper()
	var snapshot projectregistry.Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestProjectsListReturnsReadOnlySnapshot(t *testing.T) {
	handler := registryhttp.NewServer(fixtureRoots(t))
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q", got)
	}
	snapshot := decodeSnapshot(t, response)
	if snapshot.Kind != "ProjectRegistrySnapshot" || snapshot.SchemaVersion != "project-registry/v1" {
		t.Fatalf("unexpected snapshot header: %#v", snapshot)
	}
	if len(snapshot.Projects) != 2 || !snapshot.Projects[0].ReadOnly || !snapshot.Projects[1].ReadOnly {
		t.Fatalf("unexpected projects: %#v", snapshot.Projects)
	}
}

func TestProjectDetailReturnsIndependentRecord(t *testing.T) {
	handler := registryhttp.NewServer(fixtureRoots(t))
	request := httptest.NewRequest(http.MethodGet, "/v1/projects/support-insights", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	snapshot := decodeSnapshot(t, response)
	if len(snapshot.Projects) != 1 || snapshot.Projects[0].Project.ID != "support-insights" {
		t.Fatalf("unexpected detail response: %#v", snapshot.Projects)
	}
	if snapshot.Projects[0].Context.Status != "STALE" || snapshot.Projects[0].Services[0].Runtime != "UNDECLARED" {
		t.Fatalf("uncertain status was not preserved: %#v", snapshot.Projects[0])
	}
}

func TestProjectsListPreservesAllGovernedStatuses(t *testing.T) {
	root := t.TempDir()
	manifest := `apiVersion: context-rail/v1alpha1
kind: Project
metadata:
  id: status-fixture
  name: Status Fixture
spec:
  documents:
    - path: current.md
      kind: source
      status: CURRENT
    - path: stale.md
      kind: source
      status: STALE
    - path: missing.md
      kind: source
      status: MISSING
    - path: conflict.md
      kind: source
      status: CONFLICT
    - path: unknown.md
      kind: source
      status: UNKNOWN
    - path: undeclared.md
      kind: source
      status: UNDECLARED
`
	if err := os.WriteFile(filepath.Join(root, "project.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := registryhttp.NewServer([]string{root})
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	snapshot := decodeSnapshot(t, response)
	if len(snapshot.Projects) != 1 {
		t.Fatalf("project count = %d", len(snapshot.Projects))
	}
	statuses := map[string]string{}
	for _, document := range snapshot.Projects[0].Documents {
		statuses[document.Path] = document.Status
	}
	for path, expected := range map[string]string{
		"current.md": "CURRENT", "stale.md": "STALE", "missing.md": "MISSING",
		"conflict.md": "CONFLICT", "unknown.md": "UNKNOWN", "undeclared.md": "UNDECLARED",
	} {
		if statuses[path] != expected {
			t.Fatalf("document %s status = %q, want %q", path, statuses[path], expected)
		}
	}
}

func TestLocalFixtureScenariosExposeExplicitEmptyAndErrors(t *testing.T) {
	testCases := []struct {
		name       string
		scenario   string
		statusCode int
		code       string
	}{
		{name: "empty", scenario: "empty", statusCode: http.StatusOK},
		{name: "invalid", scenario: "invalid", statusCode: http.StatusUnprocessableEntity, code: "PROJECT_INVALID"},
		{name: "unavailable", scenario: "unavailable", statusCode: http.StatusServiceUnavailable, code: "PROJECT_UNAVAILABLE"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := registryhttp.NewServerWithOptions(fixtureRoots(t), registryhttp.ServerOptions{Scenario: testCase.scenario})
			request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.statusCode {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if testCase.code == "" {
				snapshot := decodeSnapshot(t, response)
				if len(snapshot.Projects) != 0 {
					t.Fatalf("project count = %d", len(snapshot.Projects))
				}
				return
			}
			assertError(t, response, testCase.statusCode, testCase.code)
		})
	}
}

func TestLocalFixtureScenarioCanDelayResponses(t *testing.T) {
	handler := registryhttp.NewServerWithOptions(fixtureRoots(t), registryhttp.ServerOptions{Delay: 25 * time.Millisecond})
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	response := httptest.NewRecorder()
	started := time.Now()
	handler.ServeHTTP(response, request)
	if elapsed := time.Since(started); elapsed < 25*time.Millisecond {
		t.Fatalf("response returned after %s; want at least 25ms", elapsed)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestUnknownProjectReturnsNotFoundError(t *testing.T) {
	handler := registryhttp.NewServer(fixtureRoots(t))
	request := httptest.NewRequest(http.MethodGet, "/v1/projects/not-configured", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertError(t, response, http.StatusNotFound, "PROJECT_NOT_FOUND")
}

func TestInvalidProjectReturnsInvalidError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.yaml"), []byte("kind: NotAProject\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := registryhttp.NewServer([]string{root})
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertError(t, response, http.StatusUnprocessableEntity, "PROJECT_INVALID")
}

func TestUnavailableProjectReturnsUnavailableError(t *testing.T) {
	handler := registryhttp.NewServer([]string{filepath.Join(t.TempDir(), "missing")})
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertError(t, response, http.StatusServiceUnavailable, "PROJECT_UNAVAILABLE")
}

func TestEmptyRegistryReturnsUnavailableError(t *testing.T) {
	handler := registryhttp.NewServer(nil)
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertError(t, response, http.StatusServiceUnavailable, "REGISTRY_UNAVAILABLE")
}

func TestWriteMethodsAreNotRoutes(t *testing.T) {
	handler := registryhttp.NewServer(fixtureRoots(t))
	request := httptest.NewRequest(http.MethodPost, "/v1/projects", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertError(t, response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var value struct {
		Kind     string `json:"kind"`
		Code     string `json:"code"`
		ReadOnly bool   `json:"read_only"`
	}
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if value.Kind != "ProjectRegistryError" || value.Code != code || !value.ReadOnly {
		t.Fatalf("unexpected error: %#v", value)
	}
}
