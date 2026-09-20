// Package topology implements the versioned Environment Topology for a
// governed Project: every add, edit, reorder, retire or restore creates a new
// immutable topology version, records an audit event and — when the change is
// material — marks dependent decisions as STALE with a visible reason.
//
// The package never mutates the consumer Project's manifest. Version 1 is a
// bootstrap snapshot of the manifest environments; later versions are
// ContextRail governance state kept in the configured state store.
package topology

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

const (
	Kind          = "EnvironmentTopologyState"
	SchemaVersion = "environment-topology/v1"
)

// Status is the environment lifecycle state.
type Status string

const (
	StatusDraft   Status = "DRAFT"
	StatusActive  Status = "ACTIVE"
	StatusPaused  Status = "PAUSED"
	StatusRetired Status = "RETIRED"
)

// Standard environment types. Display names are free; the standard type is
// what promotion policy and protection rules reason about.
var StandardTypes = []string{"development", "testing", "staging", "production", "uat", "other"}

// ProtectionBlockedInP0 marks a production environment: it may exist in the
// topology, be renamed for display, but cannot be retired, re-typed or
// executed against in the P0 prototype.
const ProtectionBlockedInP0 = "READ_ONLY_BLOCKED_IN_P0"

// Environment is one promotion node inside a topology version.
type Environment struct {
	ID               string   `json:"id"`
	DisplayName      string   `json:"display_name"`
	Type             string   `json:"type"`
	Sequence         int      `json:"sequence"`
	TargetRef        string   `json:"target_ref"`
	Owner            string   `json:"owner"`
	RequiredEvidence []string `json:"required_evidence"`
	ApproverPolicy   string   `json:"approver_policy"`
	Status           Status   `json:"status"`
	Action           *string  `json:"action"`
	Protection       string   `json:"protection,omitempty"`
}

// FieldChange is one attribute diff between two versions.
type FieldChange struct {
	Field    string `json:"field"`
	From     any    `json:"from"`
	To       any    `json:"to"`
	Material bool   `json:"material"`
}

// Change describes why a version exists.
type Change struct {
	Operation     string        `json:"operation"` // BOOTSTRAP, ADD, EDIT, REORDER, RETIRE, RESTORE
	EnvironmentID string        `json:"environment_id,omitempty"`
	FieldsChanged []FieldChange `json:"fields_changed,omitempty"`
	Material      bool          `json:"material"`
	Reason        string        `json:"reason"`
	Impact        string        `json:"impact"`
}

// Version is an immutable ordered snapshot of the Project's environments.
type Version struct {
	Version         int           `json:"version"`
	PreviousVersion int           `json:"previous_version"`
	CreatedAt       string        `json:"created_at"`
	Actor           string        `json:"actor"`
	Origin          string        `json:"origin"` // manifest or operator
	Change          Change        `json:"change"`
	Environments    []Environment `json:"environments"`
	ConfigHash      string        `json:"config_hash"`
}

// Invalidation records that a decision depends on a topology fact that has
// changed. It is ContextRail state; the consumer's DecisionRecord file is not
// touched, and the decision is not silently re-approved.
type Invalidation struct {
	DecisionID      string `json:"decision_id"`
	EnvironmentID   string `json:"environment_id,omitempty"`
	TopologyVersion int    `json:"topology_version"`
	Status          string `json:"status"` // STALE
	Reason          string `json:"reason"`
	At              string `json:"at"`
}

// AuditEvent is append-only governance history.
type AuditEvent struct {
	Sequence int    `json:"sequence"`
	At       string `json:"at"`
	Actor    string `json:"actor"`
	Action   string `json:"action"`
	Object   string `json:"object"`
	Version  int    `json:"version"`
	Reason   string `json:"reason"`
}

// State is the complete persisted topology state of one Project.
type State struct {
	Kind           string         `json:"kind"`
	SchemaVersion  string         `json:"schema_version"`
	ProjectID      string         `json:"project_id"`
	CurrentVersion int            `json:"current_version"`
	Versions       []Version      `json:"versions"`
	Invalidations  []Invalidation `json:"invalidations"`
	Audit          []AuditEvent   `json:"audit"`
}

// Current returns the current version snapshot.
func (state *State) Current() *Version {
	for index := range state.Versions {
		if state.Versions[index].Version == state.CurrentVersion {
			return &state.Versions[index]
		}
	}
	return nil
}

// ConfigHash is a deterministic hash of the ordered environment set. It lets
// later decisions bind to an exact topology configuration.
func ConfigHash(environments []Environment) string {
	sorted := append([]Environment(nil), environments...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Sequence < sorted[j].Sequence })
	encoded, _ := json.Marshal(sorted)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneEnvironments(environments []Environment) []Environment {
	out := make([]Environment, len(environments))
	for index, environment := range environments {
		out[index] = environment
		out[index].RequiredEvidence = append([]string(nil), environment.RequiredEvidence...)
		if environment.Action != nil {
			action := *environment.Action
			out[index].Action = &action
		}
	}
	return out
}
