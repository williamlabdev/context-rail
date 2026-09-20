package document

import (
	"context-rail/internal/projectregistry"
)

// RegistrySource adapts the read-only Project Registry importer as the
// document baseline source. Each call re-imports the explicit roots so the
// baseline always reflects the manifest actually on disk.
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
		declared, err := projectregistry.DeclaredDocuments(entry.Project.Root)
		if err != nil {
			return nil, newError(CodeStateUnavailable, "cannot read document declarations: %v", err)
		}
		baseline := &Baseline{ProjectID: projectID, Root: entry.Project.Root, ObservedStatuses: map[string]ObservedStatus{}, DeclaredContext: DeclaredContext{Path: "docs/ai/context-pack.json", Status: entry.Context.Status}}
		for _, document := range declared {
			baseline.Documents = append(baseline.Documents, Declared{Path: document.Path, Kind: document.Kind, SourceOfTruth: document.SourceOfTruth, RequiredFor: document.RequiredFor})
		}
		for _, document := range entry.Documents {
			baseline.ObservedStatuses[document.Path] = ObservedStatus{Status: document.Status, StaleSources: document.StaleSources}
		}
		seen := map[string]bool{}
		for _, decision := range entry.Decisions {
			if decision.DecisionID != "" && !seen[decision.DecisionID] {
				seen[decision.DecisionID] = true
				baseline.DecisionIDs = append(baseline.DecisionIDs, decision.DecisionID)
			}
		}
		return baseline, nil
	}
	return nil, nil
}
