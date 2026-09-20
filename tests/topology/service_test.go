package topology_test

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"context-rail/internal/topology"
)

// fixedSource is a deterministic baseline: four manifest environments and two
// observed decisions, mirroring the demo Project's shape.
type fixedSource struct{}

func (fixedSource) Baseline(projectID string) (*topology.Baseline, error) {
	if projectID != "demo" {
		return nil, nil
	}
	blocked := "blocked-in-demo"
	return &topology.Baseline{
		ProjectID: "demo",
		Environments: []topology.Environment{
			{ID: "development", Type: "development", Sequence: 1, TargetRef: "local", RequiredEvidence: []string{"local-test"}},
			{ID: "testing", Type: "testing", Sequence: 2, TargetRef: "ci", RequiredEvidence: []string{"test", "build"}},
			{ID: "staging", Type: "staging", Sequence: 3, TargetRef: "cloud-run/demo-staging", RequiredEvidence: []string{"decision-record", "smoke"}},
			{ID: "production", Type: "production", Sequence: 4, TargetRef: "protected/read-only", RequiredEvidence: []string{"release-approval"}, Action: &blocked},
		},
		DecisionIDs: []string{"DR-001", "DR-002"},
	}, nil
}

func newService(t *testing.T) *topology.Service {
	t.Helper()
	store, err := topology.NewFileStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	tick := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	return topology.NewService(store, fixedSource{}).WithClock(func() time.Time {
		tick = tick.Add(time.Second)
		return tick
	})
}

func mutation(version int, reason string) topology.Mutation {
	return topology.Mutation{ExpectedVersion: version, Actor: "operator-1", Reason: reason}
}

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var typed *topology.Error
	if !errors.As(err, &typed) {
		t.Fatalf("expected topology.Error, got %v", err)
	}
	return typed.Code
}

func ids(environments []topology.Environment) []string {
	out := make([]string, len(environments))
	for index, environment := range environments {
		out[index] = environment.ID
	}
	return out
}

func TestBootstrapFromManifestIsVersionOne(t *testing.T) {
	service := newService(t)
	state, err := service.Get("demo")
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentVersion != 1 || len(state.Versions) != 1 || state.Versions[0].Origin != "manifest" {
		t.Fatalf("expected manifest bootstrap v1, got %+v", state)
	}
	current := state.Current()
	if got := ids(current.Environments); len(got) != 4 || got[3] != "production" {
		t.Fatalf("unexpected environments: %v", got)
	}
	if current.Environments[3].Protection != topology.ProtectionBlockedInP0 {
		t.Fatalf("production must carry P0 protection, got %q", current.Environments[3].Protection)
	}
	if current.ConfigHash == "" || len(state.Invalidations) != 0 || len(state.Audit) != 1 {
		t.Fatalf("bootstrap must hash, not invalidate and audit once: %+v", state)
	}
	again, _ := service.Get("demo")
	if again.CurrentVersion != 1 || len(again.Versions) != 1 {
		t.Fatal("Get must not create a new version")
	}
}

func TestUnknownProjectIsNotBootstrapped(t *testing.T) {
	service := newService(t)
	_, err := service.Get("nope")
	if codeOf(t, err) != topology.CodeProjectNotFound {
		t.Fatalf("expected PROJECT_NOT_FOUND, got %v", err)
	}
}

