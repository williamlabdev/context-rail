package release_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"context-rail/internal/change"
	"context-rail/internal/release"
	"context-rail/internal/topology"
)

// harness wires topology → changes → releases with in-memory stores.
type harness struct {
	topology       *topology.Service
	changes        *change.Service
	releases       *release.Service
	singleOperator bool
}

type topologySource struct{ singleOperator bool }

func (source topologySource) Baseline(projectID string) (*topology.Baseline, error) {
	if projectID != "demo" {
		return nil, nil
	}
	staging := []string{"decision-record", "review", "test", "build", "smoke"}
	if source.singleOperator {
		staging = append(staging, "single-operator-controls")
	}
	return &topology.Baseline{ProjectID: "demo", Environments: []topology.Environment{
		{ID: "development", Type: "development", Sequence: 1, TargetRef: "local", RequiredEvidence: []string{"local-test"}},
		{ID: "testing", Type: "testing", Sequence: 2, TargetRef: "ci", RequiredEvidence: []string{"test", "build"}},
		{ID: "staging", Type: "staging", Sequence: 3, TargetRef: "cloud-run/demo-staging", RequiredEvidence: staging},
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
		ProjectID: "demo", ProjectName: "Demo", RepositoryURL: "github.com/example/demo", DefaultBranch: "develop",
		Documents:     []change.DocumentFact{{Path: "project.yaml", Status: "CURRENT", SourceOfTruth: true}},
		ContextStatus: "DERIVED", DecisionInputs: change.ReadinessFact{Status: "READY", Reasons: []string{}}, Topology: state,
	}, nil
}

func newHarness(t *testing.T, singleOperator bool) *harness {
	t.Helper()
	h := &harness{singleOperator: singleOperator}
	tick := time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time { tick = tick.Add(time.Second); return tick }
	h.topology = topology.NewService(topology.NewMemoryStore(), topologySource{singleOperator}).WithClock(clock)
	h.changes = change.NewService(change.NewMemoryStore(), h, nil).WithClock(clock)
	h.releases = release.NewService(release.NewMemoryStore(), h.changes, h.topology).WithClock(clock)
	return h
}

// acceptedChange drives a change through decision → work order → accepted candidate.
func (h *harness) acceptedChange(t *testing.T, title, headCommit string, reviewer string) string {
	t.Helper()
	view, err := h.changes.Create("demo", change.CreateRequest{Mutation: change.Mutation{Actor: "requester", Reason: "open"}, Request: change.Request{
		Title: title, Objective: "do " + title, AcceptanceCriteria: []change.AcceptanceCriterion{{Text: "works"}},
		AllowedPaths: []string{"main.go", "web/"}, TargetEnvironmentID: "staging",
		BusinessConstraints: map[string]string{"data_classification": "internal", "expected_monthly_volume": "100"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	id := view.Change.ChangeID
	if _, err := h.changes.Decide("demo", id, change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "ok"}); err != nil {
		t.Fatal(err)
	}
	view, err = h.changes.CompileWorkOrder("demo", id, change.WorkOrderRequest{Mutation: change.Mutation{Actor: "founder-001", Reason: "issue"}})
	if err != nil {
		t.Fatal(err)
	}
	order := view.WorkOrder
	observation := change.Observation{
		Branch: order.TargetBranch, BaseBranch: order.BaseBranch, HeadCommit: headCommit, ChangedPaths: []string{"main.go"},
		Checks:  []change.CheckResult{{Name: "test", Status: "PASS"}, {Name: "build", Status: "PASS"}},
		Reviews: []change.ReviewEvidence{{Reviewer: reviewer, Kind: "human", Verdict: "APPROVED"}},
	}
	if _, err := h.changes.SubmitCandidate("demo", id, change.SubmitCandidateRequest{Mutation: change.Mutation{Actor: "dev-1", Reason: "run done"}, Run: change.AgentRun{RunID: "ARR-" + id, WorkOrderID: order.WorkOrderID, WorkOrderHash: order.WorkOrderHash, StartedBy: "dev-1", StartedAt: "2026-09-21T08:00:00Z", Agent: change.AgentIdentity{Name: "claude-code"}}, Observation: observation}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.changes.DecideCandidate("demo", id, "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", Rationale: "ok"}); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	return id
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var typed *release.Error
	if !errors.As(err, &typed) {
		t.Fatalf("expected release.Error, got %v", err)
	}
	return typed.Code
}

func gate(view *release.View, name string) release.GateResult {
	for _, result := range view.LiveGates {
		if result.Gate == name {
			return result
		}
	}
	return release.GateResult{Gate: name, Status: "ABSENT"}
}

func lastAttempt(view *release.View) release.DeploymentRecord {
	return view.Release.Deployments[len(view.Release.Deployments)-1]
}

func attemptGate(record release.DeploymentRecord, name string) string {
	for _, result := range record.Gates {
		if result.Gate == name {
			return result.Status
		}
	}
	return "ABSENT"
}

const digest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"

func mutation(reason string) release.Mutation {
	return release.Mutation{Actor: "release-mgr", Reason: reason}
}

func TestNormalPromotionProducesReceiptWithFullLineage(t *testing.T) {
	h := newHarness(t, false)
	changeID := h.acceptedChange(t, "manual review", "abc123", "reviewer-2")

	view, err := h.releases.Create("demo", release.CreateRequest{Mutation: mutation("release 1"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.ReleaseID != "REL-001" || view.Release.Status != release.StatusAwaitingBuild {
		t.Fatalf("expected REL-001 AWAITING_BUILD, got %s %s: %+v", view.Release.ReleaseID, view.Release.Status, view.Release.Gates)
	}
	manifest := view.Release.Manifest
	if manifest.Transition != "testing-to-staging" || manifest.Environment.TargetRef != "cloud-run/demo-staging" || manifest.Environment.ConfigHash == "" || manifest.ManifestHash == "" {
		t.Fatalf("manifest incomplete: %+v", manifest)
	}
	if manifest.Changes[0].HeadCommit != "abc123" || manifest.Changes[0].CandidateID != "CAND-001" || manifest.Changes[0].DecisionID != "DEC-001" {
		t.Fatalf("manifest must carry change lineage: %+v", manifest.Changes[0])
	}
	// Approval before build is refused.
	if _, err := h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "go"}); codeOf(t, err) != release.CodeNotApprovable {
		t.Fatalf("approval without build must be refused: %v", err)
	}
	// Build from the wrong commit is refused.
	if _, err := h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: digest, SourceCommit: "zzz999"}); codeOf(t, err) != release.CodeBuildMismatch {
		t.Fatalf("build from another commit must be refused: %v", err)
	}
	if _, err := h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: "latest", SourceCommit: "abc123"}); codeOf(t, err) != release.CodeInvalidRequest {
		t.Fatalf("mutable tag must be refused: %v", err)
	}
	view, err = h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("cloud build finished"), ImageDigest: digest, ImageRef: "asia-east1-docker.pkg.dev/p/r/app", BuildID: "b-1", SourceCommit: "abc123", EvidenceRef: "cloudbuild/b-1"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusReadyForApproval {
		t.Fatalf("expected READY_FOR_APPROVAL, got %s", view.Release.Status)
	}
	boundHash := view.Release.Manifest.ManifestHash
	// Deploying before approval is recorded as a failed promotion, not a receipt of success.
	view, err = h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{Mutation: mutation("oops deployed early"), IdempotencyKey: "k-early", Revision: "app-00001", DeployedDigest: digest, DeployedTargetRef: "cloud-run/demo-staging", Smoke: &release.SmokeResult{Status: "PASS"}})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusPromotionFailed || attemptGate(lastAttempt(view), "release_approval") != release.GateBlocked {
		t.Fatalf("deployment without approval must be PROMOTION_FAILED: %s %+v", view.Release.Status, lastAttempt(view).Gates)
	}
	// Third human decision.
	view, err = h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Role: "release_manager", Rationale: "manifest reviewed"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusApproved || view.Release.Approval.ManifestHash != boundHash || view.Release.Approval.ExpiresAt == "" {
		t.Fatalf("approval must bind the manifest hash: %+v", view.Release.Approval)
	}
	// UI-12: deployment matches → PROMOTED with a receipt linking everything.
	view, err = h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{
		Mutation: mutation("deployed via scripts/deploy-cloud-run.sh"), IdempotencyKey: "k-1", OperationID: "op-1", Revision: "app-00002", ServiceURL: "https://app.a.run.app",
		DeployedDigest: digest, DeployedTargetRef: "cloud-run/demo-staging", Smoke: &release.SmokeResult{Status: "PASS", EvidenceRef: "smoke.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusPromoted || view.Release.Receipt == nil || view.Release.Receipt.Status != release.StatusPromoted {
		t.Fatalf("expected PROMOTED with receipt, got %s %+v", view.Release.Status, lastAttempt(view).Gates)
	}
	receipt := view.Release.Receipt
	if receipt.ReceiptID != "RR-002" || receipt.ReceiptHash == "" || receipt.Deployment.Revision != "app-00002" || receipt.Build.ImageDigest != digest || receipt.Approval.Actor != "approver-1" || receipt.Changes[0].DecisionID != "DEC-001" || receipt.Changes[0].WorkOrderID != "AWO-001" || receipt.Environment.ConfigHash == "" {
		t.Fatalf("receipt must link requirement, decision, work order, candidate, digest, target config, revision and approval: %+v", receipt)
	}
	for _, name := range []string{"digest_drift", "target_drift", "revision_observed", "smoke", "release_approval"} {
		if attemptGate(lastAttempt(view), name) != release.GatePass {
			t.Fatalf("gate %s should PASS: %+v", name, lastAttempt(view).Gates)
		}
	}
	// Idempotent replay: same key → same state, no new attempt.
	replay, err := h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{Mutation: mutation("retry after timeout"), IdempotencyKey: "k-1", Revision: "app-00002", DeployedDigest: digest, DeployedTargetRef: "cloud-run/demo-staging"})
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.Release.Deployments) != 2 || replay.Release.Receipt.ReceiptID != "RR-002" {
		t.Fatalf("replay must not add an attempt or receipt: %d attempts, receipt %s", len(replay.Release.Deployments), replay.Release.Receipt.ReceiptID)
	}
	// A second deployment with a new key is refused: the release is done.
	if _, err := h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{Mutation: mutation("again"), IdempotencyKey: "k-2", Revision: "app-00003", DeployedDigest: digest, DeployedTargetRef: "cloud-run/demo-staging"}); codeOf(t, err) != release.CodeAlreadyPromoted {
		t.Fatalf("double promotion must be refused: %v", err)
	}
}

