package change

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

// Candidate review (VS-005): an agent run produces a candidate; ContextRail
// cross-checks what was observed (branch, diff paths, checks, reviews) against
// the issued Work Order and the current decision, and blocks instead of
// trusting the run's own claims. A human then accepts or rejects the candidate;
// the gate never accepts by itself.

// Candidate verdicts.
const (
	VerdictAcceptable    = "CANDIDATE_ACCEPTABLE"
	VerdictBlocked       = "BLOCKED"
	VerdictNeedsEvidence = "NEEDS_EVIDENCE"
	VerdictNeedsReview   = "NEEDS_REVIEW"
)

// Gate statuses.
const (
	GatePass          = "PASS"
	GateBlocked       = "BLOCKED"
	GateNeedsEvidence = "NEEDS_EVIDENCE"
	GateNeedsReview   = "NEEDS_REVIEW"
	GateWaived        = "WAIVED"
	GateDeferred      = "DEFERRED_TO_PROMOTION"
)

// Checks that are only meaningful after a deployment; the candidate gate
// defers them to the promotion gate (VS-006) instead of blocking on them.
var promotionOnlyChecks = map[string]bool{"smoke": true, "staging-receipt": true, "release-approval": true, "staging-smoke": true}

// AgentIdentity is the declared agent that produced the candidate.
type AgentIdentity struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Version  string `json:"version"`
}

// AgentRun is the run declaration (agent-run-record/v1). It is a claim made
// by whoever started the run; the gate verifies it against observations.
type AgentRun struct {
	Kind             string        `json:"kind"`
	SchemaVersion    string        `json:"schema_version"`
	RunID            string        `json:"run_id"`
	WorkOrderID      string        `json:"work_order_id"`
	WorkOrderHash    string        `json:"work_order_hash"`
	Agent            AgentIdentity `json:"agent"`
	Adapter          string        `json:"adapter"`
	StartedBy        string        `json:"started_by"`
	StartedAt        string        `json:"started_at"`
	FinishedAt       string        `json:"finished_at"`
	DeclaredCommands []string      `json:"declared_commands"`
	DeclaredPaths    []string      `json:"declared_changed_paths"`
}

// CheckResult is one observed check (test, build, vet, ...).
type CheckResult struct {
	Name        string `json:"name"`
	Status      string `json:"status"` // PASS or FAIL
	EvidenceRef string `json:"evidence_ref"`
	ObservedAt  string `json:"observed_at"`
}

// ReviewEvidence is one observed review.
type ReviewEvidence struct {
	Reviewer    string `json:"reviewer"`
	Kind        string `json:"kind"`    // human or ai
	Verdict     string `json:"verdict"` // APPROVED or CHANGES_REQUESTED
	EvidenceRef string `json:"evidence_ref"`
	At          string `json:"at"`
}

// CompensatingControls is the documented single-operator control set that a
// low-risk staging policy may accept in place of a second human reviewer.
type CompensatingControls struct {
	Declared    bool   `json:"declared"`
	EvidenceRef string `json:"evidence_ref"`
	Reason      string `json:"reason"`
}

// Observation is what was actually read back about the candidate. Source
// says where it came from: "declared" (typed in by the operator) or a
// read-back adapter such as "github".
type Observation struct {
	Source               string                `json:"source"`
	Repository           string                `json:"repository"`
	Branch               string                `json:"branch"`
	BaseBranch           string                `json:"base_branch"`
	BaseCommit           string                `json:"base_commit"`
	HeadCommit           string                `json:"head_commit"`
	PullRequest          string                `json:"pull_request"`
	ChangedPaths         []string              `json:"changed_paths"`
	Checks               []CheckResult         `json:"checks"`
	Reviews              []ReviewEvidence      `json:"reviews"`
	CompensatingControls *CompensatingControls `json:"compensating_controls"`
	ObservedAt           string                `json:"observed_at"`
}

// GateResult is one deterministic check of the candidate against the contract.
type GateResult struct {
	Gate    string   `json:"gate"`
	Status  string   `json:"status"`
	Detail  string   `json:"detail"`
	Paths   []string `json:"paths,omitempty"`
	Missing []string `json:"missing,omitempty"`
}

// Violation is a hard rule breach the gate found.
type Violation struct {
	Rule   string `json:"rule"`
	Path   string `json:"path,omitempty"`
	Detail string `json:"detail"`
}

