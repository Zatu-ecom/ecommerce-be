package errors

import (
	"fmt"
	"net/http"

	"ecommerce-be/product/utils"
)

// AppError represents a structured application error with HTTP status and error code
type AppError struct {
	Code       string // Error code for client identification
	Message    string // Human-readable error message
	StatusCode int    // HTTP status code
}

// Error implements the error interface
func (e *AppError) Error() string {
	return e.Message
}

// NewAppError creates a new AppError instance
func NewAppError(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// WithMessage creates a new error with a custom message
func (e *AppError) WithMessage(message string) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    message,
		StatusCode: e.StatusCode,
	}
}

// WithMessagef creates a new error with a formatted message
func (e *AppError) WithMessagef(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: e.StatusCode,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError attempts to cast an error to AppError
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}

// Common/Validation Errors

var (
	// ErrValidation is a generic validation error
	ErrValidation = &AppError{
		Code:       utils.VALIDATION_ERROR_CODE,
		Message:    utils.VALIDATION_FAILED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidID is returned when an ID parameter is invalid
	ErrInvalidID = &AppError{
		Code:       utils.VALIDATION_ERROR_CODE,
		Message:    "Invalid ID parameter",
		StatusCode: http.StatusBadRequest,
	}
)