func TestBundleWithOneFailingChangeIsBlockedAsAWhole(t *testing.T) {
	h := newHarness(t, false)
	good := h.acceptedChange(t, "good change", "aaa111", "reviewer-2")
	// Second change: decided and issued but candidate NOT accepted.
	view, _ := h.changes.Create("demo", change.CreateRequest{Mutation: change.Mutation{Actor: "requester", Reason: "open"}, Request: change.Request{
		Title: "half done", Objective: "x", AcceptanceCriteria: []change.AcceptanceCriterion{{Text: "works"}}, AllowedPaths: []string{"main.go"}, TargetEnvironmentID: "staging",
		BusinessConstraints: map[string]string{"data_classification": "internal", "expected_monthly_volume": "1"},
	}})
	halfDone := view.Change.ChangeID
	_, _ = h.changes.Decide("demo", halfDone, change.DecideRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", SelectedOption: "minimal_reversible_slice", Rationale: "ok"})
	_, _ = h.changes.CompileWorkOrder("demo", halfDone, change.WorkOrderRequest{Mutation: change.Mutation{Actor: "founder-001", Reason: "issue"}})

	rel, err := h.releases.Create("demo", release.CreateRequest{Mutation: mutation("bundle"), ChangeIDs: []string{good, halfDone}, TargetEnvironmentID: "staging"})
	if err != nil {
		t.Fatal(err)
	}
	if rel.Release.Status != release.StatusGateBlocked {
		t.Fatalf("bundle with a failing change must be GATE_BLOCKED: %s", rel.Release.Status)
	}
	statuses := map[string]string{}
	for _, result := range rel.LiveGates {
		if result.Gate == "change" {
			statuses[result.ChangeID] = result.Status
		}
	}
	if statuses[good] != release.GatePass || statuses[halfDone] != release.GateBlocked {
		t.Fatalf("per-change gates must show which change blocks: %v", statuses)
	}
	if _, err := h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: digest, SourceCommit: "merge1", IncludesCommits: []string{"aaa111"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "ship the good one"}); codeOf(t, err) != release.CodeNotApprovable {
		t.Fatalf("no partial promotion: approval must be refused while one change blocks: %v", err)
	}
	// Once the second candidate is accepted, the same bundle can proceed.
	order, _ := h.changes.Get("demo", halfDone)
	_, _ = h.changes.SubmitCandidate("demo", halfDone, change.SubmitCandidateRequest{Mutation: change.Mutation{Actor: "dev-1", Reason: "run"}, Run: change.AgentRun{RunID: "ARR-2", WorkOrderID: order.WorkOrder.WorkOrderID, WorkOrderHash: order.WorkOrder.WorkOrderHash, StartedBy: "dev-1", StartedAt: "2026-09-21T08:00:00Z"}, Observation: change.Observation{Branch: order.WorkOrder.TargetBranch, HeadCommit: "bbb222", ChangedPaths: []string{"main.go"}, Checks: []change.CheckResult{{Name: "test", Status: "PASS"}, {Name: "build", Status: "PASS"}}, Reviews: []change.ReviewEvidence{{Reviewer: "reviewer-2", Kind: "human", Verdict: "APPROVED"}}}})
	if _, err := h.changes.DecideCandidate("demo", halfDone, "CAND-001", change.CandidateDecisionRequest{Mutation: change.Mutation{Actor: "founder-001"}, Decision: "ACCEPT", Rationale: "ok"}); err != nil {
		t.Fatal(err)
	}
	rel, _ = h.releases.Get("demo", "REL-001")
	if gate(rel, "change").Status == release.GateBlocked {
		t.Fatalf("live gates must reflect the now-accepted candidate: %+v", rel.LiveGates)
	}
	// The build recorded earlier did not include bbb222, so approval must still refuse via the build check on re-record.
	if _, err := h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("rebuilt"), ImageDigest: digest, SourceCommit: "merge2", IncludesCommits: []string{"aaa111"}}); codeOf(t, err) != release.CodeBuildMismatch {
		t.Fatalf("build must include both commits: %v", err)
	}
	if _, err := h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("rebuilt"), ImageDigest: digest, SourceCommit: "merge2", IncludesCommits: []string{"aaa111", "bbb222"}}); err != nil {
		t.Fatal(err)
	}
	rel, err = h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "both changes accepted"})
	if err != nil || rel.Release.Status != release.StatusApproved {
		t.Fatalf("bundle should now be approvable: %v %s", err, rel.Release.Status)
	}
}

