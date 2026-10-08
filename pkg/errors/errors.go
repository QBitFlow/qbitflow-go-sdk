package qberrors

import (
	"fmt"
	"strings"
)

// QBitFlowError represents a custom error for the QBitFlow SDK
type QBitFlowError struct {
	Message    string
	StatusCode int
	Err        error
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

// NotFoundError represents a 404 not found error
type NotFoundError struct {
	Message string
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("Not Found: %s", e.Message)
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{Message: message}
}

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

// ValidationErrors is the wire envelope the API uses for a 400 validation response.
type ValidationErrors struct {
	Errors []FieldError `json:"errors"`
}

// ValidationError represents a validation error, raised either by the API (HTTP 400/422)
// or by the SDK before a request is sent.
type ValidationError struct {
	// Message is a human-readable summary. For an API validation response it lists
	// every field that failed, so no failure is hidden behind the first one.
	Message string
	// Fields carries the per-field failures when the API supplied them. It is empty for
	// validation errors raised client-side.
	Fields []FieldError
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

// NewValidationErrorFromList creates a ValidationError from an API validation response.
//
// Every field failure is retained, both in the summary message and in Fields — reporting
// only the first one hides the rest of what the caller needs to fix.
func NewValidationErrorFromList(errors ValidationErrors) *ValidationError {
	parts := make([]string, 0, len(errors.Errors))
	for _, fieldErr := range errors.Errors {
		switch {
		case fieldErr.Field != "" && fieldErr.Message != "":
			parts = append(parts, fmt.Sprintf("%s: %s", fieldErr.Field, fieldErr.Message))
		case fieldErr.Message != "":
			parts = append(parts, fieldErr.Message)
		case fieldErr.Field != "":
			parts = append(parts, fieldErr.Field)
		}
	}

	message := strings.Join(parts, "; ")
	if message == "" {
		message = "request failed validation"
	}

	return &ValidationError{Message: message, Fields: errors.Errors}
}
