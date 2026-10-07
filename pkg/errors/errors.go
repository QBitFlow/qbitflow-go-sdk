// Package qberrors defines the error types returned by the QBitFlow Go SDK.
//
// Every failure the SDK reports is one of the types below, so callers can branch on the
// kind of failure with errors.As rather than by inspecting message text:
//
//	var rateLimited *qberrors.RateLimitError
//	if errors.As(err, &rateLimited) { ... }
//
// The mapping from HTTP status to type is fixed and identical across the QBitFlow SDKs:
//
//	400, 422           ValidationError      (also raised client-side before a request)
//	401                AuthenticationError
//	403                ForbiddenError
//	404                NotFoundError
//	409                ConflictError
//	429                RateLimitError       (carries Retry-After when the API sends it)
//	other 4xx          QBitFlowError        (the base type, with StatusCode set)
//	5xx, 3xx, bad 2xx  ServerError          (bad 2xx: empty or not the expected JSON)
//	transport failure  NetworkError         (after the retry policy is exhausted)
//
// Every type unwraps to a *QBitFlowError, so `errors.As(err, &qbErr)` with a
// *QBitFlowError target matches any SDK error and exposes its Message, StatusCode and
// Fields. AuthenticationError, ForbiddenError, ConflictError, RateLimitError, ServerError
// and NetworkError embed it; ValidationError and NotFoundError keep their own fields and
// unwrap to an equivalent *QBitFlowError view.
package qberrors

import (
	"errors"
	"fmt"
	"strings"
)

// FieldError is a single field-level validation failure.
//
// The API reports validation failures as a list of these, one per offending field:
//
//	{"errors":[{"field":"ProductName","message":"ProductName is too short"},
//	           {"field":"Price","message":"Price is too short"}]}
type FieldError struct {
	// Field is the name of the offending field, as the API names it.
	Field string `json:"field"`
	// Message is the API's explanation for that field.
	Message string `json:"message"`
}

// ValidationErrors is the wire envelope the API uses for a field-level validation response.
type ValidationErrors struct {
	Errors []FieldError `json:"errors"`
}

// JoinFieldErrors renders a field-error list as a single human-readable summary,
// "Field: message; Field2: message2". It returns "" for an empty list.
func JoinFieldErrors(fields []FieldError) string {
	parts := make([]string, 0, len(fields))
	for _, fieldErr := range fields {
		switch {
		case fieldErr.Field != "" && fieldErr.Message != "":
			parts = append(parts, fmt.Sprintf("%s: %s", fieldErr.Field, fieldErr.Message))
		case fieldErr.Message != "":
			parts = append(parts, fieldErr.Message)
		case fieldErr.Field != "":
			parts = append(parts, fieldErr.Field)
		}
	}
	return strings.Join(parts, "; ")
}

// QBitFlowError is the base error for every failure the SDK reports.
//
// It is returned directly for HTTP statuses that have no more specific type (for example
// 405 or 410). Every typed error below unwraps to it, so `errors.As` with a *QBitFlowError
// target matches any SDK error.
type QBitFlowError struct {
	// Message is the API's explanation, or the SDK's when no response was received.
	Message string
	// StatusCode is the HTTP status of the response; 0 when no response was received.
	StatusCode int
	// Fields carries the per-field failures when the API supplied an errors[] list.
	// It is populated for any status, not only validation failures.
	Fields []FieldError
	// Err is the underlying cause (a transport or decoding error), if any.
	Err error
}

// Error implements the error interface
func (e *QBitFlowError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("QBitFlow Error (Status %d): %s - %v", e.StatusCode, e.Message, e.Err)
	}
	return fmt.Sprintf("QBitFlow Error (Status %d): %s", e.StatusCode, e.Message)
}

// Unwrap returns the underlying error
func (e *QBitFlowError) Unwrap() error {
	return e.Err
}

