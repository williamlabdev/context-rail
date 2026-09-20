package change_test

import (
	"strings"
	"testing"

	"context-rail/internal/change"
	"context-rail/internal/topology"
)

// acceptedWithOrder drives a change to an issued Work Order and returns it.
func acceptedWithOrder(t *testing.T, h *harness) *change.AgentWorkOrder {
	t.Helper()
	if _, err := h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: completeRequest()}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "ok"}); err != nil {
		t.Fatal(err)
	}
	view, err := h.changes.CompileWorkOrder("demo", "CHG-001", change.WorkOrderRequest{Mutation: mutation("issue"), Issuer: "founder-001"})
	if err != nil {
		t.Fatal(err)
	}
	return view.WorkOrder
}

func goodRun(order *change.AgentWorkOrder) change.AgentRun {
	return change.AgentRun{
		RunID: "ARR-001", WorkOrderID: order.WorkOrderID, WorkOrderHash: order.WorkOrderHash,
		Agent:   change.AgentIdentity{Name: "claude-code", Provider: "anthropic", Model: "claude-fable-5-1", Version: "1.0"},
		Adapter: "manual-declaration", StartedBy: "dev-1", StartedAt: "2026-09-21T02:00:00Z",
		DeclaredCommands: []string{"go test ./..."}, DeclaredPaths: []string{"main.go", "main_test.go"},
	}
}

func goodObservation(order *change.AgentWorkOrder) change.Observation {
	return change.Observation{
		Source: "declared", Branch: order.TargetBranch, BaseBranch: order.BaseBranch, HeadCommit: "abc1234def5678", BaseCommit: "0000000",
		ChangedPaths: []string{"main.go", "main_test.go", "web/index.html"},
		Checks: []change.CheckResult{
			{Name: "test", Status: "PASS", EvidenceRef: "evidence/test-output.txt"},
			{Name: "build", Status: "PASS", EvidenceRef: "evidence/build-output.txt"},
			{Name: "code-review", Status: "PASS"},
			{Name: "single-operator-controls", Status: "PASS"},
		},
		Reviews: []change.ReviewEvidence{{Reviewer: "reviewer-2", Kind: "human", Verdict: "APPROVED", EvidenceRef: "pr#1"}},
	}
}

func gateStatus(candidate change.Candidate, gate string) string {
	for _, result := range candidate.Gates {
		if result.Gate == gate {
			return result.Status
		}
	}
	return "ABSENT"
}

func lastCandidate(view *change.View) change.Candidate {
	return view.Change.Candidates[len(view.Change.Candidates)-1]
}

func TestCandidateWithinContractIsAcceptableAndHumanAccepts(t *testing.T) {
	h := newHarness(t)
	order := acceptedWithOrder(t, h)
	if !strings.Contains(strings.Join(order.RequiredChecks, ","), "smoke") {
		t.Fatalf("staging target should require smoke: %v", order.RequiredChecks)
	}
	view, err := h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("agent finished"), Run: goodRun(order), Observation: goodObservation(order)})
	if err != nil {
		t.Fatal(err)
	}
	candidate := lastCandidate(view)
	if candidate.CandidateID != "CAND-001" || candidate.Verdict != change.VerdictAcceptable {
		t.Fatalf("expected CAND-001 acceptable, got %s %s: %+v", candidate.CandidateID, candidate.Verdict, candidate.Gates)
	}
	if candidate.Lineage != order.Lineage {
		t.Fatal("candidate must carry the work order lineage")
	}
	for gate, want := range map[string]string{"work_order_bound": change.GatePass, "allowed_paths": change.GatePass, "required_checks": change.GatePass, "independent_review": change.GatePass, "promotion_checks": change.GateDeferred, "work_order_expiry": change.GatePass} {
		if got := gateStatus(candidate, gate); got != want {
			t.Fatalf("gate %s: expected %s, got %s (%+v)", gate, want, got, candidate.Gates)
		}
	}
	if gateStatus(candidate, "declaration_matches_diff") != change.GatePass {
		t.Fatal("undeclared web/index.html must be recorded as observed drift, not hidden")
	}
	// The run starter cannot accept their own candidate without waiver.
	_, err = h.changes.DecideCandidate("demo", "CHG-001", "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "dev-1"}, Decision: "ACCEPT", Rationale: "mine"})
	if codeOf(t, err) != change.CodeCandidateBlocked {
		t.Fatalf("self-acceptance must be refused: %v", err)
	}
	view, err = h.changes.DecideCandidate("demo", "CHG-001", "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", Role: "solution_architect", Rationale: "gates green, review independent"})
	if err != nil {
		t.Fatal(err)
	}
	candidate = lastCandidate(view)
	if candidate.Status != "ACCEPTED_FOR_PROMOTION" || candidate.HumanDecision == nil || view.Change.Status != change.StatusCandidateAccepted {
		t.Fatalf("expected accepted candidate, got %s / %s", candidate.Status, view.Change.Status)
	}
	if _, err := h.changes.DecideCandidate("demo", "CHG-001", "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", Rationale: "again"}); codeOf(t, err) != change.CodeAlreadyDecided {
		t.Fatalf("double decision must be refused: %v", err)
	}
}

