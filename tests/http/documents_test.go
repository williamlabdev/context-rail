package http_test

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"context-rail/internal/change"
	"context-rail/internal/document"
	registryhttp "context-rail/internal/http"
	"context-rail/internal/topology"
)

// newDocumentsMux wires topology + documents + changes the way main does:
// the document baseline overlays the change facts and the change ledger
// feeds decision refs back into the Context Pack.
func newDocumentsMux(t *testing.T) *http.ServeMux {
	t.Helper()
	roots := fixtureRoots(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	topologyStore, _ := topology.NewFileStore(stateDir)
	changeStore, _ := change.NewFileStore(stateDir)
	documentStore, _ := document.NewFileStore(stateDir)
	topologyService := topology.NewService(topologyStore, topology.NewRegistrySource(roots))
	documentService := document.NewService(documentStore, document.NewRegistrySource(roots)).WithTopology(topologyService)
	changeService := change.NewService(changeStore, change.NewRegistrySource(roots, topologyService).WithDocuments(documentService), nil)
	documentService.WithDecisions(changeService)
	mux := http.NewServeMux()
	registryhttp.NewTopologyHandler(topologyService).Register(mux)
	registryhttp.NewDocumentsHandler(documentService).Register(mux)
	registryhttp.NewChangesHandler(changeService).Register(mux)
	mux.Handle("/v1/", registryhttp.NewServer(roots))
	return mux
}

func TestDocumentBaselineAndContextPackOverHTTP(t *testing.T) {
	mux := newDocumentsMux(t)
	root := fixtureRoots(t)[0]
	before := hashTree(t, root)
	documents := "/v1/projects/order-operations-portal/documents"

	response, payload := call(t, mux, http.MethodGet, documents, nil)
	if response.Code != http.StatusOK || payload["current_version"] != float64(1) || payload["live_context_status"] != "DERIVED" {
		t.Fatalf("GET documents: %d %s", response.Code, response.Body.String())
	}
	readiness := payload["readiness"].(map[string]any)
	if readiness["decision"].(map[string]any)["status"] != "READY" {
		t.Fatalf("fixture documents are all present: %v", readiness)
	}

	// Declare a missing runbook required for decision and staging.
	response, payload = call(t, mux, http.MethodPost, documents, map[string]any{"expected_version": 1, "actor": "architect", "reason": "staging needs a runbook", "path": "docs/operations/runbook.md", "kind": "runbook", "required_for": []string{"decision", "staging"}})
	if response.Code != http.StatusOK || payload["current_version"] != float64(2) {
		t.Fatalf("declare: %d %s", response.Code, response.Body.String())
	}
	readiness = payload["readiness"].(map[string]any)
	if readiness["decision"].(map[string]any)["status"] != "NEEDS_INPUT" || readiness["staging"].(map[string]any)["status"] != "NEEDS_INPUT" || readiness["development"].(map[string]any)["status"] != "READY" {
		t.Fatalf("missing runbook must block decision and staging only: %v", readiness)
	}
	// Stale version is refused.
	response, _ = call(t, mux, http.MethodPost, documents, map[string]any{"expected_version": 1, "actor": "architect", "reason": "again", "path": "docs/x.md", "required_for": []string{"decision"}})
	if response.Code != http.StatusConflict {
		t.Fatalf("stale expected_version must be 409: %d %s", response.Code, response.Body.String())
	}

	// A new Change now needs input: the missing document is reported, never filled in.
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/changes", map[string]any{"reason": "open", "request": map[string]any{
		"title": "attachments", "objective": "add attachments", "owner_summary": "Reviewers can attach files to an order exception.", "acceptance_criteria": []map[string]string{{"text": "works"}}, "allowed_paths": []string{"main.go"},
		"target_environment_id": "staging", "business_constraints": map[string]string{"data_classification": "internal", "expected_monthly_volume": "100"},
	}})
	if response.Code != http.StatusOK {
		t.Fatalf("create change: %d %s", response.Code, response.Body.String())
	}
	changeView := payload["change"].(map[string]any)
	if changeView["status"] != "NEEDS_INPUT" {
		t.Fatalf("change must be NEEDS_INPUT while a decision document is missing: %v", changeView["status"])
	}
	found := false
	versions := changeView["versions"].([]any)
	evaluation := versions[len(versions)-1].(map[string]any)["evaluation"].(map[string]any)
	for _, entry := range evaluation["missing_inputs"].([]any) {
		item := entry.(map[string]any)
		if item["field"] == "decision_documents" && strings.Contains(item["reason"].(string), "runbook.md is MISSING") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing input must name the runbook: %s", response.Body.String())
	}

	// Rebuild the Context Pack: PARTIAL, missing listed, nothing synthesized.
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/context-pack/rebuild", map[string]any{"actor": "architect", "reason": "rebuild with the gap"})
	if response.Code != http.StatusOK {
		t.Fatalf("rebuild: %d %s", response.Code, response.Body.String())
	}
	pack := payload["latest_pack"].(map[string]any)
	if pack["pack_id"] != "CP-001" || pack["status"] != "PARTIAL" || payload["live_context_status"] != "PARTIAL" || len(pack["missing"].([]any)) != 1 {
		t.Fatalf("expected CP-001 PARTIAL: %s", response.Body.String())
	}
	response, packPayload := call(t, mux, http.MethodGet, "/v1/projects/order-operations-portal/context-pack/CP-001", nil)
	if response.Code != http.StatusOK || packPayload["source_snapshot_hash"] == "" || packPayload["pack_hash"] == "" {
		t.Fatalf("pack must be re-readable with its hashes: %d %s", response.Code, response.Body.String())
	}

	// Withdraw the declaration and rebuild: DERIVED again; the Change is DECISION_READY on re-evaluation.
	response, payload = call(t, mux, http.MethodPost, documents+"/withdraw", map[string]any{"expected_version": 2, "actor": "architect", "reason": "runbook deferred to P1", "path": "docs/operations/runbook.md"})
	if response.Code != http.StatusOK || payload["current_version"] != float64(3) {
		t.Fatalf("withdraw: %d %s", response.Code, response.Body.String())
	}
	response, _ = call(t, mux, http.MethodPost, documents+"/withdraw", map[string]any{"expected_version": 3, "actor": "architect", "reason": "no", "path": "project.yaml"})
	if response.Code != http.StatusConflict {
		t.Fatalf("manifest documents are protected: %d %s", response.Code, response.Body.String())
	}
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/context-pack/rebuild", map[string]any{"actor": "architect", "reason": "runbook withdrawn"})
	if response.Code != http.StatusOK || payload["latest_pack"].(map[string]any)["status"] != "DERIVED" || payload["live_context_status"] != "DERIVED" {
		t.Fatalf("rebuild after withdraw: %d %s", response.Code, response.Body.String())
	}
	// Supplying inputs re-evaluates the Change against the live baseline: no missing document any more.
	response, payload = call(t, mux, http.MethodPost, "/v1/projects/order-operations-portal/changes/CHG-001/inputs", map[string]any{"reason": "re-evaluate after the runbook was withdrawn", "objective": "add attachments (re-evaluated)"})
	if response.Code != http.StatusOK || payload["change"].(map[string]any)["status"] == "NEEDS_INPUT" {
		t.Fatalf("change must leave NEEDS_INPUT once the document is no longer required: %d %s", response.Code, response.Body.String())
	}
	if after := hashTree(t, root); after != before {
		t.Fatal("document governance must never write into the consumer Project")
	}
	response, _ = call(t, mux, http.MethodDelete, documents, nil)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE must be 405: %d", response.Code)
	}
}
