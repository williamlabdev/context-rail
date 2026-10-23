package change_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-rail/internal/change"
)

// anthropicEnvelope wraps innerJSON the way the Messages API does: the
// proposal is the text of the first content block, not a nested object.
func anthropicEnvelope(t *testing.T, innerJSON, stopReason string) []byte {
	t.Helper()
	envelope := map[string]any{
		"type":        "message",
		"role":        "assistant",
		"content":     []map[string]any{{"type": "text", "text": innerJSON}},
		"stop_reason": stopReason,
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

// capturedAnthropicRequest records what the adapter actually sent so the
// tests can assert the wire contract (path, headers, model, output_config)
// rather than only the parsed result.
type capturedAnthropicRequest struct {
	Path    string
	Headers http.Header
	Body    map[string]any
}

// fakeAnthropicServer stands in for api.anthropic.com. As with the Gemini
// fake, this is the only coverage the request/response handling gets in the
// test suite; the single real call is recorded as evidence (EB-016).
func fakeAnthropicServer(t *testing.T, status int, response []byte, captured *capturedAnthropicRequest) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if captured != nil {
			captured.Path = r.URL.Path
			captured.Headers = r.Header.Clone()
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &captured.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(response)
	}))
	t.Cleanup(server.Close)
	return server
}

func anthropicAdvisorAt(server *httptest.Server) *change.AnthropicAdvisor {
	return &change.AnthropicAdvisor{APIKey: "fake-key", Model: "claude-test", Endpoint: server.URL, Client: server.Client()}
}

// TestAnthropicAdvisorWireContract asserts the request the adapter sends:
// POST /v1/messages, key in the x-api-key header only, the version header,
// the configured model and a json_schema output format whose pricing_ref
// enum is the fixed choice list. A successful answer with a valid
// pricing_ref reaches EstimateOptionCost and comes back ESTIMATED.
func TestAnthropicAdvisorWireContract(t *testing.T) {
	inner := `{"options":[{"id":"claude_storage_candidate","title":"Claude storage option","summary":"s","cost_drivers":["download egress"],"risks":["r"],"recommended":true,"pricing_ref":"managed_storage_with_signed_urls"}],"unknowns":["volume unknown"]}`
	var captured capturedAnthropicRequest
	advisor := anthropicAdvisorAt(fakeAnthropicServer(t, http.StatusOK, anthropicEnvelope(t, inner, "end_turn"), &captured))
	options, unknowns, err := advisor.Propose(completeRequest(), geminiFacts())
	if err != nil {
		t.Fatal(err)
	}
	if captured.Path != "/v1/messages" {
		t.Fatalf("expected POST /v1/messages, got %s", captured.Path)
	}
	if captured.Headers.Get("x-api-key") != "fake-key" || captured.Headers.Get("anthropic-version") == "" {
		t.Fatalf("expected x-api-key and anthropic-version headers, got %v", captured.Headers)
	}
	if captured.Body["model"] != "claude-test" {
		t.Fatalf("expected model claude-test in body, got %v", captured.Body["model"])
	}
	format, _ := captured.Body["output_config"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" {
		t.Fatalf("expected output_config.format.type json_schema, got %v", format)
	}
	schemaJSON, _ := json.Marshal(format["schema"])
	if !strings.Contains(string(schemaJSON), `"managed_storage_with_signed_urls"`) || !strings.Contains(string(schemaJSON), `"additionalProperties":false`) {
		t.Fatalf("expected pricing_ref enum and additionalProperties:false in schema, got %s", schemaJSON)
	}
	if len(options) != 1 || options[0].AdvisorSource != "anthropic:claude-test" || options[0].PricingRef != "managed_storage_with_signed_urls" {
		t.Fatalf("expected one option attributed to anthropic:claude-test with pricing_ref, got %+v", options)
	}
	if len(unknowns) == 0 || unknowns[0] != "volume unknown" {
		t.Fatalf("expected model unknowns to survive and come first, got %v", unknowns)
	}
	estimate := change.EstimateOptionCost(options[0].ID, options[0].PricingRef, storageConstraints("500 GB"))
	if estimate.Status != change.CostEstimated {
		t.Fatalf("expected ESTIMATED via pricing_ref, got %s: %+v", estimate.Status, estimate)
	}
}

// TestAnthropicAdvisorHTTPErrorNamesType confirms a non-200 answer is an
// error naming the status and the API error type, so the service observation
// says why the fallback happened without echoing the key or the message.
func TestAnthropicAdvisorHTTPErrorNamesType(t *testing.T) {
	body := []byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	advisor := anthropicAdvisorAt(fakeAnthropicServer(t, http.StatusUnauthorized, body, nil))
	_, _, err := advisor.Propose(completeRequest(), geminiFacts())
	if err == nil {
		t.Fatal("expected an error on HTTP 401")
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "authentication_error") {
		t.Fatalf("expected status and error type in message, got %v", err)
	}
	if strings.Contains(err.Error(), "fake-key") || strings.Contains(err.Error(), "invalid x-api-key") {
		t.Fatalf("error must not echo the key or the provider message, got %v", err)
	}
}

