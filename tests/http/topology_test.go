package http_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	registryhttp "context-rail/internal/http"
	"context-rail/internal/topology"
)

// newWorkspaceMux mirrors the production wiring: topology routes first, the
// read-only registry under /v1/, both against the same explicit fixture roots.
func newWorkspaceMux(t *testing.T) *http.ServeMux {
	t.Helper()
	roots := fixtureRoots(t)
	store, err := topology.NewFileStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registryhttp.NewTopologyHandler(topology.NewService(store, topology.NewRegistrySource(roots))).Register(mux)
	mux.Handle("/v1/", registryhttp.NewServer(roots))
	return mux
}

func call(t *testing.T, mux *http.ServeMux, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set(registryhttp.ActorHeader, "operator-http")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	var payload map[string]any
	if response.Body.Len() > 0 {
		_ = json.Unmarshal(response.Body.Bytes(), &payload)
	}
	return response, payload
}

func hashTree(t *testing.T, root string) string {
	t.Helper()
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash.Write([]byte(path))
		hash.Write(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func TestTopologyGetBootstrapsFromManifestAndRegistryStaysReadable(t *testing.T) {
	mux := newWorkspaceMux(t)
	response, payload := call(t, mux, http.MethodGet, "/v1/projects/order-operations-portal/environments", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	if payload["kind"] != topology.Kind || payload["current_version"] != float64(1) {
		t.Fatalf("expected bootstrap v1 state, got %v", payload)
	}
	versions := payload["versions"].([]any)
	environments := versions[0].(map[string]any)["environments"].([]any)
	if len(environments) != 4 {
		t.Fatalf("expected 4 manifest environments, got %d", len(environments))
	}
	// Registry routes still resolve through the same mux.
	response, _ = call(t, mux, http.MethodGet, "/v1/projects", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("registry list broke: %d", response.Code)
	}
	response, _ = call(t, mux, http.MethodGet, "/v1/projects/order-operations-portal", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("registry detail broke: %d", response.Code)
	}
}

func TestTopologyMutationJourneyDoesNotTouchConsumerSources(t *testing.T) {
	mux := newWorkspaceMux(t)
	before := hashTree(t, fixtureRoots(t)[0])

	// add
	response, payload := call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/environments", map[string]any{
		"expected_version": 1, "reason": "add uat", "id": "uat", "type": "uat", "sequence": 3, "target_ref": "cloud-run/oop-uat", "owner": "qa",
	})
	if response.Code != http.StatusOK || payload["current_version"] != float64(2) {
		t.Fatalf("add: expected 200 v2, got %d %s", response.Code, response.Body.String())
	}
	audit := payload["audit"].([]any)
	if audit[len(audit)-1].(map[string]any)["actor"] != "operator-http" {
		t.Fatalf("actor header must be recorded: %v", audit[len(audit)-1])
	}
	// stale expected_version → 409
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/environments", map[string]any{
		"expected_version": 1, "reason": "dup", "id": "qa", "type": "other",
	})
	if response.Code != http.StatusConflict || payload["code"] != topology.CodeVersionConflict {
		t.Fatalf("expected 409 TOPOLOGY_VERSION_CONFLICT, got %d %s", response.Code, response.Body.String())
	}
	// edit (material)
	response, payload = call(t, mux, http.MethodPatch, "/v1/projects/order-operations-portal/environments/staging", map[string]any{
		"expected_version": 2, "reason": "new staging service", "target_ref": "cloud-run/oop-staging-2",
	})
	if response.Code != http.StatusOK || payload["current_version"] != float64(3) {
		t.Fatalf("edit: expected 200 v3, got %d %s", response.Code, response.Body.String())
	}
	if invalidations := payload["invalidations"].([]any); len(invalidations) == 0 {
		t.Fatal("material edit must publish invalidations")
	}
	// reorder
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/environments/reorder", map[string]any{
		"expected_version": 3, "reason": "uat after staging", "order": []string{"development", "testing", "staging", "uat", "production"},
	})
	if response.Code != http.StatusOK || payload["current_version"] != float64(4) {
		t.Fatalf("reorder: expected 200 v4, got %d %s", response.Code, response.Body.String())
	}
	// retire + restore
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/environments/uat/retire", map[string]any{"expected_version": 4, "reason": "uat merged into staging"})
	if response.Code != http.StatusOK || payload["current_version"] != float64(5) {
		t.Fatalf("retire: expected 200 v5, got %d %s", response.Code, response.Body.String())
	}
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/environments/production/retire", map[string]any{"expected_version": 5, "reason": "no"})
	if response.Code != http.StatusConflict || payload["code"] != topology.CodeProductionProtected {
		t.Fatalf("production retire must be 409 PRODUCTION_PROTECTED, got %d %s", response.Code, response.Body.String())
	}
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/environments/uat/restore", map[string]any{"expected_version": 5, "reason": "uat needed again"})
	if response.Code != http.StatusOK || payload["current_version"] != float64(6) {
		t.Fatalf("restore: expected 200 v6, got %d %s", response.Code, response.Body.String())
	}

	after := hashTree(t, fixtureRoots(t)[0])
	if before != after {
		t.Fatal("topology mutations must never modify the consumer Project fixture")
	}
	// Registry detail still reports the manifest environments, not the
	// governance topology; the two are separate surfaces.
	response, payload = call(t, mux, http.MethodGet, "/v1/projects/order-operations-portal", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("registry detail broke after mutations: %d", response.Code)
	}
	entry := payload["projects"].([]any)[0].(map[string]any)
	if len(entry["environments"].([]any)) != 4 {
		t.Fatal("registry must keep reporting the manifest's 4 environments")
	}
}

func TestTopologyErrorMapping(t *testing.T) {
	mux := newWorkspaceMux(t)
	cases := []struct {
		name   string
		method string
		path   string
		body   any
		status int
		code   string
	}{
		{"unknown project", http.MethodGet, "/v1/projects/nope/environments", nil, http.StatusNotFound, topology.CodeProjectNotFound},
		{"unknown environment", http.MethodPatch, "/v1/projects/order-operations-portal/environments/ghost", map[string]any{"expected_version": 1, "reason": "x", "owner": "y"}, http.StatusNotFound, topology.CodeEnvironmentNotFound},
		{"missing reason", http.MethodPost, "/v1/projects/order-operations-portal/environments", map[string]any{"expected_version": 1, "id": "qa", "type": "other"}, http.StatusUnprocessableEntity, topology.CodeReasonRequired},
		{"bad order", http.MethodPost, "/v1/projects/order-operations-portal/environments/reorder", map[string]any{"expected_version": 1, "reason": "x", "order": []string{"staging"}}, http.StatusUnprocessableEntity, topology.CodeInvalidOrder},
		{"unknown field", http.MethodPost, "/v1/projects/order-operations-portal/environments", map[string]any{"expected_version": 1, "reason": "x", "id": "qa", "type": "other", "hard_delete": true}, http.StatusBadRequest, "INVALID_MUTATION_BODY"},
		{"delete not allowed", http.MethodDelete, "/v1/projects/order-operations-portal/environments/staging", nil, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
		{"put not allowed", http.MethodPut, "/v1/projects/order-operations-portal/environments", nil, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
	}
	for _, tc := range cases {
		response, payload := call(t, mux, tc.method, tc.path, tc.body)
		if response.Code != tc.status || payload["code"] != tc.code {
			t.Fatalf("%s: expected %d %s, got %d %s", tc.name, tc.status, tc.code, response.Code, response.Body.String())
		}
	}
}
