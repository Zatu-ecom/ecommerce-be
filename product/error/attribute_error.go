package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

// Attribute Definition Errors

var (
	// ErrAttributeExists is returned when an attribute with the same key already exists
	ErrAttributeExists = &commonError.AppError{
		Code:       utils.ATTRIBUTE_DEFINITION_EXISTS_CODE,
		Message:    utils.ATTRIBUTE_DEFINITION_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrAttributeNotFound is returned when an attribute definition is not found
	ErrAttributeNotFound = &commonError.AppError{
		Code:       utils.ATTRIBUTE_DEFINITION_NOT_FOUND_CODE,
		Message:    utils.ATTRIBUTE_DEFINITION_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrInvalidAttributeKey is returned when attribute key format is invalid
	ErrInvalidAttributeKey = &commonError.AppError{
		Code:       utils.ATTRIBUTE_KEY_EXISTS_CODE,
		Message:    utils.ATTRIBUTE_KEY_FORMAT_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidDataType is returned when attribute data type is invalid
	ErrInvalidDataType = &commonError.AppError{
		Code:       utils.ATTRIBUTE_DATA_TYPE_INVALID_CODE,
		Message:    utils.ATTRIBUTE_DATA_TYPE_INVALID_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrPhysicalSpecFamilyConflict is returned when a second unit key from
	// an already-represented measurement family is attached to a product.
	ErrPhysicalSpecFamilyConflict = &commonError.AppError{
		Code:       utils.PHYSICAL_SPEC_FAMILY_CONFLICT_CODE,
		Message:    utils.PHYSICAL_SPEC_FAMILY_CONFLICT_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidScope is returned when a scope query parameter is missing
	// or not supported (the definitions catalog only serves fulfillment).
	ErrInvalidScope = &commonError.AppError{
		Code:       utils.INVALID_SCOPE_CODE,
		Message:    utils.INVALID_SCOPE_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
