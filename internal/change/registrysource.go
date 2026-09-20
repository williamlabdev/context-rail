package change

import (
	"context-rail/internal/projectregistry"
	"context-rail/internal/topology"
)

// RegistrySource composes the read-only registry import with the topology
// service to produce ProjectFacts. Reading the topology may bootstrap its
// version 1 from the manifest; that is the only side effect, and it never
// touches the consumer Project.
type RegistrySource struct {
	roots    []string
	topology *topology.Service
}

func NewRegistrySource(roots []string, topologyService *topology.Service) *RegistrySource {
	return &RegistrySource{roots: append([]string(nil), roots...), topology: topologyService}
}

func (source *RegistrySource) Facts(projectID string) (*ProjectFacts, error) {
	snapshot, err := projectregistry.Import(source.roots)
	if err != nil {
		return nil, newError(CodeStateUnavailable, "project registry import failed: %v", err)
	}
	for _, entry := range snapshot.Projects {
		if entry.Project.ID != projectID {
			continue
		}
		facts := &ProjectFacts{
			ProjectID: projectID, ProjectName: entry.Project.Name, ContextStatus: entry.Context.Status,
			DecisionInputs: ReadinessFact{Status: entry.Readiness.ReadyForDecision.Status, Reasons: entry.Readiness.ReadyForDecision.Reasons},
		}
		if len(entry.Repositories) > 0 {
			facts.RepositoryURL = entry.Repositories[0].URL
			facts.DefaultBranch = entry.Repositories[0].DefaultBranch
		}
		for _, document := range entry.Documents {
			facts.Documents = append(facts.Documents, DocumentFact{Path: document.Path, Status: document.Status, SourceOfTruth: document.SourceOfTruth})
		}
		if source.topology != nil {
			state, err := source.topology.Get(projectID)
			if err != nil {
				return nil, err
			}
			facts.Topology = state
		}
		return facts, nil
	}
	return nil, nil
}
