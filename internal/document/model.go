// Package document implements the governed document baseline and the
// Context Pack rebuild (UI-20): which documents a Project declares as
// required for which readiness stage, whether each one is actually present
// now, and a re-buildable, hashed Context Pack that lists every source with
// its path, version and content hash — and lists what is MISSING instead of
// inventing it. A pack with a missing required source is PARTIAL.
//
// The package reads the consumer Project (stat + hash of declared files)
// and never writes into it; its state lives in the governance state dir.
package document

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	Kind          = "DocumentBaselineState"
	SchemaVersion = "document-baseline/v1"
	PackSchema    = "context-pack/v1"
	PolicyVersion = "project-workspace-v1"
)

// Document statuses as observed now.
const (
	StatusCurrent    = "CURRENT"
	StatusMissing    = "MISSING"
	StatusDerived    = "DERIVED"
	StatusStale      = "STALE"
	StatusNeedsInput = "NEEDS_INPUT"
)

// Context Pack statuses.
const (
	PackDerived = "DERIVED" // every declared source present and hashed
	PackPartial = "PARTIAL" // at least one declared source is missing; nothing synthesized
)

// Readiness stages a document can be required for.
var Stages = []string{"decision", "development", "work-order", "staging"}

// Declared is one document in the baseline: what the manifest (origin
// manifest) or an operator (origin operator) says the Project must have.
type Declared struct {
	Path              string   `json:"path"`
	Kind              string   `json:"kind"`
	SourceOfTruth     bool     `json:"source_of_truth"`
	RequiredFor       []string `json:"required_for"`
	Origin            string   `json:"origin"` // manifest | operator
	DeclaredInVersion int      `json:"declared_in_version"`
}

// BaselineChange explains why a baseline version exists.
type BaselineChange struct {
	Operation string `json:"operation"` // BOOTSTRAP, DECLARE, WITHDRAW
	Path      string `json:"path,omitempty"`
	Reason    string `json:"reason"`
	Impact    string `json:"impact"`
}

// BaselineVersion is an immutable snapshot of the declared document set.
type BaselineVersion struct {
	Version         int            `json:"version"`
	PreviousVersion int            `json:"previous_version"`
	CreatedAt       string         `json:"created_at"`
	Actor           string         `json:"actor"`
	Origin          string         `json:"origin"` // manifest | operator
	Change          BaselineChange `json:"change"`
	Documents       []Declared     `json:"documents"`
	BaselineHash    string         `json:"baseline_hash"`
}

// Source is one document as observed for a pack or a live view.
type Source struct {
	Path          string   `json:"path"`
	Kind          string   `json:"kind"`
	SourceOfTruth bool     `json:"source_of_truth"`
	RequiredFor   []string `json:"required_for"`
	Origin        string   `json:"origin"`
	Version       string   `json:"version"` // baseline version the declaration comes from, e.g. v1
	Status        string   `json:"status"`
	ContentHash   string   `json:"content_hash,omitempty"`
	Bytes         int64    `json:"bytes,omitempty"`
	ObservedAt    string   `json:"observed_at"`
	StaleSources  []string `json:"stale_sources,omitempty"`
	AccessScope   string   `json:"access_scope"`
}

// MissingSource is a declared document that does not exist now, with the
// readiness stages it blocks. It is listed, never filled in.
type MissingSource struct {
	Path            string   `json:"path"`
	Kind            string   `json:"kind"`
	RequiredFor     []string `json:"required_for"`
	ReadinessImpact []string `json:"readiness_impact"`
	Origin          string   `json:"origin"`
}

// Readiness is the document readiness of one stage.
type Readiness struct {
	Status  string   `json:"status"` // READY or NEEDS_INPUT
	Reasons []string `json:"reasons"`
}

// ContextPack is the rebuilt derived context. Its shape extends the
// consumer fixture's docs/ai/context-pack.json (context-pack/v1).
type ContextPack struct {
	Kind               string               `json:"kind"`
	SchemaVersion      string               `json:"schema_version"`
	Derived            bool                 `json:"derived"`
	Status             string               `json:"status"`
	ProjectID          string               `json:"project_id"`
	PackID             string               `json:"pack_id"`
	GeneratedAt        string               `json:"generated_at"`
	GeneratedBy        string               `json:"generated_by"`
	Reason             string               `json:"reason"`
	BaselineVersion    int                  `json:"baseline_version"`
	BaselineHash       string               `json:"baseline_hash"`
	TopologyVersion    int                  `json:"topology_version,omitempty"`
	TopologyConfigHash string               `json:"topology_config_hash,omitempty"`
	SourceSnapshotHash string               `json:"source_snapshot_hash"`
	Sources            []Source             `json:"sources"`
	Missing            []MissingSource      `json:"missing"`
	Readiness          map[string]Readiness `json:"readiness"`
	DecisionRefs       []string             `json:"decision_refs"`
	Limitations        []string             `json:"limitations"`
	PolicyVersion      string               `json:"policy_version"`
	PackHash           string               `json:"pack_hash"`
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

// State is the persisted document governance state of one Project.
type State struct {
	Kind           string            `json:"kind"`
	SchemaVersion  string            `json:"schema_version"`
	ProjectID      string            `json:"project_id"`
	CurrentVersion int               `json:"current_version"`
	Versions       []BaselineVersion `json:"versions"`
	NextPack       int               `json:"next_pack"`
	Packs          []ContextPack     `json:"packs"`
	Audit          []AuditEvent      `json:"audit"`
}

// Current returns the current baseline version.
func (state *State) Current() *BaselineVersion {
	for index := range state.Versions {
		if state.Versions[index].Version == state.CurrentVersion {
			return &state.Versions[index]
		}
	}
	return nil
}

// LatestPack returns the most recent rebuilt pack, or nil.
func (state *State) LatestPack() *ContextPack {
	if len(state.Packs) == 0 {
		return nil
	}
	return &state.Packs[len(state.Packs)-1]
}

// Drift explains why the latest pack no longer reflects the sources.
type Drift struct {
	Stale  bool   `json:"stale"`
	Reason string `json:"reason,omitempty"`
}

// DeclaredContext is the consumer's own derived context file as the
// read-only registry observes it (docs/ai/context-pack.json).
type DeclaredContext struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// View is the API shape: baseline, live document statuses, readiness by
// stage, the latest rebuilt pack and whether it drifted.
type View struct {
	ProjectID        string               `json:"project_id"`
	CurrentVersion   int                  `json:"current_version"`
	Baseline         BaselineVersion      `json:"baseline"`
	Documents        []Source             `json:"documents"`
	Missing          []MissingSource      `json:"missing"`
	Readiness        map[string]Readiness `json:"readiness"`
	DeclaredContext  DeclaredContext      `json:"declared_context"`
	LiveContext      string               `json:"live_context_status"`
	LatestPack       *ContextPack         `json:"latest_pack"`
	PackDrift        Drift                `json:"pack_drift"`
	Packs            []string             `json:"packs"`
	Versions         []BaselineVersion    `json:"versions"`
	Audit            []AuditEvent         `json:"audit"`
	ObservedAt       string               `json:"observed_at"`
	ObservedRoot     string               `json:"observed_root"`
	ReadOnlyConsumer bool                 `json:"read_only_consumer"`
}

func hashJSON(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
