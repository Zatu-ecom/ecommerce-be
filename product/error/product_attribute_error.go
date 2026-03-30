package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

var (
	// ErrProductAttributeNotFound is returned when a product attribute is not found
	ErrProductAttributeNotFound = &commonError.AppError{
		Code:       utils.PRODUCT_ATTRIBUTE_NOT_FOUND_CODE,
		Message:    utils.PRODUCT_ATTRIBUTE_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrProductAttributeExists is returned when a product attribute already exists
	ErrProductAttributeExists = &commonError.AppError{
		Code:       utils.PRODUCT_ATTRIBUTE_EXISTS_CODE,
		Message:    utils.PRODUCT_ATTRIBUTE_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrInvalidAttributeValue is returned when attribute value is invalid
	ErrInvalidAttributeValue = &commonError.AppError{
		Code:       utils.INVALID_ATTRIBUTE_VALUE_CODE,
		Message:    utils.INVALID_ATTRIBUTE_VALUE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrUnauthorizedAttributeAccess is returned when user tries to access attribute they don't own
	ErrUnauthorizedAttributeAccess = &commonError.AppError{
		Code:       utils.UNAUTHORIZED_ATTRIBUTE_ACCESS_CODE,
		Message:    utils.UNAUTHORIZED_ATTRIBUTE_ACCESS_MSG,
		StatusCode: http.StatusForbidden,
	}
)
