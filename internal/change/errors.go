package change

import "fmt"

// Error carries a stable code the HTTP boundary maps to a status.
type Error struct {
	Code    string
	Message string
}

func (err *Error) Error() string { return fmt.Sprintf("%s: %s", err.Code, err.Message) }

const (
	CodeProjectNotFound  = "PROJECT_NOT_FOUND"
	CodeChangeNotFound   = "CHANGE_NOT_FOUND"
	CodeInvalidRequest   = "INVALID_CHANGE_REQUEST"
	CodeReasonRequired   = "REASON_REQUIRED"
	CodeActorRequired    = "ACTOR_REQUIRED"
	CodeNotDecisionReady = "NEEDS_INPUT"
	CodeInvalidOption    = "INVALID_OPTION"
	CodeAlreadyDecided   = "ALREADY_DECIDED"
	CodeChangeClosed     = "CHANGE_CLOSED"
	CodeNotAccepted      = "DECISION_NOT_ACCEPTED"
	CodeDecisionStale    = "DECISION_STALE"
	CodeAlreadyIssued    = "WORK_ORDER_ALREADY_ISSUED"
	CodeStateUnavailable = "CHANGE_STATE_UNAVAILABLE"
)

func newError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