func TestTargetConfigDriftAfterApprovalMakesApprovalStaleAndBlocksPromotion(t *testing.T) {
	h := newHarness(t, false)
	changeID := h.acceptedChange(t, "drift", "abc123", "reviewer-2")
	_, _ = h.releases.Create("demo", release.CreateRequest{Mutation: mutation("release"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"})
	_, _ = h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: digest, SourceCommit: "abc123"})
	if _, err := h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "ok"}); err != nil {
		t.Fatal(err)
	}
	// Operator changes the staging target in the topology after approval.
	target := "cloud-run/demo-staging-blue"
	if _, err := h.topology.Edit("demo", "staging", topology.EditRequest{Mutation: topology.Mutation{ExpectedVersion: 1, Actor: "ops", Reason: "blue/green"}, TargetRef: &target}); err != nil {
		t.Fatal(err)
	}
	view, _ := h.releases.Get("demo", "REL-001")
	if !view.Staleness.Stale || view.Release.Status != release.StatusStale || view.Release.Approval.Decision != "STALE" || gate(view, "target_config_drift").Status != release.GateStale {
		t.Fatalf("UI-14: drift must show STALE approval: stale=%v status=%s approval=%s gates=%+v", view.Staleness.Stale, view.Release.Status, view.Release.Approval.Decision, view.LiveGates)
	}
	// Deploying anyway (to the old target) is recorded as a failed promotion and the stored approval is invalidated.
	view, err := h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{Mutation: mutation("deployed"), IdempotencyKey: "k-1", Revision: "r1", DeployedDigest: digest, DeployedTargetRef: "cloud-run/demo-staging", Smoke: &release.SmokeResult{Status: "PASS"}})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusPromotionFailed || view.Release.Receipt.Status != release.StatusPromotionFailed {
		t.Fatalf("expected PROMOTION_FAILED receipt, got %s", view.Release.Status)
	}
	if attemptGate(lastAttempt(view), "target_config_drift") != release.GateStale {
		t.Fatalf("environment drift must be visible on the attempt: %+v", lastAttempt(view).Gates)
	}
	if view.Release.Approval.Decision != "STALE" {
		t.Fatal("stored approval must be marked STALE after drift")
	}
	// Re-approval binds the new manifest? No: the manifest still holds the old
	// environment config; a new release must be cut against the new topology.
	if _, err := h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "re-approve"}); codeOf(t, err) != release.CodeNotApprovable {
		t.Fatalf("stale manifest cannot simply be re-approved: %v", err)
	}
	fresh, err := h.releases.Create("demo", release.CreateRequest{Mutation: mutation("release 2 against v2"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"})
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Release.Manifest.Environment.TargetRef != target || fresh.Release.Manifest.Environment.TopologyVersion != 2 {
		t.Fatalf("new release must bind the current topology: %+v", fresh.Release.Manifest.Environment)
	}
	// ...but the change decision bound topology v1 and the target changed materially, so the change itself is STALE.
	if gate(fresh, "change").Status != release.GateStale {
		t.Fatalf("change decided against v1 must be STALE after a material topology change: %+v", fresh.LiveGates)
	}
}

func TestDigestDriftAndFailedSmokeBlockPromotion(t *testing.T) {
	h := newHarness(t, false)
	changeID := h.acceptedChange(t, "digest", "abc123", "reviewer-2")
	_, _ = h.releases.Create("demo", release.CreateRequest{Mutation: mutation("release"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"})
	_, _ = h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: digest, SourceCommit: "abc123"})
	_, _ = h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "ok"})
	other := "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	view, err := h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{Mutation: mutation("deployed wrong image"), IdempotencyKey: "k-1", Revision: "r1", DeployedDigest: other, DeployedTargetRef: "cloud-run/demo-staging", Smoke: &release.SmokeResult{Status: "PASS"}})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusPromotionFailed || attemptGate(lastAttempt(view), "digest_drift") != release.GateBlocked {
		t.Fatalf("digest drift must fail promotion: %s %+v", view.Release.Status, lastAttempt(view).Gates)
	}
	// Approval is untouched by digest drift (not a target/config drift); a corrected deploy can proceed.
	if view.Release.Approval.Decision != "APPROVED" {
		t.Fatalf("digest drift should not invalidate the approval: %s", view.Release.Approval.Decision)
	}
	view, err = h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{Mutation: mutation("redeployed right image, smoke failed"), IdempotencyKey: "k-2", Revision: "r2", DeployedDigest: digest, DeployedTargetRef: "cloud-run/demo-staging", Smoke: &release.SmokeResult{Status: "FAIL", EvidenceRef: "smoke-2.txt"}})
	if err != nil || view.Release.Status != release.StatusPromotionFailed || attemptGate(lastAttempt(view), "smoke") != release.GateBlocked {
		t.Fatalf("failed smoke must fail promotion: %v %s", err, view.Release.Status)
	}
	view, err = h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{Mutation: mutation("smoke fixed"), IdempotencyKey: "k-3", Revision: "r2", DeployedDigest: digest, DeployedTargetRef: "cloud-run/demo-staging", Smoke: &release.SmokeResult{Status: "PASS", EvidenceRef: "smoke-3.txt"}})
	if err != nil || view.Release.Status != release.StatusPromoted || len(view.Release.Deployments) != 3 {
		t.Fatalf("third attempt should promote: %v %s (%d attempts)", err, view.Release.Status, len(view.Release.Deployments))
	}
}

