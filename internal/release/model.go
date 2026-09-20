// Package release implements staging promotion (VS-006): one or more
// accepted candidates are bundled into a Release with a manifest (commits,
// image digest, target, config hash, policy), a third human approval bound to
// that manifest, a deployment record that is checked for digest, target and
// config drift, and a re-readable Release Receipt.
//
// The package never deploys anything itself. It decides whether a deployment
// may proceed and verifies what was observed afterwards; the deploy is run by
// the operator (or a later adapter) and recorded here with an idempotency key.
package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	Kind          = "ReleaseLedgerState"
	SchemaVersion = "release-ledger/v1"
	PolicyVersion = "project-workspace-v1"
)

// Release statuses.
const (
	StatusGateBlocked      = "GATE_BLOCKED"
	StatusAwaitingBuild    = "AWAITING_BUILD"
	StatusReadyForApproval = "READY_FOR_APPROVAL"
	StatusApproved         = "APPROVED"
	StatusPromoted         = "PROMOTED"
	StatusPromotionFailed  = "PROMOTION_FAILED"
	StatusRejected         = "REJECTED"
	StatusStale            = "STALE" // view-only
)

// Gate statuses.
const (
	GatePass          = "PASS"
	GateBlocked       = "BLOCKED"
	GateNeedsEvidence = "NEEDS_EVIDENCE"
	GateNeedsApproval = "NEEDS_APPROVAL"
	GateStale         = "STALE"
)

// ApprovalTTL bounds how long an approval binds a manifest.
const ApprovalTTLHours = 72

// GateResult is one deterministic check.
type GateResult struct {
	Gate     string `json:"gate"`
	ChangeID string `json:"change_id,omitempty"`
	Status   string `json:"status"`
	Detail   string `json:"detail"`
}

// ReleaseChange is the per-change lineage carried into the manifest and receipt.
type ReleaseChange struct {
	ChangeID            string `json:"change_id"`
	Title               string `json:"title"`
	DecisionID          string `json:"decision_id"`
	DecisionVersion     int    `json:"decision_version"`
	WorkOrderID         string `json:"work_order_id"`
	WorkOrderHash       string `json:"work_order_hash"`
	CandidateID         string `json:"candidate_id"`
	RunID               string `json:"run_id"`
	RunStartedBy        string `json:"run_started_by"`
	Branch              string `json:"branch"`
	HeadCommit          string `json:"head_commit"`
	ReviewGate          string `json:"review_gate"`
	SourceSnapshotHash  string `json:"source_snapshot_hash"`
	TargetEnvironment   string `json:"target_environment_id"`
	CandidateAcceptedBy string `json:"candidate_accepted_by"`
}

// EnvironmentConfig is the target environment configuration a release binds
// to; its hash is what drift is measured against.
type EnvironmentConfig struct {
	EnvironmentID    string   `json:"environment_id"`
	Type             string   `json:"type"`
	TargetRef        string   `json:"target_ref"`
	RequiredEvidence []string `json:"required_evidence"`
	ApproverPolicy   string   `json:"approver_policy"`
	TopologyVersion  int      `json:"topology_version"`
	ConfigHash       string   `json:"config_hash"`
}

// BuildEvidence is the observed build that produced the image to promote.
type BuildEvidence struct {
	ImageDigest     string   `json:"image_digest"`
	ImageRef        string   `json:"image_ref"`
	BuildID         string   `json:"build_id"`
	SourceCommit    string   `json:"source_commit"`
	IncludesCommits []string `json:"includes_commits"`
	EvidenceRef     string   `json:"evidence_ref"`
	ObservedAt      string   `json:"observed_at"`
}

// Manifest is the release identity. Its hash is what the approval binds.
type Manifest struct {
	ReleaseID         string            `json:"release_id"`
	ProjectID         string            `json:"project_id"`
	Changes           []ReleaseChange   `json:"changes"`
	Transition        string            `json:"transition"`
	SourceEnvironment string            `json:"source_environment_id"`
	Environment       EnvironmentConfig `json:"environment"`
	Build             *BuildEvidence    `json:"build"`
	PolicyVersion     string            `json:"policy_version"`
	ManifestHash      string            `json:"manifest_hash"`
}