// NewQBitFlowError creates a new QBitFlowError
func NewQBitFlowError(message string, statusCode int, err error) *QBitFlowError {
	return &QBitFlowError{
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// NewQBitFlowErrorWithFields creates a QBitFlowError carrying the API's field-error list.
func NewQBitFlowErrorWithFields(message string, statusCode int, fields []FieldError) *QBitFlowError {
	return &QBitFlowError{
		Message:    message,
		StatusCode: statusCode,
		Fields:     fields,
	}
}

// NotFoundError represents a 404 not found error.
//
// The API answers 404 both for a resource that does not exist and for one the caller does
// not own (existence is never revealed across organizations or users).
type NotFoundError struct {
	// Message is the API's explanation.
	Message string
	// StatusCode is always 404.
	StatusCode int
	// Fields carries the API's field-error list when it sent one (rare on a 404).
	Fields []FieldError
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("Not Found: %s", e.Message)
}

// Unwrap returns an equivalent *QBitFlowError, so an `errors.As` with a *QBitFlowError
// target matches a NotFoundError too.
func (e *NotFoundError) Unwrap() error {
	return &QBitFlowError{Message: e.Message, StatusCode: e.StatusCode, Fields: e.Fields}
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{Message: message, StatusCode: 404}
}

// NewNotFoundErrorWithFields creates a NotFoundError carrying the API's field-error list.
func NewNotFoundErrorWithFields(message string, fields []FieldError) *NotFoundError {
	return &NotFoundError{Message: message, StatusCode: 404, Fields: fields}
}

// ValidationError represents a validation error, raised either by the API (HTTP 400/422)
// or by the SDK before a request is sent.
//
// The SDK mirrors the API's binding rules client-side (identifier guards, product text,
// alphanumeric names, e-mail format, roles, fee bounds, redirect URLs, accounting dates),
// so the same field rejected locally and rejected by the API surfaces as this one type.
type ValidationError struct {
	// Message is a human-readable summary. For an API validation response it lists
	// every field that failed, so no failure is hidden behind the first one.
	Message string
	// StatusCode is 400 or 422 for an API-side failure and 0 for one raised client-side.
	StatusCode int
	// Fields carries the per-field failures when the API supplied them. It is empty for
	// validation errors raised client-side.
	Fields []FieldError
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", e.Message)
}

// Unwrap returns an equivalent *QBitFlowError, so an `errors.As` with a *QBitFlowError
// target matches a ValidationError too (StatusCode 0 for a client-side failure).
func (e *ValidationError) Unwrap() error {
	return &QBitFlowError{Message: e.Message, StatusCode: e.StatusCode, Fields: e.Fields}
}

// NewValidationError creates a client-side ValidationError (StatusCode 0).
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

// NewAPIValidationError creates a ValidationError for an API 400/422 response.
func NewAPIValidationError(message string, statusCode int, fields []FieldError) *ValidationError {
	return &ValidationError{Message: message, StatusCode: statusCode, Fields: fields}
}

// NewValidationErrorFromList creates a ValidationError from an API validation response.
//
// Every field failure is retained, both in the summary message and in Fields — reporting
// only the first one hides the rest of what the caller needs to fix. The status is
// recorded as 400; use NewAPIValidationError to record 422.
func NewValidationErrorFromList(list ValidationErrors) *ValidationError {
	message := JoinFieldErrors(list.Errors)
	if message == "" {
		message = "request failed validation"
	}

	return &ValidationError{Message: message, StatusCode: 400, Fields: list.Errors}
}

// AuthenticationError is returned for HTTP 401: the API key is missing, invalid, expired
// or deleted. Fix the credential; retrying does not help.
type AuthenticationError struct {
	*QBitFlowError
}

// Error implements the error interface
func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("Authentication Error (Status %d): %s", e.StatusCode, e.Message)
}

// Unwrap returns the embedded *QBitFlowError.
func (e *AuthenticationError) Unwrap() error { return e.QBitFlowError }

// NewAuthenticationError creates an AuthenticationError.
func NewAuthenticationError(message string, statusCode int, fields []FieldError) *AuthenticationError {
	return &AuthenticationError{QBitFlowError: NewQBitFlowErrorWithFields(message, statusCode, fields)}
}

// ForbiddenError is returned for HTTP 403: the credential is valid but its role is below
// what the route requires, or the target is visible in the tenant but not the caller's to
// act on (a peer user's record, the organization owner, another user's API key).
type ForbiddenError struct {
	*QBitFlowError
}

// Error implements the error interface
func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("Forbidden (Status %d): %s", e.StatusCode, e.Message)
}

// Unwrap returns the embedded *QBitFlowError.
func (e *ForbiddenError) Unwrap() error { return e.QBitFlowError }

// NewForbiddenError creates a ForbiddenError.
func NewForbiddenError(message string, statusCode int, fields []FieldError) *ForbiddenError {
	return &ForbiddenError{QBitFlowError: NewQBitFlowErrorWithFields(message, statusCode, fields)}
}

// ConflictError is returned for HTTP 409: the request is well-formed but conflicts with
// the resource's current state — for example executing a test billing cycle on a
// subscription that is not yet due.
type ConflictError struct {
	*QBitFlowError
}

