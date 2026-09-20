package document

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"context-rail/internal/topology"
)

// Baseline is what the read-only registry declares for a Project: its root
// (read, never written), the manifest documents with the stages they are
// required for, the observed status of derived documents, the consumer's own
// context file status and the decisions already recorded in the repository.
type Baseline struct {
	ProjectID        string
	Root             string
	Documents        []Declared
	ObservedStatuses map[string]ObservedStatus // registry statuses (derived docs carry stale sources)
	DeclaredContext  DeclaredContext
	DecisionIDs      []string
}

// ObservedStatus is the registry's view of one declared document.
type ObservedStatus struct {
	Status       string
	StaleSources []string
}

// BaselineSource resolves the baseline; nil, nil means the Project is not configured.
type BaselineSource interface {
	Baseline(projectID string) (*Baseline, error)
}

// TopologyReader lets the pack record the topology version it was built against.
type TopologyReader interface {
	Get(projectID string) (*topology.State, error)
}

// DecisionLister lets the pack reference decisions from the governed change
// ledger without importing it (late-bound to avoid a wiring cycle).
type DecisionLister interface {
	AcceptedDecisionIDs(projectID string) ([]string, error)
}

// Service applies the governed document baseline and rebuilds Context Packs.
type Service struct {
	store     Store
	source    BaselineSource
	topology  TopologyReader
	decisions DecisionLister
	now       func() time.Time
	mu        sync.Mutex
}

