package document

import "context-rail/internal/change"

// DocumentFacts implements change.DocumentOverlay: the live declared
// document set (manifest + operator declarations, with their status now),
// the live context status (latest rebuilt pack, STALE when it drifted) and
// the document readiness for decisions. A missing required document makes a
// new Change NEEDS_INPUT; it is never filled in by the advisor.
func (service *Service) DocumentFacts(projectID string) ([]change.DocumentFact, string, change.ReadinessFact, bool, error) {
	facts, err := service.Facts(projectID)
	if err != nil {
		var typed *Error
		if asError(err, &typed) && typed.Code == CodeProjectNotFound {
			return nil, "", change.ReadinessFact{}, false, nil
		}
		return nil, "", change.ReadinessFact{}, false, err
	}
	if facts == nil {
		return nil, "", change.ReadinessFact{}, false, nil
	}
	documents := make([]change.DocumentFact, 0, len(facts.Documents))
	for _, source := range facts.Documents {
		documents = append(documents, change.DocumentFact{Path: source.Path, Status: source.Status, SourceOfTruth: source.SourceOfTruth})
	}
	return documents, facts.ContextStatus, change.ReadinessFact{Status: facts.Decision.Status, Reasons: facts.Decision.Reasons}, true, nil
}

func asError(err error, target **Error) bool {
	typed, ok := err.(*Error)
	if ok {
		*target = typed
	}
	return ok
}