// Error implements the error interface
func (e *ConflictError) Error() string {
	return fmt.Sprintf("Conflict (Status %d): %s", e.StatusCode, e.Message)
}

// Unwrap returns the embedded *QBitFlowError.
func (e *ConflictError) Unwrap() error { return e.QBitFlowError }

// NewConflictError creates a ConflictError.
func NewConflictError(message string, statusCode int, fields []FieldError) *ConflictError {
	return &ConflictError{QBitFlowError: NewQBitFlowErrorWithFields(message, statusCode, fields)}
}

// RateLimitError is returned for HTTP 429. The SDK never retries a 429 itself; honour
// RetryAfter when it is set.
type RateLimitError struct {
	*QBitFlowError
	// RetryAfter is the number of seconds the API asked the caller to wait, taken from the
	// Retry-After header. Nil when the header was absent or not an integer.
	RetryAfter *int
}

// Error implements the error interface
func (e *RateLimitError) Error() string {
	if e.RetryAfter != nil {
		return fmt.Sprintf("Rate Limited (Status %d): %s (retry after %ds)", e.StatusCode, e.Message, *e.RetryAfter)
	}
	return fmt.Sprintf("Rate Limited (Status %d): %s", e.StatusCode, e.Message)
}

// Unwrap returns the embedded *QBitFlowError.
func (e *RateLimitError) Unwrap() error { return e.QBitFlowError }

// NewRateLimitError creates a RateLimitError. retryAfter may be nil.
func NewRateLimitError(message string, statusCode int, fields []FieldError, retryAfter *int) *RateLimitError {
	return &RateLimitError{
		QBitFlowError: NewQBitFlowErrorWithFields(message, statusCode, fields),
		RetryAfter:    retryAfter,
	}
}

// ServerError is returned when the API or the path to it misbehaved: a 5xx response
// (after the retry policy is exhausted), a 3xx (redirects are never followed — usually a
// misconfigured base URL), or a 2xx whose body is empty or not the JSON the SDK expects
// (a proxy's HTML page served with a 200, or a field of the wrong JSON type). StatusCode
// is the response's status.
type ServerError struct {
	*QBitFlowError
}

// Error implements the error interface
func (e *ServerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Server Error (Status %d): %s - %v", e.StatusCode, e.Message, e.Err)
	}
	return fmt.Sprintf("Server Error (Status %d): %s", e.StatusCode, e.Message)
}

// Unwrap returns the embedded *QBitFlowError.
func (e *ServerError) Unwrap() error { return e.QBitFlowError }

// NewServerError creates a ServerError. err is the decoding failure for a bad 2xx body,
// nil otherwise.
func NewServerError(message string, statusCode int, err error) *ServerError {
	return &ServerError{QBitFlowError: NewQBitFlowError(message, statusCode, err)}
}

// NewServerErrorWithFields creates a ServerError carrying the API's field-error list.
func NewServerErrorWithFields(message string, statusCode int, fields []FieldError) *ServerError {
	return &ServerError{QBitFlowError: NewQBitFlowErrorWithFields(message, statusCode, fields)}
}

// NetworkError is returned when no HTTP response was received: connection refused, DNS
// failure, TLS failure, or the client timeout elapsed — after the retry policy is
// exhausted for a retriable request. StatusCode is always 0 and Err holds the transport
// error.
type NetworkError struct {
	*QBitFlowError
}

// Error implements the error interface
func (e *NetworkError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Network Error: %s - %v", e.Message, e.Err)
	}
	return fmt.Sprintf("Network Error: %s", e.Message)
}

// Unwrap returns the embedded *QBitFlowError.
func (e *NetworkError) Unwrap() error { return e.QBitFlowError }

// NewNetworkError creates a NetworkError wrapping the transport failure.
func NewNetworkError(message string, err error) *NetworkError {
	return &NetworkError{QBitFlowError: NewQBitFlowError(message, 0, err)}
}

// StatusCodeOf returns the HTTP status code carried by an SDK error, or 0 when the error
// is not an SDK error or no response was received. The error may be wrapped
// (fmt.Errorf("...: %w", err)).
func StatusCodeOf(err error) int {
	var qbErr *QBitFlowError
	if errors.As(err, &qbErr) {
		return qbErr.StatusCode
	}
	return 0
}

// FieldsOf returns the API's field-error list carried by an SDK error, or nil. The error may
// be wrapped.
func FieldsOf(err error) []FieldError {
	var qbErr *QBitFlowError
	if errors.As(err, &qbErr) {
		return qbErr.Fields
	}
	return nil
}
