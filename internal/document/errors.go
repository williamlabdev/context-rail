package document

import "fmt"

// Error carries a stable code the HTTP boundary maps to a status.
type Error struct {
	Code    string
	Message string
}

func (err *Error) Error() string { return fmt.Sprintf("%s: %s", err.Code, err.Message) }

const (
	CodeProjectNotFound   = "PROJECT_NOT_FOUND"
	CodePackNotFound      = "CONTEXT_PACK_NOT_FOUND"
	CodeInvalidRequest    = "INVALID_DOCUMENT_REQUEST"
	CodeReasonRequired    = "REASON_REQUIRED"
	CodeVersionConflict   = "BASELINE_VERSION_CONFLICT"
	CodeAlreadyDeclared   = "DOCUMENT_ALREADY_DECLARED"
	CodeNotDeclared       = "DOCUMENT_NOT_DECLARED"
	CodeManifestProtected = "MANIFEST_DOCUMENT_PROTECTED"
	CodeStateUnavailable  = "DOCUMENT_STATE_UNAVAILABLE"
)

func newError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
