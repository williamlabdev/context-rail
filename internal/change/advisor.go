package change

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Rule-advisor notes are fixed sentences with at most one value. Each has a
// code so the workspace can show it in the reader's language; the English
// text stays the recorded value. The generator and NoteCode share this table,
// so a sentence cannot change without its code.
var ruleNotes = []struct{ code, prefix, suffix string }{
	{"constraint_not_declared", "", " not declared; cost status UNKNOWN"},
	{"project_context_not_current", "project context is ", "; some architecture facts may be outdated"},
	{"repository_unverified", "repository URL is a fixture placeholder; real remote and branch protection unverified", ""},
	{"no_out_of_scope_list", "no explicit out-of-scope list; the agent boundary relies on allowed_paths and forbidden_actions only", ""},
}

func ruleNote(code, value string) string {
	for _, note := range ruleNotes {
		if note.code == code {
			return note.prefix + value + note.suffix
		}
	}
	panic("unknown rule note " + code)
}

// NoteCode returns the code and value of a rule-advisor sentence, or "" for
// any other text (a model's unknown, a human's words).
func NoteCode(text string) (code, value string) {
	for _, note := range ruleNotes {
		if note.suffix == "" {
			if text == note.prefix {
				return note.code, ""
			}
			continue
		}
		if strings.HasPrefix(text, note.prefix) && strings.HasSuffix(text, note.suffix) && len(text) > len(note.prefix)+len(note.suffix) {
			return note.code, text[len(note.prefix) : len(text)-len(note.suffix)]
		}
	}
	return "", ""
}

// ruleOptionIDs are the options RuleAdvisor proposes with fixed wording.
var ruleOptionIDs = map[string]bool{
	"minimal_reversible_slice": true, "defer": true,
	"managed_storage_with_signed_urls": true, "expanded_scope_new_infrastructure": true,
}

// RuleAdvisor is the deterministic default. It proposes the same candidates
// for the same request and never claims facts the request did not state.
type RuleAdvisor struct{}

func (RuleAdvisor) Name() string { return "rule-advisor" }

func (RuleAdvisor) Propose(request Request, facts *ProjectFacts) ([]Option, []string, error) {
	unknowns := []string{}
	for _, constraint := range requiredConstraints {
		if strings.TrimSpace(request.BusinessConstraints[constraint.key]) == "" {
			unknowns = append(unknowns, ruleNote("constraint_not_declared", constraint.key))
		}
	}
	if facts != nil && facts.ContextStatus != "" && facts.ContextStatus != "CURRENT" && facts.ContextStatus != "DERIVED" {
		unknowns = append(unknowns, ruleNote("project_context_not_current", facts.ContextStatus))
	}
	if facts != nil && strings.Contains(facts.RepositoryURL, "example") {
		unknowns = append(unknowns, ruleNote("repository_unverified", ""))
	}
	if len(request.ScopeExcluded) == 0 {
		unknowns = append(unknowns, ruleNote("no_out_of_scope_list", ""))
	}
	text := strings.ToLower(strings.Join(append([]string{request.Objective}, request.ScopeIncluded...), " "))
	storageLike := strings.Contains(text, "attach") || strings.Contains(text, "upload") || strings.Contains(text, "file") || strings.Contains(text, "storage")

	options := []Option{
		{
			ID: "minimal_reversible_slice", Title: "Minimal reversible slice",
			Summary:     "Implement only the accepted scope inside the allowed paths, without new infrastructure, persistence or IAM changes; verify in the target environment before anything else is added.",
			CostDrivers: []string{"compute of the existing service", "build and test minutes"},
			Risks:       []string{"may not cover the full business need on its own", "state resets if the slice is stateless"},
			Recommended: true, AdvisorSource: "rule-advisor",
		},
		{
			ID: "defer", Title: "Defer / keep current process",
			Summary:       "Do not change the system now; keep the manual process and revisit when the unknowns are resolved.",
			CostDrivers:   []string{"ongoing manual effort (not measured)"},
			Risks:         []string{"business impact of waiting is not quantified"},
			AdvisorSource: "rule-advisor",
		},
	}
	if storageLike {
		options = append(options, Option{
			ID: "managed_storage_with_signed_urls", Title: "Managed object storage with short-lived signed URLs",
			Summary:       "Store objects in a private bucket and have the service authorize first, then issue short-lived signed URLs; needs retention, region and access-pattern inputs before its cost can be known.",
			CostDrivers:   []string{"stored bytes over retention", "download egress", "metadata reads/writes"},
			Risks:         []string{"URL replay and revocation limits", "IAM and bucket policy must be provisioned outside the agent's scope"},
			AdvisorSource: "rule-advisor",
		})
	} else {
		options = append(options, Option{
			ID: "expanded_scope_new_infrastructure", Title: "Expanded scope with new infrastructure",
			Summary:       "Solve the broader need with new services or persistence; requires its own architecture decision, cost evidence and IAM plan before acceptance.",
			CostDrivers:   []string{"new managed services", "operations and on-call load"},
			Risks:         []string{"scope creep", "cost unknown until sized", "longer time to first evidence"},
			AdvisorSource: "rule-advisor",
		})
	}
	return options, unknowns, nil
}

