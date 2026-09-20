package topology

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Baseline is what the service needs from the Project registry to bootstrap
// and to know which decisions may depend on the topology. It is read-only
// input; the service never writes back to the registry or the manifest.
type Baseline struct {
	ProjectID    string
	Environments []Environment
	DecisionIDs  []string
}

// Source resolves a Project baseline. The HTTP layer adapts the registry
// importer; tests use a fixed map.
type Source interface {
	Baseline(projectID string) (*Baseline, error) // nil, nil when the Project is not configured
}

// Service applies governed topology operations.
type Service struct {
	store Store
	src   Source
	now   func() time.Time
	mu    sync.Mutex
}

func NewService(store Store, src Source) *Service {
	return &Service{store: store, src: src, now: func() time.Time { return time.Now().UTC() }}
}

// WithClock overrides the timestamp source (tests).
func (service *Service) WithClock(now func() time.Time) *Service {
	service.now = now
	return service
}

// Mutation carries the fields every operation needs.
type Mutation struct {
	ExpectedVersion int    `json:"expected_version"`
	Actor           string `json:"actor"`
	Reason          string `json:"reason"`
}

// AddRequest creates a new environment in a new topology version.
type AddRequest struct {
	Mutation
	ID               string   `json:"id"`
	DisplayName      string   `json:"display_name"`
	Type             string   `json:"type"`
	Sequence         int      `json:"sequence"`
	TargetRef        string   `json:"target_ref"`
	Owner            string   `json:"owner"`
	RequiredEvidence []string `json:"required_evidence"`
	ApproverPolicy   string   `json:"approver_policy"`
}

// EditRequest patches an environment. Nil pointers leave a field unchanged.
type EditRequest struct {
	Mutation
	DisplayName      *string   `json:"display_name"`
	Type             *string   `json:"type"`
	Sequence         *int      `json:"sequence"`
	TargetRef        *string   `json:"target_ref"`
	Owner            *string   `json:"owner"`
	RequiredEvidence *[]string `json:"required_evidence"`
	ApproverPolicy   *string   `json:"approver_policy"`
}

// ReorderRequest supplies the complete new order of environment IDs.
type ReorderRequest struct {
	Mutation
	Order []string `json:"order"`
}

var environmentIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,39}$`)

// Material fields — type, sequence, target_ref, required_evidence,
// approver_policy and status — invalidate dependent decisions when changed.
// display_name and owner are display-only and stay informational.

// Get returns the Project's topology state, bootstrapping version 1 from the
// manifest when no governance state exists yet.
func (service *Service) Get(projectID string) (*State, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, _, err := service.load(projectID)
	return state, err
}

// Add creates an environment in a new version.
func (service *Service) Add(projectID string, request AddRequest) (*State, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if err := service.guard(state, request.Mutation); err != nil {
		return nil, err
	}
	current := state.Current()
	if !environmentIDPattern.MatchString(request.ID) {
		return nil, newError(CodeInvalidEnvironment, "environment id must match %s", environmentIDPattern.String())
	}
	if findEnvironment(current.Environments, request.ID) >= 0 {
		return nil, newError(CodeEnvironmentExists, "environment %q already exists in topology v%d", request.ID, current.Version)
	}
	if !validType(request.Type) {
		return nil, newError(CodeInvalidEnvironment, "type must be one of %s", strings.Join(StandardTypes, ", "))
	}
	environment := Environment{
		ID:               request.ID,
		DisplayName:      firstNonEmpty(strings.TrimSpace(request.DisplayName), request.ID),
		Type:             request.Type,
		TargetRef:        strings.TrimSpace(request.TargetRef),
		Owner:            strings.TrimSpace(request.Owner),
		RequiredEvidence: normalizeEvidence(request.RequiredEvidence),
		ApproverPolicy:   strings.TrimSpace(request.ApproverPolicy),
	}
	environment.Status = StatusActive
	if environment.TargetRef == "" {
		environment.Status = StatusDraft // a node without a declared target cannot be promoted to
	}
	applyProtection(&environment)

	next := cloneEnvironments(current.Environments)
	sequence := request.Sequence
	if sequence <= 0 || sequence > len(next)+1 {
		sequence = len(next) + 1
	}
	environment.Sequence = sequence
	next = insertAtSequence(next, environment)

	impact := fmt.Sprintf("environment %s added at position %d; promotion path now has %d nodes", environment.ID, sequence, len(next))
	if environment.Status == StatusDraft {
		impact += "; DRAFT until a target_ref is declared"
	}
	change := Change{Operation: "ADD", EnvironmentID: environment.ID, Material: true, Reason: request.Reason, Impact: impact}
	return service.commit(projectID, state, baseline, request.Mutation, change, next)
}

// Edit changes one environment's attributes in a new version.
func (service *Service) Edit(projectID, environmentID string, request EditRequest) (*State, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if err := service.guard(state, request.Mutation); err != nil {
		return nil, err
	}
	current := state.Current()
	index := findEnvironment(current.Environments, environmentID)
	if index < 0 {
		return nil, newError(CodeEnvironmentNotFound, "environment %q is not in topology v%d", environmentID, current.Version)
	}
	next := cloneEnvironments(current.Environments)
	target := &next[index]
	if target.Status == StatusRetired {
		return nil, newError(CodeEnvironmentRetired, "environment %q is RETIRED; restore it through a new version before editing", environmentID)
	}

	var fields []FieldChange
	if request.DisplayName != nil && strings.TrimSpace(*request.DisplayName) != target.DisplayName {
		value := strings.TrimSpace(*request.DisplayName)
		if value == "" {
			return nil, newError(CodeInvalidEnvironment, "display_name cannot be empty")
		}
		fields = append(fields, FieldChange{Field: "display_name", From: target.DisplayName, To: value})
		target.DisplayName = value
	}
	if request.Type != nil && *request.Type != target.Type {
		if !validType(*request.Type) {
			return nil, newError(CodeInvalidEnvironment, "type must be one of %s", strings.Join(StandardTypes, ", "))
		}
		if target.Type == "production" || *request.Type == "production" {
			return nil, newError(CodeProductionProtected, "the production standard type cannot be changed in P0; rename for display instead")
		}
		fields = append(fields, FieldChange{Field: "type", From: target.Type, To: *request.Type, Material: true})
		target.Type = *request.Type
	}
	if request.TargetRef != nil && strings.TrimSpace(*request.TargetRef) != target.TargetRef {
		value := strings.TrimSpace(*request.TargetRef)
		fields = append(fields, FieldChange{Field: "target_ref", From: target.TargetRef, To: value, Material: true})
		target.TargetRef = value
	}
	if request.Owner != nil && strings.TrimSpace(*request.Owner) != target.Owner {
		value := strings.TrimSpace(*request.Owner)
		fields = append(fields, FieldChange{Field: "owner", From: target.Owner, To: value})
		target.Owner = value
	}
	if request.RequiredEvidence != nil {
		value := normalizeEvidence(*request.RequiredEvidence)
		if strings.Join(value, ",") != strings.Join(target.RequiredEvidence, ",") {
			fields = append(fields, FieldChange{Field: "required_evidence", From: target.RequiredEvidence, To: value, Material: true})
			target.RequiredEvidence = value
		}
	}
	if request.ApproverPolicy != nil && strings.TrimSpace(*request.ApproverPolicy) != target.ApproverPolicy {
		value := strings.TrimSpace(*request.ApproverPolicy)
		fields = append(fields, FieldChange{Field: "approver_policy", From: target.ApproverPolicy, To: value, Material: true})
		target.ApproverPolicy = value
	}
	if request.Sequence != nil && *request.Sequence != target.Sequence {
		if *request.Sequence < 1 || *request.Sequence > len(next) {
			return nil, newError(CodeInvalidOrder, "sequence must be between 1 and %d", len(next))
		}
		fields = append(fields, FieldChange{Field: "sequence", From: target.Sequence, To: *request.Sequence, Material: true})
		moved := *target
		next = removeEnvironment(next, environmentID)
		moved.Sequence = *request.Sequence
		next = insertAtSequence(next, moved)
		target = &next[findEnvironment(next, environmentID)]
	}
	// DRAFT ↔ ACTIVE follows the declared target; this is a status consequence
	// of the edit, not a separate operation.
	previousStatus := target.Status
	if target.TargetRef != "" && target.Status == StatusDraft {
		target.Status = StatusActive
	} else if target.TargetRef == "" && target.Status == StatusActive {
		target.Status = StatusDraft
	}
	if target.Status != previousStatus {
		fields = append(fields, FieldChange{Field: "status", From: previousStatus, To: target.Status, Material: true})
	}
	applyProtection(target)

	if len(fields) == 0 {
		return nil, newError(CodeInvalidEnvironment, "no attribute of %q would change", environmentID)
	}
	material := false
	var materialNames []string
	for _, field := range fields {
		if field.Material {
			material = true
			materialNames = append(materialNames, field.Field)
		}
	}
	impact := "display-only change; existing decisions remain valid (informational)"
	if material {
		impact = fmt.Sprintf("material change to %s; dependent decisions and approvals are marked STALE", strings.Join(materialNames, ", "))
	}
	change := Change{Operation: "EDIT", EnvironmentID: environmentID, FieldsChanged: fields, Material: material, Reason: request.Reason, Impact: impact}
	return service.commit(projectID, state, baseline, request.Mutation, change, next)
}

// Reorder rebuilds the sequence from a complete permutation of environment IDs.
func (service *Service) Reorder(projectID string, request ReorderRequest) (*State, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if err := service.guard(state, request.Mutation); err != nil {
		return nil, err
	}
	current := state.Current()
	if len(request.Order) != len(current.Environments) {
		return nil, newError(CodeInvalidOrder, "order must list all %d environments exactly once", len(current.Environments))
	}
	seen := map[string]bool{}
	next := make([]Environment, 0, len(current.Environments))
	var fields []FieldChange
	for position, id := range request.Order {
		if seen[id] {
			return nil, newError(CodeInvalidOrder, "environment %q listed twice", id)
		}
		seen[id] = true
		index := findEnvironment(current.Environments, id)
		if index < 0 {
			return nil, newError(CodeInvalidOrder, "environment %q is not in topology v%d", id, current.Version)
		}
		moved := cloneEnvironments(current.Environments[index : index+1])[0]
		if moved.Sequence != position+1 {
			fields = append(fields, FieldChange{Field: "sequence", From: moved.Sequence, To: position + 1, Material: true})
		}
		moved.Sequence = position + 1
		next = append(next, moved)
	}
	if len(fields) == 0 {
		return nil, newError(CodeInvalidOrder, "order is identical to topology v%d", current.Version)
	}
	change := Change{Operation: "REORDER", FieldsChanged: fields, Material: true, Reason: request.Reason,
		Impact: fmt.Sprintf("promotion path reordered to %s; predecessor/successor transitions must be revalidated", strings.Join(request.Order, " → "))}
	return service.commit(projectID, state, baseline, request.Mutation, change, next)
}

// Retire stops the environment from being selected by new Changes. History
// stays readable; there is no hard delete.
func (service *Service) Retire(projectID, environmentID string, request Mutation) (*State, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if err := service.guard(state, request); err != nil {
		return nil, err
	}
	current := state.Current()
	index := findEnvironment(current.Environments, environmentID)
	if index < 0 {
		return nil, newError(CodeEnvironmentNotFound, "environment %q is not in topology v%d", environmentID, current.Version)
	}
	next := cloneEnvironments(current.Environments)
	target := &next[index]
	if target.Type == "production" {
		return nil, newError(CodeProductionProtected, "production cannot be retired in P0; it stays in the topology as a protected node")
	}
	if target.Status == StatusRetired {
		return nil, newError(CodeEnvironmentRetired, "environment %q is already RETIRED", environmentID)
	}
	fields := []FieldChange{{Field: "status", From: target.Status, To: StatusRetired, Material: true}}
	target.Status = StatusRetired
	change := Change{Operation: "RETIRE", EnvironmentID: environmentID, FieldsChanged: fields, Material: true, Reason: request.Reason,
		Impact: fmt.Sprintf("%s is RETIRED: new Changes cannot select it; historical evidence and receipts remain readable", environmentID)}
	return service.commit(projectID, state, baseline, request, change, next)
}

// Restore reactivates a retired environment after re-validating its target.
func (service *Service) Restore(projectID, environmentID string, request Mutation) (*State, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, baseline, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if err := service.guard(state, request); err != nil {
		return nil, err
	}
	current := state.Current()
	index := findEnvironment(current.Environments, environmentID)
	if index < 0 {
		return nil, newError(CodeEnvironmentNotFound, "environment %q is not in topology v%d", environmentID, current.Version)
	}
	next := cloneEnvironments(current.Environments)
	target := &next[index]
	if target.Status != StatusRetired {
		return nil, newError(CodeEnvironmentActive, "environment %q is %s, not RETIRED", environmentID, target.Status)
	}
	if target.TargetRef == "" {
		return nil, newError(CodeInvalidEnvironment, "environment %q has no target_ref; declare a target before restoring", environmentID)
	}
	restored := StatusActive
	fields := []FieldChange{{Field: "status", From: StatusRetired, To: restored, Material: true}}
	target.Status = restored
	change := Change{Operation: "RESTORE", EnvironmentID: environmentID, FieldsChanged: fields, Material: true, Reason: request.Reason,
		Impact: fmt.Sprintf("%s restored as %s under the current policy; it does not reinstate any decision made against an earlier version", environmentID, restored)}
	return service.commit(projectID, state, baseline, request, change, next)
}

// --- internals ---------------------------------------------------------------

func (service *Service) load(projectID string) (*State, *Baseline, error) {
	baseline, err := service.src.Baseline(projectID)
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
	environments := cloneEnvironments(baseline.Environments)
	sort.SliceStable(environments, func(i, j int) bool { return environments[i].Sequence < environments[j].Sequence })
	for index := range environments {
		environments[index].Sequence = index + 1
		if environments[index].DisplayName == "" {
			environments[index].DisplayName = environments[index].ID
		}
		if environments[index].Status == "" {
			environments[index].Status = StatusActive
			if environments[index].TargetRef == "" {
				environments[index].Status = StatusDraft
			}
		}
		if environments[index].RequiredEvidence == nil {
			environments[index].RequiredEvidence = []string{}
		}
		applyProtection(&environments[index])
	}
	at := service.now().Format(time.RFC3339)
	version := Version{
		Version: 1, PreviousVersion: 0, CreatedAt: at, Actor: "system", Origin: "manifest",
		Change: Change{Operation: "BOOTSTRAP", Material: false, Reason: "imported from project.yaml",
			Impact: fmt.Sprintf("baseline topology with %d environments observed from the Project manifest", len(environments))},
		Environments: environments, ConfigHash: ConfigHash(environments),
	}
	return &State{
		Kind: Kind, SchemaVersion: SchemaVersion, ProjectID: baseline.ProjectID, CurrentVersion: 1,
		Versions: []Version{version}, Invalidations: []Invalidation{},
		Audit: []AuditEvent{{Sequence: 1, At: at, Actor: "system", Action: "topology.bootstrap", Object: "topology", Version: 1, Reason: "imported from project.yaml"}},
	}
}

func (service *Service) guard(state *State, mutation Mutation) error {
	if strings.TrimSpace(mutation.Reason) == "" {
		return newError(CodeReasonRequired, "a reason is required for every topology change")
	}
	if mutation.ExpectedVersion != state.CurrentVersion {
		return newError(CodeVersionConflict, "expected topology v%d but current is v%d; reload and retry", mutation.ExpectedVersion, state.CurrentVersion)
	}
	return nil
}

func (service *Service) commit(projectID string, state *State, baseline *Baseline, mutation Mutation, change Change, environments []Environment) (*State, error) {
	at := service.now().Format(time.RFC3339)
	actor := firstNonEmpty(strings.TrimSpace(mutation.Actor), "local-operator")
	number := state.CurrentVersion + 1
	version := Version{
		Version: number, PreviousVersion: state.CurrentVersion, CreatedAt: at, Actor: actor, Origin: "operator",
		Change: change, Environments: environments, ConfigHash: ConfigHash(environments),
	}
	state.Versions = append(state.Versions, version)
	state.CurrentVersion = number
	state.Audit = append(state.Audit, AuditEvent{
		Sequence: len(state.Audit) + 1, At: at, Actor: actor,
		Action: "topology." + strings.ToLower(change.Operation), Object: firstNonEmpty(change.EnvironmentID, "topology"),
		Version: number, Reason: change.Reason,
	})
	if change.Material {
		for _, decisionID := range baseline.DecisionIDs {
			state.Invalidations = append(state.Invalidations, Invalidation{
				DecisionID: decisionID, EnvironmentID: change.EnvironmentID, TopologyVersion: number, Status: "STALE",
				Reason: fmt.Sprintf("topology v%d (%s): %s", number, strings.ToLower(change.Operation), change.Impact), At: at,
			})
		}
	}
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	return state, nil
}

func applyProtection(environment *Environment) {
	if environment.Type == "production" {
		environment.Protection = ProtectionBlockedInP0
	} else {
		environment.Protection = ""
	}
}

func validType(value string) bool {
	for _, standard := range StandardTypes {
		if standard == value {
			return true
		}
	}
	return false
}

func normalizeEvidence(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func findEnvironment(environments []Environment, id string) int {
	for index, environment := range environments {
		if environment.ID == id {
			return index
		}
	}
	return -1
}

func removeEnvironment(environments []Environment, id string) []Environment {
	out := make([]Environment, 0, len(environments))
	for _, environment := range environments {
		if environment.ID != id {
			out = append(out, environment)
		}
	}
	return renumber(out)
}

func insertAtSequence(environments []Environment, environment Environment) []Environment {
	sort.SliceStable(environments, func(i, j int) bool { return environments[i].Sequence < environments[j].Sequence })
	position := environment.Sequence - 1
	if position < 0 {
		position = 0
	}
	if position > len(environments) {
		position = len(environments)
	}
	out := make([]Environment, 0, len(environments)+1)
	out = append(out, environments[:position]...)
	out = append(out, environment)
	out = append(out, environments[position:]...)
	return renumber(out)
}

func renumber(environments []Environment) []Environment {
	for index := range environments {
		environments[index].Sequence = index + 1
	}
	return environments
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
