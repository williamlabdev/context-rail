package release_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context-rail/internal/release"
	"context-rail/internal/topology"
)

// fakeObserver answers like a running local instance would, and counts calls
// so a replay can be shown not to re-read the service.
type fakeObserver struct {
	answers map[string]*release.LiveObservation
	err     error
	calls   int
}

func (f *fakeObserver) Name() string { return "fake" }

func (f *fakeObserver) Observe(ctx context.Context, serviceURL string) (*release.LiveObservation, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if answer, ok := f.answers[serviceURL]; ok {
		copied := *answer
		return &copied, nil
	}
	return nil, errors.New("unreachable")
}

func localInstance(environment, revision, buildDigest string) *release.LiveObservation {
	return &release.LiveObservation{HTTPStatus: 200, Status: "ok", Service: "demo", Environment: environment, Revision: revision, BuildDigest: buildDigest, TargetRef: release.LocalTargetRef("demo", environment), ObservedAt: "2026-10-26T00:00:00Z"}
}

// localHarness is the standard harness with staging retargeted to a local
// instance and an isolated local prod-demo added through the topology API,
// exactly as the EB-017 run does over HTTP (project.yaml stays untouched).
func localHarness(t *testing.T, observer release.Observer) *harness {
	t.Helper()
	h := newHarness(t, false)
	h.releases.WithObserver(observer)
	stagingTarget := release.LocalTargetRef("demo", "staging")
	if _, err := h.topology.Edit("demo", "staging", topology.EditRequest{Mutation: topology.Mutation{ExpectedVersion: 1, Actor: "ops", Reason: "DR-015: staging is a local instance"}, TargetRef: &stagingTarget}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.topology.Add("demo", topology.AddRequest{
		Mutation: topology.Mutation{ExpectedVersion: 2, Actor: "ops", Reason: "DR-015: isolated local prod-demo"},
		ID:       "prod-demo", DisplayName: "Prod demo (local)", Type: "prod-demo", Sequence: 4, TargetRef: release.LocalTargetRef("demo", "prod-demo"),
		RequiredEvidence: []string{"release-approval", "staging-receipt", "smoke"}, ApproverPolicy: "distinct human approver",
	}); err != nil {
		t.Fatal(err)
	}
	return h
}

func approvedStagingRelease(t *testing.T, h *harness) {
	t.Helper()
	changeID := h.acceptedChange(t, "local promotion", "abc123", "reviewer-2")
	if _, err := h.releases.Create("demo", release.CreateRequest{Mutation: mutation("staging release"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.releases.RecordBuild("demo", "REL-001", release.BuildRequest{Mutation: mutation("built"), ImageDigest: digest, ImageRef: "file://bin/demo", SourceCommit: "abc123", BuildID: "local-1", EvidenceRef: "shasum"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.releases.Approve("demo", "REL-001", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "staging ok"}); err != nil {
		t.Fatal(err)
	}
}

func localDeployment(key, serviceURL string) release.DeploymentRequest {
	return release.DeploymentRequest{Mutation: mutation("observed local deployment"), IdempotencyKey: key, Source: release.SourceLocal, ServiceURL: serviceURL}
}

func TestLocalDeploymentIsObservedNotDeclared(t *testing.T) {
	observer := &fakeObserver{answers: map[string]*release.LiveObservation{"http://127.0.0.1:8091": localInstance("staging", "staging-r1", digest)}}
	h := localHarness(t, observer)
	approvedStagingRelease(t, h)

	// Declaring identity on a local record is refused: the read-back owns it.
	for name, declare := range map[string]func(*release.DeploymentRequest){
		"revision":             func(r *release.DeploymentRequest) { r.Revision = "staging-r1" },
		"deployed_digest":      func(r *release.DeploymentRequest) { r.DeployedDigest = digest },
		"deployed_target_ref":  func(r *release.DeploymentRequest) { r.DeployedTargetRef = "local/demo-staging" },
		"deployed_config_hash": func(r *release.DeploymentRequest) { r.DeployedConfigHash = "sha256:declared" },
		"smoke":                func(r *release.DeploymentRequest) { r.Smoke = &release.SmokeResult{Status: "PASS"} },
	} {
		declared := localDeployment("k-declared-"+name, "http://127.0.0.1:8091")
		declare(&declared)
		if _, err := h.releases.RecordDeployment("demo", "REL-001", declared); codeOf(t, err) != release.CodeInvalidRequest {
			t.Fatalf("declared %s on a local deployment must be refused: %v", name, err)
		}
	}
	if _, err := h.releases.RecordDeployment("demo", "REL-001", localDeployment("k-nourl", "")); codeOf(t, err) != release.CodeInvalidRequest {
		t.Fatalf("missing service_url must be refused: %v", err)
	}
	if observer.calls != 0 {
		t.Fatalf("refused requests must not read the service: %d calls", observer.calls)
	}

	view, err := h.releases.RecordDeployment("demo", "REL-001", localDeployment("k-staging", "http://127.0.0.1:8091"))
	if err != nil {
		t.Fatal(err)
	}
	attempt := lastAttempt(view)
	if view.Release.Status != release.StatusPromoted || attempt.Source != release.SourceLocal || attempt.Observation == nil {
		t.Fatalf("observed local deployment must promote with its observation stored: %s %+v", view.Release.Status, attempt.Gates)
	}
	if attempt.Revision != "staging-r1" || attempt.DeployedDigest != digest || attempt.DeployedTargetRef != "local/demo-staging" || attempt.Smoke == nil || attempt.Smoke.Status != "PASS" || !strings.HasPrefix(attempt.Smoke.EvidenceRef, "healthz:http://127.0.0.1:8091") {
		t.Fatalf("identity must come from the read-back: %+v", attempt)
	}
	for _, name := range []string{"digest_drift", "target_drift", "revision_observed", "smoke"} {
		if attemptGate(attempt, name) != release.GatePass {
			t.Fatalf("gate %s must PASS: %+v", name, attempt.Gates)
		}
	}
	if view.Release.Receipt == nil || view.Release.Receipt.Deployment.Observation == nil || view.Release.Receipt.Deployment.Observation.Revision != "staging-r1" {
		t.Fatalf("the receipt must carry the observation: %+v", view.Release.Receipt)
	}
}

func TestLocalDeploymentReplayDoesNotReObserve(t *testing.T) {
	observer := &fakeObserver{answers: map[string]*release.LiveObservation{"http://127.0.0.1:8091": localInstance("staging", "staging-r1", digest)}}
	h := localHarness(t, observer)
	approvedStagingRelease(t, h)
	first, err := h.releases.RecordDeployment("demo", "REL-001", localDeployment("k-same", "http://127.0.0.1:8091"))
	if err != nil {
		t.Fatal(err)
	}
	// The instance restarts (new revision) but the retry carries the same key.
	observer.answers["http://127.0.0.1:8091"] = localInstance("staging", "staging-r2", digest)
	again, err := h.releases.RecordDeployment("demo", "REL-001", localDeployment("k-same", "http://127.0.0.1:8091"))
	if err != nil {
		t.Fatal(err)
	}
	if observer.calls != 1 {
		t.Fatalf("a replay must not re-read the service: %d calls", observer.calls)
	}
	if len(again.Release.Deployments) != 1 || again.Release.Receipt.ReceiptID != first.Release.Receipt.ReceiptID || again.Release.Receipt.ReceiptHash != first.Release.Receipt.ReceiptHash || lastAttempt(again).Revision != "staging-r1" {
		t.Fatalf("replay must return the same attempt and receipt: %+v", again.Release.Receipt)
	}
}

func TestLocalDeploymentDriftIsCaughtFromTheReadBack(t *testing.T) {
	other := "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	observer := &fakeObserver{answers: map[string]*release.LiveObservation{
		"http://127.0.0.1:8091": localInstance("staging", "staging-r1", digest),
		"http://127.0.0.1:8092": localInstance("prod-demo", "prod-demo-r1", digest),
		"http://127.0.0.1:8093": localInstance("prod-demo", "prod-demo-drift", other),
	}}
	h := localHarness(t, observer)
	approvedStagingRelease(t, h)
	if _, err := h.releases.RecordDeployment("demo", "REL-001", localDeployment("k-staging", "http://127.0.0.1:8091")); err != nil {
		t.Fatal(err)
	}
	if _, err := h.releases.Create("demo", release.CreateRequest{Mutation: mutation("promote"), SourceReleaseID: "REL-001", TargetEnvironmentID: "prod-demo"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.releases.Approve("demo", "REL-002", release.ApprovalRequest{Mutation: release.Mutation{Actor: "approver-1"}, Decision: "APPROVE", Rationale: "delta ok"}); err != nil {
		t.Fatal(err)
	}
	// A different build answering as prod-demo: digest drift, observed.
	view, err := h.releases.RecordDeployment("demo", "REL-002", localDeployment("k-drift", "http://127.0.0.1:8093"))
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusPromotionFailed || attemptGate(lastAttempt(view), "digest_drift") != release.GateBlocked || attemptGate(lastAttempt(view), "target_drift") != release.GatePass {
		t.Fatalf("another build on the right target must fail digest_drift only: %s %+v", view.Release.Status, lastAttempt(view).Gates)
	}
	// The staging instance answering for prod-demo: target drift and a moved revision.
	view, err = h.releases.RecordDeployment("demo", "REL-002", localDeployment("k-wrong-instance", "http://127.0.0.1:8091"))
	if err != nil {
		t.Fatal(err)
	}
	if view.Release.Status != release.StatusPromotionFailed || attemptGate(lastAttempt(view), "target_drift") != release.GateBlocked || attemptGate(lastAttempt(view), "new_revision") != release.GateBlocked || attemptGate(lastAttempt(view), "digest_drift") != release.GatePass {
		t.Fatalf("the staging instance must fail target_drift and new_revision: %s %+v", view.Release.Status, lastAttempt(view).Gates)
	}
	// The right build on the right instance: promoted, chained to the staging receipt.
	view, err = h.releases.RecordDeployment("demo", "REL-002", localDeployment("k-prod-demo", "http://127.0.0.1:8092"))
	if err != nil {
		t.Fatal(err)
	}
	receipt := view.Release.Receipt
	if view.Release.Status != release.StatusPromoted || receipt == nil || receipt.PreviousReceiptID != "RR-001" || receipt.Deployment.Revision != "prod-demo-r1" || receipt.Environment.TargetRef != "local/demo-prod-demo" {
		t.Fatalf("expected a chained PROMOTED receipt: %s %+v", view.Release.Status, lastAttempt(view).Gates)
	}
}

func TestLocalDeploymentUnreachableOrUnhealthyIsNotPromoted(t *testing.T) {
	observer := &fakeObserver{answers: map[string]*release.LiveObservation{
		"http://127.0.0.1:8095": {HTTPStatus: 503, Status: "degraded", Service: "demo", Environment: "staging", Revision: "staging-r9", BuildDigest: digest, TargetRef: "local/demo-staging"},
	}}
	h := localHarness(t, observer)
	approvedStagingRelease(t, h)
	view, err := h.releases.RecordDeployment("demo", "REL-001", localDeployment("k-down", "http://127.0.0.1:8094"))
	if err != nil {
		t.Fatal(err)
	}
	attempt := lastAttempt(view)
	if view.Release.Status != release.StatusPromotionFailed || attempt.Smoke == nil || attempt.Smoke.Status != "FAIL" || !strings.Contains(attempt.Smoke.Detail, "unreachable") || attempt.DeployedDigest != "" {
		t.Fatalf("an unreachable instance must be a failed, empty-identity attempt: %s %+v", view.Release.Status, attempt)
	}
	if attemptGate(attempt, "digest_drift") != release.GateNeedsEvidence || attemptGate(attempt, "smoke") != release.GateBlocked {
		t.Fatalf("no identity observed must stay NEEDS_EVIDENCE, smoke BLOCKED: %+v", attempt.Gates)
	}
	view, err = h.releases.RecordDeployment("demo", "REL-001", localDeployment("k-unhealthy", "http://127.0.0.1:8095"))
	if err != nil {
		t.Fatal(err)
	}
	attempt = lastAttempt(view)
	if view.Release.Status != release.StatusPromotionFailed || attemptGate(attempt, "smoke") != release.GateBlocked || !strings.Contains(attempt.Smoke.Detail, "503") || attemptGate(attempt, "digest_drift") != release.GatePass {
		t.Fatalf("an unhealthy instance keeps its identity but fails smoke: %+v", attempt)
	}
}

func TestLocalDeploymentNeedsAnObserver(t *testing.T) {
	h := newHarness(t, false)
	changeID := h.acceptedChange(t, "no observer", "abc123", "reviewer-2")
	if _, err := h.releases.Create("demo", release.CreateRequest{Mutation: mutation("staging release"), ChangeIDs: []string{changeID}, TargetEnvironmentID: "staging"}); err != nil {
		t.Fatal(err)
	}
	_, err := h.releases.RecordDeployment("demo", "REL-001", localDeployment("k", "http://127.0.0.1:8091"))
	if codeOf(t, err) != release.CodeInvalidRequest || !strings.Contains(err.Error(), "none is configured") {
		t.Fatalf("without an observer a local deployment must be refused: %v", err)
	}
}

func TestLocalReadBackReadsHealthzOnLoopbackOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"order-operations-portal","environment":"prod-demo","revision":"prod-demo-20261026t010203z-42","build_digest":"` + digest + `","started_at":"2026-10-26T01:02:03Z"}`))
	}))
	defer server.Close()
	reader := release.NewLocalReadBack()
	observation, err := reader.Observe(context.Background(), server.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	if observation.HTTPStatus != 200 || observation.Status != "ok" || observation.Environment != "prod-demo" || observation.Revision != "prod-demo-20261026t010203z-42" || observation.BuildDigest != digest || observation.TargetRef != "local/order-operations-portal-prod-demo" || observation.ServiceURL != server.URL || observation.ObservedAt == "" {
		t.Fatalf("unexpected observation: %+v", observation)
	}
	for _, bad := range []string{"http://example.com", "http://10.0.0.1:8091", "http://169.254.169.254/", "ftp://127.0.0.1", "http://user@127.0.0.1:8091", "http://127.0.0.1:8091/?x=1", "127.0.0.1:8091", ""} {
		if _, err := reader.Observe(context.Background(), bad); err == nil {
			t.Fatalf("%q must be refused before any request is made", bad)
		}
	}
	for _, good := range []string{"http://localhost:1", "http://[::1]:1", "http://127.0.0.2:1"} {
		// Loopback addresses pass the check; nothing listens, so the error is a connection error.
		if _, err := reader.Observe(context.Background(), good); err == nil || !strings.Contains(err.Error(), "unreachable") {
			t.Fatalf("%q must be accepted as loopback and fail only on connect: %v", good, err)
		}
	}
	// A reachable but unhealthy instance is an observation, not an error.
	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"status":"degraded"}`))
	}))
	defer unhealthy.Close()
	observation, err = reader.Observe(context.Background(), unhealthy.URL)
	if err != nil || observation.HTTPStatus != 503 || observation.Status != "degraded" || observation.TargetRef != "" {
		t.Fatalf("unhealthy instance: %v %+v", err, observation)
	}
	// A redirect is never followed, even to another loopback origin: the
	// read-back reports the 3xx it got and reads nothing else.
	followed := false
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed = true }))
	defer elsewhere.Close()
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/healthz", http.StatusFound)
	}))
	defer redirecting.Close()
	observation, err = reader.Observe(context.Background(), redirecting.URL)
	if err != nil || observation.HTTPStatus != http.StatusFound || followed {
		t.Fatalf("redirect must not be followed: followed=%v %v %+v", followed, err, observation)
	}
}
