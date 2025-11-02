package qberrors

import "fmt"

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

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// NewValidationError creates a new ValidationError from a list of validation errors
func NewValidationErrorFromList(errors ValidationErrors) *ValidationError {
	messages := ""
	for _, err := range errors.Errors {
		messages += err.Error() + "; "
	}
	return &ValidationError{Message: messages}
}