// Candidate is one immutable evaluation of a submitted run + observation.
type Candidate struct {
	CandidateID       string         `json:"candidate_id"`
	ChangeID          string         `json:"change_id"`
	WorkOrderID       string         `json:"work_order_id"`
	Lineage           Lineage        `json:"lineage"`
	Run               AgentRun       `json:"run"`
	Observation       Observation    `json:"observation"`
	Gates             []GateResult   `json:"gates"`
	Violations        []Violation    `json:"violations"`
	Verdict           string         `json:"verdict"`
	Status            string         `json:"status"` // EVALUATED, ACCEPTED_FOR_PROMOTION, REJECTED, STALE (view-only)
	RecommendedAction string         `json:"recommended_action"`
	HumanDecision     *HumanDecision `json:"human_decision"`
	SubmittedBy       string         `json:"submitted_by"`
	Reason            string         `json:"reason"`
	CreatedAt         string         `json:"created_at"`
}

// SubmitCandidateRequest carries the run declaration and the observation.
type SubmitCandidateRequest struct {
	Mutation
	Run         AgentRun    `json:"run"`
	Observation Observation `json:"observation"`
}

// CandidateDecisionRequest is the human decision on an evaluated candidate.
type CandidateDecisionRequest struct {
	Mutation
	Decision  string `json:"decision"` // ACCEPT or REJECT
	Role      string `json:"role"`
	Rationale string `json:"rationale"`
}

// SubmitCandidate evaluates a candidate against the issued Work Order and
// records it. Every submission is a new immutable candidate.
func (service *Service) SubmitCandidate(projectID, changeID string, request SubmitCandidateRequest) (*View, error) {
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
		return nil, newError(CodeReasonRequired, "a reason is required to submit a candidate")
	}
	if change.WorkOrder == nil {
		return nil, newError(CodeNotAccepted, "change %s has no issued Work Order; a candidate needs the contract it was produced under", changeID)
	}
	decision := latestDecision(change)
	staleness := service.staleness(change, decision, facts)
	at := service.stamp()
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "local-operator")
	run := normalizeRun(request.Run, change.WorkOrder, at)
	observation := normalizeObservation(request.Observation, at)
	candidate := Candidate{
		CandidateID: fmt.Sprintf("CAND-%03d", len(change.Candidates)+1), ChangeID: change.ChangeID, WorkOrderID: change.WorkOrder.WorkOrderID,
		Lineage: change.WorkOrder.Lineage, Run: run, Observation: observation, Status: "EVALUATED",
		SubmittedBy: actor, Reason: request.Reason, CreatedAt: at,
	}
	candidate.Gates, candidate.Violations, candidate.Verdict, candidate.RecommendedAction = evaluateCandidate(change.WorkOrder, staleness, run, observation)
	change.Candidates = append(change.Candidates, candidate)
	change.UpdatedAt = at
	service.audit(state, at, actor, "candidate.submit", candidate.CandidateID, fmt.Sprintf("%s: %s", candidate.Verdict, request.Reason))
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(change, facts)
	return &view, nil
}

