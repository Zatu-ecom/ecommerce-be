package errors

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

var (
	// ErrProductAttributeNotFound is returned when a product attribute is not found
	ErrProductAttributeNotFound = &commonerrors.AppError{
		Code:       utils.PRODUCT_ATTRIBUTE_NOT_FOUND_CODE,
		Message:    utils.PRODUCT_ATTRIBUTE_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrProductAttributeExists is returned when a product attribute already exists
	ErrProductAttributeExists = &commonerrors.AppError{
		Code:       utils.PRODUCT_ATTRIBUTE_EXISTS_CODE,
		Message:    utils.PRODUCT_ATTRIBUTE_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrInvalidAttributeValue is returned when attribute value is invalid
	ErrInvalidAttributeValue = &commonerrors.AppError{
		Code:       utils.INVALID_ATTRIBUTE_VALUE_CODE,
		Message:    utils.INVALID_ATTRIBUTE_VALUE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrUnauthorizedAttributeAccess is returned when user tries to access attribute they don't own
	ErrUnauthorizedAttributeAccess = &commonerrors.AppError{
		Code:       utils.UNAUTHORIZED_ATTRIBUTE_ACCESS_CODE,
		Message:    utils.UNAUTHORIZED_ATTRIBUTE_ACCESS_MSG,
		StatusCode: http.StatusForbidden,
	}
)
