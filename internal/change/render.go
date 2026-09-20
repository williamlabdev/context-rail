package change

import (
	"fmt"
	"sort"
	"strings"
)

// RenderBrief renders the human Change Decision Brief from a DecisionRecord.
// It adds no facts: every line maps to a field of the record or to the live
// staleness verdict.
func RenderBrief(decision *DecisionRecord, facts *ProjectFacts, staleness Staleness) ChangeDecisionBrief {
	lineage := lineageOf(decision)
	headline := fmt.Sprintf("Proceed with %q in %s; verify in %s before anything further.", decision.SelectedOption, decision.TargetEnvironment.ID, decision.TargetEnvironment.ID)
	if staleness.Stale {
		headline = "This decision is STALE and must be re-made: " + staleness.Reason
	}
	projectName := decision.ProjectID
	if facts != nil && facts.ProjectName != "" {
		projectName = facts.ProjectName
	}
	sections := []BriefSection{
		{Heading: "Decision identity", Lines: []string{
			"Project: " + projectName + " (" + decision.ProjectID + ")",
			fmt.Sprintf("Change: %s · decision %s v%d", decision.ChangeID, decision.DecisionID, decision.Version),
			"Status: " + decision.Status + " · risk " + decision.RiskLevel,
			"Source snapshot: " + decision.SourceSnapshotHash,
			fmt.Sprintf("Environment topology: v%d (%s) · policy %s", decision.TopologyVersion, decision.TopologyConfigHash, decision.PolicyVersion),
		}},
		{Heading: "Where this change sits", Lines: []string{
			transitionLine(decision),
			"Production: " + decision.ProductionAction + " — this decision does not authorize any production change.",
		}},
		{Heading: "One-line conclusion", Lines: []string{headline}},
		{Heading: "Why", Lines: []string{decision.Rationale}},
		{Heading: "Objective", Lines: []string{decision.Objective}},
		{Heading: "Approved this time", Lines: orDefault(decision.AcceptedScope, "No explicit scope lines; the objective and acceptance criteria bound the work.")},
		{Heading: "Not approved / out of scope", Lines: orDefault(decision.OutOfScope, "No explicit out-of-scope lines; the forbidden actions below still apply.")},
		{Heading: "What must be true before the next gate", Lines: nextGateLines(decision)},
		{Heading: "Unknowns you accepted knowingly", Lines: orDefault(decision.Unknowns, "None recorded.")},
		{Heading: "Who decided", Lines: []string{fmt.Sprintf("%s (%s) %s at %s", decision.HumanDecision.Actor, decision.HumanDecision.Role, decision.HumanDecision.Decision, decision.HumanDecision.At)}},
	}
	if len(decision.Alternatives) > 1 {
		lines := []string{}
		for _, option := range decision.Alternatives {
			if option.ID == decision.SelectedOption {
				continue
			}
			lines = append(lines, option.Title+" — "+option.Summary)
		}
		sections = append(sections, BriefSection{Heading: "Alternatives considered, not selected", Lines: lines})
	}
	return ChangeDecisionBrief{
		ArtifactType: "change_decision_brief", Lineage: lineage,
		Audience: "business owner, engineering manager, PM", Headline: headline, Sections: sections, RenderedAt: decision.CreatedAt,
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

func transitionLine(decision *DecisionRecord) string {
	if decision.SourceEnvironment != nil {
		return fmt.Sprintf("Next transition: %s → %s (%s); target %s", decision.SourceEnvironment.ID, decision.TargetEnvironment.ID, decision.AllowedTransition, decision.TargetEnvironment.TargetRef)
	}
	return fmt.Sprintf("Next transition: entry → %s; target %s", decision.TargetEnvironment.ID, decision.TargetEnvironment.TargetRef)
}

func nextGateLines(decision *DecisionRecord) []string {
	lines := []string{}
	evidence := append([]string{}, decision.TargetEnvironment.RequiredEvidence...)
	sort.Strings(evidence)
	if len(evidence) > 0 {
		lines = append(lines, "Required evidence for "+decision.TargetEnvironment.ID+": "+strings.Join(evidence, ", "))
	}
	lines = append(lines, "An independent review and an allowed-paths diff check of the agent candidate.")
	lines = append(lines, "The same decision_id, version and source snapshot on the candidate; any input, topology or source change makes this decision STALE.")
	return lines
}

func orDefault(values []string, fallback string) []string {
	if len(values) == 0 {
		return []string{fallback}
	}
	return values
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