// DecideCandidate records the second human decision: accept the candidate
// for promotion, or reject it. Only CANDIDATE_ACCEPTABLE can be accepted.
func (service *Service) DecideCandidate(projectID, changeID, candidateID string, request CandidateDecisionRequest) (*View, error) {
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
	actor := strings.TrimSpace(request.Actor)
	if actor == "" {
		return nil, newError(CodeActorRequired, "a candidate decision must name its actor")
	}
	rationale := firstNonEmpty(strings.TrimSpace(request.Rationale), strings.TrimSpace(request.Reason))
	if rationale == "" {
		return nil, newError(CodeReasonRequired, "a rationale is required for a candidate decision")
	}
	var candidate *Candidate
	for index := range change.Candidates {
		if change.Candidates[index].CandidateID == candidateID {
			candidate = &change.Candidates[index]
		}
	}
	if candidate == nil {
		return nil, newError(CodeCandidateNotFound, "candidate %q is not recorded on %s", candidateID, changeID)
	}
	if candidate.HumanDecision != nil {
		return nil, newError(CodeAlreadyDecided, "candidate %s already has a human decision (%s)", candidateID, candidate.HumanDecision.Decision)
	}
	staleness := service.staleness(change, latestDecision(change), facts)
	at := service.stamp()
	decision := strings.ToUpper(strings.TrimSpace(request.Decision))
	switch decision {
	case "REJECT", "REJECTED":
		candidate.Status = "REJECTED"
		candidate.HumanDecision = &HumanDecision{Actor: actor, Role: firstNonEmpty(strings.TrimSpace(request.Role), "undeclared"), At: at, Decision: "REJECTED", Reason: rationale}
	case "ACCEPT", "ACCEPTED":
		if staleness.Stale {
			return nil, newError(CodeDecisionStale, "the decision behind this candidate is STALE: %s", staleness.Reason)
		}
		if candidate.Verdict != VerdictAcceptable {
			return nil, newError(CodeCandidateBlocked, "candidate %s is %s and cannot be accepted; resolve the gate results and submit a new candidate", candidateID, candidate.Verdict)
		}
		if actor == candidate.Run.StartedBy && !candidateHasWaiver(candidate) {
			return nil, newError(CodeCandidateBlocked, "the actor who started the run cannot also accept the candidate without documented single-operator controls")
		}
		candidate.Status = "ACCEPTED_FOR_PROMOTION"
		candidate.HumanDecision = &HumanDecision{Actor: actor, Role: firstNonEmpty(strings.TrimSpace(request.Role), "undeclared"), At: at, Decision: "ACCEPTED", Reason: rationale}
		change.Status = StatusCandidateAccepted
	default:
		return nil, newError(CodeInvalidRequest, "decision must be ACCEPT or REJECT")
	}
	change.UpdatedAt = at
	service.audit(state, at, actor, "candidate."+strings.ToLower(candidate.HumanDecision.Decision), candidate.CandidateID, rationale)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(change, facts)
	return &view, nil
}

