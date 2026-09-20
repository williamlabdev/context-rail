package topology

import "fmt"

// Error is a typed governance/validation error with a stable code that the
// HTTP boundary maps to a status without re-interpreting the message.
type Error struct {
	Code    string
	Message string
}

func (err *Error) Error() string { return fmt.Sprintf("%s: %s", err.Code, err.Message) }

const (
	CodeProjectNotFound     = "PROJECT_NOT_FOUND"
	CodeVersionConflict     = "TOPOLOGY_VERSION_CONFLICT"
	CodeEnvironmentNotFound = "ENVIRONMENT_NOT_FOUND"
	CodeEnvironmentExists   = "ENVIRONMENT_EXISTS"
	CodeInvalidEnvironment  = "INVALID_ENVIRONMENT"
	CodeInvalidOrder        = "INVALID_ORDER"
	CodeReasonRequired      = "REASON_REQUIRED"
	CodeProductionProtected = "PRODUCTION_PROTECTED"
	CodeEnvironmentRetired  = "ENVIRONMENT_RETIRED"
	CodeEnvironmentActive   = "ENVIRONMENT_NOT_RETIRED"
	CodeStateUnavailable    = "TOPOLOGY_STATE_UNAVAILABLE"
)

func newError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
