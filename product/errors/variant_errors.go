package errors

import (
	"net/http"

	"ecommerce-be/product/utils"
)

// Variant Errors

var (
	// ErrVariantNotFound is returned when a variant is not found
	ErrVariantNotFound = &AppError{
		StatusCode: http.StatusNotFound,
		Code:       utils.VARIANT_NOT_FOUND_CODE,
		Message:    utils.VARIANT_NOT_FOUND_MSG,
	}

	// ErrVariantSKUExists is returned when a variant SKU already exists
	ErrVariantSKUExists = &AppError{
		StatusCode: http.StatusConflict,
		Code:       utils.VARIANT_SKU_EXISTS_CODE,
		Message:    utils.VARIANT_SKU_EXISTS_MSG,
	}

	// ErrVariantCombinationExists is returned when a variant with the same option combination already exists
	ErrVariantCombinationExists = &AppError{
		StatusCode: http.StatusConflict,
		Code:       utils.VARIANT_OPTION_COMBINATION_EXISTS_CODE,
		Message:    utils.VARIANT_OPTION_COMBINATION_EXISTS_MSG,
	}

	// ErrProductHasNoOptions is returned when a product has no options defined
	ErrProductHasNoOptions = &AppError{
		StatusCode: http.StatusBadRequest,
		Code:       utils.INVALID_OPTION_CODE,
		Message:    utils.PRODUCT_HAS_NO_OPTIONS_MSG,
	}

	// ErrLastVariantDeleteNotAllowed is returned when trying to delete the last variant
	ErrLastVariantDeleteNotAllowed = &AppError{
		StatusCode: http.StatusBadRequest,
		Code:       utils.LAST_VARIANT_DELETE_NOT_ALLOWED_CODE,
		Message:    utils.LAST_VARIANT_DELETE_NOT_ALLOWED_MSG,
	}

	// ErrInvalidStockOperation is returned when an invalid stock operation is requested
	ErrInvalidStockOperation = &AppError{
		StatusCode: http.StatusBadRequest,
		Code:       utils.INVALID_STOCK_OPERATION_CODE,
		Message:    utils.INVALID_STOCK_OPERATION_MSG,
	}

	// ErrInsufficientStockForOperation is returned when there's not enough stock for the requested operation
	ErrInsufficientStockForOperation = &AppError{
		StatusCode: http.StatusBadRequest,
		Code:       utils.INSUFFICIENT_STOCK_FOR_OPERATION_CODE,
		Message:    utils.INSUFFICIENT_STOCK_FOR_OPERATION_MSG,
	}

	// ErrBulkUpdateEmptyList is returned when trying to perform a bulk update with an empty list
	ErrBulkUpdateEmptyList = &AppError{
		StatusCode: http.StatusBadRequest,
		Code:       utils.BULK_UPDATE_EMPTY_LIST_CODE,
		Message:    utils.BULK_UPDATE_EMPTY_LIST_MSG,
	}

	// ErrBulkUpdateVariantNotFound is returned when one or more variants in a bulk update are not found
	ErrBulkUpdateVariantNotFound = &AppError{
		StatusCode: http.StatusNotFound,
		Code:       utils.BULK_UPDATE_VARIANT_NOT_FOUND_CODE,
		Message:    utils.BULK_UPDATE_VARIANT_NOT_FOUND_MSG,
	}
)