// evaluateCandidate is deterministic: same contract + observation → same gates.
func evaluateCandidate(order *AgentWorkOrder, staleness Staleness, run AgentRun, observation Observation) ([]GateResult, []Violation, string, string) {
	gates := []GateResult{}
	violations := []Violation{}
	add := func(gate GateResult) { gates = append(gates, gate) }

	// 1. The run must declare exactly the issued contract.
	switch {
	case run.WorkOrderID != order.WorkOrderID:
		add(GateResult{Gate: "work_order_bound", Status: GateBlocked, Detail: fmt.Sprintf("run declares %s but the issued Work Order is %s", run.WorkOrderID, order.WorkOrderID)})
		violations = append(violations, Violation{Rule: "work_order_mismatch", Detail: "the run was not produced under the issued Work Order"})
	case run.WorkOrderHash != order.WorkOrderHash:
		add(GateResult{Gate: "work_order_bound", Status: GateBlocked, Detail: "run declares a different work_order_hash than the issued contract; the agent may have worked from an edited or stale order"})
		violations = append(violations, Violation{Rule: "work_order_hash_mismatch", Detail: "declared hash " + shortHash(run.WorkOrderHash) + " ≠ issued " + shortHash(order.WorkOrderHash)})
	default:
		add(GateResult{Gate: "work_order_bound", Status: GatePass, Detail: order.WorkOrderID + " " + shortHash(order.WorkOrderHash)})
	}

	// 2. The decision behind the order must still be current.
	if staleness.Stale {
		add(GateResult{Gate: "decision_current", Status: GateBlocked, Detail: staleness.Reason})
		violations = append(violations, Violation{Rule: "decision_stale", Detail: staleness.Reason})
	} else {
		add(GateResult{Gate: "decision_current", Status: GatePass, Detail: fmt.Sprintf("%s v%d bound to topology v%d", order.Lineage.DecisionID, order.Lineage.DecisionVersion, order.Lineage.TopologyVersion)})
	}

	// 3. Branch and base.
	switch {
	case observation.Branch == "":
		add(GateResult{Gate: "branch_matches", Status: GateNeedsEvidence, Detail: "no branch observed", Missing: []string{"branch"}})
	case observation.Branch != order.TargetBranch:
		add(GateResult{Gate: "branch_matches", Status: GateBlocked, Detail: fmt.Sprintf("observed branch %s, Work Order requires %s", observation.Branch, order.TargetBranch)})
		violations = append(violations, Violation{Rule: "branch_mismatch", Detail: observation.Branch + " ≠ " + order.TargetBranch})
	case observation.BaseBranch != "" && observation.BaseBranch != order.BaseBranch:
		add(GateResult{Gate: "branch_matches", Status: GateBlocked, Detail: fmt.Sprintf("observed base %s, Work Order requires %s", observation.BaseBranch, order.BaseBranch)})
		violations = append(violations, Violation{Rule: "base_branch_mismatch", Detail: observation.BaseBranch + " ≠ " + order.BaseBranch})
	default:
		add(GateResult{Gate: "branch_matches", Status: GatePass, Detail: observation.Branch + " ← " + firstNonEmpty(observation.BaseBranch, order.BaseBranch)})
	}

	// 4. Head commit must be observed; the run's declaration alone is not evidence.
	if observation.HeadCommit == "" {
		add(GateResult{Gate: "commit_observed", Status: GateNeedsEvidence, Detail: "no head commit observed for the candidate", Missing: []string{"head_commit"}})
	} else {
		add(GateResult{Gate: "commit_observed", Status: GatePass, Detail: shortHash(observation.HeadCommit)})
	}

	// 5. Every changed path must fall inside allowed_paths (UI-10).
	if len(observation.ChangedPaths) == 0 {
		add(GateResult{Gate: "allowed_paths", Status: GateNeedsEvidence, Detail: "no changed paths observed; a candidate with no diff cannot be reviewed", Missing: []string{"changed_paths"}})
	} else {
		outside := []string{}
		for _, changed := range observation.ChangedPaths {
			if !pathAllowed(changed, order.AllowedPaths) {
				outside = append(outside, changed)
			}
		}
		if len(outside) > 0 {
			add(GateResult{Gate: "allowed_paths", Status: GateBlocked, Detail: fmt.Sprintf("%d path(s) outside the issued allowed_paths", len(outside)), Paths: outside})
			for _, changed := range outside {
				violations = append(violations, Violation{Rule: "path_out_of_scope", Path: changed, Detail: "changed outside allowed_paths " + strings.Join(order.AllowedPaths, ", ")})
			}
		} else {
			add(GateResult{Gate: "allowed_paths", Status: GatePass, Detail: fmt.Sprintf("%d changed path(s) all within allowed_paths", len(observation.ChangedPaths))})
		}
		// Declared vs observed drift is informational but recorded.
		if len(run.DeclaredPaths) > 0 {
			undeclared := difference(observation.ChangedPaths, run.DeclaredPaths)
			if len(undeclared) > 0 {
				add(GateResult{Gate: "declaration_matches_diff", Status: GatePass, Detail: "observed diff includes paths the run did not declare; the observed diff governs", Paths: undeclared})
			}
		}
	}

	// 6. Forbidden actions of the form modify:<prefix> are checked against the diff.
	forbiddenHits := []string{}
	for _, action := range order.ForbiddenActions {
		if !strings.HasPrefix(action, "modify:") {
			continue
		}
		prefix := strings.TrimPrefix(action, "modify:")
		for _, changed := range observation.ChangedPaths {
			if pathAllowed(changed, []string{prefix}) {
				forbiddenHits = append(forbiddenHits, changed)
				violations = append(violations, Violation{Rule: "forbidden_action", Path: changed, Detail: "matches forbidden action " + action})
			}
		}
	}
	if len(forbiddenHits) > 0 {
		add(GateResult{Gate: "forbidden_actions", Status: GateBlocked, Detail: "diff touches a forbidden area", Paths: forbiddenHits})
	} else {
		add(GateResult{Gate: "forbidden_actions", Status: GatePass, Detail: "no forbidden path touched (non-path actions are enforced at promotion)"})
	}

	// 7. Required checks: PASS evidence for each, except review-type and promotion-only ones.
	checksByName := map[string]CheckResult{}
	for _, check := range observation.Checks {
		checksByName[strings.ToLower(strings.TrimSpace(check.Name))] = check
	}
	missing, failed, deferred := []string{}, []string{}, []string{}
	for _, required := range order.RequiredChecks {
		name := strings.ToLower(required)
		switch {
		case name == "allowed-paths-diff-check", name == "independent-review", name == "decision-record", name == "review", name == "single-operator-controls", name == "ai-review", name == "code-review":
			continue // satisfied by gates 1, 2, 5, 8 or the review evidence below
		case promotionOnlyChecks[name]:
			deferred = append(deferred, required)
			continue
		}
		check, ok := checksByName[name]
		switch {
		case !ok:
			missing = append(missing, required)
		case strings.ToUpper(check.Status) != "PASS":
			failed = append(failed, required)
		}
	}
	switch {
	case len(failed) > 0:
		add(GateResult{Gate: "required_checks", Status: GateBlocked, Detail: "required check(s) FAILED: " + strings.Join(failed, ", "), Missing: missing})
		violations = append(violations, Violation{Rule: "check_failed", Detail: strings.Join(failed, ", ")})
	case len(missing) > 0:
		add(GateResult{Gate: "required_checks", Status: GateNeedsEvidence, Detail: "no PASS evidence for: " + strings.Join(missing, ", "), Missing: missing})
	default:
		add(GateResult{Gate: "required_checks", Status: GatePass, Detail: "all required pre-promotion checks have PASS evidence"})
	}
	if len(deferred) > 0 {
		add(GateResult{Gate: "promotion_checks", Status: GateDeferred, Detail: "verified by the promotion gate after deployment: " + strings.Join(deferred, ", ")})
	}

	// 8. Independent review (UI-11): a human other than the run starter, or
	// documented single-operator controls plus an AI review when policy allows.
	reviewGate := independentReviewGate(order, run, observation)
	add(reviewGate)

	// 9. Work Order expiry.
	if order.ExpiresAt != "" && run.StartedAt != "" {
		expires, errE := time.Parse(time.RFC3339, order.ExpiresAt)
		started, errS := time.Parse(time.RFC3339, run.StartedAt)
		if errE == nil && errS == nil && started.After(expires) {
			add(GateResult{Gate: "work_order_expiry", Status: GateBlocked, Detail: "run started after the Work Order expired at " + order.ExpiresAt})
			violations = append(violations, Violation{Rule: "work_order_expired", Detail: order.ExpiresAt})
		} else {
			add(GateResult{Gate: "work_order_expiry", Status: GatePass, Detail: "run started before " + order.ExpiresAt})
		}
	}

	verdict := VerdictAcceptable
	for _, gate := range gates {
		switch gate.Status {
		case GateBlocked:
			verdict = VerdictBlocked
		case GateNeedsEvidence:
			if verdict != VerdictBlocked {
				verdict = VerdictNeedsEvidence
			}
		case GateNeedsReview:
			if verdict == VerdictAcceptable {
				verdict = VerdictNeedsReview
			}
		}
	}
	return gates, violations, verdict, recommendedAction(verdict, gates)
}

