package release

import "fmt"

// Error carries a stable code the HTTP boundary maps to a status.
type Error struct {
	Code    string
	Message string
}

func (err *Error) Error() string { return fmt.Sprintf("%s: %s", err.Code, err.Message) }

const (
	CodeProjectNotFound  = "PROJECT_NOT_FOUND"
	CodeReleaseNotFound  = "RELEASE_NOT_FOUND"
	CodeInvalidRequest   = "INVALID_RELEASE_REQUEST"
	CodeReasonRequired   = "REASON_REQUIRED"
	CodeActorRequired    = "ACTOR_REQUIRED"
	CodeGateBlocked      = "PROMOTION_BLOCKED"
	CodeBuildMismatch    = "BUILD_SOURCE_MISMATCH"
	CodeNotApprovable    = "NOT_READY_FOR_APPROVAL"
	CodeApprovalInvalid  = "APPROVAL_INVALID"
	CodeApproverConflict = "APPROVER_SEPARATION"
	CodeAlreadyApproved  = "ALREADY_APPROVED"
	CodeAlreadyPromoted  = "ALREADY_PROMOTED"
	CodeReleaseClosed    = "RELEASE_CLOSED"
	CodeStale            = "RELEASE_STALE"
	CodeStateUnavailable = "RELEASE_STATE_UNAVAILABLE"
)

func newError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