func TestCandidateOutsideAllowedPathsIsBlockedWithThePath(t *testing.T) {
	h := newHarness(t)
	order := acceptedWithOrder(t, h)
	observation := goodObservation(order)
	observation.ChangedPaths = append(observation.ChangedPaths, "infra/iam/public-bucket.yaml", "internal/auth/login.go")
	view, err := h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("agent finished"), Run: goodRun(order), Observation: observation})
	if err != nil {
		t.Fatal(err)
	}
	candidate := lastCandidate(view)
	if candidate.Verdict != change.VerdictBlocked || gateStatus(candidate, "allowed_paths") != change.GateBlocked {
		t.Fatalf("expected BLOCKED on allowed_paths: %s %+v", candidate.Verdict, candidate.Gates)
	}
	paths := []string{}
	for _, violation := range candidate.Violations {
		if violation.Rule == "path_out_of_scope" {
			paths = append(paths, violation.Path)
		}
	}
	if strings.Join(paths, ",") != "infra/iam/public-bucket.yaml,internal/auth/login.go" {
		t.Fatalf("violations must name exactly the out-of-scope paths: %v", paths)
	}
	if !strings.Contains(candidate.RecommendedAction, "green test suite cannot override") {
		t.Fatalf("recommended action must say tests do not override policy: %s", candidate.RecommendedAction)
	}
	if _, err := h.changes.DecideCandidate("demo", "CHG-001", "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", Rationale: "tests pass"}); codeOf(t, err) != change.CodeCandidateBlocked {
		t.Fatalf("blocked candidate must not be acceptable: %v", err)
	}
	// Rejecting is still a recorded human act.
	view, err = h.changes.DecideCandidate("demo", "CHG-001", "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "REJECT", Rationale: "out of scope"})
	if err != nil || lastCandidate(view).Status != "REJECTED" {
		t.Fatalf("reject must be recorded: %v", err)
	}
}

func TestCandidateWithoutIndependentReviewNeedsReview(t *testing.T) {
	h := newHarness(t)
	order := acceptedWithOrder(t, h)
	observation := goodObservation(order)
	observation.Reviews = []change.ReviewEvidence{{Reviewer: "dev-1", Kind: "human", Verdict: "APPROVED"}} // the run starter reviewing themselves
	view, err := h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("agent finished"), Run: goodRun(order), Observation: observation})
	if err != nil {
		t.Fatal(err)
	}
	candidate := lastCandidate(view)
	if candidate.Verdict != change.VerdictNeedsReview || gateStatus(candidate, "independent_review") != change.GateNeedsReview {
		t.Fatalf("self-review must not count: %s %+v", candidate.Verdict, candidate.Gates)
	}
	if _, err := h.changes.DecideCandidate("demo", "CHG-001", "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", Rationale: "looks fine"}); codeOf(t, err) != change.CodeCandidateBlocked {
		t.Fatalf("NEEDS_REVIEW cannot be accepted: %v", err)
	}
	// Single-operator policy: AI review + declared controls → WAIVED, acceptable.
	observation.Reviews = []change.ReviewEvidence{{Reviewer: "ctr-ai-review", Kind: "ai", Verdict: "APPROVED", EvidenceRef: "evidence/code-review.md"}}
	observation.CompensatingControls = &change.CompensatingControls{Declared: true, EvidenceRef: "evidence/single-operator-controls.md", Reason: "solo founder, low-risk staging"}
	view, err = h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("resubmitted with controls"), Run: goodRun(order), Observation: observation})
	if err != nil {
		t.Fatal(err)
	}
	candidate = lastCandidate(view)
	if candidate.CandidateID != "CAND-002" || candidate.Verdict != change.VerdictAcceptable || gateStatus(candidate, "independent_review") != change.GateWaived {
		t.Fatalf("expected CAND-002 acceptable with WAIVED review: %s %s", candidate.Verdict, gateStatus(candidate, "independent_review"))
	}
	// With the waiver the single operator may accept their own candidate.
	view, err = h.changes.DecideCandidate("demo", "CHG-001", "CAND-002", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "dev-1"}, Decision: "ACCEPT", Rationale: "single-operator controls documented"})
	if err != nil || lastCandidate(view).Status != "ACCEPTED_FOR_PROMOTION" {
		t.Fatalf("waived self-acceptance should pass: %v", err)
	}
}

