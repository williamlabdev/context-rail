package document_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"context-rail/internal/document"
)

// source is a fake registry baseline over a temp root that stands in for the
// consumer repository. The tests only ever read from it — except where they
// simulate the consumer adding a file, which is what a real Project would do.
type source struct{ root string }

func (s source) Baseline(projectID string) (*document.Baseline, error) {
	if projectID != "demo" {
		return nil, nil
	}
	return &document.Baseline{
		ProjectID: "demo", Root: s.root,
		Documents: []document.Declared{
			{Path: "project.yaml", Kind: "project-manifest", SourceOfTruth: true, RequiredFor: []string{"decision", "development", "staging"}},
			{Path: "architecture.md", Kind: "architecture-baseline", SourceOfTruth: true, RequiredFor: []string{"decision", "development"}},
			{Path: "docs/ai/context-pack.json", Kind: "derived-context", SourceOfTruth: false, RequiredFor: []string{"work-order"}},
		},
		ObservedStatuses: map[string]document.ObservedStatus{"docs/ai/context-pack.json": {Status: "STALE", StaleSources: []string{"architecture.md"}}},
		DeclaredContext:  document.DeclaredContext{Path: "docs/ai/context-pack.json", Status: "STALE"},
		DecisionIDs:      []string{"DR-001"},
	}, nil
}

type lister struct{}

func (lister) AcceptedDecisionIDs(string) ([]string, error) { return []string{"DEC-001"}, nil }

func newService(t *testing.T) (*document.Service, string) {
	t.Helper()
	root := t.TempDir()
	write(t, root, "project.yaml", "kind: Project\n")
	write(t, root, "architecture.md", "# arch\n")
	write(t, root, "docs/ai/context-pack.json", `{"kind":"ContextPack","derived":true}`)
	tick := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { tick = tick.Add(time.Second); return tick }
	return document.NewService(document.NewMemoryStore(), source{root}).WithClock(clock).WithDecisions(lister{}), root
}

func write(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var typed *document.Error
	if !errors.As(err, &typed) {
		t.Fatalf("expected document.Error, got %v", err)
	}
	return typed.Code
}

func status(view *document.View, path string) document.Source {
	for _, entry := range view.Documents {
		if entry.Path == path {
			return entry
		}
	}
	return document.Source{Path: path, Status: "ABSENT"}
}

func TestBaselineBootstrapsFromManifestAndReportsLiveStatus(t *testing.T) {
	service, _ := newService(t)
	view, err := service.Get("demo")
	if err != nil {
		t.Fatal(err)
	}
	if view.CurrentVersion != 1 || view.Baseline.Change.Operation != "BOOTSTRAP" || len(view.Documents) != 3 {
		t.Fatalf("bootstrap must import the manifest documents as v1: %+v", view.Baseline)
	}
	if status(view, "project.yaml").Status != document.StatusCurrent || status(view, "project.yaml").ContentHash == "" || status(view, "project.yaml").Origin != "manifest" {
		t.Fatalf("manifest document must be CURRENT with a content hash: %+v", status(view, "project.yaml"))
	}
	if status(view, "docs/ai/context-pack.json").Status != "STALE" || status(view, "docs/ai/context-pack.json").StaleSources[0] != "architecture.md" {
		t.Fatalf("derived document keeps the registry verdict: %+v", status(view, "docs/ai/context-pack.json"))
	}
	if view.Readiness["decision"].Status != "READY" || view.Readiness["work-order"].Status != "NEEDS_INPUT" || view.LiveContext != "STALE" || view.LatestPack != nil {
		t.Fatalf("readiness and context before any rebuild: %+v live=%s", view.Readiness, view.LiveContext)
	}
}