func independentReviewGate(order *AgentWorkOrder, run AgentRun, observation Observation) GateResult {
	for _, review := range observation.Reviews {
		if strings.EqualFold(review.Kind, "human") && strings.EqualFold(review.Verdict, "APPROVED") && review.Reviewer != "" && review.Reviewer != run.StartedBy && review.Reviewer != run.Agent.Name {
			return GateResult{Gate: "independent_review", Status: GatePass, Detail: "approved by " + review.Reviewer + " (independent of " + firstNonEmpty(run.StartedBy, "the run starter") + ")"}
		}
	}
	for _, review := range observation.Reviews {
		if strings.EqualFold(review.Kind, "human") && strings.EqualFold(review.Verdict, "CHANGES_REQUESTED") {
			return GateResult{Gate: "independent_review", Status: GateBlocked, Detail: review.Reviewer + " requested changes"}
		}
	}
	policyAllowsSingleOperator := false
	for _, check := range order.RequiredChecks {
		if strings.EqualFold(check, "single-operator-controls") {
			policyAllowsSingleOperator = true
		}
	}
	aiApproved := false
	for _, review := range observation.Reviews {
		if strings.EqualFold(review.Kind, "ai") && strings.EqualFold(review.Verdict, "APPROVED") {
			aiApproved = true
		}
	}
	if policyAllowsSingleOperator && aiApproved && observation.CompensatingControls != nil && observation.CompensatingControls.Declared {
		return GateResult{Gate: "independent_review", Status: GateWaived, Detail: "no second human; accepted under documented single-operator controls (" + firstNonEmpty(observation.CompensatingControls.EvidenceRef, "undeclared ref") + ") plus AI technical review — low-risk staging only, never production"}
	}
	missing := []string{"human review by someone other than " + firstNonEmpty(run.StartedBy, "the run starter")}
	if policyAllowsSingleOperator {
		missing = append(missing, "or: AI review APPROVED + declared single-operator controls")
	}
	return GateResult{Gate: "independent_review", Status: GateNeedsReview, Detail: "no independent review observed; the candidate cannot enter " + order.TargetEnvironmentID, Missing: missing}
}