func TestAddCreatesNewVersionAndInvalidatesDecisions(t *testing.T) {
	service := newService(t)
	state, err := service.Add("demo", topology.AddRequest{
		Mutation: mutation(1, "add UAT before staging"), ID: "uat", DisplayName: "UAT", Type: "uat", Sequence: 3,
		TargetRef: "cloud-run/demo-uat", Owner: "qa-lead", RequiredEvidence: []string{"test", "uat-signoff"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentVersion != 2 || state.Versions[1].PreviousVersion != 1 {
		t.Fatalf("expected v2 chained to v1, got %+v", state.Versions)
	}
	if got := ids(state.Current().Environments); got[2] != "uat" || got[3] != "staging" || got[4] != "production" {
		t.Fatalf("uat must sit at position 3: %v", got)
	}
	for index, environment := range state.Current().Environments {
		if environment.Sequence != index+1 {
			t.Fatalf("sequences must be renumbered: %+v", state.Current().Environments)
		}
	}
	if state.Current().Environments[2].Status != topology.StatusActive {
		t.Fatalf("uat with target must be ACTIVE, got %s", state.Current().Environments[2].Status)
	}
	if len(state.Invalidations) != 2 || state.Invalidations[0].DecisionID != "DR-001" || state.Invalidations[0].Status != "STALE" || state.Invalidations[0].TopologyVersion != 2 {
		t.Fatalf("both observed decisions must be STALE against v2: %+v", state.Invalidations)
	}
	if state.Versions[0].ConfigHash == state.Versions[1].ConfigHash {
		t.Fatal("config hash must change when the topology changes")
	}
	if len(state.Audit) != 2 || state.Audit[1].Action != "topology.add" || state.Audit[1].Actor != "operator-1" {
		t.Fatalf("audit must record the add with its actor: %+v", state.Audit)
	}
}

func TestAddWithoutTargetIsDraft(t *testing.T) {
	service := newService(t)
	state, err := service.Add("demo", topology.AddRequest{Mutation: mutation(1, "placeholder"), ID: "sandbox", Type: "other"})
	if err != nil {
		t.Fatal(err)
	}
	last := state.Current().Environments[len(state.Current().Environments)-1]
	if last.ID != "sandbox" || last.Status != topology.StatusDraft || last.DisplayName != "sandbox" {
		t.Fatalf("expected DRAFT sandbox appended, got %+v", last)
	}
}

func TestAddValidation(t *testing.T) {
	service := newService(t)
	cases := []struct {
		name string
		req  topology.AddRequest
		code string
	}{
		{"duplicate", topology.AddRequest{Mutation: mutation(1, "x"), ID: "staging", Type: "staging"}, topology.CodeEnvironmentExists},
		{"bad id", topology.AddRequest{Mutation: mutation(1, "x"), ID: "Bad ID", Type: "other"}, topology.CodeInvalidEnvironment},
		{"bad type", topology.AddRequest{Mutation: mutation(1, "x"), ID: "qa", Type: "galaxy"}, topology.CodeInvalidEnvironment},
		{"no reason", topology.AddRequest{Mutation: mutation(1, "  "), ID: "qa", Type: "other"}, topology.CodeReasonRequired},
		{"stale version", topology.AddRequest{Mutation: mutation(7, "x"), ID: "qa", Type: "other"}, topology.CodeVersionConflict},
	}
	for _, tc := range cases {
		_, err := service.Add("demo", tc.req)
		if codeOf(t, err) != tc.code {
			t.Fatalf("%s: expected %s, got %v", tc.name, tc.code, err)
		}
	}
	state, _ := service.Get("demo")
	if state.CurrentVersion != 1 {
		t.Fatal("rejected mutations must not create versions")
	}
}

func TestDisplayOnlyEditIsInformational(t *testing.T) {
	service := newService(t)
	name := "Pre-production"
	owner := "release-manager"
	state, err := service.Edit("demo", "staging", topology.EditRequest{Mutation: mutation(1, "rename for clarity"), DisplayName: &name, Owner: &owner})
	if err != nil {
		t.Fatal(err)
	}
	change := state.Current().Change
	if change.Material || len(change.FieldsChanged) != 2 {
		t.Fatalf("display-only edit must not be material: %+v", change)
	}
	if len(state.Invalidations) != 0 {
		t.Fatalf("display-only edit must not invalidate decisions: %+v", state.Invalidations)
	}
	if state.CurrentVersion != 2 {
		t.Fatal("even informational edits create a new immutable version")
	}
}

func TestMaterialEditInvalidatesWithVisibleReason(t *testing.T) {
	service := newService(t)
	target := "cloud-run/demo-staging-v2"
	evidence := []string{"decision-record", "smoke", "load-test"}
	state, err := service.Edit("demo", "staging", topology.EditRequest{Mutation: mutation(1, "move staging service"), TargetRef: &target, RequiredEvidence: &evidence})
	if err != nil {
		t.Fatal(err)
	}
	change := state.Current().Change
	if !change.Material {
		t.Fatalf("target change must be material: %+v", change)
	}
	if len(state.Invalidations) != 2 || state.Invalidations[1].DecisionID != "DR-002" || state.Invalidations[1].EnvironmentID != "staging" {
		t.Fatalf("expected DR-001 and DR-002 STALE for staging: %+v", state.Invalidations)
	}
	if reason := state.Invalidations[0].Reason; !strings.Contains(reason, "target_ref") {
		t.Fatalf("invalidation reason must name the changed field: %q", reason)
	}
	if state.Versions[0].Environments[2].TargetRef != "cloud-run/demo-staging" {
		t.Fatal("previous version must remain immutable")
	}
}

func TestEditRejectsNoop(t *testing.T) {
	service := newService(t)
	same := "staging"
	_, err := service.Edit("demo", "staging", topology.EditRequest{Mutation: mutation(1, "nothing"), DisplayName: &same})
	if codeOf(t, err) != topology.CodeInvalidEnvironment {
		t.Fatalf("expected INVALID_ENVIRONMENT for no-op edit, got %v", err)
	}
}

func TestProductionTypeAndRetireAreProtected(t *testing.T) {
	service := newService(t)
	other := "other"
	_, err := service.Edit("demo", "production", topology.EditRequest{Mutation: mutation(1, "downgrade"), Type: &other})
	if codeOf(t, err) != topology.CodeProductionProtected {
		t.Fatalf("expected PRODUCTION_PROTECTED on type change, got %v", err)
	}
	_, err = service.Retire("demo", "production", mutation(1, "remove prod"))
	if codeOf(t, err) != topology.CodeProductionProtected {
		t.Fatalf("expected PRODUCTION_PROTECTED on retire, got %v", err)
	}
	name := "Live"
	state, err := service.Edit("demo", "production", topology.EditRequest{Mutation: mutation(1, "display rename"), DisplayName: &name})
	if err != nil {
		t.Fatalf("production may be renamed for display: %v", err)
	}
	if state.Current().Environments[3].Type != "production" || state.Current().Environments[3].Protection != topology.ProtectionBlockedInP0 {
		t.Fatal("rename must keep the production type and protection")
	}
}

func TestReorderRebuildsSequenceAndRequiresPermutation(t *testing.T) {
	service := newService(t)
	_, err := service.Reorder("demo", topology.ReorderRequest{Mutation: mutation(1, "partial"), Order: []string{"testing", "development"}})
	if codeOf(t, err) != topology.CodeInvalidOrder {
		t.Fatalf("expected INVALID_ORDER for partial order, got %v", err)
	}
	_, err = service.Reorder("demo", topology.ReorderRequest{Mutation: mutation(1, "same"), Order: []string{"development", "testing", "staging", "production"}})
	if codeOf(t, err) != topology.CodeInvalidOrder {
		t.Fatalf("expected INVALID_ORDER for identical order, got %v", err)
	}
	state, err := service.Reorder("demo", topology.ReorderRequest{Mutation: mutation(1, "test before dev"), Order: []string{"testing", "development", "staging", "production"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(state.Current().Environments); got[0] != "testing" || got[1] != "development" {
		t.Fatalf("unexpected order %v", got)
	}
	if state.Current().Environments[0].Sequence != 1 || state.Current().Environments[1].Sequence != 2 {
		t.Fatal("sequence must be rebuilt")
	}
	if !state.Current().Change.Material || len(state.Invalidations) != 2 {
		t.Fatal("reorder is material and invalidates dependent decisions")
	}
}

func TestRetireRestoreLifecycle(t *testing.T) {
	service := newService(t)
	state, err := service.Retire("demo", "testing", mutation(1, "CI now runs in development"))
	if err != nil {
		t.Fatal(err)
	}
	if state.Current().Environments[1].Status != topology.StatusRetired || len(state.Current().Environments) != 4 {
		t.Fatalf("retire must keep the node with RETIRED status: %+v", state.Current().Environments)
	}
	if len(state.Invalidations) != 2 {
		t.Fatal("retire is material")
	}
	name := "x"
	if _, err := service.Edit("demo", "testing", topology.EditRequest{Mutation: mutation(2, "edit retired"), DisplayName: &name}); codeOf(t, err) != topology.CodeEnvironmentRetired {
		t.Fatalf("editing a retired environment must fail: %v", err)
	}
	if _, err := service.Retire("demo", "testing", mutation(2, "again")); codeOf(t, err) != topology.CodeEnvironmentRetired {
		t.Fatalf("double retire must fail: %v", err)
	}
	if _, err := service.Restore("demo", "staging", mutation(2, "not retired")); codeOf(t, err) != topology.CodeEnvironmentActive {
		t.Fatalf("restoring an active environment must fail: %v", err)
	}
	state, err = service.Restore("demo", "testing", mutation(2, "CI split again"))
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentVersion != 3 || state.Current().Environments[1].Status != topology.StatusActive {
		t.Fatalf("restore must create v3 with ACTIVE testing: %+v", state.Current())
	}
	if state.Versions[1].Environments[1].Status != topology.StatusRetired {
		t.Fatal("v2 must still show testing as RETIRED")
	}
}

func TestStateSurvivesRestart(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	store, _ := topology.NewFileStore(dir)
	first := topology.NewService(store, fixedSource{})
	if _, err := first.Add("demo", topology.AddRequest{Mutation: mutation(1, "add"), ID: "uat", Type: "uat", TargetRef: "cloud-run/uat"}); err != nil {
		t.Fatal(err)
	}
	reopened, _ := topology.NewFileStore(dir)
	second := topology.NewService(reopened, fixedSource{})
	state, err := second.Get("demo")
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentVersion != 2 || len(state.Versions) != 2 || len(state.Invalidations) != 2 {
		t.Fatalf("persisted state must be reloaded intact: v%d versions=%d", state.CurrentVersion, len(state.Versions))
	}
}

func TestRegistrySourceReadsDemoFixtures(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	source := topology.NewRegistrySource([]string{filepath.Join(root, "demo", "order-operations-portal"), filepath.Join(root, "examples", "support-insights")})
	baseline, err := source.Baseline("order-operations-portal")
	if err != nil {
		t.Fatal(err)
	}
	if baseline == nil || len(baseline.Environments) != 4 || baseline.Environments[3].Type != "production" {
		t.Fatalf("unexpected demo baseline: %+v", baseline)
	}
	if len(baseline.DecisionIDs) != 1 || baseline.DecisionIDs[0] != "DR-001" {
		t.Fatalf("demo Project exposes DR-001 once even though several views share the id: %v", baseline.DecisionIDs)
	}
	missing, err := source.Baseline("not-configured")
	if err != nil || missing != nil {
		t.Fatalf("unknown project must resolve to nil, nil: %v %v", missing, err)
	}
}
