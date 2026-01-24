package errors

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

// Product Errors

var (
	// ErrProductExists is returned when a product with the same SKU already exists
	ErrProductExists = &commonerrors.AppError{
		Code:       utils.PRODUCT_EXISTS_CODE,
		Message:    utils.PRODUCT_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrProductNotFound is returned when a product is not found
	ErrProductNotFound = &commonerrors.AppError{
		Code:       utils.PRODUCT_NOT_FOUND_CODE,
		Message:    utils.PRODUCT_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrProductSKUExists is returned when a product SKU already exists
	ErrProductSKUExists = &commonerrors.AppError{
		Code:       utils.PRODUCT_SKU_EXISTS_CODE,
		Message:    utils.PRODUCT_SKU_UNIQUE_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrInvalidCategory is returned when product category is invalid
	ErrInvalidCategory = &commonerrors.AppError{
		Code:       utils.PRODUCT_CATEGORY_INVALID_CODE,
		Message:    utils.PRODUCT_CATEGORY_INVALID_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrInvalidAttributes is returned when product attributes are invalid
	ErrInvalidAttributes = &commonerrors.AppError{
		Code:       utils.PRODUCT_ATTRIBUTES_INVALID_CODE,
		Message:    utils.PRODUCT_ATTRIBUTES_REQUIRED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrUnauthorizedProductAccess = &commonerrors.AppError{
		Code:       utils.UNAUTHORIZED_PRODUCT_ACCESS_CODE,
		Message:    utils.PRODUCT_UNAUTHORIZED_ACCESS_MSG,
		StatusCode: http.StatusForbidden,
	}

	// ErrInvalidStrategy is returned when strategy parameter is invalid
	ErrInvalidStrategy = &commonerrors.AppError{
		Code:       utils.INVALID_STRATEGY_CODE,
		Message:    utils.INVALID_STRATEGY_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