// TestAnthropicAdvisorRefusalIsError confirms stop_reason=refusal is an
// error (so the rule fallback runs) rather than a parse of whatever text came
// back, which the API says may not match the schema.
func TestAnthropicAdvisorRefusalIsError(t *testing.T) {
	advisor := anthropicAdvisorAt(fakeAnthropicServer(t, http.StatusOK, anthropicEnvelope(t, "I cannot help with that.", "refusal"), nil))
	_, _, err := advisor.Propose(completeRequest(), geminiFacts())
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("expected a refusal error, got %v", err)
	}
}

// TestAnthropicAdvisorTruncatedIsError confirms a max_tokens stop is an
// error even when the partial text happens to parse.
func TestAnthropicAdvisorTruncatedIsError(t *testing.T) {
	inner := `{"options":[{"id":"a","title":"t","summary":"s","cost_drivers":[],"risks":[],"recommended":false,"pricing_ref":"none"}],"unknowns":[]}`
	advisor := anthropicAdvisorAt(fakeAnthropicServer(t, http.StatusOK, anthropicEnvelope(t, inner, "max_tokens"), nil))
	_, _, err := advisor.Propose(completeRequest(), geminiFacts())
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected a truncation error, got %v", err)
	}
}

// TestAnthropicAdvisorNoOptionsIsError mirrors the Gemini rule: an answer
// with zero options is a failure, never an empty candidate list.
func TestAnthropicAdvisorNoOptionsIsError(t *testing.T) {
	advisor := anthropicAdvisorAt(fakeAnthropicServer(t, http.StatusOK, anthropicEnvelope(t, `{"options":[],"unknowns":["x"]}`, "end_turn"), nil))
	_, _, err := advisor.Propose(completeRequest(), geminiFacts())
	if err == nil {
		t.Fatal("expected an error when no options are returned")
	}
}

func TestNewAnthropicAdvisorDefaultModel(t *testing.T) {
	advisor := change.NewAnthropicAdvisor("k", "")
	if advisor.Model != "claude-opus-5-5" || advisor.Name() != "anthropic:claude-opus-5-5" {
		t.Fatalf("expected default model claude-opus-5-5, got %q / %q", advisor.Model, advisor.Name())
	}
	if change.NewAnthropicAdvisor("k", "claude-custom").Name() != "anthropic:claude-custom" {
		t.Fatal("expected explicit model to be honoured")
	}
}

// TestSelectAdvisor pins the DR-016 selection rule: explicit provider wins
// and needs its key; otherwise exactly one key picks the provider, none means
// rule-advisor, and both keys without a choice is a blocked start.
func TestSelectAdvisor(t *testing.T) {
	cases := []struct {
		name      string
		config    change.AdvisorConfig
		wantLabel string
		wantErr   string
	}{
		{"no keys", change.AdvisorConfig{}, "rule-advisor", ""},
		{"gemini key only", change.AdvisorConfig{GeminiKey: "g"}, "gemini:gemini-3.6-flash (falls back to rule-advisor on error)", ""},
		{"anthropic key only", change.AdvisorConfig{AnthropicKey: "a"}, "anthropic:claude-opus-5-5 (falls back to rule-advisor on error)", ""},
		{"both keys, no choice", change.AdvisorConfig{GeminiKey: "g", AnthropicKey: "a"}, "", "CONTEXT_RAIL_ADVISOR"},
		{"both keys, anthropic chosen", change.AdvisorConfig{Provider: "anthropic", GeminiKey: "g", AnthropicKey: "a", AnthropicModel: "claude-x"}, "anthropic:claude-x (falls back to rule-advisor on error)", ""},
		{"both keys, gemini chosen", change.AdvisorConfig{Provider: "gemini", GeminiKey: "g", AnthropicKey: "a"}, "gemini:gemini-3.6-flash (falls back to rule-advisor on error)", ""},
		{"explicit rule ignores keys", change.AdvisorConfig{Provider: "rule", GeminiKey: "g", AnthropicKey: "a"}, "rule-advisor", ""},
		{"anthropic chosen without key", change.AdvisorConfig{Provider: "anthropic", GeminiKey: "g"}, "", "ANTHROPIC_API_KEY is empty"},
		{"gemini chosen without key", change.AdvisorConfig{Provider: "gemini", AnthropicKey: "a"}, "", "GEMINI_API_KEY is empty"},
		{"unknown provider", change.AdvisorConfig{Provider: "openai"}, "", "unknown CONTEXT_RAIL_ADVISOR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			advisor, label, err := change.SelectAdvisor(tc.config)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if label != tc.wantLabel {
				t.Fatalf("expected label %q, got %q", tc.wantLabel, label)
			}
			if advisor == nil {
				t.Fatal("expected a non-nil advisor")
			}
		})
	}
}