func NewService(store Store, source BaselineSource) *Service {
	return &Service{store: store, source: source, now: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) WithClock(now func() time.Time) *Service { service.now = now; return service }

func (service *Service) WithTopology(reader TopologyReader) *Service {
	service.topology = reader
	return service
}

func (service *Service) WithDecisions(lister DecisionLister) *Service {
	service.decisions = lister
	return service
}

// Mutation carries the fields every operation needs.
type Mutation struct {
	ExpectedVersion int    `json:"expected_version"`
	Actor           string `json:"actor"`
	Reason          string `json:"reason"`
}

// DeclareRequest adds a required document to the baseline (new version).
type DeclareRequest struct {
	Mutation
	Path        string   `json:"path"`
	Kind        string   `json:"kind"`
	RequiredFor []string `json:"required_for"`
}

// WithdrawRequest removes an operator-declared document (new version).
type WithdrawRequest struct {
	Mutation
	Path string `json:"path"`
}

// RebuildRequest regenerates the Context Pack from the sources as they are now.
type RebuildRequest struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

var documentPathPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,199}$`)

// Get returns the live view: baseline, statuses now, readiness, latest pack.
func (service *Service) Get(projectID string) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	view := service.view(state, baseline)
	return &view, nil
}

// Declare records that the Project must have a document it does not
// necessarily have yet. The missing file is reported, never created.
func (service *Service) Declare(projectID string, request DeclareRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if err := service.guard(state, request.Mutation); err != nil {
		return nil, err
	}
	path := strings.TrimSpace(request.Path)
	if !documentPathPattern.MatchString(path) || strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
		return nil, newError(CodeInvalidRequest, "path must be a relative repository path (letters, digits, . _ / -), got %q", request.Path)
	}
	current := state.Current()
	for _, document := range current.Documents {
		if document.Path == path {
			return nil, newError(CodeAlreadyDeclared, "%s is already declared (%s) in baseline v%d", path, document.Origin, current.Version)
		}
	}
	stages := normalizeStages(request.RequiredFor)
	if len(stages) == 0 {
		return nil, newError(CodeInvalidRequest, "required_for needs at least one of %s", strings.Join(Stages, ", "))
	}
	at := service.stamp()
	next := append([]Declared{}, current.Documents...)
	declared := Declared{Path: path, Kind: firstNonEmpty(strings.TrimSpace(request.Kind), "document"), SourceOfTruth: true, RequiredFor: stages, Origin: "operator", DeclaredInVersion: current.Version + 1}
	next = append(next, declared)
	status := service.observe(baseline, declared, at).Status
	impact := fmt.Sprintf("%s is %s now; required for %s", path, status, strings.Join(stages, ", "))
	if status != StatusCurrent {
		impact += " — those stages are NEEDS_INPUT until the file exists; the Context Pack lists it as missing and does not synthesize it"
	}
	service.commit(state, at, request.Actor, "operator", BaselineChange{Operation: "DECLARE", Path: path, Reason: request.Reason, Impact: impact}, next)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(state, baseline)
	return &view, nil
}

// Withdraw removes an operator declaration. Manifest documents are the
// consumer's contract and cannot be withdrawn here.
func (service *Service) Withdraw(projectID string, request WithdrawRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if err := service.guard(state, request.Mutation); err != nil {
		return nil, err
	}
	path := strings.TrimSpace(request.Path)
	current := state.Current()
	next := []Declared{}
	var found *Declared
	for index := range current.Documents {
		if current.Documents[index].Path == path {
			found = &current.Documents[index]
			continue
		}
		next = append(next, current.Documents[index])
	}
	switch {
	case found == nil:
		return nil, newError(CodeNotDeclared, "%s is not declared in baseline v%d", path, current.Version)
	case found.Origin != "operator":
		return nil, newError(CodeManifestProtected, "%s is declared by the Project manifest; edit project.yaml in the consumer repository, not the governance baseline", path)
	}
	at := service.stamp()
	service.commit(state, at, request.Actor, "operator", BaselineChange{Operation: "WITHDRAW", Path: path, Reason: request.Reason, Impact: path + " is no longer required; stages it blocked are re-evaluated"}, next)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(state, baseline)
	return &view, nil
}

// Rebuild regenerates the Context Pack from the declared sources as they are
// on disk right now. Missing sources make the pack PARTIAL; they are listed
// with the stages they block and nothing is generated in their place.
func (service *Service) Rebuild(projectID string, request RebuildRequest) (*View, error) {
	// Ask the change ledger first: it reads our Facts while resolving its own
	// project facts, so it must not run under our lock.
	ledgerDecisions := []string{}
	if service.decisions != nil {
		if ids, err := service.decisions.AcceptedDecisionIDs(projectID); err == nil {
			ledgerDecisions = ids
		}
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Reason) == "" {
		return nil, newError(CodeReasonRequired, "a reason is required to rebuild the Context Pack")
	}
	at := service.stamp()
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "local-operator")
	current := state.Current()
	sources, missing, readiness := service.observeAll(baseline, current, at)
	pack := ContextPack{
		Kind: "ContextPack", SchemaVersion: PackSchema, Derived: true, Status: PackDerived, ProjectID: projectID,
		PackID: fmt.Sprintf("CP-%03d", state.NextPack), GeneratedAt: at, GeneratedBy: actor, Reason: request.Reason,
		BaselineVersion: current.Version, BaselineHash: current.BaselineHash, Sources: sources, Missing: missing, Readiness: readiness,
		DecisionRefs: []string{}, Limitations: []string{"Generated from declared repository sources; external Git, CI and runtime evidence is not included"},
		PolicyVersion: PolicyVersion,
	}
	if service.topology != nil {
		if topologyState, err := service.topology.Get(projectID); err == nil && topologyState != nil && topologyState.Current() != nil {
			pack.TopologyVersion = topologyState.Current().Version
			pack.TopologyConfigHash = topologyState.Current().ConfigHash
		}
	}
	pack.DecisionRefs = append(pack.DecisionRefs, baseline.DecisionIDs...)
	pack.DecisionRefs = append(pack.DecisionRefs, ledgerDecisions...)
	sort.Strings(pack.DecisionRefs)
	if len(missing) > 0 {
		pack.Status = PackPartial
		paths := make([]string, 0, len(missing))
		for _, entry := range missing {
			paths = append(paths, entry.Path)
		}
		pack.Limitations = append(pack.Limitations, fmt.Sprintf("%d declared source(s) missing: %s — nothing was synthesized in their place; the pack is PARTIAL until they exist", len(missing), strings.Join(paths, ", ")))
	}
	for _, source := range sources {
		if !source.SourceOfTruth {
			pack.Limitations = append(pack.Limitations, source.Path+" is a derived document, not a source of truth; it is listed, not trusted")
		}
	}
	pack.SourceSnapshotHash = snapshotHash(projectID, current, sources)
	pack.PackHash = hashJSON(pack)
	state.Packs = append(state.Packs, pack)
	state.NextPack++
	service.audit(state, at, actor, "context_pack.rebuild", pack.PackID, fmt.Sprintf("%s: %s", pack.Status, request.Reason))
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(state, baseline)
	return &view, nil
}

// Pack returns one rebuilt pack by id.
func (service *Service) Pack(projectID, packID string) (*ContextPack, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, _, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	for index := range state.Packs {
		if state.Packs[index].PackID == packID {
			pack := state.Packs[index]
			return &pack, nil
		}
	}
	return nil, newError(CodePackNotFound, "context pack %q is not recorded for %s", packID, projectID)
}

// Facts is the overlay the change ledger consumes: live document set,
// live context status and document readiness for decisions.
type Facts struct {
	Documents     []Source
	ContextStatus string
	Decision      Readiness
}

// Facts returns the document overlay without changing state.
func (service *Service) Facts(projectID string) (*Facts, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}
	view := service.view(state, baseline)
	return &Facts{Documents: view.Documents, ContextStatus: view.LiveContext, Decision: view.Readiness["decision"]}, nil
}

// --- observation ------------------------------------------------------------------

func (service *Service) observeAll(baseline *Baseline, current *BaselineVersion, at string) ([]Source, []MissingSource, map[string]Readiness) {
	sources := make([]Source, 0, len(current.Documents))
	missing := []MissingSource{}
	reasons := map[string][]string{}
	for _, stage := range Stages {
		reasons[stage] = []string{}
	}
	for _, declared := range current.Documents {
		source := service.observe(baseline, declared, at)
		sources = append(sources, source)
		if source.Status == StatusCurrent || source.Status == StatusDerived {
			continue
		}
		impact := []string{}
		for _, stage := range declared.RequiredFor {
			if _, known := reasons[stage]; known {
				reasons[stage] = append(reasons[stage], fmt.Sprintf("%s is %s (%s, declared by %s)", declared.Path, source.Status, declared.Kind, declared.Origin))
				impact = append(impact, stage)
			}
		}
		if source.Status == StatusMissing {
			missing = append(missing, MissingSource{Path: declared.Path, Kind: declared.Kind, RequiredFor: declared.RequiredFor, ReadinessImpact: impact, Origin: declared.Origin})
		}
	}
	readiness := map[string]Readiness{}
	for _, stage := range Stages {
		if len(reasons[stage]) > 0 {
			readiness[stage] = Readiness{Status: "NEEDS_INPUT", Reasons: reasons[stage]}
		} else {
			readiness[stage] = Readiness{Status: "READY", Reasons: []string{}}
		}
	}
	return sources, missing, readiness
}

// observe stats and hashes one declared document under the consumer root.
// It reads only; a document that is not a regular file is MISSING.
func (service *Service) observe(baseline *Baseline, declared Declared, at string) Source {
	source := Source{
		Path: declared.Path, Kind: declared.Kind, SourceOfTruth: declared.SourceOfTruth, RequiredFor: declared.RequiredFor, Origin: declared.Origin,
		Version: fmt.Sprintf("v%d", declared.DeclaredInVersion), Status: StatusMissing, ObservedAt: at, AccessScope: "repository-read",
	}
	full, ok := safeJoin(baseline.Root, declared.Path)
	if !ok {
		source.Status = StatusNeedsInput
		return source
	}
	info, err := os.Stat(full)
	if err != nil || !info.Mode().IsRegular() {
		return source
	}
	hash, size, err := fileHash(full)
	if err != nil {
		source.Status = StatusNeedsInput
		return source
	}
	source.ContentHash, source.Bytes, source.Status = hash, size, StatusCurrent
	if observed, ok := baseline.ObservedStatuses[declared.Path]; ok {
		switch {
		case !declared.SourceOfTruth:
			// Derived documents keep the registry's verdict (DERIVED / STALE / NEEDS_INPUT).
			source.Status = observed.Status
			source.StaleSources = observed.StaleSources
		case observed.Status == StatusStale || observed.Status == StatusNeedsInput || observed.Status == "CONFLICT":
			// The manifest itself declares an uncertain state; the registry preserves it and so do we.
			source.Status = observed.Status
		}
	}
	return source
}

func safeJoin(root, relative string) (string, bool) {
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Join(root, clean), true
}

func fileHash(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), size, nil
}

func snapshotHash(projectID string, current *BaselineVersion, sources []Source) string {
	type entry struct {
		Path   string `json:"path"`
		Status string `json:"status"`
		Hash   string `json:"content_hash"`
	}
	entries := make([]entry, 0, len(sources))
	for _, source := range sources {
		entries = append(entries, entry{Path: source.Path, Status: source.Status, Hash: source.ContentHash})
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return hashJSON(map[string]any{"project_id": projectID, "baseline_version": current.Version, "baseline_hash": current.BaselineHash, "sources": entries, "policy_version": PolicyVersion})
}

// --- view ----------------------------------------------------------------------

func (service *Service) view(state *State, baseline *Baseline) View {
	at := service.stamp()
	current := state.Current()
	sources, missing, readiness := service.observeAll(baseline, current, at)
	view := View{
		ProjectID: state.ProjectID, CurrentVersion: state.CurrentVersion, Baseline: *current, Documents: sources, Missing: missing, Readiness: readiness,
		DeclaredContext: baseline.DeclaredContext, Versions: state.Versions, Audit: state.Audit, ObservedAt: at, ObservedRoot: baseline.Root, ReadOnlyConsumer: true,
		Packs: make([]string, 0, len(state.Packs)),
	}
	for _, pack := range state.Packs {
		view.Packs = append(view.Packs, pack.PackID)
	}
	view.LiveContext = firstNonEmpty(baseline.DeclaredContext.Status, StatusMissing)
	if latest := state.LatestPack(); latest != nil {
		pack := *latest
		view.LatestPack = &pack
		view.LiveContext = pack.Status
		if pack.BaselineVersion != current.Version {
			view.PackDrift = Drift{Stale: true, Reason: fmt.Sprintf("%s was built against baseline v%d; the baseline is now v%d — rebuild to reflect the declared document set", pack.PackID, pack.BaselineVersion, current.Version)}
		} else if now := snapshotHash(state.ProjectID, current, sources); now != pack.SourceSnapshotHash {
			changed := changedPaths(pack.Sources, sources)
			view.PackDrift = Drift{Stale: true, Reason: fmt.Sprintf("%s no longer matches the sources (%s changed since %s); rebuild before relying on it", pack.PackID, strings.Join(changed, ", "), pack.GeneratedAt)}
		}
		if view.PackDrift.Stale {
			view.LiveContext = StatusStale
		}
	}
	return view
}

func changedPaths(before, after []Source) []string {
	previous := map[string]Source{}
	for _, source := range before {
		previous[source.Path] = source
	}
	changed := []string{}
	for _, source := range after {
		if old, ok := previous[source.Path]; !ok || old.Status != source.Status || old.ContentHash != source.ContentHash {
			changed = append(changed, source.Path)
		}
	}
	if len(changed) == 0 {
		changed = append(changed, "declared set")
	}
	return changed
}

// --- internals -----------------------------------------------------------------

func (service *Service) load(projectID string) (*State, *Baseline, error) {
	baseline, err := service.source.Baseline(projectID)
	if err != nil {
		return nil, nil, err
	}
	if baseline == nil {
		return nil, nil, newError(CodeProjectNotFound, "project %q is not configured", projectID)
	}
	state, err := service.store.Load(projectID)
	if err != nil {
		return nil, nil, err
	}
	if state == nil {
		state = service.bootstrap(baseline)
		if err := service.store.Save(projectID, state); err != nil {
			return nil, nil, err
		}
	}
	return state, baseline, nil
}

func (service *Service) bootstrap(baseline *Baseline) *State {
	at := service.stamp()
	documents := make([]Declared, 0, len(baseline.Documents))
	for _, declared := range baseline.Documents {
		copied := declared
		copied.Origin = "manifest"
		copied.DeclaredInVersion = 1
		copied.RequiredFor = normalizeStages(declared.RequiredFor)
		documents = append(documents, copied)
	}
	state := &State{Kind: Kind, SchemaVersion: SchemaVersion, ProjectID: baseline.ProjectID, NextPack: 1, Versions: []BaselineVersion{}, Packs: []ContextPack{}, Audit: []AuditEvent{}}
	service.commit(state, at, "context-rail", "manifest", BaselineChange{Operation: "BOOTSTRAP", Reason: "baseline imported from project.yaml", Impact: fmt.Sprintf("%d declared documents", len(documents))}, documents)
	return state
}

func (service *Service) commit(state *State, at, actor, origin string, change BaselineChange, documents []Declared) {
	version := BaselineVersion{Version: state.CurrentVersion + 1, PreviousVersion: state.CurrentVersion, CreatedAt: at, Actor: firstNonEmpty(strings.TrimSpace(actor), "local-operator"), Origin: origin, Change: change, Documents: documents}
	version.BaselineHash = hashJSON(documents)
	state.Versions = append(state.Versions, version)
	state.CurrentVersion = version.Version
	service.audit(state, at, version.Actor, "baseline."+strings.ToLower(change.Operation), firstNonEmpty(change.Path, fmt.Sprintf("v%d", version.Version)), change.Reason)
}

func (service *Service) guard(state *State, mutation Mutation) error {
	if strings.TrimSpace(mutation.Reason) == "" {
		return newError(CodeReasonRequired, "a reason is required for every baseline change")
	}
	if mutation.ExpectedVersion != state.CurrentVersion {
		return newError(CodeVersionConflict, "expected baseline v%d but the current version is v%d; reload and retry", mutation.ExpectedVersion, state.CurrentVersion)
	}
	return nil
}

func (service *Service) stamp() string { return service.now().Format(time.RFC3339) }

func (service *Service) audit(state *State, at, actor, action, object, reason string) {
	state.Audit = append(state.Audit, AuditEvent{Sequence: len(state.Audit) + 1, At: at, Actor: actor, Action: action, Object: object, Reason: reason})
}

func normalizeStages(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		stage := strings.ToLower(strings.TrimSpace(value))
		known := false
		for _, candidate := range Stages {
			if candidate == stage {
				known = true
			}
		}
		if stage == "" || !known || seen[stage] {
			continue
		}
		seen[stage] = true
		out = append(out, stage)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
