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

	// ErrNoFieldsProvided is returned when no fields are provided in update request
	ErrNoFieldsProvided = &AppError{
		Code:       constants.NO_FIELDS_PROVIDED_CODE,
		Message:    constants.NO_FIELDS_PROVIDED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidRequestStruct is returned when request structure is invalid
	ErrInvalidRequestStruct = &AppError{
		Code:       constants.INVALID_REQUEST_STRUCT_CODE,
		Message:    constants.INVALID_REQUEST_STRUCT_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrRoleDataMissing is returned when role data is missing in context
	ErrRoleDataMissing = &AppError{
		Code:       constants.ROLE_DATA_MISSING_CODE,
		Message:    constants.ROLE_DATA_MISSING_MSG,
		StatusCode: http.StatusInternalServerError,
	}

	// UnauthorizedError is returned when access is unauthorized
	UnauthorizedError = &AppError{
		Code:       constants.UNAUTHORIZED_ERROR_CODE,
		Message:    constants.UNAUTHORIZED_ERROR_MSG,
		StatusCode: http.StatusUnauthorized,
	}

	ErrSellerDataMissing = &AppError{
		Code:       constants.SELLER_DATA_MISSING_CODE,
		Message:    constants.SELLER_DATA_MISSING_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrRequiredQueryParam is returned when a required query parameter is missing
	ErrRequiredQueryParam = &AppError{
		Code:       constants.REQUIRED_QUERY_PARAM_CODE,
		Message:    constants.REQUIRED_QUERY_PARAM_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidLimit is returned when limit parameter is invalid
	ErrInvalidLimit = &AppError{
		Code:       constants.INVALID_LIMIT_CODE,
		Message:    constants.INVALID_LIMIT_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrUserDataMissing = &AppError{
		Code:       constants.USER_DATA_MISSING_CODE,
		Message:    constants.USER_DATA_MISSING_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrCorrelationIDMissing = &AppError{
		Code:       constants.CORRELATION_ID_MISSING,
		Message:    constants.CORRELATION_ID_MISSING_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
