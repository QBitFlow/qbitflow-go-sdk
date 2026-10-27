package qbitflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Error codes the SDK branches on (the API's stable `code` values, README §8).
const (
	codeValidationFailed     = "validation_failed"
	codeNotFound             = "not_found"
	codeIdempotencyKeyInUse  = "idempotency_key_in_use"
	codeIdempotencyKeyReused = "idempotency_key_reused"
	codeInvalidSignature     = "invalid_signature"
)

// FieldError is one failing input of a validation error.
type FieldError struct {
	// Field is the input's wire name, dotted when nested (e.g. "frequency.unit").
	Field string `json:"field"`
	// Message explains what is wrong with it.
	Message string `json:"message"`
}

// QBitFlowError is implemented by every error the SDK returns. Match a concrete type with
// [errors.As] (e.g. *NotFoundError), or *APIError for the fields every error carries.
type QBitFlowError interface {
	error
	// apiError returns the error's common payload; unexported so the set of SDK errors is closed.
	apiError() *APIError
}

// APIError is the payload every SDK error carries, and the error returned as is for an HTTP
// error without a more specific type (e.g. 413 request_too_large, 405 method_not_allowed).
//
// Every typed error (*NotFoundError, *RateLimitError, …) embeds it and unwraps to it, so
// errors.As(err, &apiErr) with a *APIError target matches any SDK error. StatusCode is 0 when
// no response was received (client-side validation, network errors, local webhook checks).
type APIError struct {
	// StatusCode is the HTTP status; 0 without a response.
	StatusCode int
	// Code is the API's machine-readable code (e.g. "not_found", "unique_violation"); "" when absent.
	Code string
	// Message is the API's human-readable message (never branch on it), else a status default.
	Message string
	// Details is the error body's structured data (e.g. details.field); never nil.
	Details map[string]any
	// RequestID is the request's id (the body's requestId, else the X-Request-Id response header):
	// quote it to support.
	RequestID string
	// FieldErrors lists the failing inputs (details.errors, or the SDK's own checks).
	FieldErrors []FieldError
	// RawBody is the error response's body as received (it may carry a dev server's debug text).
	RawBody []byte
	// Err is the underlying cause, if any (a transport, timeout or decoding error).
	Err error
}

// Error renders "<message> (status <status>, code <code>, request <requestId>)" followed by
// "; <field>: <message>" for each field error. Parts without a value are left out.
func (e *APIError) Error() string {
	var b strings.Builder
	b.WriteString(e.Message)
	if b.Len() == 0 {
		b.WriteString("qbitflow error")
	}

	var meta []string
	if e.StatusCode != 0 {
		meta = append(meta, "status "+strconv.Itoa(e.StatusCode))
	}
	if e.Code != "" {
		meta = append(meta, "code "+e.Code)
	}
	if e.RequestID != "" {
		meta = append(meta, "request "+e.RequestID)
	}
	if len(meta) > 0 {
		b.WriteString(" (" + strings.Join(meta, ", ") + ")")
	}

	for _, fe := range e.FieldErrors {
		b.WriteString("; " + fe.Field + ": " + fe.Message)
	}

	if e.Err != nil {
		b.WriteString(": " + e.Err.Error())
	}
	return b.String()
}

// Unwrap returns the underlying cause.
func (e *APIError) Unwrap() error { return e.Err }

func (e *APIError) apiError() *APIError { return e }

// ValidationError is a 400 validation_failed, or an input the SDK refused before sending
// anything (StatusCode 0). FieldErrors names each failing input by its wire name.
type ValidationError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *ValidationError) Unwrap() error { return e.APIError }

// BadRequestError is any other 400 (foreign_key_violation, test_mode_only, sender_is_receiver, …).
type BadRequestError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *BadRequestError) Unwrap() error { return e.APIError }

// AuthenticationError is a 401: the API key is missing, unknown, expired or revoked.
type AuthenticationError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *AuthenticationError) Unwrap() error { return e.APIError }

// PermissionDeniedError is a 403 (forbidden, policy_disabled with details.policy, plan_required).
type PermissionDeniedError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *PermissionDeniedError) Unwrap() error { return e.APIError }

// NotFoundError is a 404: the resource does not exist, or is outside the request's space.
type NotFoundError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *NotFoundError) Unwrap() error { return e.APIError }

// ConflictError is a 409 (unique_violation with details.field, tx_already_sent,
// merchant_not_ready with details.reason, held_funds_pending, already_joined, payment_not_due,
// subscription_already_stopped_or_inactive, idempotency_key_in_use, conflict, …).
type ConflictError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *ConflictError) Unwrap() error { return e.APIError }

// GoneError is a 410 (merchant_closed): the merchant's space is closed.
type GoneError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *GoneError) Unwrap() error { return e.APIError }

// IdempotencyError is a 422 idempotency_key_reused: the Idempotency-Key was already used for
// a different request. A caller bug; never retried.
type IdempotencyError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *IdempotencyError) Unwrap() error { return e.APIError }

