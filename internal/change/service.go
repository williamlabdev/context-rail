package change

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"context-rail/internal/topology"
)

// ProjectFacts is the read-only context a decision is evaluated against.
// It is observed, never edited, by this package.
type ProjectFacts struct {
	ProjectID      string
	ProjectName    string
	RepositoryURL  string
	DefaultBranch  string
	Documents      []DocumentFact
	ContextStatus  string
	DecisionInputs ReadinessFact
	Topology       *topology.State
}

type DocumentFact struct {
	Path          string `json:"path"`
	Status        string `json:"status"`
	SourceOfTruth bool   `json:"source_of_truth"`
}

type ReadinessFact struct {
	Status  string   `json:"status"`
	Reasons []string `json:"reasons"`
}

// Source resolves ProjectFacts; nil, nil means the Project is not configured.
type Source interface {
	Facts(projectID string) (*ProjectFacts, error)
}

// Advisor proposes candidate options and unknowns. Its output is never
// accepted automatically; a human selects an option in Accept.
type Advisor interface {
	Name() string
	Propose(request Request, facts *ProjectFacts) ([]Option, []string, error)
}

// Store persists one ledger per Project.
type Store interface {
	Load(projectID string) (*State, error)
	Save(projectID string, state *State) error
}

// Service applies the governed Change → Decision → Work Order path.
type Service struct {
	store   Store
	src     Source
	advisor Advisor
	now     func() time.Time
	mu      sync.Mutex
}

func NewService(store Store, src Source, advisor Advisor) *Service {
	if advisor == nil {
		advisor = RuleAdvisor{}
	}
	return &Service{store: store, src: src, advisor: advisor, now: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) WithClock(now func() time.Time) *Service {
	service.now = now
	return service
}

// Required business constraints: cost and data handling cannot be reasoned
// about without them, so their absence is NEEDS_INPUT, not a guess.
var requiredConstraints = []struct{ key, owner, reason string }{
	{"data_classification", "business_owner", "the decision cannot judge access, retention or residency risk without a declared data classification"},
	{"expected_monthly_volume", "business_owner", "cost drivers stay UNKNOWN until an expected volume is declared; the calculator will not fill this in"},
}

// Policy defaults every Agent Context Pack and Work Order carries.
var defaultForbiddenActions = []string{
	"deploy:production",
	"modify:iam",
	"read:deployment-secrets",
	"merge-or-approve-release",
	"change-scope-without-new-decision-version",
}

// Actor / reason envelope for mutations.
type Mutation struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

type CreateRequest struct {
	Mutation
	Request Request `json:"request"`
}

// InputsRequest supplies or corrects request inputs; nil/empty fields are
// left unchanged, business_constraints are merged by key.
type InputsRequest struct {
	Mutation
	Objective           *string                `json:"objective"`
	OwnerSummary        *string                `json:"owner_summary"`
	ScopeIncluded       *[]string              `json:"scope_included"`
	ScopeExcluded       *[]string              `json:"scope_excluded"`
	AcceptanceCriteria  *[]AcceptanceCriterion `json:"acceptance_criteria"`
	AllowedPaths        *[]string              `json:"allowed_paths"`
	ForbiddenActions    *[]string              `json:"forbidden_actions"`
	TargetEnvironmentID *string                `json:"target_environment_id"`
	BusinessConstraints map[string]string      `json:"business_constraints"`
}

type DecideRequest struct {
	Mutation
	Decision       string `json:"decision"` // ACCEPT or REJECT
	Role           string `json:"role"`
	SelectedOption string `json:"selected_option"`
	Rationale      string `json:"rationale"`
	RiskLevel      string `json:"risk_level"`
}

type WorkOrderRequest struct {
	Mutation
	Issuer       string `json:"issuer"`
	TargetBranch string `json:"target_branch"`
}

// List returns every Change of a Project with live staleness.
func (service *Service) List(projectID string) ([]View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, facts, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	views := make([]View, 0, len(state.Changes))
	for index := range state.Changes {
		views = append(views, service.view(&state.Changes[index], facts))
	}
	return views, nil
}

// AcceptedDecisionIDs lists the decision ids of every Change whose latest
// decision is accepted, for Context Pack lineage (document.DecisionLister).
func (service *Service) AcceptedDecisionIDs(projectID string) ([]string, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, _, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for index := range state.Changes {
		if decision := latestDecision(&state.Changes[index]); decision != nil && decision.Status == "ACCEPTED_FOR_DEVELOPMENT" {
			ids = append(ids, decision.DecisionID)
		}
	}
	return ids, nil
}

// Get returns one Change with its rendered artifacts.
func (service *Service) Get(projectID, changeID string) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, facts, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	change := findChange(state, changeID)
	if change == nil {
		return nil, newError(CodeChangeNotFound, "change %q is not in the ledger of %s", changeID, projectID)
	}
	view := service.view(change, facts)
	return &view, nil
}

