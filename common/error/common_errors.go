package error

import (
	"net/http"

	"ecommerce-be/common/constants"
)

// Common validation error codes
const (
	VALIDATION_ERROR_CODE = "VALIDATION_ERROR"
	INVALID_ID_CODE       = "INVALID_ID"
)

// Common validation error messages
const (
	VALIDATION_FAILED_MSG = "Validation failed"
	INVALID_ID_MSG        = "Invalid ID parameter"
)

// Common/Validation Errors

var (
	// ErrValidation is a generic validation error
	ErrValidation = &AppError{
		Code:       VALIDATION_ERROR_CODE,
		Message:    VALIDATION_FAILED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidID is returned when an ID parameter is invalid
	ErrInvalidID = &AppError{
		Code:       INVALID_ID_CODE,
		Message:    INVALID_ID_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrRoleDataMissing is returned when role data is missing in context
	ErrRoleDataMissing = &AppError{
		Code:       constants.ROLE_DATA_MISSING_CODE,
		Message:    constants.ROLE_DATA_MISSING_MESSAGE,
		StatusCode: http.StatusInternalServerError,
	}

	// UnauthorizedError is returned when access is unauthorized
	UnauthorizedError = &AppError{
		Code:       constants.UNAUTHORIZED_ERROR_CODE,
		Message:    constants.UNAUTHORIZED_ERROR_MSG,
		StatusCode: http.StatusUnauthorized,
	}
)
