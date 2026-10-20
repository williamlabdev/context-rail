package change

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Anthropic Messages API constants. The version header is part of the wire
// contract, not a model choice; bump it only when the API changelog says so.
const (
	anthropicDefaultModel    = "claude-opus-5-5"
	anthropicDefaultEndpoint = "https://api.anthropic.com"
	anthropicVersion         = "2023-06-01"
	// anthropicMaxTokens bounds the answer. A 2-3 option proposal with
	// unknowns is a few hundred tokens of JSON; the headroom is for the
	// model, not for us, and a max_tokens stop is treated as a failure below.
	anthropicMaxTokens = 4096
	// anthropicTimeout is longer than Gemini's 30s because Opus-class models
	// think before answering, but it stays under the HTTP server's 60s
	// WriteTimeout (cmd/context-rail/main.go) so that a slow model still
	// leaves time for the rule-advisor fallback to be written to the client.
	// EB-016 measured 13s for one real call.
	anthropicTimeout = 45 * time.Second
	anthropicSystem  = "You are the ContextRail decision advisor. Answer with the strict JSON object the user message describes and nothing else."
)

// AnthropicAdvisor asks Claude (Anthropic Messages API) to propose candidates
// and unknowns as strict JSON. It is a Advisor in the same sense as
// GeminiAdvisor: candidates only, never the decision, and the service falls
// back to RuleAdvisor on any error. DR-016 added it so the project is not
// bound to one provider; both adapters send advisorPrompt verbatim.
type AnthropicAdvisor struct {
	APIKey   string
	Model    string
	Endpoint string
	Client   *http.Client
}

// NewAnthropicAdvisor returns an advisor for the given key and model; an empty
// model selects anthropicDefaultModel.
func NewAnthropicAdvisor(apiKey, model string) *AnthropicAdvisor {
	if model == "" {
		model = anthropicDefaultModel
	}
	return &AnthropicAdvisor{
		APIKey:   apiKey,
		Model:    model,
		Endpoint: anthropicDefaultEndpoint,
		Client:   &http.Client{Timeout: anthropicTimeout},
	}
}

// Name identifies the provider and model so Option.advisor_source and the
// startup line say which model produced the candidates.
func (advisor *AnthropicAdvisor) Name() string {
	return "anthropic:" + advisor.Model
}

// anthropicResponse is the subset of the Messages API response we read.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason  string `json:"stop_reason"`
	StopDetails *struct {
		Category string `json:"category"`
	} `json:"stop_details"`
}

// anthropicError is the body the API sends with a non-2xx status. Only the
// error type is surfaced; the message is kept out of observations so a
// provider-side string can never leak request content into the ledger.
type anthropicError struct {
	Error struct {
		Type string `json:"type"`
	} `json:"error"`
}

// Propose sends the shared advisor prompt to the Messages API with a JSON
// schema output format and returns the parsed proposal. Any deviation from
// the contract (HTTP error, refusal, truncated answer, no options) is an
// error so the service can fall back to RuleAdvisor and say so.
func (advisor *AnthropicAdvisor) Propose(request Request, facts *ProjectFacts) ([]Option, []string, error) {
	if advisor.APIKey == "" {
		return nil, nil, errors.New("ANTHROPIC_API_KEY not configured")
	}
	pricingRefEnum := pricingRefChoices()
	body := map[string]any{
		"model":      advisor.Model,
		"max_tokens": anthropicMaxTokens,
		"system":     anthropicSystem,
		"messages": []map[string]any{
			{"role": "user", "content": advisorPrompt(request, facts, pricingRefEnum)},
		},
		"output_config": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"schema": anthropicResponseSchema(pricingRefEnum),
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	client := advisor.Client
	if client == nil {
		client = &http.Client{Timeout: anthropicTimeout}
	}
	// The key travels only in the header, never the URL or the body: a logged
	// request line or an error message must never contain it.
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(advisor.Endpoint, "/")+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", advisor.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr anthropicError
		_ = json.Unmarshal(raw, &apiErr)
		if apiErr.Error.Type != "" {
			return nil, nil, fmt.Errorf("anthropic returned HTTP %d (%s)", resp.StatusCode, apiErr.Error.Type)
		}
		return nil, nil, fmt.Errorf("anthropic returned HTTP %d", resp.StatusCode)
	}
	var envelope anthropicResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, nil, fmt.Errorf("anthropic response unreadable: %w", err)
	}
	switch envelope.StopReason {
	case "refusal":
		category := "unspecified"
		if envelope.StopDetails != nil && envelope.StopDetails.Category != "" {
			category = envelope.StopDetails.Category
		}
		return nil, nil, fmt.Errorf("anthropic refused the request (%s)", category)
	case "max_tokens":
		return nil, nil, fmt.Errorf("anthropic answer truncated at %d tokens", anthropicMaxTokens)
	}
	var text string
	for _, block := range envelope.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	if strings.TrimSpace(text) == "" {
		return nil, nil, errors.New("anthropic returned no text block")
	}
	var proposal modelProposal
	if err := json.Unmarshal([]byte(text), &proposal); err != nil {
		return nil, nil, fmt.Errorf("anthropic proposal is not valid JSON: %w", err)
	}
	if len(proposal.Options) == 0 {
		return nil, nil, errors.New("anthropic returned no options")
	}
	for i := range proposal.Options {
		proposal.Options[i].AdvisorSource = advisor.Name()
		if proposal.Options[i].ID == "" {
			proposal.Options[i].ID = fmt.Sprintf("anthropic_option_%d", i+1)
		}
	}
	_, ruleUnknowns, _ := RuleAdvisor{}.Propose(request, facts)
	return proposal.Options, mergeUnique(proposal.Unknowns, ruleUnknowns), nil
}