// Create records a new Change and evaluates it immediately.
func (service *Service) Create(projectID string, request CreateRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, facts, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Reason) == "" {
		return nil, newError(CodeReasonRequired, "a reason is required to open a Change")
	}
	title := strings.TrimSpace(request.Request.Title)
	if title == "" {
		return nil, newError(CodeInvalidRequest, "a Change needs a title")
	}
	at := service.stamp()
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "local-operator")
	change := Change{
		ChangeID: fmt.Sprintf("CHG-%03d", state.NextChange), ProjectID: projectID, Title: title,
		CreatedAt: at, UpdatedAt: at,
	}
	state.NextChange++
	normalized := normalizeRequest(request.Request)
	normalized.RequestedBy = firstNonEmpty(normalized.RequestedBy, actor)
	version, err := service.evaluateVersion(1, normalized, facts, actor, request.Reason, at)
	if err != nil {
		return nil, err
	}
	change.Versions = []ChangeVersion{version}
	change.CurrentVersion = 1
	change.Status = version.Evaluation.Status
	state.Changes = append(state.Changes, change)
	service.audit(state, at, actor, "change.create", change.ChangeID, request.Reason)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(&state.Changes[len(state.Changes)-1], facts)
	return &view, nil
}

// SupplyInputs merges new inputs into a new Change version and re-evaluates.
// An accepted decision made against an older version becomes stale; it is
// not silently re-approved.
func (service *Service) SupplyInputs(projectID, changeID string, request InputsRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, facts, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Reason) == "" {
		return nil, newError(CodeReasonRequired, "a reason is required to change request inputs")
	}
	change := findChange(state, changeID)
	if change == nil {
		return nil, newError(CodeChangeNotFound, "change %q is not in the ledger of %s", changeID, projectID)
	}
	if change.Status == StatusRejected {
		return nil, newError(CodeChangeClosed, "change %s was REJECTED; open a new Change instead of editing a closed one", changeID)
	}
	current := change.Versions[len(change.Versions)-1]
	next := current.Request
	changed := false
	if request.Objective != nil && strings.TrimSpace(*request.Objective) != next.Objective {
		next.Objective = strings.TrimSpace(*request.Objective)
		changed = true
	}
	if request.OwnerSummary != nil && strings.TrimSpace(*request.OwnerSummary) != next.OwnerSummary {
		next.OwnerSummary = strings.TrimSpace(*request.OwnerSummary)
		changed = true
	}
	if request.ScopeIncluded != nil {
		next.ScopeIncluded = cleanList(*request.ScopeIncluded)
		changed = true
	}
	if request.ScopeExcluded != nil {
		next.ScopeExcluded = cleanList(*request.ScopeExcluded)
		changed = true
	}
	if request.AcceptanceCriteria != nil {
		next.AcceptanceCriteria = cleanCriteria(*request.AcceptanceCriteria)
		changed = true
	}
	if request.AllowedPaths != nil {
		next.AllowedPaths = cleanList(*request.AllowedPaths)
		changed = true
	}
	if request.ForbiddenActions != nil {
		next.ForbiddenActions = cleanList(*request.ForbiddenActions)
		changed = true
	}
	if request.TargetEnvironmentID != nil && strings.TrimSpace(*request.TargetEnvironmentID) != next.TargetEnvironmentID {
		next.TargetEnvironmentID = strings.TrimSpace(*request.TargetEnvironmentID)
		changed = true
	}
	if len(request.BusinessConstraints) > 0 {
		merged := map[string]string{}
		for key, value := range next.BusinessConstraints {
			merged[key] = value
		}
		for key, value := range request.BusinessConstraints {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key == "" {
				continue
			}
			if value == "" {
				delete(merged, key)
			} else {
				merged[key] = value
			}
		}
		next.BusinessConstraints = merged
		changed = true
	}
	if !changed {
		return nil, newError(CodeInvalidRequest, "no request input would change")
	}
	at := service.stamp()
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "local-operator")
	version, err := service.evaluateVersion(change.CurrentVersion+1, next, facts, actor, request.Reason, at)
	if err != nil {
		return nil, err
	}
	change.Versions = append(change.Versions, version)
	change.CurrentVersion = version.Version
	change.Status = version.Evaluation.Status
	change.UpdatedAt = at
	service.audit(state, at, actor, "change.inputs", change.ChangeID, request.Reason)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(change, facts)
	return &view, nil
}

