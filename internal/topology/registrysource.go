package topology

import (
	"context-rail/internal/projectregistry"
)

// RegistrySource adapts the read-only Project Registry importer as a
// topology baseline source. Each call re-imports the explicit roots so the
// bootstrap always reflects the manifest actually on disk.
type RegistrySource struct {
	roots []string
}

func NewRegistrySource(roots []string) *RegistrySource {
	return &RegistrySource{roots: append([]string(nil), roots...)}
}

func (source *RegistrySource) Baseline(projectID string) (*Baseline, error) {
	snapshot, err := projectregistry.Import(source.roots)
	if err != nil {
		return nil, newError(CodeStateUnavailable, "project registry import failed: %v", err)
	}
	for _, entry := range snapshot.Projects {
		if entry.Project.ID != projectID {
			continue
		}
		baseline := &Baseline{ProjectID: projectID}
		for _, environment := range entry.Environments {
			baseline.Environments = append(baseline.Environments, Environment{
				ID:               environment.ID,
				DisplayName:      environment.ID,
				Type:             environment.Type,
				Sequence:         environment.Sequence,
				TargetRef:        environment.TargetRef,
				RequiredEvidence: append([]string{}, environment.RequiredEvidence...),
				Action:           environment.Action,
			})
		}
		seen := map[string]bool{}
		for _, decision := range entry.Decisions {
			// A decision may be observed through several audience views that
			// share one decision_id; invalidate it once.
			if decision.DecisionID != "" && !seen[decision.DecisionID] {
				seen[decision.DecisionID] = true
				baseline.DecisionIDs = append(baseline.DecisionIDs, decision.DecisionID)
			}
		}
		return baseline, nil
	}
	return nil, nil
}
