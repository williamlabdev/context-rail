// Package change implements the Change Decision Pack (VS-004): a Request
// becomes a Change, a deterministic readiness evaluation either blocks it as
// NEEDS_INPUT or lets a human accept a DecisionRecord, and the accepted
// record is the single source from which both the human Change Decision
// Brief and the Agent Context Pack are rendered. An accepted, non-stale
// record compiles into a hashed Agent Work Order.
//
// Nothing here invents a missing fact: unknown inputs stay NEEDS_INPUT, the
// advisor only proposes candidates, and every derived artifact carries the
// decision_id, version, source_snapshot_hash, topology_version and
// policy_version it was rendered from.
package change

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	Kind          = "ChangeLedgerState"
	SchemaVersion = "change-ledger/v1"
	PolicyVersion = "project-workspace-v1"
)

// Change statuses.
const (
	StatusDraft             = "DRAFT"
	StatusNeedsInput        = "NEEDS_INPUT"
	StatusDecisionReady     = "DECISION_READY"
	StatusAccepted          = "ACCEPTED"
	StatusCandidateAccepted = "CANDIDATE_ACCEPTED"
	StatusRejected          = "REJECTED"
	StatusStale             = "STALE"
)

// AcceptanceCriterion is one testable expectation supplied by the requester.
type AcceptanceCriterion struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Request is the requester-supplied input of a Change. Everything the
// decision needs must be declared here or arrive later through Inputs.
type Request struct {
	Title     string `json:"title"`
	Objective string `json:"objective"`
	// OwnerSummary is the requester's plain-language statement of what the
	// change does, written for the business owner. The Brief leads with it;
	// nothing downstream may synthesize it.
	OwnerSummary        string                `json:"owner_summary,omitempty"`
	ScopeIncluded       []string              `json:"scope_included"`
	ScopeExcluded       []string              `json:"scope_excluded"`
	AcceptanceCriteria  []AcceptanceCriterion `json:"acceptance_criteria"`
	AllowedPaths        []string              `json:"allowed_paths"`
	ForbiddenActions    []string              `json:"forbidden_actions"`
	TargetEnvironmentID string                `json:"target_environment_id"`
	BusinessConstraints map[string]string     `json:"business_constraints"`
	RequestedBy         string                `json:"requested_by"`
}

// MissingInput names one thing a human must supply before a decision.
type MissingInput struct {
	Field     string `json:"field"`
	OwnerRole string `json:"owner_role"`
	Reason    string `json:"reason"`
}

// Evaluation is the deterministic readiness result for one Change version.
type Evaluation struct {
	Status              string          `json:"status"` // DECISION_READY or NEEDS_INPUT
	MissingInputs       []MissingInput  `json:"missing_inputs"`
	Observations        []string        `json:"observations"`
	SourceSnapshotHash  string          `json:"source_snapshot_hash"`
	TopologyVersion     int             `json:"topology_version"`
	TopologyConfigHash  string          `json:"topology_config_hash"`
	TargetEnvironment   *EnvironmentRef `json:"target_environment"`
	SourceEnvironment   *EnvironmentRef `json:"source_environment"`
	AllowedTransition   string          `json:"allowed_transition"`
	ProjectContext      string          `json:"project_context_status"`
	DecisionInputsReady string          `json:"decision_inputs_readiness"`
	EvaluatedAt         string          `json:"evaluated_at"`
}

// EnvironmentRef is the topology node a Change targets or departs from.
type EnvironmentRef struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Sequence         int      `json:"sequence"`
	TargetRef        string   `json:"target_ref"`
	RequiredEvidence []string `json:"required_evidence"`
	Status           string   `json:"status"`
	Protection       string   `json:"protection,omitempty"`
}

// Option is a candidate the advisor proposes; a human selects one.
type Option struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	CostDrivers   []string `json:"cost_drivers"`
	Risks         []string `json:"risks"`
	Recommended   bool     `json:"recommended"`
	AdvisorSource string   `json:"advisor_source"` // rule-advisor or gemini
}

// ChangeVersion is one immutable evaluation of the request inputs.
type ChangeVersion struct {
	Version    int        `json:"version"`
	CreatedAt  string     `json:"created_at"`
	Actor      string     `json:"actor"`
	Reason     string     `json:"reason"`
	Request    Request    `json:"request"`
	Evaluation Evaluation `json:"evaluation"`
	Options    []Option   `json:"options"`
	Unknowns   []string   `json:"unknowns"`
}