// Decide records the human decision on the current Change version. Only a
// DECISION_READY version can be accepted; a NEEDS_INPUT version can only be
// rejected.
func (service *Service) Decide(projectID, changeID string, request DecideRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, facts, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	change := findChange(state, changeID)
	if change == nil {
		return nil, newError(CodeChangeNotFound, "change %q is not in the ledger of %s", changeID, projectID)
	}
	if strings.TrimSpace(request.Reason) == "" && strings.TrimSpace(request.Rationale) == "" {
		return nil, newError(CodeReasonRequired, "a rationale is required for a human decision")
	}
	actor := strings.TrimSpace(request.Actor)
	if actor == "" {
		return nil, newError(CodeActorRequired, "a human decision must name its actor; it cannot be recorded anonymously")
	}
	current := change.Versions[len(change.Versions)-1]
	if change.Status == StatusRejected {
		return nil, newError(CodeChangeClosed, "change %s is already REJECTED", changeID)
	}
	if latest := latestDecision(change); latest != nil && latest.ChangeVersion == current.Version && latest.Status != "REJECTED" {
		return nil, newError(CodeAlreadyDecided, "change version %d already has %s; supply new inputs to create a version that can be decided again", current.Version, latest.DecisionID)
	}
	at := service.stamp()
	decision := strings.ToUpper(strings.TrimSpace(request.Decision))
	rationale := firstNonEmpty(strings.TrimSpace(request.Rationale), strings.TrimSpace(request.Reason))
	record := DecisionRecord{
		DecisionID: fmt.Sprintf("DEC-%03d", state.NextDecision), ProjectID: projectID, ChangeID: change.ChangeID,
		ChangeVersion: current.Version, Version: len(change.Decisions) + 1,
		RiskLevel:    firstNonEmpty(strings.ToLower(strings.TrimSpace(request.RiskLevel)), "low"),
		Alternatives: current.Options, Rationale: rationale,
		Objective: current.Request.Objective, OwnerSummary: current.Request.OwnerSummary, AcceptedScope: current.Request.ScopeIncluded, OutOfScope: current.Request.ScopeExcluded,
		AllowedPaths:       current.Request.AllowedPaths,
		ForbiddenActions:   mergeUnique(defaultForbiddenActions, current.Request.ForbiddenActions),
		AcceptanceCriteria: current.Request.AcceptanceCriteria, BusinessConstraints: current.Request.BusinessConstraints,
		Unknowns:          current.Unknowns,
		SourceEnvironment: current.Evaluation.SourceEnvironment, AllowedTransition: current.Evaluation.AllowedTransition,
		ProductionAction:   "forbidden",
		SourceSnapshotHash: current.Evaluation.SourceSnapshotHash, TopologyVersion: current.Evaluation.TopologyVersion,
		TopologyConfigHash: current.Evaluation.TopologyConfigHash, PolicyVersion: PolicyVersion,
		EvidenceRefs: []string{}, CreatedAt: at,
		HumanDecision: HumanDecision{Actor: actor, Role: firstNonEmpty(strings.TrimSpace(request.Role), "undeclared"), At: at, Reason: rationale},
	}
	if current.Evaluation.TargetEnvironment != nil {
		record.TargetEnvironment = *current.Evaluation.TargetEnvironment
	}
	switch decision {
	case "REJECT", "REJECTED":
		record.Status = "REJECTED"
		record.HumanDecision.Decision = "REJECTED"
		record.SelectedOption = "none"
		change.Status = StatusRejected
	case "ACCEPT", "ACCEPTED":
		if current.Evaluation.Status != StatusDecisionReady {
			return nil, newError(CodeNotDecisionReady, "change version %d is %s; supply the missing inputs before accepting (%d missing)", current.Version, current.Evaluation.Status, len(current.Evaluation.MissingInputs))
		}
		selected := strings.TrimSpace(request.SelectedOption)
		if !hasOption(current.Options, selected) {
			return nil, newError(CodeInvalidOption, "selected_option %q is not one of the proposed candidates for version %d", selected, current.Version)
		}
		record.Status = "ACCEPTED_FOR_DEVELOPMENT"
		record.HumanDecision.Decision = "ACCEPTED"
		record.SelectedOption = selected
		change.Status = StatusAccepted
	default:
		return nil, newError(CodeInvalidRequest, "decision must be ACCEPT or REJECT")
	}
	state.NextDecision++
	change.Decisions = append(change.Decisions, record)
	change.UpdatedAt = at
	service.audit(state, at, actor, "decision."+strings.ToLower(record.HumanDecision.Decision), record.DecisionID, rationale)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(change, facts)
	return &view, nil
}

