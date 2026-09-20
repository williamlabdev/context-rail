package change_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"context-rail/internal/change"
	"context-rail/internal/topology"
)

// harness wires a change service to an in-memory topology so tests can move
// the topology and watch decisions go stale.
type harness struct {
	topology *topology.Service
	changes  *change.Service
	context  string
	repo     string
}

type topologySource struct{}

func (topologySource) Baseline(projectID string) (*topology.Baseline, error) {
	if projectID != "demo" {
		return nil, nil
	}
	return &topology.Baseline{ProjectID: "demo", Environments: []topology.Environment{
		{ID: "development", Type: "development", Sequence: 1, TargetRef: "local", RequiredEvidence: []string{"local-test"}},
		{ID: "testing", Type: "testing", Sequence: 2, TargetRef: "ci", RequiredEvidence: []string{"test", "build"}},
		{ID: "staging", Type: "staging", Sequence: 3, TargetRef: "cloud-run/demo-staging", RequiredEvidence: []string{"decision-record", "smoke"}},
		{ID: "production", Type: "production", Sequence: 4, TargetRef: "protected", RequiredEvidence: []string{"release-approval"}},
	}}, nil
}

func (h *harness) Facts(projectID string) (*change.ProjectFacts, error) {
	if projectID != "demo" {
		return nil, nil
	}
	state, err := h.topology.Get(projectID)
	if err != nil {
		return nil, err
	}
	return &change.ProjectFacts{
		ProjectID: "demo", ProjectName: "Demo", RepositoryURL: h.repo, DefaultBranch: "develop",
		Documents:      []change.DocumentFact{{Path: "project.yaml", Status: "CURRENT", SourceOfTruth: true}, {Path: "architecture.md", Status: "CURRENT", SourceOfTruth: true}},
		ContextStatus:  h.context,
		DecisionInputs: change.ReadinessFact{Status: "READY", Reasons: []string{}},
		Topology:       state,
	}, nil
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{context: "DERIVED", repo: "github.com/example/demo"}
	tick := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { tick = tick.Add(time.Second); return tick }
	h.topology = topology.NewService(topology.NewMemoryStore(), topologySource{}).WithClock(clock)
	h.changes = change.NewService(change.NewMemoryStore(), h, nil).WithClock(clock)
	return h
}

func completeRequest() change.Request {
	return change.Request{
		Title: "Manual order review", Objective: "Let operations staff approve or reject an order exception with a note.",
		ScopeIncluded: []string{"review action", "required note"}, ScopeExcluded: []string{"payment", "fulfillment"},
		AcceptanceCriteria: []change.AcceptanceCriterion{{Text: "approve and reject require a note"}, {ID: "AC-404", Text: "unknown order returns 404"}},
		AllowedPaths:       []string{"main.go", "web/index.html", "main_test.go"}, ForbiddenActions: []string{"authentication"},
		TargetEnvironmentID: "staging",
		BusinessConstraints: map[string]string{"data_classification": "internal", "expected_monthly_volume": "500 reviews"},
	}
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var typed *change.Error
	if !errors.As(err, &typed) {
		t.Fatalf("expected change.Error, got %v", err)
	}
	return typed.Code
}

func mutation(reason string) change.Mutation {
	return change.Mutation{Actor: "requester-1", Reason: reason}
}

func TestIncompleteRequestIsNeedsInputWithOwners(t *testing.T) {
	h := newHarness(t)
	view, err := h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: change.Request{Title: "Attachments", Objective: "Upload attachments"}})
	if err != nil {
		t.Fatal(err)
	}
	if view.Change.Status != change.StatusNeedsInput || view.Change.ChangeID != "CHG-001" {
		t.Fatalf("expected CHG-001 NEEDS_INPUT, got %+v", view.Change)
	}
	evaluation := view.Change.Versions[0].Evaluation
	fields := map[string]string{}
	for _, missing := range evaluation.MissingInputs {
		fields[missing.Field] = missing.OwnerRole
	}
	for field, owner := range map[string]string{"acceptance_criteria": "requester", "allowed_paths": "tech_lead", "target_environment_id": "tech_lead", "business_constraints.data_classification": "business_owner", "business_constraints.expected_monthly_volume": "business_owner"} {
		if fields[field] != owner {
			t.Fatalf("missing input %s must be owned by %s: %v", field, owner, fields)
		}
	}
	if view.Decision != nil || view.Brief != nil || view.Pack != nil {
		t.Fatal("no artifacts may be rendered before a decision")
	}
	if len(view.Change.Versions[0].Options) < 2 {
		t.Fatal("advisor must still propose candidates so the human can see the shape of the decision")
	}
	if !strings.Contains(strings.Join(view.Change.Versions[0].Unknowns, " "), "UNKNOWN") {
		t.Fatalf("unknown cost drivers must be stated: %v", view.Change.Versions[0].Unknowns)
	}
	// Accepting a NEEDS_INPUT version is refused; nothing is invented.
	_, err = h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: mutation("go"), Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "just do it"})
	if codeOf(t, err) != change.CodeNotDecisionReady {
		t.Fatalf("expected NEEDS_INPUT refusal, got %v", err)
	}
}

