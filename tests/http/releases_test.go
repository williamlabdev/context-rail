package http_test

import (
	"net/http"
	"path/filepath"
	"testing"

	"context-rail/internal/change"
	registryhttp "context-rail/internal/http"
	"context-rail/internal/release"
	"context-rail/internal/topology"
)

func newFullMux(t *testing.T) *http.ServeMux {
	t.Helper()
	roots := fixtureRoots(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	topologyStore, _ := topology.NewFileStore(stateDir)
	changeStore, _ := change.NewFileStore(stateDir)
	releaseStore, _ := release.NewFileStore(stateDir)
	topologyService := topology.NewService(topologyStore, topology.NewRegistrySource(roots))
	changeService := change.NewService(changeStore, change.NewRegistrySource(roots, topologyService), nil)
	mux := http.NewServeMux()
	registryhttp.NewTopologyHandler(topologyService).Register(mux)
	registryhttp.NewChangesHandler(changeService).Register(mux)
	registryhttp.NewReleasesHandler(release.NewService(releaseStore, changeService, topologyService)).Register(mux)
	mux.Handle("/v1/", registryhttp.NewServer(roots))
	return mux
}

func TestPromotionJourneyOverHTTP(t *testing.T) {
	mux := newFullMux(t)
	order := issueWorkOrderOverHTTP(t, mux)
	before := hashTree(t, fixtureRoots(t)[0])
	candidates := "/v1/projects/order-operations-portal/changes/CHG-001/candidates"
	run := map[string]any{"run_id": "ARR-001", "work_order_id": order["work_order_id"], "work_order_hash": order["work_order_hash"], "started_by": "dev-1", "started_at": "2026-09-21T02:00:00Z"}
	response, _ := call(t, mux, http.MethodPost, candidates, map[string]any{"reason": "done", "run": run, "observation": map[string]any{
		"branch": order["target_branch"], "base_branch": "main", "head_commit": "abc123", "changed_paths": []string{"main.go"},
		"checks":  []map[string]string{{"name": "test", "status": "PASS"}, {"name": "build", "status": "PASS"}},
		"reviews": []map[string]string{{"reviewer": "reviewer-2", "kind": "human", "verdict": "APPROVED"}},
	}})
	if response.Code != http.StatusOK {
		t.Fatalf("candidate: %s", response.Body.String())
	}
	response, _ = call(t, mux, http.MethodPost, candidates+"/CAND-001/decision", map[string]any{"decision": "ACCEPT", "rationale": "ok"})
	if response.Code != http.StatusOK {
		t.Fatalf("accept candidate: %s", response.Body.String())
	}

	releases := "/v1/projects/order-operations-portal/releases"
	response, payload := call(t, mux, http.MethodPost, releases, map[string]any{"reason": "release", "change_ids": []string{"CHG-001"}, "target_environment_id": "staging"})
	if response.Code != http.StatusOK || payload["release"].(map[string]any)["status"] != release.StatusAwaitingBuild {
		t.Fatalf("create release: %d %s", response.Code, response.Body.String())
	}
	digest := "sha256:4444444444444444444444444444444444444444444444444444444444444444"
	response, payload = call(t, mux, http.MethodPost, releases+"/REL-001/build", map[string]any{"reason": "built", "image_digest": digest, "source_commit": "abc123", "build_id": "b1", "evidence_ref": "cloudbuild/b1"})
	if response.Code != http.StatusOK || payload["release"].(map[string]any)["status"] != release.StatusReadyForApproval {
		t.Fatalf("build: %d %s", response.Code, response.Body.String())
	}
	response, payload = call(t, mux, http.MethodPost, releases+"/REL-001/approval", map[string]any{"decision": "APPROVE", "actor": "approver-1", "role": "release_manager", "rationale": "manifest reviewed"})
	if response.Code != http.StatusOK || payload["release"].(map[string]any)["status"] != release.StatusApproved {
		t.Fatalf("approve: %d %s", response.Code, response.Body.String())
	}
	response, payload = call(t, mux, http.MethodPost, releases+"/REL-001/deployment", map[string]any{
		"reason": "deployed", "idempotency_key": "k1", "operation_id": "op1", "revision": "oop-00003", "service_url": "https://oop.a.run.app",
		"deployed_digest": digest, "deployed_target_ref": "cloud-run/order-operations-portal-staging", "smoke": map[string]string{"status": "PASS", "evidence_ref": "smoke.txt"},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("deployment: %d %s", response.Code, response.Body.String())
	}
	rel := payload["release"].(map[string]any)
	receipt := rel["receipt"].(map[string]any)
	if rel["status"] != release.StatusPromoted || receipt["status"] != release.StatusPromoted || receipt["receipt_hash"] == "" {
		t.Fatalf("expected PROMOTED receipt: %s", response.Body.String())
	}
	if receipt["changes"].([]any)[0].(map[string]any)["decision_id"] != "DEC-001" || receipt["deployment"].(map[string]any)["revision"] != "oop-00003" {
		t.Fatalf("receipt must link decision and revision: %v", receipt)
	}
	// Replay with the same key returns the same receipt; a new key is refused.
	response, payload = call(t, mux, http.MethodPost, releases+"/REL-001/deployment", map[string]any{"reason": "retry", "idempotency_key": "k1", "revision": "oop-00003", "deployed_digest": digest, "deployed_target_ref": "cloud-run/order-operations-portal-staging"})
	if response.Code != http.StatusOK || payload["release"].(map[string]any)["receipt"].(map[string]any)["receipt_id"] != receipt["receipt_id"] {
		t.Fatalf("replay must return the same receipt: %d %s", response.Code, response.Body.String())
	}
	response, payload = call(t, mux, http.MethodPost, releases+"/REL-001/deployment", map[string]any{"reason": "again", "idempotency_key": "k2", "revision": "oop-00004", "deployed_digest": digest, "deployed_target_ref": "cloud-run/order-operations-portal-staging"})
	if response.Code != http.StatusConflict || payload["code"] != release.CodeAlreadyPromoted {
		t.Fatalf("second promotion must be 409 ALREADY_PROMOTED: %d %s", response.Code, response.Body.String())
	}
	if after := hashTree(t, fixtureRoots(t)[0]); after != before {
		t.Fatal("release operations must never modify the consumer Project fixture")
	}
	response, payload = call(t, mux, http.MethodGet, releases, nil)
	if response.Code != http.StatusOK || payload["kind"] != "ReleaseList" || len(payload["releases"].([]any)) != 1 {
		t.Fatalf("list: %d %s", response.Code, response.Body.String())
	}
	response, payload = call(t, mux, http.MethodGet, releases+"/REL-009", nil)
	if response.Code != http.StatusNotFound || payload["code"] != release.CodeReleaseNotFound {
		t.Fatalf("unknown release: %d", response.Code)
	}
}
