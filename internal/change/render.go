package change

import (
	"sort"
	"strings"
)

// RenderBrief renders the human Change Decision Brief from a DecisionRecord.
// It adds no facts: every field is copied from the record, the option list
// it was decided against, or the live staleness verdict.
func RenderBrief(decision *DecisionRecord, staleness Staleness) ChangeDecisionBrief {
	state := "ACCEPTED"
	if staleness.Stale {
		state = StatusStale
	}
	selected := BriefOption{ID: decision.SelectedOption, Title: decision.SelectedOption}
	alternatives := []BriefOption{}
	for _, option := range decision.Alternatives {
		entry := BriefOption{ID: option.ID, Title: option.Title, Summary: option.Summary}
		if option.AdvisorSource == "rule-advisor" && ruleOptionIDs[option.ID] {
			entry.Code = option.ID
		}
		if option.ID == decision.SelectedOption {
			selected = entry
			continue
		}
		alternatives = append(alternatives, entry)
	}
	route := BriefRoute{
		TargetEnvironmentID: decision.TargetEnvironment.ID, TargetType: decision.TargetEnvironment.Type,
		TargetRef: decision.TargetEnvironment.TargetRef, Transition: decision.AllowedTransition, ProductionAction: decision.ProductionAction,
		Path: append([]PathStep{}, decision.PromotionPath...),
	}
	if decision.SourceEnvironment != nil {
		route.SourceEnvironmentID = decision.SourceEnvironment.ID
	}
	unknowns := []BriefNote{}
	for _, text := range decision.Unknowns {
		code, value := NoteCode(text)
		unknowns = append(unknowns, BriefNote{Text: text, Code: code, Value: value})
	}
	evidence := append([]string{}, decision.TargetEnvironment.RequiredEvidence...)
	sort.Strings(evidence)
	return ChangeDecisionBrief{
		ArtifactType: "change_decision_brief", SchemaVersion: "change-decision-brief/v2", Lineage: lineageOf(decision),
		Audience: "business owner, engineering manager, PM", State: state, StaleReason: staleness.Reason,
		OwnerSummary: decision.OwnerSummary, OwnerSummaryMissing: strings.TrimSpace(decision.OwnerSummary) == "",
		Objective: decision.Objective, Selected: selected, Route: route, RiskLevel: decision.RiskLevel,
		Unknowns: unknowns, InScope: nonNil(decision.AcceptedScope), OutOfScope: nonNil(decision.OutOfScope),
		RequiredEvidence: evidence,
		DecidedBy:        BriefDecider{Actor: decision.HumanDecision.Actor, Role: decision.HumanDecision.Role, At: decision.HumanDecision.At, Rationale: decision.Rationale},
		Alternatives:     alternatives, RenderedAt: decision.CreatedAt,
	}
}

// RenderAgentContextPack renders the machine-facing pack from the same
// DecisionRecord. Same lineage, same scope, no re-interpretation.
func RenderAgentContextPack(decision *DecisionRecord, staleness Staleness) AgentContextPack {
	status := "ISSUED_FOR_DEVELOPMENT_CANDIDATE"
	if staleness.Stale {
		status = "STALE"
	}
	tests := make([]AcceptanceTest, 0, len(decision.AcceptanceCriteria))
	for _, criterion := range decision.AcceptanceCriteria {
		tests = append(tests, AcceptanceTest{ID: criterion.ID, Expected: criterion.Text})
	}
	topologyBlock := map[string]any{
		"version":               decision.TopologyVersion,
		"config_hash":           decision.TopologyConfigHash,
		"target_environment_id": decision.TargetEnvironment.ID,
		"target_ref":            decision.TargetEnvironment.TargetRef,
		"allowed_transition_id": decision.AllowedTransition,
		"production_action":     decision.ProductionAction,
	}
	if decision.SourceEnvironment != nil {
		topologyBlock["source_environment_id"] = decision.SourceEnvironment.ID
	}
	constraints := decision.BusinessConstraints
	if constraints == nil {
		constraints = map[string]string{}
	}
	return AgentContextPack{
		ArtifactType: "agent_context_pack", SchemaVersion: "agent-context-pack/v1", Lineage: lineageOf(decision), Status: status,
		Objective: decision.Objective, EnvironmentTopology: topologyBlock, BusinessConstraints: constraints,
		AcceptedScope: nonNil(decision.AcceptedScope), OutOfScope: nonNil(decision.OutOfScope),
		AllowedPaths: nonNil(decision.AllowedPaths), ForbiddenActions: nonNil(decision.ForbiddenActions),
		AcceptanceTests: tests, RequiredChecks: requiredChecks(decision), Unknowns: nonNil(decision.Unknowns),
		EvidenceRefs: nonNil(decision.EvidenceRefs), RenderedAt: decision.CreatedAt,
	}
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
