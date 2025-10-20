package errors

import (
	"net/http"

	"ecommerce-be/product/utils"
)

// Attribute Definition Errors

var (
	// ErrAttributeExists is returned when an attribute with the same key already exists
	ErrAttributeExists = &AppError{
		Code:       utils.ATTRIBUTE_DEFINITION_EXISTS_CODE,
		Message:    utils.ATTRIBUTE_DEFINITION_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrAttributeNotFound is returned when an attribute definition is not found
	ErrAttributeNotFound = &AppError{
		Code:       utils.ATTRIBUTE_DEFINITION_NOT_FOUND_CODE,
		Message:    utils.ATTRIBUTE_DEFINITION_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrInvalidAttributeKey is returned when attribute key format is invalid
	ErrInvalidAttributeKey = &AppError{
		Code:       utils.ATTRIBUTE_KEY_EXISTS_CODE,
		Message:    utils.ATTRIBUTE_KEY_FORMAT_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidDataType is returned when attribute data type is invalid
	ErrInvalidDataType = &AppError{
		Code:       utils.ATTRIBUTE_DATA_TYPE_INVALID_CODE,
		Message:    utils.ATTRIBUTE_DATA_TYPE_INVALID_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