func TestSupplyingInputsCreatesNewVersionAndBecomesReady(t *testing.T) {
	h := newHarness(t)
	_, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: change.Request{Title: "Attachments", Objective: "Upload attachments", TargetEnvironmentID: "staging"}})
	criteria := []change.AcceptanceCriterion{{Text: "upload is private"}}
	paths := []string{"internal/attachments/"}
	view, err := h.changes.SupplyInputs("demo", "CHG-001", change.InputsRequest{
		Mutation:           mutation("business owner supplied volume and classification"),
		AcceptanceCriteria: &criteria, AllowedPaths: &paths,
		BusinessConstraints: map[string]string{"data_classification": "internal-confidential", "expected_monthly_volume": "10000 uploads"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Change.CurrentVersion != 2 || view.Change.Status != change.StatusDecisionReady {
		t.Fatalf("expected v2 DECISION_READY, got v%d %s (%+v)", view.Change.CurrentVersion, view.Change.Status, view.Change.Versions[1].Evaluation.MissingInputs)
	}
	if view.Change.Versions[0].Evaluation.Status != change.StatusNeedsInput {
		t.Fatal("v1 must remain readable as NEEDS_INPUT")
	}
	if view.Change.Versions[1].Evaluation.AllowedTransition != "testing-to-staging" {
		t.Fatalf("transition must be derived from the topology order: %s", view.Change.Versions[1].Evaluation.AllowedTransition)
	}
	options := view.Change.Versions[1].Options
	if options[len(options)-1].ID != "managed_storage_with_signed_urls" {
		t.Fatalf("storage-like request should get the signed-url candidate: %v", options)
	}
}

func TestAcceptRendersBriefAndPackFromTheSameRecord(t *testing.T) {
	h := newHarness(t)
	_, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: completeRequest()})
	view, err := h.changes.Decide("demo", "CHG-001", change.DecideRequest{
		Mutation: change.Mutation{Actor: "founder-001", Reason: ""}, Decision: "ACCEPT", Role: "solution_architect",
		SelectedOption: "minimal_reversible_slice", Rationale: "smallest reversible slice proves the path", RiskLevel: "low",
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Change.Status != change.StatusAccepted || view.Decision == nil || view.Decision.DecisionID != "DEC-001" || view.Decision.Status != "ACCEPTED_FOR_DEVELOPMENT" {
		t.Fatalf("expected accepted DEC-001, got %+v", view.Decision)
	}
	if view.Brief == nil || view.Pack == nil {
		t.Fatal("brief and pack must be rendered from the accepted record")
	}
	if view.Brief.Lineage != view.Pack.Lineage {
		t.Fatalf("brief and pack must share one lineage: %+v vs %+v", view.Brief.Lineage, view.Pack.Lineage)
	}
	lineage := view.Pack.Lineage
	if lineage.DecisionID != "DEC-001" || lineage.DecisionVersion != 1 || lineage.SourceSnapshotHash == "" || lineage.TopologyVersion != 1 || lineage.PolicyVersion != change.PolicyVersion {
		t.Fatalf("lineage incomplete: %+v", lineage)
	}
	if lineage.SourceSnapshotHash != view.Decision.SourceSnapshotHash {
		t.Fatal("lineage hash must equal the record's hash")
	}
	if len(view.Pack.AcceptanceTests) != 2 || view.Pack.AcceptanceTests[0].ID != "AC-001" || view.Pack.AcceptanceTests[1].ID != "AC-404" {
		t.Fatalf("acceptance tests must mirror criteria ids: %+v", view.Pack.AcceptanceTests)
	}
	joined := strings.Join(view.Pack.ForbiddenActions, ",")
	if !strings.Contains(joined, "deploy:production") || !strings.Contains(joined, "authentication") {
		t.Fatalf("policy defaults and request forbidden actions must both be present: %s", joined)
	}
	if view.Pack.EnvironmentTopology["target_environment_id"] != "staging" || view.Pack.EnvironmentTopology["production_action"] != "forbidden" {
		t.Fatalf("pack topology block wrong: %v", view.Pack.EnvironmentTopology)
	}
	briefText := ""
	for _, section := range view.Brief.Sections {
		briefText += section.Heading + " " + strings.Join(section.Lines, " ") + " "
	}
	for _, expected := range []string{"DEC-001", view.Decision.SourceSnapshotHash, "testing → staging", "founder-001", "payment"} {
		if !strings.Contains(briefText, expected) {
			t.Fatalf("brief must carry %q", expected)
		}
	}
	// A second acceptance of the same version is refused.
	_, err = h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "defer", Rationale: "again"})
	if codeOf(t, err) != change.CodeAlreadyDecided {
		t.Fatalf("expected ALREADY_DECIDED, got %v", err)
	}
}

