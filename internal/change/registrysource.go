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
	roots     []string
	topology  *topology.Service
	documents DocumentOverlay
}

// DocumentOverlay is the governed document baseline (UI-20): the live
// declared document set, the live context status (rebuilt Context Pack) and
// the document readiness for decisions. found=false leaves the registry
// facts untouched.
type DocumentOverlay interface {
	DocumentFacts(projectID string) (documents []DocumentFact, contextStatus string, decision ReadinessFact, found bool, err error)
}

func NewRegistrySource(roots []string, topologyService *topology.Service) *RegistrySource {
	return &RegistrySource{roots: append([]string(nil), roots...), topology: topologyService}
}

// WithDocuments overlays the governed document baseline on the registry facts.
func (source *RegistrySource) WithDocuments(overlay DocumentOverlay) *RegistrySource {
	source.documents = overlay
	return source
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
		if source.documents != nil {
			documents, contextStatus, decision, found, err := source.documents.DocumentFacts(projectID)
			if err != nil {
				return nil, newError(CodeStateUnavailable, "document baseline unavailable: %v", err)
			}
			if found {
				facts.Documents = documents
				if contextStatus != "" {
					facts.ContextStatus = contextStatus
				}
				if decision.Status != "" && decision.Status != "READY" {
					reasons := append([]string{}, facts.DecisionInputs.Reasons...)
					reasons = append(reasons, decision.Reasons...)
					facts.DecisionInputs = ReadinessFact{Status: decision.Status, Reasons: reasons}
				}
			}
		}
		return facts, nil
	}
	return nil, nil
}