// CompileWorkOrder turns an accepted, non-stale DecisionRecord into a hashed
// Agent Work Order. It does not run any agent.
func (service *Service) CompileWorkOrder(projectID, changeID string, request WorkOrderRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, facts, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	change := findChange(state, changeID)
	if change == nil {
		return nil, newError(CodeChangeNotFound, "change %q is not in the ledger of %s", changeID, projectID)
	}
	if strings.TrimSpace(request.Reason) == "" {
		return nil, newError(CodeReasonRequired, "a reason is required to issue a Work Order")
	}
	decision := latestDecision(change)
	if decision == nil || decision.Status != "ACCEPTED_FOR_DEVELOPMENT" {
		return nil, newError(CodeNotAccepted, "change %s has no accepted DecisionRecord; a Work Order needs a human-accepted decision", changeID)
	}
	staleness := service.staleness(change, decision, facts)
	if staleness.Stale {
		return nil, newError(CodeDecisionStale, "decision %s is STALE: %s", decision.DecisionID, staleness.Reason)
	}
	if change.WorkOrder != nil && change.WorkOrder.Lineage.DecisionID == decision.DecisionID && change.WorkOrder.Lineage.DecisionVersion == decision.Version {
		return nil, newError(CodeAlreadyIssued, "work order %s is already issued for %s v%d", change.WorkOrder.WorkOrderID, decision.DecisionID, decision.Version)
	}
	if facts.RepositoryURL == "" {
		return nil, newError(CodeNotDecisionReady, "the Project declares no repository; a Work Order cannot bind an agent to UNLINKED_REPOSITORY")
	}
	at := service.stamp()
	issued := service.now()
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "local-operator")
	acceptanceIDs := make([]string, 0, len(decision.AcceptanceCriteria))
	for _, criterion := range decision.AcceptanceCriteria {
		acceptanceIDs = append(acceptanceIDs, criterion.ID)
	}
	order := AgentWorkOrder{
		Kind: "AgentWorkOrder", SchemaVersion: "agent-work-order/v1",
		WorkOrderID: fmt.Sprintf("AWO-%03d", state.NextWorkOrder),
		Lineage:     lineageOf(decision), Status: "ISSUED", HumanGate: decision.Status,
		Issuer:     firstNonEmpty(strings.TrimSpace(request.Issuer), actor),
		Repository: facts.RepositoryURL, BaseBranch: firstNonEmpty(facts.DefaultBranch, "main"),
		TargetBranch:        firstNonEmpty(strings.TrimSpace(request.TargetBranch), "change/"+strings.ToLower(change.ChangeID)),
		TargetEnvironmentID: decision.TargetEnvironment.ID, AllowedTransition: decision.AllowedTransition,
		AllowedPaths: decision.AllowedPaths, ForbiddenActions: decision.ForbiddenActions, AcceptanceIDs: acceptanceIDs,
		RequiredChecks: requiredChecks(decision), OutputPaths: []string{"runs/", "evidence/"},
		IssuedAt: at, ExpiresAt: issued.Add(14 * 24 * time.Hour).Format(time.RFC3339),
	}
	order.WorkOrderHash = hashJSON(order) // computed with WorkOrderHash empty
	state.NextWorkOrder++
	change.WorkOrder = &order
	change.UpdatedAt = at
	service.audit(state, at, actor, "work-order.issue", order.WorkOrderID, request.Reason)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(change, facts)
	return &view, nil
}