// RateLimitError is a 429. It is returned once the retries are exhausted, or at once when the
// API asks to wait more than 60 seconds.
type RateLimitError struct {
	*APIError
	// RetryAfter is how long the API asked to wait (the Retry-After header, else
	// details.retryAfterSeconds); 0 when unknown.
	RetryAfter time.Duration
	// Limit is the number of requests allowed per period (details.limit); 0 when unknown.
	Limit int
	// PeriodSeconds is the limit's period, in seconds (details.periodSeconds); 0 when unknown.
	PeriodSeconds int
}

// Unwrap returns the embedded *APIError.
func (e *RateLimitError) Unwrap() error { return e.APIError }

// ServerError is a 5xx (503 network_unavailable and 504 timeout included), an unexpected 3xx
// (redirects are never followed), or a 2xx whose body is not the expected JSON.
type ServerError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *ServerError) Unwrap() error { return e.APIError }

// NetworkError means no response was received (DNS, connection, TLS, timeout, cancelled
// context). Err holds the cause: errors.Is(err, context.DeadlineExceeded) works through it.
type NetworkError struct{ *APIError }

// Unwrap returns the embedded *APIError.
func (e *NetworkError) Unwrap() error { return e.APIError }

// WebhookSignatureReason says why a webhook signature was refused.
type WebhookSignatureReason string

// Webhook signature failure reasons.
const (
	// WebhookSignatureMissingHeader: the QBitFlow-Signature header is empty.
	WebhookSignatureMissingHeader WebhookSignatureReason = "missingHeader"
	// WebhookSignatureMalformedHeader: no t, a t that is not an integer, or no v1.
	WebhookSignatureMalformedHeader WebhookSignatureReason = "malformedHeader"
	// WebhookSignatureTimestampOutsideTolerance: t is too far from now (either direction).
	WebhookSignatureTimestampOutsideTolerance WebhookSignatureReason = "timestampOutsideTolerance"
	// WebhookSignatureNoMatchingSignature: no v1 matches the expected signature.
	WebhookSignatureNoMatchingSignature WebhookSignatureReason = "noMatchingSignature"
	// WebhookSignatureInvalidSignature: the API's check (POST /webhooks/verify) refused it.
	WebhookSignatureInvalidSignature WebhookSignatureReason = "invalidSignature"
)

// WebhookSignatureError means a webhook's signature could not be verified: locally
// (VerifyWebhook, StatusCode 0) or by the API (400 invalid_signature).
type WebhookSignatureError struct {
	*APIError
	// Reason says why.
	Reason WebhookSignatureReason
}

// Unwrap returns the embedded *APIError.
func (e *WebhookSignatureError) Unwrap() error { return e.APIError }

// IsRetryable reports whether err is a failure the SDK's retry policy treats as transient: a
// network error or timeout, a 5xx, a 429, or a 409 idempotency_key_in_use. It does not say
// whether the method may be retried (only reads and the idempotent creates are).
func IsRetryable(err error) bool {
	var (
		netErr  *NetworkError
		srvErr  *ServerError
		rateErr *RateLimitError
		confErr *ConflictError
	)
	switch {
	case errors.As(err, &netErr), errors.As(err, &rateErr):
		return true
	case errors.As(err, &srvErr):
		return srvErr.StatusCode >= 500
	case errors.As(err, &confErr):
		return confErr.Code == codeIdempotencyKeyInUse
	}
	return false
}

// asWebhookSignatureError turns the API's 400 invalid_signature (POST /webhooks/verify) into a
// *WebhookSignatureError with reason invalidSignature; any other error is returned unchanged.
func asWebhookSignatureError(err error) error {
	var br *BadRequestError
	if errors.As(err, &br) && br.Code == codeInvalidSignature {
		return &WebhookSignatureError{APIError: br.APIError, Reason: WebhookSignatureInvalidSignature}
	}
	return err
}

// newValidationError builds a client-side ValidationError (no request was sent).
func newValidationError(message string, fields ...FieldError) *ValidationError {
	return &ValidationError{&APIError{Message: message, Details: map[string]any{}, FieldErrors: fields}}
}

// newFieldError builds a client-side ValidationError for one input; message follows the
// field's name ("timeout" + "must be positive").
func newFieldError(field, message string) *ValidationError {
	return newValidationError("validation failed", FieldError{Field: field, Message: field + " " + message})
}

// newNetworkError wraps a transport failure.
func newNetworkError(message string, cause error) *NetworkError {
	return &NetworkError{&APIError{Message: message, Details: map[string]any{}, Err: cause}}
}

// newServerError builds a ServerError for a response the SDK could not use.
func newServerError(message string, resp *response, cause error) *ServerError {
	e := &APIError{Message: message, Details: map[string]any{}, Err: cause}
	if resp != nil {
		e.StatusCode = resp.status
		e.RequestID = resp.header.Get("X-Request-Id")
	}
	return &ServerError{e}
}