func TestMissingDeclaredDocumentMapsToReadinessAndPartialPack(t *testing.T) {
	service, root := newService(t)
	// Declare a runbook the Project does not have. The file is NOT created.
	view, err := service.Declare("demo", document.DeclareRequest{Mutation: document.Mutation{ExpectedVersion: 1, Actor: "architect", Reason: "staging needs a runbook"}, Path: "docs/operations/runbook.md", Kind: "runbook", RequiredFor: []string{"decision", "staging", "bogus"}})
	if err != nil {
		t.Fatal(err)
	}
	if view.CurrentVersion != 2 || view.Baseline.Change.Operation != "DECLARE" || !strings.Contains(view.Baseline.Change.Impact, "MISSING") {
		t.Fatalf("declaring must create baseline v2 with the impact spelled out: %+v", view.Baseline)
	}
	runbook := status(view, "docs/operations/runbook.md")
	if runbook.Status != document.StatusMissing || runbook.ContentHash != "" || runbook.Origin != "operator" || runbook.Version != "v2" || strings.Join(runbook.RequiredFor, ",") != "decision,staging" {
		t.Fatalf("missing declaration: %+v", runbook)
	}
	if view.Readiness["decision"].Status != "NEEDS_INPUT" || view.Readiness["staging"].Status != "NEEDS_INPUT" || view.Readiness["development"].Status != "READY" {
		t.Fatalf("missing document must map to the stages it is required for: %+v", view.Readiness)
	}
	if len(view.Missing) != 1 || strings.Join(view.Missing[0].ReadinessImpact, ",") != "decision,staging" {
		t.Fatalf("missing list must carry the readiness impact: %+v", view.Missing)
	}
	if _, statErr := os.Stat(filepath.Join(root, "docs/operations/runbook.md")); !os.IsNotExist(statErr) {
		t.Fatal("declaring a document must never create it in the consumer repository")
	}
	// Rebuild: PARTIAL, lists the gap, synthesizes nothing.
	view, err = service.Rebuild("demo", document.RebuildRequest{Actor: "architect", Reason: "rebuild after declaring the runbook"})
	if err != nil {
		t.Fatal(err)
	}
	pack := view.LatestPack
	if pack == nil || pack.PackID != "CP-001" || pack.Status != document.PackPartial || !pack.Derived || pack.BaselineVersion != 2 || pack.PackHash == "" || pack.SourceSnapshotHash == "" {
		t.Fatalf("expected CP-001 PARTIAL: %+v", pack)
	}
	if len(pack.Missing) != 1 || pack.Missing[0].Path != "docs/operations/runbook.md" || len(pack.Sources) != 4 {
		t.Fatalf("pack must list the missing source among the declared ones: %+v", pack.Missing)
	}
	for _, entry := range pack.Sources {
		if entry.Path == "docs/operations/runbook.md" && (entry.ContentHash != "" || entry.Status != document.StatusMissing) {
			t.Fatalf("a missing source has no content hash and stays MISSING: %+v", entry)
		}
		if entry.Path == "architecture.md" && (entry.ContentHash == "" || entry.Version != "v1" || entry.AccessScope != "repository-read") {
			t.Fatalf("present sources carry path, version and hash: %+v", entry)
		}
	}
	joined := strings.Join(pack.Limitations, " | ")
	if !strings.Contains(joined, "nothing was synthesized") || !strings.Contains(joined, "derived document") {
		t.Fatalf("limitations must say what was not done: %s", joined)
	}
	if strings.Join(pack.DecisionRefs, ",") != "DEC-001,DR-001" {
		t.Fatalf("decision refs from repository and ledger: %v", pack.DecisionRefs)
	}
	if pack.Readiness["decision"].Status != "NEEDS_INPUT" || view.LiveContext != document.PackPartial || view.PackDrift.Stale {
		t.Fatalf("live context follows the pack: %s drift=%+v", view.LiveContext, view.PackDrift)
	}
	if len(view.Audit) != 3 || view.Audit[2].Action != "context_pack.rebuild" || view.Audit[1].Action != "baseline.declare" {
		t.Fatalf("audit must record bootstrap, declare and rebuild: %+v", view.Audit)
	}
	// The overlay for the change ledger reports the gap as a decision input.
	documents, contextStatus, decision, found, err := service.DocumentFacts("demo")
	if err != nil || !found || contextStatus != document.PackPartial || decision.Status != "NEEDS_INPUT" || !strings.Contains(strings.Join(decision.Reasons, " "), "runbook.md is MISSING") || len(documents) != 4 {
		t.Fatalf("overlay: found=%v ctx=%s decision=%+v docs=%d err=%v", found, contextStatus, decision, len(documents), err)
	}

	// The consumer adds the file: the baseline is satisfied, the pack drifted, a rebuild is DERIVED.
	write(t, root, "docs/operations/runbook.md", "# runbook\n")
	view, _ = service.Get("demo")
	if status(view, "docs/operations/runbook.md").Status != document.StatusCurrent || view.Readiness["decision"].Status != "READY" || len(view.Missing) != 0 {
		t.Fatalf("present file must satisfy the declaration: %+v %+v", status(view, "docs/operations/runbook.md"), view.Readiness)
	}
	if !view.PackDrift.Stale || !strings.Contains(view.PackDrift.Reason, "runbook.md") || view.LiveContext != document.StatusStale {
		t.Fatalf("the PARTIAL pack no longer matches the sources: %+v live=%s", view.PackDrift, view.LiveContext)
	}
	view, err = service.Rebuild("demo", document.RebuildRequest{Actor: "architect", Reason: "runbook landed"})
	if err != nil {
		t.Fatal(err)
	}
	if view.LatestPack.PackID != "CP-002" || view.LatestPack.Status != document.PackDerived || len(view.LatestPack.Missing) != 0 || view.PackDrift.Stale || view.LiveContext != document.PackDerived {
		t.Fatalf("expected CP-002 DERIVED: %+v", view.LatestPack)
	}
	if pack, err := service.Pack("demo", "CP-001"); err != nil || pack.Status != document.PackPartial {
		t.Fatalf("earlier packs stay readable: %+v %v", pack, err)
	}
}