// --- evaluation ---------------------------------------------------------------

func (service *Service) evaluateVersion(number int, request Request, facts *ProjectFacts, actor, reason, at string) (ChangeVersion, error) {
	evaluation := evaluate(request, facts, at)
	options, unknowns, err := service.advisor.Propose(request, facts)
	if err != nil {
		// The advisor is advisory: fall back to the deterministic rules and say so.
		options, unknowns, _ = RuleAdvisor{}.Propose(request, facts)
		evaluation.Observations = append(evaluation.Observations, fmt.Sprintf("advisor %s unavailable (%v); rule-based candidates used", service.advisor.Name(), err))
	}
	return ChangeVersion{
		Version: number, CreatedAt: at, Actor: actor, Reason: reason, Request: request,
		Evaluation: evaluation, Options: options, Unknowns: unknowns,
	}, nil
}

// evaluate is deterministic: same request + same facts → same result.
func evaluate(request Request, facts *ProjectFacts, at string) Evaluation {
	evaluation := Evaluation{
		Status: StatusDecisionReady, MissingInputs: []MissingInput{}, Observations: []string{},
		ProjectContext: facts.ContextStatus, DecisionInputsReady: facts.DecisionInputs.Status, EvaluatedAt: at,
	}
	missing := func(field, owner, reason string) {
		evaluation.MissingInputs = append(evaluation.MissingInputs, MissingInput{Field: field, OwnerRole: owner, Reason: reason})
	}
	if strings.TrimSpace(request.Objective) == "" {
		missing("objective", "requester", "the change has no stated objective; the decision cannot judge intent")
	}
	if strings.TrimSpace(request.OwnerSummary) == "" {
		missing("owner_summary", "requester", "the business owner needs a plain-language summary of what this change does; ContextRail will not write one on the requester's behalf")
	}
	if len(request.AcceptanceCriteria) == 0 {
		missing("acceptance_criteria", "requester", "at least one acceptance criterion is needed; an agent cannot be handed a change with no testable expectation")
	}
	if len(request.AllowedPaths) == 0 {
		missing("allowed_paths", "tech_lead", "a Work Order must bound the agent to explicit repository paths")
	}
	for _, constraint := range requiredConstraints {
		if strings.TrimSpace(request.BusinessConstraints[constraint.key]) == "" {
			missing("business_constraints."+constraint.key, constraint.owner, constraint.reason)
		}
	}

	// Topology binding.
	var current *topology.Version
	if facts.Topology != nil {
		current = facts.Topology.Current()
	}
	if current == nil {
		missing("environment_topology", "project_owner", "the Project has no environment topology version to bind the change to")
	} else {
		evaluation.TopologyVersion = current.Version
		evaluation.TopologyConfigHash = current.ConfigHash
		if strings.TrimSpace(request.TargetEnvironmentID) == "" {
			missing("target_environment_id", "tech_lead", "the change must name the environment it is meant to reach next")
		} else {
			ordered := append([]topology.Environment(nil), current.Environments...)
			sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Sequence < ordered[j].Sequence })
			targetIndex := -1
			for index, environment := range ordered {
				if environment.ID == request.TargetEnvironmentID {
					targetIndex = index
				}
			}
			switch {
			case targetIndex < 0:
				missing("target_environment_id", "tech_lead", fmt.Sprintf("environment %q is not in topology v%d", request.TargetEnvironmentID, current.Version))
			default:
				target := ordered[targetIndex]
				evaluation.TargetEnvironment = refOf(target)
				switch {
				case target.Status == topology.StatusRetired:
					missing("target_environment_id", "tech_lead", fmt.Sprintf("environment %s is RETIRED in topology v%d; choose an active environment or restore it", target.ID, current.Version))
				case target.Type == "production":
					missing("target_environment_id", "release_approver", "production is READ_ONLY/BLOCKED in P0; a change can target at most the environment before it")
				case target.Status == topology.StatusDraft:
					missing("target_environment_id", "tech_lead", fmt.Sprintf("environment %s has no declared target_ref (DRAFT); declare it in the topology first", target.ID))
				}
				for index := targetIndex - 1; index >= 0; index-- {
					if ordered[index].Status != topology.StatusRetired {
						evaluation.SourceEnvironment = refOf(ordered[index])
						evaluation.AllowedTransition = ordered[index].ID + "-to-" + target.ID
						break
					}
				}
				if evaluation.SourceEnvironment == nil {
					evaluation.AllowedTransition = "entry-to-" + target.ID
				}
			}
		}
	}

	// Project context and decision inputs from the registry.
	switch facts.ContextStatus {
	case "MISSING":
		missing("project_context", "project_owner", "the Project has no derived context; rebuild the Context Pack from current sources")
	case "STALE", "CONFLICT":
		evaluation.Observations = append(evaluation.Observations, fmt.Sprintf("project context is %s; the decision is made against sources that may have moved", facts.ContextStatus))
	}
	if facts.DecisionInputs.Status != "" && facts.DecisionInputs.Status != "READY" {
		missing("decision_documents", "project_owner", "required decision documents are not all CURRENT: "+strings.Join(facts.DecisionInputs.Reasons, "; "))
	}

	evaluation.SourceSnapshotHash = snapshotHash(facts, current)
	if len(evaluation.MissingInputs) > 0 {
		evaluation.Status = StatusNeedsInput
	}
	return evaluation
}