func TestDecisionValidation(t *testing.T) {
	h := newHarness(t)
	_, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: completeRequest()})
	if _, err := h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: ""}, Decision: "ACCEPT", SelectedOption: "defer", Rationale: "x"}); codeOf(t, err) != change.CodeActorRequired {
		t.Fatalf("anonymous decision must be refused: %v", err)
	}
	if _, err := h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "a"}, Decision: "ACCEPT", SelectedOption: "not-a-candidate", Rationale: "x"}); codeOf(t, err) != change.CodeInvalidOption {
		t.Fatalf("unknown option must be refused: %v", err)
	}
	if _, err := h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "a"}, Decision: "ACCEPT", SelectedOption: "defer"}); codeOf(t, err) != change.CodeReasonRequired {
		t.Fatalf("rationale is required: %v", err)
	}
	view, err := h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "a"}, Decision: "REJECT", Rationale: "not now"})
	if err != nil || view.Change.Status != change.StatusRejected || view.Decision.Status != "REJECTED" {
		t.Fatalf("reject must close the change: %v %+v", err, view)
	}
	if view.Brief != nil || view.Pack != nil {
		t.Fatal("a rejected decision renders no handoff artifacts")
	}
	if _, err := h.changes.SupplyInputs("demo", "CHG-001", change.InputsRequest{Mutation: mutation("edit"), Objective: strPtr("new")}); codeOf(t, err) != change.CodeChangeClosed {
		t.Fatalf("closed change cannot be edited: %v", err)
	}
}

func TestProductionTargetAndRetiredTargetBlock(t *testing.T) {
	h := newHarness(t)
	request := completeRequest()
	request.TargetEnvironmentID = "production"
	view, _ := h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: request})
	if view.Change.Status != change.StatusNeedsInput || !strings.Contains(view.Change.Versions[0].Evaluation.MissingInputs[0].Reason, "production") {
		t.Fatalf("production target must be blocked: %+v", view.Change.Versions[0].Evaluation)
	}
	if _, err := h.topology.Retire("demo", "staging", topology.Mutation{ExpectedVersion: 1, Actor: "ops", Reason: "merge staging into testing"}); err != nil {
		t.Fatal(err)
	}
	request.TargetEnvironmentID = "staging"
	view, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: request})
	if view.Change.Status != change.StatusNeedsInput || !strings.Contains(view.Change.Versions[0].Evaluation.MissingInputs[0].Reason, "RETIRED") {
		t.Fatalf("retired target must be blocked: %+v", view.Change.Versions[0].Evaluation)
	}
}