// errorBody is the API's error envelope: {error, code, details, requestId, debug}.
type errorBody struct {
	Error     string         `json:"error"`
	Code      string         `json:"code"`
	Details   map[string]any `json:"details"`
	RequestID string         `json:"requestId"`
}

// parseAPIError builds the *APIError of a non-2xx response. Each key is read on its own so a
// malformed one never hides the others; field errors come from details.errors only.
func parseAPIError(resp *response) *APIError {
	e := &APIError{StatusCode: resp.status, Details: map[string]any{}, RawBody: resp.body}

	var raw map[string]json.RawMessage
	if json.Unmarshal(resp.body, &raw) == nil {
		var body errorBody
		if v, ok := raw["error"]; ok {
			_ = json.Unmarshal(v, &body.Error)
		}
		if v, ok := raw["code"]; ok {
			_ = json.Unmarshal(v, &body.Code)
		}
		if v, ok := raw["details"]; ok {
			_ = json.Unmarshal(v, &body.Details)
		}
		if v, ok := raw["requestId"]; ok {
			_ = json.Unmarshal(v, &body.RequestID)
		}
		e.Message, e.Code, e.RequestID = body.Error, body.Code, body.RequestID
		if body.Details != nil {
			e.Details = body.Details
		}
		e.FieldErrors = fieldErrorsFromDetails(e.Details)
	}

	if e.Message == "" {
		e.Message = statusMessage(resp.status)
	}
	if e.RequestID == "" {
		e.RequestID = resp.header.Get("X-Request-Id")
	}
	return e
}

// statusMessage is the default message for a status without a body message.
func statusMessage(status int) string {
	if text := http.StatusText(status); text != "" {
		return strings.ToLower(text)
	}
	return fmt.Sprintf("http status %d", status)
}

// fieldErrorsFromDetails reads details.errors ([{field, message}]), skipping malformed entries.
func fieldErrorsFromDetails(details map[string]any) []FieldError {
	list, _ := details["errors"].([]any)
	var out []FieldError
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		field, _ := entry["field"].(string)
		message, _ := entry["message"].(string)
		if field == "" && message == "" {
			continue
		}
		out = append(out, FieldError{Field: field, Message: message})
	}
	return out
}

// errorFromResponse maps a non-2xx response to the SDK's typed error.
func errorFromResponse(resp *response, now time.Time) error {
	e := parseAPIError(resp)
	switch status := resp.status; {
	case status == http.StatusBadRequest && e.Code == codeValidationFailed:
		return &ValidationError{e}
	case status == http.StatusBadRequest:
		return &BadRequestError{e}
	case status == http.StatusUnauthorized:
		return &AuthenticationError{e}
	case status == http.StatusForbidden:
		return &PermissionDeniedError{e}
	case status == http.StatusNotFound:
		return &NotFoundError{e}
	case status == http.StatusConflict:
		return &ConflictError{e}
	case status == http.StatusGone:
		return &GoneError{e}
	case status == http.StatusUnprocessableEntity && e.Code == codeIdempotencyKeyReused:
		return &IdempotencyError{e}
	case status == http.StatusTooManyRequests:
		return &RateLimitError{
			APIError:      e,
			RetryAfter:    retryAfter(resp.header, e.Details, now),
			Limit:         detailInt(e.Details, "limit"),
			PeriodSeconds: detailInt(e.Details, "periodSeconds"),
		}
	case status >= 500, status < 200, status >= 300 && status < 400:
		return &ServerError{e}
	}
	return e
}

// retryAfter reads the Retry-After header (delta-seconds or an HTTP-date), else
// details.retryAfterSeconds. It returns 0 when neither is usable.
func retryAfter(header http.Header, details map[string]any, now time.Time) time.Duration {
	if value := strings.TrimSpace(header.Get("Retry-After")); value != "" {
		if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
			if seconds >= 0 {
				return time.Duration(min(seconds, maxRetryAfterSeconds)) * time.Second
			}
		} else if when, err := http.ParseTime(value); err == nil {
			// An HTTP-date has a one-second resolution: round the wait up.
			wait := when.Sub(now)
			if wait < 0 {
				return 0
			}
			return (wait + time.Second - 1).Truncate(time.Second)
		}
	}
	if seconds := detailInt(details, "retryAfterSeconds"); seconds > 0 {
		seconds = min(seconds, maxRetryAfterSeconds)
		return time.Duration(seconds) * time.Second
	}
	return 0
}

// maxRetryAfterSeconds bounds a Retry-After value so it never overflows a time.Duration
// (about a year: far above the 60 s the SDK is willing to wait anyway).
const maxRetryAfterSeconds = 365 * 24 * 3600

// detailInt reads an integer from details (JSON numbers decode as float64); 0 when absent.
func detailInt(details map[string]any, key string) int {
	switch v := details[key].(type) {
	case float64:
		if v >= 0 && v < 1<<31 {
			return int(v)
		}
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}