func TestApproverSeparation(t *testing.T) {
	strict := newHarness(t, false)
	changeID := strict.acceptedChange(t, "sep", "abc123", "reviewer-2")
	_, _ = strict.releases.Create("demo", release.CreateRequest{Mutation: mutation("release"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"})
	_, _ = strict.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: digest, SourceCommit: "abc123"})
	if _, err := strict.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "dev-1"}, Decision: "APPROVE", Rationale: "my own run"}); codeOf(t, err) != release.CodeApproverConflict {
		t.Fatalf("run starter cannot approve without single-operator policy: %v", err)
	}
	if _, err := strict.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: ""}, Decision: "APPROVE", Rationale: "x"}); codeOf(t, err) != release.CodeActorRequired {
		t.Fatalf("anonymous approval refused: %v", err)
	}

	solo := newHarness(t, true)
	changeID = solo.acceptedChange(t, "solo", "abc123", "reviewer-2")
	_, _ = solo.releases.Create("demo", release.CreateRequest{Mutation: mutation("release"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"})
	_, _ = solo.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: digest, SourceCommit: "abc123"})
	view, err := solo.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "dev-1"}, Decision: "APPROVE", Rationale: "single operator"})
	if err != nil || view.Release.Approval.Waiver == "" {
		t.Fatalf("single-operator policy must allow with a visible waiver: %v %+v", err, view.Release.Approval)
	}
	if _, err := solo.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "dev-1"}, Decision: "APPROVE", Rationale: "again"}); codeOf(t, err) != release.CodeAlreadyApproved {
		t.Fatalf("double approval refused: %v", err)
	}
	// A new build after approval invalidates it.
	view, err = solo.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("rebuilt"), ImageDigest: "sha256:3333333333333333333333333333333333333333333333333333333333333333", SourceCommit: "abc123"})
	if err != nil || view.Release.Approval != nil || view.Release.Status != release.StatusReadyForApproval {
		t.Fatalf("new build must drop the approval: %v %+v %s", err, view.Release.Approval, view.Release.Status)
	}
}