func TestBaselineGuards(t *testing.T) {
	service, _ := newService(t)
	if _, err := service.Declare("demo", document.DeclareRequest{Mutation: document.Mutation{ExpectedVersion: 0, Actor: "a", Reason: "r"}, Path: "x.md", RequiredFor: []string{"decision"}}); codeOf(t, err) != document.CodeVersionConflict {
		t.Fatalf("stale expected_version must be refused: %v", err)
	}
	if _, err := service.Declare("demo", document.DeclareRequest{Mutation: document.Mutation{ExpectedVersion: 1, Actor: "a"}, Path: "x.md", RequiredFor: []string{"decision"}}); codeOf(t, err) != document.CodeReasonRequired {
		t.Fatalf("reason required: %v", err)
	}
	if _, err := service.Declare("demo", document.DeclareRequest{Mutation: document.Mutation{ExpectedVersion: 1, Actor: "a", Reason: "r"}, Path: "../escape.md", RequiredFor: []string{"decision"}}); codeOf(t, err) != document.CodeInvalidRequest {
		t.Fatalf("path escaping the root must be refused: %v", err)
	}
	if _, err := service.Declare("demo", document.DeclareRequest{Mutation: document.Mutation{ExpectedVersion: 1, Actor: "a", Reason: "r"}, Path: "x.md", RequiredFor: []string{"nowhere"}}); codeOf(t, err) != document.CodeInvalidRequest {
		t.Fatalf("unknown stage must be refused: %v", err)
	}
	if _, err := service.Declare("demo", document.DeclareRequest{Mutation: document.Mutation{ExpectedVersion: 1, Actor: "a", Reason: "r"}, Path: "architecture.md", RequiredFor: []string{"decision"}}); codeOf(t, err) != document.CodeAlreadyDeclared {
		t.Fatalf("manifest document is already declared: %v", err)
	}
	if _, err := service.Withdraw("demo", document.WithdrawRequest{Mutation: document.Mutation{ExpectedVersion: 1, Actor: "a", Reason: "r"}, Path: "architecture.md"}); codeOf(t, err) != document.CodeManifestProtected {
		t.Fatalf("manifest documents cannot be withdrawn here: %v", err)
	}
	if _, err := service.Withdraw("demo", document.WithdrawRequest{Mutation: document.Mutation{ExpectedVersion: 1, Actor: "a", Reason: "r"}, Path: "nope.md"}); codeOf(t, err) != document.CodeNotDeclared {
		t.Fatalf("unknown document: %v", err)
	}
	view, err := service.Declare("demo", document.DeclareRequest{Mutation: document.Mutation{ExpectedVersion: 1, Actor: "a", Reason: "r"}, Path: "docs/runbook.md", Kind: "runbook", RequiredFor: []string{"staging"}})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.Withdraw("demo", document.WithdrawRequest{Mutation: document.Mutation{ExpectedVersion: view.CurrentVersion, Actor: "a", Reason: "not needed"}, Path: "docs/runbook.md"})
	if err != nil {
		t.Fatal(err)
	}
	if view.CurrentVersion != 3 || len(view.Documents) != 3 || view.Readiness["staging"].Status != "READY" {
		t.Fatalf("withdraw creates v3 and re-evaluates: v%d %d docs %+v", view.CurrentVersion, len(view.Documents), view.Readiness["staging"])
	}
	if _, err := service.Rebuild("demo", document.RebuildRequest{Actor: "a"}); codeOf(t, err) != document.CodeReasonRequired {
		t.Fatalf("rebuild needs a reason: %v", err)
	}
	if _, err := service.Get("unknown"); codeOf(t, err) != document.CodeProjectNotFound {
		t.Fatalf("unknown project: %v", err)
	}
	if _, err := service.Pack("demo", "CP-009"); codeOf(t, err) != document.CodePackNotFound {
		t.Fatalf("unknown pack: %v", err)
	}
}