func candidateHasWaiver(candidate *Candidate) bool {
	for _, gate := range candidate.Gates {
		if gate.Gate == "independent_review" && gate.Status == GateWaived {
			return true
		}
	}
	return false
}

func recommendedAction(verdict string, gates []GateResult) string {
	switch verdict {
	case VerdictAcceptable:
		return "A human may accept this candidate for promotion; acceptance is a separate decision from the requirement and from release."
	case VerdictBlocked:
		reasons := []string{}
		for _, gate := range gates {
			if gate.Status == GateBlocked {
				reasons = append(reasons, gate.Gate)
			}
		}
		return "Blocked by " + strings.Join(reasons, ", ") + ". Remove the out-of-scope change or re-run under a new decision version; a green test suite cannot override a policy violation."
	case VerdictNeedsEvidence:
		return "Supply the missing observations (branch, commit, diff or check results) and submit a new candidate; the gate does not assume they passed."
	default:
		return "Obtain an independent review (or the documented single-operator controls where policy allows) and submit a new candidate."
	}
}

// pathAllowed reports whether changed lies under one of the allowed entries.
// An entry may be an exact file, a directory prefix (with or without trailing
// slash) or a glob such as internal/attachments/**.
func pathAllowed(changed string, allowed []string) bool {
	changed = strings.TrimPrefix(path.Clean("/"+changed), "/")
	for _, entry := range allowed {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.HasSuffix(entry, "/**") {
			prefix := strings.TrimSuffix(entry, "/**")
			if changed == prefix || strings.HasPrefix(changed, prefix+"/") {
				return true
			}
			continue
		}
		if strings.HasSuffix(entry, "/") {
			if strings.HasPrefix(changed, entry) {
				return true
			}
			continue
		}
		if changed == entry || strings.HasPrefix(changed, entry+"/") {
			return true
		}
		if matched, _ := path.Match(entry, changed); matched {
			return true
		}
	}
	return false
}

func difference(observed, declared []string) []string {
	set := map[string]bool{}
	for _, item := range declared {
		set[strings.TrimPrefix(path.Clean("/"+item), "/")] = true
	}
	out := []string{}
	for _, item := range observed {
		if !set[strings.TrimPrefix(path.Clean("/"+item), "/")] {
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeRun(run AgentRun, order *AgentWorkOrder, at string) AgentRun {
	run.Kind = "AgentRunRecord"
	run.SchemaVersion = "agent-run-record/v1"
	run.RunID = strings.TrimSpace(run.RunID)
	if run.RunID == "" {
		run.RunID = "ARR-" + strings.TrimPrefix(order.WorkOrderID, "AWO-")
	}
	run.WorkOrderID = strings.TrimSpace(run.WorkOrderID)
	run.WorkOrderHash = strings.TrimSpace(run.WorkOrderHash)
	run.Adapter = firstNonEmpty(strings.TrimSpace(run.Adapter), "manual-declaration")
	run.StartedBy = strings.TrimSpace(run.StartedBy)
	if run.StartedAt == "" {
		run.StartedAt = at
	}
	run.DeclaredCommands = cleanList(run.DeclaredCommands)
	run.DeclaredPaths = cleanList(run.DeclaredPaths)
	return run
}

func normalizeObservation(observation Observation, at string) Observation {
	observation.Source = firstNonEmpty(strings.TrimSpace(observation.Source), "declared")
	observation.Branch = strings.TrimSpace(observation.Branch)
	observation.BaseBranch = strings.TrimSpace(observation.BaseBranch)
	observation.HeadCommit = strings.TrimSpace(observation.HeadCommit)
	observation.BaseCommit = strings.TrimSpace(observation.BaseCommit)
	observation.ChangedPaths = cleanList(observation.ChangedPaths)
	if observation.Checks == nil {
		observation.Checks = []CheckResult{}
	}
	if observation.Reviews == nil {
		observation.Reviews = []ReviewEvidence{}
	}
	if observation.ObservedAt == "" {
		observation.ObservedAt = at
	}
	return observation
}

func shortHash(value string) string {
	if len(value) > 19 {
		return value[:19]
	}
	return value
}