// Approval is the third human decision: approve this exact manifest for
// this exact environment, for a bounded time.
type Approval struct {
	Actor        string `json:"actor"`
	Role         string `json:"role"`
	At           string `json:"at"`
	Decision     string `json:"decision"` // APPROVED or REJECTED
	Reason       string `json:"reason"`
	ManifestHash string `json:"manifest_hash"`
	ExpiresAt    string `json:"expires_at"`
	Waiver       string `json:"waiver,omitempty"` // single-operator note when the approver also ran the agent
}

// SmokeResult is the post-deploy verification outcome.
type SmokeResult struct {
	Status      string `json:"status"` // PASS or FAIL
	EvidenceRef string `json:"evidence_ref"`
	At          string `json:"at"`
}

// DeploymentRecord is what was observed after the operator deployed.
type DeploymentRecord struct {
	AttemptID          string       `json:"attempt_id"`
	IdempotencyKey     string       `json:"idempotency_key"`
	OperationID        string       `json:"operation_id"`
	Source             string       `json:"source"` // declared or cloud-run
	Revision           string       `json:"revision"`
	ServiceURL         string       `json:"service_url"`
	DeployedDigest     string       `json:"deployed_digest"`
	DeployedTargetRef  string       `json:"deployed_target_ref"`
	DeployedConfigHash string       `json:"deployed_config_hash"`
	DeployedAt         string       `json:"deployed_at"`
	DeployedBy         string       `json:"deployed_by"`
	Smoke              *SmokeResult `json:"smoke"`
	Gates              []GateResult `json:"gates"`
	Outcome            string       `json:"outcome"` // PROMOTED, PROMOTION_FAILED, REPLAYED
	RecordedAt         string       `json:"recorded_at"`
}

// Receipt is the re-readable result of a governed release.
type Receipt struct {
	Kind          string            `json:"kind"`
	SchemaVersion string            `json:"schema_version"`
	ReceiptID     string            `json:"receipt_id"`
	ReleaseID     string            `json:"release_id"`
	ProjectID     string            `json:"project_id"`
	Status        string            `json:"status"` // PROMOTED or PROMOTION_FAILED
	Transition    string            `json:"transition"`
	Environment   EnvironmentConfig `json:"environment"`
	Changes       []ReleaseChange   `json:"changes"`
	Build         BuildEvidence     `json:"build"`
	Approval      Approval          `json:"approval"`
	Deployment    DeploymentRecord  `json:"deployment"`
	ManifestHash  string            `json:"manifest_hash"`
	EvidenceRefs  []string          `json:"evidence_refs"`
	IssuedAt      string            `json:"issued_at"`
	ReceiptHash   string            `json:"receipt_hash"`
}

// Release is the ledger entry.
type Release struct {
	ReleaseID   string             `json:"release_id"`
	ProjectID   string             `json:"project_id"`
	Status      string             `json:"status"`
	Manifest    Manifest           `json:"manifest"`
	Gates       []GateResult       `json:"gates"`
	Verdict     string             `json:"verdict"`
	Approval    *Approval          `json:"approval"`
	Deployments []DeploymentRecord `json:"deployments"`
	Receipt     *Receipt           `json:"receipt"`
	Reason      string             `json:"reason"`
	CreatedBy   string             `json:"created_by"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

// Staleness explains why an approval or manifest no longer binds.
type Staleness struct {
	Stale  bool   `json:"stale"`
	Reason string `json:"reason,omitempty"`
}

// View is the API shape: the release plus live gate re-evaluation.
type View struct {
	Release   Release      `json:"release"`
	LiveGates []GateResult `json:"live_gates"`
	Staleness Staleness    `json:"staleness"`
}

// AuditEvent is append-only history.
type AuditEvent struct {
	Sequence int    `json:"sequence"`
	At       string `json:"at"`
	Actor    string `json:"actor"`
	Action   string `json:"action"`
	Object   string `json:"object"`
	Reason   string `json:"reason"`
}

// State is the persisted release ledger of one Project.
type State struct {
	Kind          string       `json:"kind"`
	SchemaVersion string       `json:"schema_version"`
	ProjectID     string       `json:"project_id"`
	NextRelease   int          `json:"next_release"`
	NextReceipt   int          `json:"next_receipt"`
	Releases      []Release    `json:"releases"`
	Audit         []AuditEvent `json:"audit"`
}

func hashJSON(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