func TestCandidateMissingChecksNeedsEvidenceAndFailedCheckBlocks(t *testing.T) {
	h := newHarness(t)
	order := acceptedWithOrder(t, h)
	observation := goodObservation(order)
	observation.Checks = []change.CheckResult{{Name: "test", Status: "PASS"}}
	view, _ := h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("partial"), Run: goodRun(order), Observation: observation})
	candidate := lastCandidate(view)
	if candidate.Verdict != change.VerdictNeedsEvidence || gateStatus(candidate, "required_checks") != change.GateNeedsEvidence {
		t.Fatalf("missing build must be NEEDS_EVIDENCE: %s %+v", candidate.Verdict, candidate.Gates)
	}
	observation.Checks = append(observation.Checks, change.CheckResult{Name: "build", Status: "FAIL"})
	view, _ = h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("build failed"), Run: goodRun(order), Observation: observation})
	candidate = lastCandidate(view)
	if candidate.Verdict != change.VerdictBlocked || gateStatus(candidate, "required_checks") != change.GateBlocked {
		t.Fatalf("failed build must BLOCK: %s", candidate.Verdict)
	}
}

func TestCandidateContractMismatchesBlock(t *testing.T) {
	h := newHarness(t)
	order := acceptedWithOrder(t, h)
	run := goodRun(order)
	run.WorkOrderHash = "sha256:tampered"
	view, _ := h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("x"), Run: run, Observation: goodObservation(order)})
	if c := lastCandidate(view); c.Verdict != change.VerdictBlocked || gateStatus(c, "work_order_bound") != change.GateBlocked {
		t.Fatalf("hash mismatch must block: %+v", c.Gates)
	}
	observation := goodObservation(order)
	observation.Branch = "feature/somewhere-else"
	view, _ = h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("x"), Run: goodRun(order), Observation: observation})
	if c := lastCandidate(view); gateStatus(c, "branch_matches") != change.GateBlocked {
		t.Fatalf("wrong branch must block: %+v", c.Gates)
	}
	run = goodRun(order)
	run.StartedAt = "2027-01-01T00:00:00Z"
	view, _ = h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("x"), Run: run, Observation: goodObservation(order)})
	if c := lastCandidate(view); gateStatus(c, "work_order_expiry") != change.GateBlocked {
		t.Fatalf("expired order must block: %+v", c.Gates)
	}
}

func TestCandidateForbiddenModifyPrefixBlocks(t *testing.T) {
	h := newHarness(t)
	request := completeRequest()
	request.AllowedPaths = append(request.AllowedPaths, "infra/")
	request.ForbiddenActions = append(request.ForbiddenActions, "modify:infra/iam")
	_, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: request})
	_, _ = h.changes.Decide("demo", "CHG-001", change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "ok"})
	view, _ := h.changes.CompileWorkOrder("demo", "CHG-001", change.WorkOrderRequest{Mutation: mutation("issue")})
	order := view.WorkOrder
	observation := goodObservation(order)
	observation.ChangedPaths = []string{"main.go", "infra/iam/public-bucket.yaml"}
	view, _ = h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("x"), Run: goodRun(order), Observation: observation})
	c := lastCandidate(view)
	if gateStatus(c, "allowed_paths") != change.GatePass || gateStatus(c, "forbidden_actions") != change.GateBlocked || c.Verdict != change.VerdictBlocked {
		t.Fatalf("path inside allowed_paths but under a modify: forbidden prefix must block via forbidden_actions: %+v", c.Gates)
	}
}

func TestCandidateUnderStaleDecisionIsBlockedAndShownStale(t *testing.T) {
	h := newHarness(t)
	order := acceptedWithOrder(t, h)
	view, _ := h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("ok"), Run: goodRun(order), Observation: goodObservation(order)})
	if lastCandidate(view).Verdict != change.VerdictAcceptable {
		t.Fatal("precondition: acceptable")
	}
	target := "cloud-run/demo-staging-2"
	if _, err := h.topology.Edit("demo", "staging", topology.EditRequest{Mutation: topology.Mutation{ExpectedVersion: 1, Actor: "ops", Reason: "move"}, TargetRef: &target}); err != nil {
		t.Fatal(err)
	}
	view, _ = h.changes.Get("demo", "CHG-001")
	if lastCandidate(view).Status != "STALE" {
		t.Fatalf("evaluated candidate must show STALE once the decision is stale: %s", lastCandidate(view).Status)
	}
	if _, err := h.changes.DecideCandidate("demo", "CHG-001", "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", Rationale: "go"}); codeOf(t, err) != change.CodeDecisionStale {
		t.Fatalf("stale decision must block acceptance: %v", err)
	}
	view, _ = h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("retry"), Run: goodRun(order), Observation: goodObservation(order)})
	if c := lastCandidate(view); c.Verdict != change.VerdictBlocked || gateStatus(c, "decision_current") != change.GateBlocked {
		t.Fatalf("new candidate under stale decision must be BLOCKED: %+v", c.Gates)
	}
}

func TestCandidateRequiresIssuedWorkOrder(t *testing.T) {
	h := newHarness(t)
	_, _ = h.changes.Create("demo", change.CreateRequest{Mutation: mutation("open"), Request: completeRequest()})
	_, err := h.changes.SubmitCandidate("demo", "CHG-001", change.SubmitCandidateRequest{Mutation: mutation("x"), Run: change.AgentRun{}, Observation: change.Observation{}})
	if codeOf(t, err) != change.CodeNotAccepted {
		t.Fatalf("candidate without work order must be refused: %v", err)
	}
}
