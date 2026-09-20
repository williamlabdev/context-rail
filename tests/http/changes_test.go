package http_test

import (
	"net/http"
	"path/filepath"
	"testing"

	"context-rail/internal/change"
	registryhttp "context-rail/internal/http"
	"context-rail/internal/topology"
)

// newGovernanceMux mirrors production wiring for registry + topology + changes.
func newGovernanceMux(t *testing.T) *http.ServeMux {
	t.Helper()
	roots := fixtureRoots(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	topologyStore, err := topology.NewFileStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	changeStore, err := change.NewFileStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	topologyService := topology.NewService(topologyStore, topology.NewRegistrySource(roots))
	changeService := change.NewService(changeStore, change.NewRegistrySource(roots, topologyService), nil)
	mux := http.NewServeMux()
	registryhttp.NewTopologyHandler(topologyService).Register(mux)
	registryhttp.NewChangesHandler(changeService).Register(mux)
	mux.Handle("/v1/", registryhttp.NewServer(roots))
	return mux
}

func TestChangeJourneyOverHTTP(t *testing.T) {
	mux := newGovernanceMux(t)
	base := "/v1/projects/order-operations-portal/changes"
	before := hashTree(t, fixtureRoots(t)[0])

	// Empty ledger.
	response, payload := call(t, mux, http.MethodGet, base, nil)
	if response.Code != http.StatusOK || payload["kind"] != "ChangeList" || len(payload["changes"].([]any)) != 0 {
		t.Fatalf("expected empty ChangeList, got %d %s", response.Code, response.Body.String())
	}

	// UI-07: an under-specified Change is NEEDS_INPUT with owners.
	response, payload = call(t, mux, http.MethodPost, base, map[string]any{
		"reason":  "ops asked for manual review",
		"request": map[string]any{"title": "Manual order review", "objective": "Approve or reject order exceptions with a note", "target_environment_id": "staging"},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	entry := payload["change"].(map[string]any)
	if entry["change_id"] != "CHG-001" || entry["status"] != "NEEDS_INPUT" {
		t.Fatalf("expected CHG-001 NEEDS_INPUT, got %v", entry)
	}
	missing := entry["versions"].([]any)[0].(map[string]any)["evaluation"].(map[string]any)["missing_inputs"].([]any)
	if len(missing) < 3 {
		t.Fatalf("expected missing inputs with owners, got %v", missing)
	}
	// Accepting now is refused with NEEDS_INPUT (422).
	response, payload = call(t, mux, http.MethodPost, base+"/CHG-001/decision", map[string]any{"decision": "ACCEPT", "selected_option": "minimal_reversible_slice", "rationale": "go"})
	if response.Code != http.StatusUnprocessableEntity || payload["code"] != change.CodeNotDecisionReady {
		t.Fatalf("expected 422 NEEDS_INPUT, got %d %s", response.Code, response.Body.String())
	}

	// UI-08: supply inputs → v2 DECISION_READY.
	response, payload = call(t, mux, http.MethodPost, base+"/CHG-001/inputs", map[string]any{
		"reason":               "requester and business owner supplied inputs",
		"acceptance_criteria":  []map[string]string{{"text": "approve and reject require a note"}, {"text": "unknown order returns 404"}},
		"allowed_paths":        []string{"main.go", "web/index.html", "main_test.go"},
		"business_constraints": map[string]string{"data_classification": "internal", "expected_monthly_volume": "500 reviews"},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("inputs: %d %s", response.Code, response.Body.String())
	}
	entry = payload["change"].(map[string]any)
	if entry["current_version"] != float64(2) || entry["status"] != "DECISION_READY" {
		t.Fatalf("expected v2 DECISION_READY, got %v", entry["status"])
	}

	// UI-06/09: human accepts; brief and pack share lineage.
	response, payload = call(t, mux, http.MethodPost, base+"/CHG-001/decision", map[string]any{
		"decision": "ACCEPT", "role": "solution_architect", "selected_option": "minimal_reversible_slice", "rationale": "smallest reversible slice", "risk_level": "low",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("decide: %d %s", response.Code, response.Body.String())
	}
	decision := payload["decision"].(map[string]any)
	brief := payload["brief"].(map[string]any)
	pack := payload["agent_context_pack"].(map[string]any)
	if decision["decision_id"] != "DEC-001" || decision["status"] != "ACCEPTED_FOR_DEVELOPMENT" {
		t.Fatalf("unexpected decision %v", decision)
	}
	if decision["human_decision"].(map[string]any)["actor"] != "operator-http" {
		t.Fatal("actor header must be recorded on the human decision")
	}
	briefLineage := brief["lineage"].(map[string]any)
	packLineage := pack["lineage"].(map[string]any)
	for _, key := range []string{"decision_id", "decision_version", "source_snapshot_hash", "topology_version", "policy_version"} {
		if briefLineage[key] != packLineage[key] {
			t.Fatalf("lineage %s differs: %v vs %v", key, briefLineage[key], packLineage[key])
		}
	}
	if packLineage["source_snapshot_hash"] != decision["source_snapshot_hash"] {
		t.Fatal("pack must carry the record's snapshot hash")
	}

	// Work order compiles with a hash and the same lineage.
	response, payload = call(t, mux, http.MethodPost, base+"/CHG-001/work-order", map[string]any{"reason": "hand to agent", "issuer": "founder-001"})
	if response.Code != http.StatusOK {
		t.Fatalf("work-order: %d %s", response.Code, response.Body.String())
	}
	order := payload["work_order"].(map[string]any)
	if order["work_order_id"] != "AWO-001" || order["status"] != "ISSUED" || order["work_order_hash"] == "" {
		t.Fatalf("unexpected work order %v", order)
	}
	if order["lineage"].(map[string]any)["decision_id"] != "DEC-001" {
		t.Fatal("work order lineage must point at DEC-001")
	}

	// A material topology change makes everything STALE and blocks reissue.
	response, _ = call(t, mux, http.MethodPatch, "/v1/projects/order-operations-portal/environments/staging", map[string]any{
		"expected_version": 1, "reason": "move staging service", "target_ref": "cloud-run/oop-staging-2",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("topology edit: %d %s", response.Code, response.Body.String())
	}
	response, payload = call(t, mux, http.MethodGet, base+"/CHG-001", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("get: %d", response.Code)
	}
	if payload["staleness"].(map[string]any)["stale"] != true || payload["change"].(map[string]any)["status"] != "STALE" || payload["work_order"].(map[string]any)["status"] != "STALE" {
		t.Fatalf("expected STALE after material topology change: %s", response.Body.String())
	}
	response, payload = call(t, mux, http.MethodPost, base+"/CHG-001/work-order", map[string]any{"reason": "reissue"})
	if response.Code != http.StatusConflict || payload["code"] != change.CodeDecisionStale {
		t.Fatalf("expected 409 DECISION_STALE, got %d %s", response.Code, response.Body.String())
	}

	if after := hashTree(t, fixtureRoots(t)[0]); after != before {
		t.Fatal("change ledger operations must never modify the consumer Project fixture")
	}
}

func TestChangeErrorMapping(t *testing.T) {
	mux := newGovernanceMux(t)
	cases := []struct {
		name   string
		method string
		path   string
		body   any
		status int
		code   string
	}{
		{"unknown project", http.MethodGet, "/v1/projects/nope/changes", nil, http.StatusNotFound, change.CodeProjectNotFound},
		{"unknown change", http.MethodGet, "/v1/projects/order-operations-portal/changes/CHG-999", nil, http.StatusNotFound, change.CodeChangeNotFound},
		{"missing reason", http.MethodPost, "/v1/projects/order-operations-portal/changes", map[string]any{"request": map[string]any{"title": "x"}}, http.StatusUnprocessableEntity, change.CodeReasonRequired},
		{"missing title", http.MethodPost, "/v1/projects/order-operations-portal/changes", map[string]any{"reason": "x", "request": map[string]any{"objective": "y"}}, http.StatusUnprocessableEntity, change.CodeInvalidRequest},
		{"unknown field", http.MethodPost, "/v1/projects/order-operations-portal/changes", map[string]any{"reason": "x", "request": map[string]any{"title": "t"}, "auto_approve": true}, http.StatusBadRequest, "INVALID_MUTATION_BODY"},
		{"delete not allowed", http.MethodDelete, "/v1/projects/order-operations-portal/changes/CHG-001", nil, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
	}
	for _, tc := range cases {
		response, payload := call(t, mux, tc.method, tc.path, tc.body)
		if response.Code != tc.status || payload["code"] != tc.code {
			t.Fatalf("%s: expected %d %s, got %d %s", tc.name, tc.status, tc.code, response.Code, response.Body.String())
		}
	}
}
