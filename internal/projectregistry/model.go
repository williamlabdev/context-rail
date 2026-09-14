// Package projectregistry provides the read-only Project Registry contract.
package projectregistry

// Snapshot is the versioned registry result for explicitly supplied Projects.
type Snapshot struct {
	Kind          string         `json:"kind"`
	SchemaVersion string         `json:"schema_version"`
	Projects      []ProjectEntry `json:"projects"`
}

// ProjectEntry contains one normalized Project and its observed governance data.
type ProjectEntry struct {
	Project      ProjectRecord     `json:"project"`
	Repositories []Repository      `json:"repositories"`
	Services     []Service         `json:"services"`
	Environments []Environment     `json:"environments"`
	Documents    []DocumentStatus  `json:"documents"`
	Decisions    []DecisionSummary `json:"decisions"`
	Readiness    ReadinessSet      `json:"readiness"`
	Context      ContextStatus     `json:"context"`
	ReadOnly     bool              `json:"read_only"`
	ObservedAt   string            `json:"observed_at"`
}

// ProjectRecord is the declared Project identity plus its explicit root.
type ProjectRecord struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Version        int               `json:"version"`
	Classification string            `json:"classification"`
	Status         string            `json:"status"`
	Purpose        string            `json:"purpose"`
	Owners         map[string]string `json:"owners"`
	Root           string            `json:"root"`
}

type Repository struct {
	Provider      string `json:"provider"`
	Visibility    string `json:"visibility"`
	URL           string `json:"url"`
	DefaultBranch string `json:"default_branch"`
	Status        string `json:"status"`
}

type Service struct {
	ID         string  `json:"id"`
	Repository *string `json:"repository"`
	Path       string  `json:"path"`
	Runtime    string  `json:"runtime"`
	Role       string  `json:"role"`
}

type Environment struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Sequence         int      `json:"sequence"`
	TargetRef        string   `json:"target_ref"`
	RequiredEvidence []string `json:"required_evidence"`
	Action           *string  `json:"action"`
}

type DocumentStatus struct {
	Path          string   `json:"path"`
	Kind          string   `json:"kind"`
	SourceOfTruth bool     `json:"source_of_truth"`
	Status        string   `json:"status"`
	StaleSources  []string `json:"stale_sources,omitempty"`
}

type DecisionSummary struct {
	DecisionID string `json:"decision_id"`
	RequestID  string `json:"request_id"`
	Status     string `json:"status"`
}

type ReadinessStatus struct {
	Status  string   `json:"status"`
	Reasons []string `json:"reasons"`
}

type ReadinessSet struct {
	ReadyForDecision         ReadinessStatus `json:"ready_for_decision"`
	ReadyForLocalDevelopment ReadinessStatus `json:"ready_for_local_development"`
	ReadyForCloudTesting     ReadinessStatus `json:"ready_for_cloud_testing"`
	ReadyForStaging          ReadinessStatus `json:"ready_for_staging"`
	StagingVerified          ReadinessStatus `json:"staging_verified"`
	Production               ReadinessStatus `json:"production"`
}

type ContextStatus struct {
	Status string `json:"status"`
}
