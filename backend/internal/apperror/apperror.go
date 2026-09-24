// Package apperror defines the small set of error codes shared between
// application services and the HTTP adapter, so a service can signal
// "not found" or "validation failed" without importing net/http.
package apperror

import "fmt"

type Code string

const (
	CodeValidationError      Code = "VALIDATION_ERROR"
	CodeUnauthorized         Code = "UNAUTHORIZED"
	CodeForbidden            Code = "FORBIDDEN"
	CodeNotFound             Code = "NOT_FOUND"
	CodeConflict             Code = "CONFLICT"
	CodePayloadTooLarge      Code = "PAYLOAD_TOO_LARGE"
	CodeUnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE"
	CodeRateLimited          Code = "RATE_LIMITED"
	CodeInternalError        Code = "INTERNAL_ERROR"
)

// Error is a typed application error carrying a stable code, a stable
// machine-readable Key, and an English Message.
//
// Key exists so a client can show a message in the user's own language
// without parsing English text: it identifies exactly which validation or
// lookup failed (e.g. "deck.name.tooLong"), the same way Code identifies
// the general class of failure. Message stays in English; it is a safe
// fallback for a client that does not recognize Key (or has none for it
// yet) and is what server-side logs use.
type Error struct {
	Code    Code
	Key     string
	Message string
	// Err is the underlying cause, kept for server-side logs only. It is
	// never included in the client-facing message.
	Err error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func newErr(code Code, key, message string) *Error {
	return &Error{Code: code, Key: key, Message: message}
}

// Validation, Unauthorized, Forbidden, NotFound, Conflict, RateLimited,
// PayloadTooLarge and UnsupportedMediaType each take a stable key (see
// Error.Key) plus the English fallback message.
func Validation(key, message string) *Error      { return newErr(CodeValidationError, key, message) }
func Unauthorized(key, message string) *Error    { return newErr(CodeUnauthorized, key, message) }
func Forbidden(key, message string) *Error       { return newErr(CodeForbidden, key, message) }
func NotFound(key, message string) *Error        { return newErr(CodeNotFound, key, message) }
func Conflict(key, message string) *Error        { return newErr(CodeConflict, key, message) }
func RateLimited(key, message string) *Error     { return newErr(CodeRateLimited, key, message) }
func PayloadTooLarge(key, message string) *Error { return newErr(CodePayloadTooLarge, key, message) }
func UnsupportedMediaType(key, message string) *Error {
	return newErr(CodeUnsupportedMediaType, key, message)
}

// Internal wraps an unexpected error with a safe client-facing message
// while preserving the original cause for logging.
func Internal(cause error) *Error {
	return &Error{Code: CodeInternalError, Key: "internal.unexpected", Message: "an unexpected error occurred", Err: cause}
}
