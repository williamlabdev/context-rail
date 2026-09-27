package change_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-rail/internal/change"
)

// geminiEnvelope wraps innerJSON exactly the way the real Generative
// Language generateContent response does: candidates[0].content.parts[0].text
// is itself a JSON-encoded string, not a nested object.
func geminiEnvelope(t *testing.T, innerJSON string) []byte {
	t.Helper()
	envelope := map[string]any{
		"candidates": []map[string]any{
			{"content": map[string]any{"parts": []map[string]any{{"text": innerJSON}}}},
		},
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

// fakeGeminiServer is an httptest fake standing in for
// generativelanguage.googleapis.com, following the same httptest.NewServer
// pattern already used elsewhere in this codebase (tests/http/*). GeminiAdvisor
// is not exercised against the real API anywhere in this workspace (no key,
// no egress) — this is the only coverage its request/response handling gets.
func fakeGeminiServer(t *testing.T, innerJSON string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(geminiEnvelope(t, innerJSON))
	}))
	t.Cleanup(server.Close)
	return server
}

func geminiAdvisorAt(server *httptest.Server) *change.GeminiAdvisor {
	return &change.GeminiAdvisor{APIKey: "fake-key", Model: "gemini-test", Endpoint: server.URL, Client: server.Client()}
}

func geminiFacts() *change.ProjectFacts {
	return &change.ProjectFacts{ProjectID: "demo", ContextStatus: "CURRENT"}
}

// TestGeminiAdvisorPricingRefEstimated confirms a Gemini candidate naming a
// valid pricing_ref (one costableOptions actually prices) reaches
// EstimateOptionCost and comes back ESTIMATED — priced via pricing_ref, not
// via the candidate's own free-form id.
func TestGeminiAdvisorPricingRefEstimated(t *testing.T) {
	inner := `{"options":[{"id":"gemini_storage_candidate","title":"Gemini storage option","summary":"s","cost_drivers":["download egress"],"risks":["r"],"recommended":true,"pricing_ref":"managed_storage_with_signed_urls"}],"unknowns":[]}`
	advisor := geminiAdvisorAt(fakeGeminiServer(t, inner))
	options, _, err := advisor.Propose(completeRequest(), geminiFacts())
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 1 || options[0].PricingRef != "managed_storage_with_signed_urls" {
		t.Fatalf("expected pricing_ref to survive Propose, got %+v", options)
	}
	estimate := change.EstimateOptionCost(options[0].ID, options[0].PricingRef, storageConstraints("500 GB"))
	if estimate.Status != change.CostEstimated {
		t.Fatalf("expected ESTIMATED via pricing_ref, got %s: %+v", estimate.Status, estimate)
	}
}

// TestGeminiAdvisorInvalidPricingRefIsUnknown confirms a pricing_ref value
// outside the fixed choice list is UNKNOWN with a reason naming the bad
// value, never a fuzzy/best-effort match — covers a model that ignores the
// prompt/response-schema constraint and returns a value not on the list.
func TestGeminiAdvisorInvalidPricingRefIsUnknown(t *testing.T) {
	inner := `{"options":[{"id":"gemini_storage_candidate","title":"t","summary":"s","cost_drivers":[],"risks":[],"recommended":false,"pricing_ref":"totally_bogus_sku"}],"unknowns":[]}`
	advisor := geminiAdvisorAt(fakeGeminiServer(t, inner))
	options, _, err := advisor.Propose(completeRequest(), geminiFacts())
	if err != nil {
		t.Fatal(err)
	}
	estimate := change.EstimateOptionCost(options[0].ID, options[0].PricingRef, storageConstraints("500 GB"))
	if estimate.Status != change.CostUnknown {
		t.Fatalf("expected UNKNOWN for an invalid pricing_ref, got %+v", estimate)
	}
	if !strings.Contains(estimate.Reason, `pricing_ref "totally_bogus_sku" is not a priced SKU`) {
		t.Fatalf("expected the reason to name the bad pricing_ref, got %q", estimate.Reason)
	}
}

// TestGeminiAdvisorMissingPricingRefIsUnknown confirms a Gemini candidate
// with no pricing_ref field at all (the model omitted it, defeating the
// prompt/response-schema constraint) is UNKNOWN, not silently priced by
// matching its own (possibly advisor.go-rewritten) id.
func TestGeminiAdvisorMissingPricingRefIsUnknown(t *testing.T) {
	inner := `{"options":[{"title":"t","summary":"s","cost_drivers":[],"risks":[],"recommended":false}],"unknowns":[]}`
	advisor := geminiAdvisorAt(fakeGeminiServer(t, inner))
	options, _, err := advisor.Propose(completeRequest(), geminiFacts())
	if err != nil {
		t.Fatal(err)
	}
	if options[0].PricingRef != "" {
		t.Fatalf("expected empty pricing_ref when the field is omitted, got %q", options[0].PricingRef)
	}
	if options[0].ID != "gemini_option_1" {
		t.Fatalf("expected the advisor.go:206 id rewrite to still apply when id is also omitted, got %q", options[0].ID)
	}
	estimate := change.EstimateOptionCost(options[0].ID, options[0].PricingRef, storageConstraints("500 GB"))
	if estimate.Status != change.CostUnknown {
		t.Fatalf("expected UNKNOWN when pricing_ref is missing, got %s: %+v", estimate.Status, estimate)
	}
}