// GeminiAdvisor asks Gemini to propose candidates and unknowns as strict
// JSON. It is an adapter over the Generative Language REST API; the caller
// (Service) falls back to RuleAdvisor if it fails, and every option it
// returns is still only a candidate until a human selects it.
//
// NOT exercised in the verification workspace (no key, host not reachable);
// treat as UNVERIFIED until a run with GEMINI_API_KEY is recorded as evidence.
type GeminiAdvisor struct {
	APIKey   string
	Model    string
	Endpoint string
	Client   *http.Client
}

func NewGeminiAdvisor(apiKey, model string) *GeminiAdvisor {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiAdvisor{APIKey: apiKey, Model: model, Endpoint: "https://generativelanguage.googleapis.com/v1beta", Client: &http.Client{Timeout: 30 * time.Second}}
}

func (advisor *GeminiAdvisor) Name() string { return "gemini:" + advisor.Model }

type geminiProposal struct {
	Options  []Option `json:"options"`
	Unknowns []string `json:"unknowns"`
}

func (advisor *GeminiAdvisor) Propose(request Request, facts *ProjectFacts) ([]Option, []string, error) {
	if advisor.APIKey == "" {
		return nil, nil, errors.New("GEMINI_API_KEY not configured")
	}
	grounding := map[string]any{
		"project_id": facts.ProjectID, "project_context_status": facts.ContextStatus,
		"documents": facts.Documents, "repository": facts.RepositoryURL,
	}
	groundingJSON, _ := json.Marshal(grounding)
	requestJSON, _ := json.Marshal(request)
	prompt := "You are the ContextRail decision advisor. Propose 2-3 candidate options for the change request below and list unknowns that block a cost or risk judgement. " +
		"Use ONLY the request and the grounding facts; if something is not stated, put it in unknowns instead of assuming it. " +
		"Respond with strict JSON: {\"options\":[{\"id\":snake_case,\"title\":..,\"summary\":..,\"cost_drivers\":[..],\"risks\":[..],\"recommended\":bool}],\"unknowns\":[..]}.\n" +
		"REQUEST: " + string(requestJSON) + "\nGROUNDING: " + string(groundingJSON)
	body := map[string]any{
		"contents":         []map[string]any{{"role": "user", "parts": []map[string]string{{"text": prompt}}}},
		"generationConfig": map[string]any{"responseMimeType": "application/json", "temperature": 0.2},
	}
	encoded, _ := json.Marshal(body)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	url := fmt.Sprintf("%s/models/%s:generateContent", advisor.Endpoint, advisor.Model)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return nil, nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("x-goog-api-key", advisor.APIKey) // header, never the URL: keys must not land in logs
	response, err := advisor.Client.Do(httpRequest)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("gemini returned HTTP %d", response.StatusCode)
	}
	var envelope struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Candidates) == 0 || len(envelope.Candidates[0].Content.Parts) == 0 {
		return nil, nil, errors.New("gemini response had no candidate text")
	}
	var proposal geminiProposal
	if err := json.Unmarshal([]byte(envelope.Candidates[0].Content.Parts[0].Text), &proposal); err != nil {
		return nil, nil, fmt.Errorf("gemini response was not the expected JSON: %w", err)
	}
	if len(proposal.Options) == 0 {
		return nil, nil, errors.New("gemini proposed no options")
	}
	for index := range proposal.Options {
		proposal.Options[index].AdvisorSource = advisor.Name()
		if proposal.Options[index].ID == "" {
			proposal.Options[index].ID = fmt.Sprintf("gemini_option_%d", index+1)
		}
	}
	// Rule-derived unknowns are always kept: the model may omit them.
	_, ruleUnknowns, _ := RuleAdvisor{}.Propose(request, facts)
	return proposal.Options, mergeUnique(ruleUnknowns, proposal.Unknowns), nil
}