// snapshotHash binds a decision to the observed document set, context
// status and topology configuration it was made against.
func snapshotHash(facts *ProjectFacts, current *topology.Version) string {
	docs := append([]DocumentFact(nil), facts.Documents...)
	sort.SliceStable(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	payload := map[string]any{
		"project_id": facts.ProjectID, "documents": docs, "context_status": facts.ContextStatus,
		"repository": facts.RepositoryURL, "policy_version": PolicyVersion,
	}
	if current != nil {
		payload["topology_version"] = current.Version
		payload["topology_config_hash"] = current.ConfigHash
	}
	return hashJSON(payload)
}

// --- views and staleness -------------------------------------------------------

func (service *Service) view(change *Change, facts *ProjectFacts) View {
	entry := *change
	view := View{Change: entry, Staleness: Staleness{}}
	decision := latestDecision(change)
	if decision == nil {
		return view
	}
	view.Decision = decision
	if decision.Status == "ACCEPTED_FOR_DEVELOPMENT" {
		view.Staleness = service.staleness(change, decision, facts)
		brief := RenderBrief(decision, view.Staleness)
		pack := RenderAgentContextPack(decision, view.Staleness)
		view.Brief = &brief
		view.Pack = &pack
		if view.Staleness.Stale {
			view.Change.Status = StatusStale
		}
	}
	if change.WorkOrder != nil {
		order := *change.WorkOrder
		if view.Staleness.Stale || order.Lineage.DecisionID != decision.DecisionID || order.Lineage.DecisionVersion != decision.Version {
			order.Status = "STALE"
		}
		view.WorkOrder = &order
	}
	if view.Staleness.Stale || (view.WorkOrder != nil && view.WorkOrder.Status == "STALE") {
		// Candidates produced under a stale contract cannot be accepted; show it.
		view.Change.Candidates = append([]Candidate(nil), change.Candidates...)
		for index := range view.Change.Candidates {
			if view.Change.Candidates[index].Status == "EVALUATED" {
				view.Change.Candidates[index].Status = "STALE"
			}
		}
	}
	return view
}

func (service *Service) staleness(change *Change, decision *DecisionRecord, facts *ProjectFacts) Staleness {
	if decision.ChangeVersion != change.CurrentVersion {
		return Staleness{Stale: true, Reason: fmt.Sprintf("request inputs changed after acceptance (change v%d, decision made against v%d); a new human decision is required", change.CurrentVersion, decision.ChangeVersion)}
	}
	if facts.Topology != nil {
		for _, version := range facts.Topology.Versions {
			if version.Version > decision.TopologyVersion && version.Change.Material {
				return Staleness{Stale: true, Reason: fmt.Sprintf("topology v%d (%s) is a material change after the decision bound topology v%d: %s", version.Version, strings.ToLower(version.Change.Operation), decision.TopologyVersion, version.Change.Impact)}
			}
		}
		if current := facts.Topology.Current(); current != nil {
			for _, environment := range current.Environments {
				if environment.ID == decision.TargetEnvironment.ID && environment.Status == topology.StatusRetired {
					return Staleness{Stale: true, Reason: fmt.Sprintf("target environment %s is RETIRED in topology v%d", environment.ID, current.Version)}
				}
			}
		}
	}
	if current := snapshotHashNow(facts, decision); current != decision.SourceSnapshotHash {
		return Staleness{Stale: true, Reason: "observed Project sources or context changed after acceptance (source_snapshot_hash differs); re-evaluate before handing off"}
	}
	return Staleness{}
}

func snapshotHashNow(facts *ProjectFacts, decision *DecisionRecord) string {
	// Recompute against the topology version the decision bound, so that a
	// display-only topology edit (new version, same environments hash) does not
	// count as source drift; material topology changes are caught above.
	if facts.Topology != nil {
		for index := range facts.Topology.Versions {
			if facts.Topology.Versions[index].Version == decision.TopologyVersion {
				return snapshotHash(facts, &facts.Topology.Versions[index])
			}
		}
	}
	return snapshotHash(facts, nil)
}

// --- internals ------------------------------------------------------------------

func (service *Service) load(projectID string) (*State, *ProjectFacts, error) {
	facts, err := service.src.Facts(projectID)
	if err != nil {
		return nil, nil, err
	}
	if facts == nil {
		return nil, nil, newError(CodeProjectNotFound, "project %q is not configured", projectID)
	}
	state, err := service.store.Load(projectID)
	if err != nil {
		return nil, nil, err
	}
	if state == nil {
		state = &State{Kind: Kind, SchemaVersion: SchemaVersion, ProjectID: projectID, NextChange: 1, NextDecision: 1, NextWorkOrder: 1, Changes: []Change{}, Audit: []AuditEvent{}}
	}
	return state, facts, nil
}

func (service *Service) stamp() string { return service.now().Format(time.RFC3339) }

func (service *Service) audit(state *State, at, actor, action, object, reason string) {
	state.Audit = append(state.Audit, AuditEvent{Sequence: len(state.Audit) + 1, At: at, Actor: actor, Action: action, Object: object, Reason: reason})
}

func findChange(state *State, changeID string) *Change {
	for index := range state.Changes {
		if state.Changes[index].ChangeID == changeID {
			return &state.Changes[index]
		}
	}
	return nil
}

func latestDecision(change *Change) *DecisionRecord {
	if len(change.Decisions) == 0 {
		return nil
	}
	return &change.Decisions[len(change.Decisions)-1]
}

func lineageOf(decision *DecisionRecord) Lineage {
	return Lineage{
		DecisionID: decision.DecisionID, DecisionVersion: decision.Version, ChangeID: decision.ChangeID, ProjectID: decision.ProjectID,
		SourceSnapshotHash: decision.SourceSnapshotHash, TopologyVersion: decision.TopologyVersion, TopologyConfigHash: decision.TopologyConfigHash,
		PolicyVersion: decision.PolicyVersion,
	}
}

func requiredChecks(decision *DecisionRecord) []string {
	checks := []string{"allowed-paths-diff-check", "independent-review"}
	return mergeUnique(checks, decision.TargetEnvironment.RequiredEvidence)
}

func refOf(environment topology.Environment) *EnvironmentRef {
	return &EnvironmentRef{
		ID: environment.ID, Type: environment.Type, Sequence: environment.Sequence, TargetRef: environment.TargetRef,
		RequiredEvidence: append([]string{}, environment.RequiredEvidence...), Status: string(environment.Status), Protection: environment.Protection,
	}
}

func normalizeRequest(request Request) Request {
	request.Title = strings.TrimSpace(request.Title)
	request.Objective = strings.TrimSpace(request.Objective)
	request.OwnerSummary = strings.TrimSpace(request.OwnerSummary)
	request.ScopeIncluded = cleanList(request.ScopeIncluded)
	request.ScopeExcluded = cleanList(request.ScopeExcluded)
	request.AcceptanceCriteria = cleanCriteria(request.AcceptanceCriteria)
	request.AllowedPaths = cleanList(request.AllowedPaths)
	request.ForbiddenActions = cleanList(request.ForbiddenActions)
	request.TargetEnvironmentID = strings.TrimSpace(request.TargetEnvironmentID)
	constraints := map[string]string{}
	for key, value := range request.BusinessConstraints {
		if k, v := strings.TrimSpace(key), strings.TrimSpace(value); k != "" && v != "" {
			constraints[k] = v
		}
	}
	request.BusinessConstraints = constraints
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	return request
}

func cleanList(values []string) []string {
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

func cleanCriteria(values []AcceptanceCriterion) []AcceptanceCriterion {
	out := make([]AcceptanceCriterion, 0, len(values))
	for index, value := range values {
		text := strings.TrimSpace(value.Text)
		if text == "" {
			continue
		}
		id := strings.TrimSpace(value.ID)
		if id == "" {
			id = fmt.Sprintf("AC-%03d", index+1)
		}
		out = append(out, AcceptanceCriterion{ID: id, Text: text})
	}
	return out
}

func mergeUnique(base []string, extra []string) []string {
	return cleanList(append(append([]string{}, base...), extra...))
}

func hasOption(options []Option, id string) bool {
	for _, option := range options {
		if option.ID == id {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