// anthropicResponseSchema is the structured-output schema for the Messages
// API. It mirrors geminiResponseSchema but every object closes with
// additionalProperties:false, which Anthropic requires, and every property is
// required, which it also requires.
func anthropicResponseSchema(pricingRefEnum []string) map[string]any {
	stringList := map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	option := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":           map[string]any{"type": "string"},
			"title":        map[string]any{"type": "string"},
			"summary":      map[string]any{"type": "string"},
			"cost_drivers": stringList,
			"risks":        stringList,
			"recommended":  map[string]any{"type": "boolean"},
			"pricing_ref":  map[string]any{"type": "string", "enum": pricingRefEnum},
		},
		"required":             []string{"id", "title", "summary", "cost_drivers", "risks", "recommended", "pricing_ref"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"options":  map[string]any{"type": "array", "items": option},
			"unknowns": stringList,
		},
		"required":             []string{"options", "unknowns"},
		"additionalProperties": false,
	}
}

// AdvisorConfig is everything the process start-up knows about which advisor
// to run. It is plain data so SelectAdvisor can be tested without env vars.
type AdvisorConfig struct {
	Provider       string // CONTEXT_RAIL_ADVISOR: "", "rule", "gemini" or "anthropic"
	GeminiKey      string
	GeminiModel    string
	AnthropicKey   string
	AnthropicModel string
}

// SelectAdvisor picks the advisor for a process (DR-016). An explicit Provider
// wins and must have its key; with no Provider, exactly one configured key
// picks that provider, no key means RuleAdvisor, and two keys is an error
// because silently preferring one provider is exactly the lock-in the record
// removes. The label is what the startup line prints.
func SelectAdvisor(config AdvisorConfig) (Advisor, string, error) {
	provider := strings.ToLower(strings.TrimSpace(config.Provider))
	if provider == "" {
		switch {
		case config.GeminiKey != "" && config.AnthropicKey != "":
			return nil, "", errors.New("both GEMINI_API_KEY and ANTHROPIC_API_KEY are set; set CONTEXT_RAIL_ADVISOR to gemini or anthropic")
		case config.GeminiKey != "":
			provider = "gemini"
		case config.AnthropicKey != "":
			provider = "anthropic"
		default:
			provider = "rule"
		}
	}
	switch provider {
	case "rule":
		return RuleAdvisor{}, "rule-advisor", nil
	case "gemini":
		if config.GeminiKey == "" {
			return nil, "", errors.New("CONTEXT_RAIL_ADVISOR=gemini but GEMINI_API_KEY is empty")
		}
		advisor := NewGeminiAdvisor(config.GeminiKey, config.GeminiModel)
		return advisor, advisor.Name() + " (falls back to rule-advisor on error)", nil
	case "anthropic":
		if config.AnthropicKey == "" {
			return nil, "", errors.New("CONTEXT_RAIL_ADVISOR=anthropic but ANTHROPIC_API_KEY is empty")
		}
		advisor := NewAnthropicAdvisor(config.AnthropicKey, config.AnthropicModel)
		return advisor, advisor.Name() + " (falls back to rule-advisor on error)", nil
	default:
		return nil, "", fmt.Errorf("unknown CONTEXT_RAIL_ADVISOR %q (use rule, gemini or anthropic)", config.Provider)
	}
}
