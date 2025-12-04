package errors

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/inventory/utils/constant"
)

var (
	// Inventory errors
	ErrInventoryNotFound = &commonerrors.AppError{
		Code:       constant.INVENTORY_NOT_FOUND_CODE,
		Message:    constant.INVENTORY_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	ErrInsufficientStock = &commonerrors.AppError{
		Code:       constant.INSUFFICIENT_STOCK_CODE,
		Message:    constant.INSUFFICIENT_STOCK_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrInvalidQuantity = &commonerrors.AppError{
		Code:       constant.INVALID_QUANTITY_CODE,
		Message:    constant.INVALID_QUANTITY_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrNegativeStock = &commonerrors.AppError{
		Code:       constant.NEGATIVE_STOCK_CODE,
		Message:    constant.NEGATIVE_STOCK_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrBelowThreshold = &commonerrors.AppError{
		Code:       constant.BELOW_THRESHOLD_CODE,
		Message:    constant.BELOW_THRESHOLD_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrInsufficientReservedStock = &commonerrors.AppError{
		Code:       constant.INSUFFICIENT_RESERVED_STOCK_CODE,
		Message:    constant.INSUFFICIENT_RESERVED_STOCK_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrVariantNotFound = &commonerrors.AppError{
		Code:       constant.VARIANT_NOT_FOUND_CODE,
		Message:    constant.VARIANT_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// Transaction errors
	ErrInvalidTransactionType = &commonerrors.AppError{
		Code:       constant.INVALID_TRANSACTION_TYPE_CODE,
		Message:    constant.INVALID_TRANSACTION_TYPE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrInvalidAdjustmentType = &commonerrors.AppError{
		Code:       constant.INVALID_ADJUSTMENT_TYPE_CODE,
		Message:    constant.INVALID_ADJUSTMENT_TYPE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrDirectionRequired = &commonerrors.AppError{
		Code:       constant.DIRECTION_REQUIRED_CODE,
		Message:    constant.DIRECTION_REQUIRED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrDirectionNotAllowed = &commonerrors.AppError{
		Code:       constant.DIRECTION_NOT_ALLOWED_CODE,
		Message:    constant.DIRECTION_NOT_ALLOWED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrNotManualTransaction = &commonerrors.AppError{
		Code:       constant.NOT_MANUAL_TRANSACTION_CODE,
		Message:    constant.NOT_MANUAL_TRANSACTION_MSG,
		StatusCode: http.StatusBadRequest,
	}

	ErrReferenceRequired = &commonerrors.AppError{
		Code:       constant.REFERENCE_REQUIRED_CODE,
		Message:    constant.REFERENCE_REQUIRED_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