func TestReleaseValidation(t *testing.T) {
	h := newHarness(t, false)
	if _, err := h.releases.Create("demo", release.CreateRequest{Mutation: mutation("x"), ChangeIDs: []string{}, TargetEnvironmentID: "staging"}); codeOf(t, err) != release.CodeInvalidRequest {
		t.Fatalf("empty bundle: %v", err)
	}
	if _, err := h.releases.Create("nope", release.CreateRequest{Mutation: mutation("x"), ChangeIDs: []string{"CHG-001"}, TargetEnvironmentID: "staging"}); codeOf(t, err) != release.CodeProjectNotFound {
		t.Fatalf("unknown project: %v", err)
	}
	changeID := h.acceptedChange(t, "prod", "abc123", "reviewer-2")
	view, err := h.releases.Create("demo", release.CreateRequest{Mutation: mutation("to prod"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "production"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusGateBlocked || !strings.Contains(gate(view, "environment").Detail, "production") {
		t.Fatalf("production release must be blocked: %s %+v", view.Release.Status, view.LiveGates)
	}
	if _, err := h.releases.Get("demo", "REL-009"); codeOf(t, err) != release.CodeReleaseNotFound {
		t.Fatalf("unknown release: %v", err)
	}
}

func TestDeploymentWithoutObservedDigestDoesNotPromote(t *testing.T) {
	h := newHarness(t, false)
	changeID := h.acceptedChange(t, "no digest", "abc123", "reviewer-2")
	_, _ = h.releases.Create("demo", release.CreateRequest{Mutation: mutation("release"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"})
	_, _ = h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: digest, SourceCommit: "abc123"})
	_, _ = h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "ok"})
	view, err := h.releases.RecordDeployment("demo", "REL-001", release.DeploymentRequest{Mutation: mutation("deployed, digest unknown"), IdempotencyKey: "k-1", Revision: "r1", DeployedTargetRef: "cloud-run/demo-staging", Smoke: &release.SmokeResult{Status: "PASS"}})
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusPromotionFailed || attemptGate(lastAttempt(view), "digest_drift") != release.GateNeedsEvidence {
		t.Fatalf("missing digest must not promote: %s %+v", view.Release.Status, lastAttempt(view).Gates)
	}
}