func TestWorkOrderCompilesWithHashAndGoesStaleOnTopologyChange(t *testing.T) {
	h := newHarness(t)
	_, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: completeRequest()})
	if _, err := h.changes.CompileWorkOrder("demo", "CHG-001", change.WorkOrderRequest{Mutation: mutation("issue")}); codeOf(t, err) != change.CodeNotAccepted {
		t.Fatalf("work order before acceptance must be refused: %v", err)
	}
	_, err := h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "ok"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := h.changes.CompileWorkOrder("demo", "CHG-001", change.WorkOrderRequest{Mutation: mutation("issue to agent"), Issuer: "founder-001"})
	if err != nil {
		t.Fatal(err)
	}
	order := view.WorkOrder
	if order == nil || order.WorkOrderID != "AWO-001" || order.Status != "ISSUED" || !strings.HasPrefix(order.WorkOrderHash, "sha256:") {
		t.Fatalf("expected issued AWO-001 with hash, got %+v", order)
	}
	if order.Lineage != view.Pack.Lineage {
		t.Fatal("work order lineage must equal the pack lineage")
	}
	if order.Repository != "github.com/example/demo" || order.BaseBranch != "develop" || order.TargetBranch != "change/chg-001" || order.TargetEnvironmentID != "staging" {
		t.Fatalf("work order binding wrong: %+v", order)
	}
	if strings.Join(order.AcceptanceIDs, ",") != "AC-001,AC-404" {
		t.Fatalf("acceptance ids wrong: %v", order.AcceptanceIDs)
	}
	if !strings.Contains(strings.Join(order.RequiredChecks, ","), "smoke") || !strings.Contains(strings.Join(order.RequiredChecks, ","), "allowed-paths-diff-check") {
		t.Fatalf("required checks must include target evidence and the diff check: %v", order.RequiredChecks)
	}
	if _, err := h.changes.CompileWorkOrder("demo", "CHG-001", change.WorkOrderRequest{Mutation: mutation("again")}); codeOf(t, err) != change.CodeAlreadyIssued {
		t.Fatalf("second issue must be refused: %v", err)
	}
	// Display-only topology edit: not material, decision stays valid.
	name := "Pre-prod"
	if _, err := h.topology.Edit("demo", "staging", topology.EditRequest{Mutation: topology.Mutation{ExpectedVersion: 1, Actor: "ops", Reason: "rename"}, DisplayName: &name}); err != nil {
		t.Fatal(err)
	}
	view, _ = h.changes.Get("demo", "CHG-001")
	if view.Staleness.Stale {
		t.Fatalf("display-only topology change must not invalidate: %s", view.Staleness.Reason)
	}
	// Material topology edit: decision, brief, pack and work order go STALE.
	target := "cloud-run/demo-staging-2"
	if _, err := h.topology.Edit("demo", "staging", topology.EditRequest{Mutation: topology.Mutation{ExpectedVersion: 2, Actor: "ops", Reason: "move service"}, TargetRef: &target}); err != nil {
		t.Fatal(err)
	}
	view, _ = h.changes.Get("demo", "CHG-001")
	if !view.Staleness.Stale || !strings.Contains(view.Staleness.Reason, "topology v3") {
		t.Fatalf("material topology change must invalidate: %+v", view.Staleness)
	}
	if view.Change.Status != change.StatusStale || view.Pack.Status != "STALE" || view.WorkOrder.Status != "STALE" || !strings.Contains(view.Brief.Headline, "STALE") {
		t.Fatalf("stale must be visible on every artifact: change=%s pack=%s awo=%s brief=%s", view.Change.Status, view.Pack.Status, view.WorkOrder.Status, view.Brief.Headline)
	}
	if _, err := h.changes.CompileWorkOrder("demo", "CHG-001", change.WorkOrderRequest{Mutation: mutation("reissue")}); codeOf(t, err) != change.CodeDecisionStale {
		t.Fatalf("stale decision cannot issue a work order: %v", err)
	}
}

func TestChangedInputsAfterAcceptanceMakeDecisionStale(t *testing.T) {
	h := newHarness(t)
	_, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: completeRequest()})
	_, _ = h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "ok"})
	view, err := h.changes.SupplyInputs("demo", "CHG-001", change.InputsRequest{Mutation: mutation("retention changed"), BusinessConstraints: map[string]string{"retention_days": "365"}})
	if err != nil {
		t.Fatal(err)
	}
	if !view.Staleness.Stale || view.Change.CurrentVersion != 2 || !strings.Contains(view.Staleness.Reason, "v2") {
		t.Fatalf("new inputs must stale the earlier decision: %+v", view.Staleness)
	}
	// Re-deciding creates DEC-002 v2 bound to change v2.
	view, err = h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "re-accepted with retention"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Staleness.Stale || view.Decision.DecisionID != "DEC-002" || view.Decision.Version != 2 || view.Decision.ChangeVersion != 2 {
		t.Fatalf("expected fresh DEC-002 v2 against change v2: %+v %+v", view.Decision, view.Staleness)
	}
	if len(view.Change.Decisions) != 2 {
		t.Fatal("earlier decision must remain readable")
	}
}

func TestSourceDriftMakesDecisionStale(t *testing.T) {
	h := newHarness(t)
	_, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: completeRequest()})
	_, _ = h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "ok"})
	h.context = "STALE" // the Project's derived context moved after acceptance
	view, _ := h.changes.Get("demo", "CHG-001")
	if !view.Staleness.Stale || !strings.Contains(view.Staleness.Reason, "source_snapshot_hash") {
		t.Fatalf("source drift must invalidate: %+v", view.Staleness)
	}
}

func TestUnknownProject(t *testing.T) {
	h := newHarness(t)
	_, err := h.changes.List("nope")
	if codeOf(t, err) != change.CodeProjectNotFound {
		t.Fatalf("expected PROJECT_NOT_FOUND, got %v", err)
	}
}

func strPtr(value string) *string { return &value }