// HumanDecision is the recorded human act; it is never inferred.
type HumanDecision struct {
	Actor    string `json:"actor"`
	Role     string `json:"role"`
	At       string `json:"at"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// DecisionRecord is the single source of truth for an accepted (or rejected)
// decision. Brief and Agent Context Pack are rendered from it and nothing else.
type DecisionRecord struct {
	DecisionID          string                `json:"decision_id"`
	ProjectID           string                `json:"project_id"`
	ChangeID            string                `json:"change_id"`
	ChangeVersion       int                   `json:"change_version"`
	Version             int                   `json:"version"`
	Status              string                `json:"status"` // ACCEPTED_FOR_DEVELOPMENT, REJECTED
	RiskLevel           string                `json:"risk_level"`
	SelectedOption      string                `json:"selected_option"`
	Alternatives        []Option              `json:"alternatives"`
	Rationale           string                `json:"rationale"`
	Objective           string                `json:"objective"`
	OwnerSummary        string                `json:"owner_summary,omitempty"`
	AcceptedScope       []string              `json:"accepted_scope"`
	OutOfScope          []string              `json:"out_of_scope"`
	AllowedPaths        []string              `json:"allowed_paths"`
	ForbiddenActions    []string              `json:"forbidden_actions"`
	AcceptanceCriteria  []AcceptanceCriterion `json:"acceptance_criteria"`
	BusinessConstraints map[string]string     `json:"business_constraints"`
	Unknowns            []string              `json:"unknowns"`
	TargetEnvironment   EnvironmentRef        `json:"target_environment"`
	SourceEnvironment   *EnvironmentRef       `json:"source_environment"`
	AllowedTransition   string                `json:"allowed_transition"`
	ProductionAction    string                `json:"production_action"`
	SourceSnapshotHash  string                `json:"source_snapshot_hash"`
	TopologyVersion     int                   `json:"topology_version"`
	TopologyConfigHash  string                `json:"topology_config_hash"`
	PolicyVersion       string                `json:"policy_version"`
	EvidenceRefs        []string              `json:"evidence_refs"`
	HumanDecision       HumanDecision         `json:"human_decision"`
	CreatedAt           string                `json:"created_at"`
}

// Lineage is the identity block every derived artifact must carry verbatim.
type Lineage struct {
	DecisionID         string `json:"decision_id"`
	DecisionVersion    int    `json:"decision_version"`
	ChangeID           string `json:"change_id"`
	ProjectID          string `json:"project_id"`
	SourceSnapshotHash string `json:"source_snapshot_hash"`
	TopologyVersion    int    `json:"topology_version"`
	TopologyConfigHash string `json:"topology_config_hash"`
	PolicyVersion      string `json:"policy_version"`
}

// BriefOption names a decision option by its human title; the ID stays for
// traceability but is never the thing a reader is asked to parse.
type BriefOption struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// BriefRoute is where the decision lets the change go next.
type BriefRoute struct {
	SourceEnvironmentID string `json:"source_environment_id,omitempty"`
	TargetEnvironmentID string `json:"target_environment_id"`
	TargetType          string `json:"target_type"`
	TargetRef           string `json:"target_ref"`
	Transition          string `json:"transition"`
	ProductionAction    string `json:"production_action"`
}

// BriefDecider is the recorded human act, as the reader sees it.
type BriefDecider struct {
	Actor     string `json:"actor"`
	Role      string `json:"role"`
	At        string `json:"at"`
	Rationale string `json:"rationale"`
}

// ChangeDecisionBrief is the human-facing rendering of a DecisionRecord.
// It is structured, not prose: the reader-facing order, labels and language
// are presentation, applied by the client; every value here is copied from
// the record or the live staleness verdict and none is synthesized.
type ChangeDecisionBrief struct {
	ArtifactType  string  `json:"artifact_type"`
	SchemaVersion string  `json:"schema_version"`
	Lineage       Lineage `json:"lineage"`
	Audience      string  `json:"audience"`
	State         string  `json:"state"` // ACCEPTED or STALE
	StaleReason   string  `json:"stale_reason,omitempty"`
	OwnerSummary  string  `json:"owner_summary"`
	// OwnerSummaryMissing marks a record decided before owner_summary was
	// required; the reader is told so instead of being shown a stand-in.
	OwnerSummaryMissing bool          `json:"owner_summary_missing"`
	Objective           string        `json:"objective"`
	Selected            BriefOption   `json:"selected"`
	Route               BriefRoute    `json:"route"`
	RiskLevel           string        `json:"risk_level"`
	Unknowns            []string      `json:"unknowns"`
	InScope             []string      `json:"in_scope"`
	OutOfScope          []string      `json:"out_of_scope"`
	RequiredEvidence    []string      `json:"required_evidence"`
	DecidedBy           BriefDecider  `json:"decided_by"`
	Alternatives        []BriefOption `json:"alternatives"`
	RenderedAt          string        `json:"rendered_at"`
}

// AcceptanceTest is the agent-facing form of an acceptance criterion.
type AcceptanceTest struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
}

// AgentContextPack is the machine-facing rendering of the same DecisionRecord.
type AgentContextPack struct {
	ArtifactType        string            `json:"artifact_type"`
	SchemaVersion       string            `json:"schema_version"`
	Lineage             Lineage           `json:"lineage"`
	Status              string            `json:"status"`
	Objective           string            `json:"objective"`
	EnvironmentTopology map[string]any    `json:"environment_topology"`
	BusinessConstraints map[string]string `json:"business_constraints"`
	AcceptedScope       []string          `json:"accepted_scope"`
	OutOfScope          []string          `json:"out_of_scope"`
	AllowedPaths        []string          `json:"allowed_paths"`
	ForbiddenActions    []string          `json:"forbidden_actions"`
	AcceptanceTests     []AcceptanceTest  `json:"acceptance_tests"`
	RequiredChecks      []string          `json:"required_checks"`
	Unknowns            []string          `json:"unknowns"`
	EvidenceRefs        []string          `json:"evidence_refs"`
	RenderedAt          string            `json:"rendered_at"`
}

// AgentWorkOrder is the bounded execution contract compiled from an accepted
// DecisionRecord. Its hash binds the agent run to exactly this content.
type AgentWorkOrder struct {
	Kind                string   `json:"kind"`
	SchemaVersion       string   `json:"schema_version"`
	WorkOrderID         string   `json:"work_order_id"`
	Lineage             Lineage  `json:"lineage"`
	Status              string   `json:"status"` // ISSUED, STALE
	HumanGate           string   `json:"human_gate"`
	Issuer              string   `json:"issuer"`
	Repository          string   `json:"repository"`
	BaseBranch          string   `json:"base_branch"`
	TargetBranch        string   `json:"target_branch"`
	TargetEnvironmentID string   `json:"target_environment_id"`
	AllowedTransition   string   `json:"allowed_transition"`
	AllowedPaths        []string `json:"allowed_paths"`
	ForbiddenActions    []string `json:"forbidden_actions"`
	AcceptanceIDs       []string `json:"acceptance_ids"`
	RequiredChecks      []string `json:"required_checks"`
	OutputPaths         []string `json:"output_paths"`
	IssuedAt            string   `json:"issued_at"`
	ExpiresAt           string   `json:"expires_at"`
	WorkOrderHash       string   `json:"work_order_hash"`
}

// Change groups the immutable versions, decision records and work order of
// one governed change.
type Change struct {
	ChangeID       string           `json:"change_id"`
	ProjectID      string           `json:"project_id"`
	Title          string           `json:"title"`
	Status         string           `json:"status"`
	CurrentVersion int              `json:"current_version"`
	Versions       []ChangeVersion  `json:"versions"`
	Decisions      []DecisionRecord `json:"decisions"`
	WorkOrder      *AgentWorkOrder  `json:"work_order"`
	Candidates     []Candidate      `json:"candidates"`
	CreatedAt      string           `json:"created_at"`
	UpdatedAt      string           `json:"updated_at"`
}

// AuditEvent is append-only governance history for the ledger.
type AuditEvent struct {
	Sequence int    `json:"sequence"`
	At       string `json:"at"`
	Actor    string `json:"actor"`
	Action   string `json:"action"`
	Object   string `json:"object"`
	Reason   string `json:"reason"`
}

// State is the persisted change ledger of one Project.
type State struct {
	Kind          string       `json:"kind"`
	SchemaVersion string       `json:"schema_version"`
	ProjectID     string       `json:"project_id"`
	NextChange    int          `json:"next_change"`
	NextDecision  int          `json:"next_decision"`
	NextWorkOrder int          `json:"next_work_order"`
	Changes       []Change     `json:"changes"`
	Audit         []AuditEvent `json:"audit"`
}

// Staleness explains why an accepted decision no longer binds.
type Staleness struct {
	Stale  bool   `json:"stale"`
	Reason string `json:"reason,omitempty"`
}

// View is what the API returns for one Change: the ledger entry plus the
// artifacts rendered from its latest DecisionRecord and the live staleness.
type View struct {
	Change    Change               `json:"change"`
	Decision  *DecisionRecord      `json:"decision"`
	Brief     *ChangeDecisionBrief `json:"brief"`
	Pack      *AgentContextPack    `json:"agent_context_pack"`
	WorkOrder *AgentWorkOrder      `json:"work_order"`
	Staleness Staleness            `json:"staleness"`
}

func hashJSON(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
