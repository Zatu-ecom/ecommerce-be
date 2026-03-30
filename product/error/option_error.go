package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

// Product Option Errors

var (
	// ErrProductOptionNotFound is returned when a product option is not found
	ErrProductOptionNotFound = &commonError.AppError{
		Code:       utils.PRODUCT_OPTION_NOT_FOUND_CODE,
		Message:    utils.PRODUCT_OPTION_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrProductOptionNameExists is returned when a product option name already exists
	ErrProductOptionNameExists = &commonError.AppError{
		Code:       utils.PRODUCT_OPTION_NAME_EXISTS_CODE,
		Message:    utils.PRODUCT_OPTION_NAME_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrProductOptionInUse is returned when trying to delete an option being used by variants
	ErrProductOptionInUse = &commonError.AppError{
		Code:       utils.PRODUCT_OPTION_IN_USE_CODE,
		Message:    utils.PRODUCT_OPTION_IN_USE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrProductOptionValueNotFound is returned when a product option value is not found
	ErrProductOptionValueNotFound = &commonError.AppError{
		Code:       utils.PRODUCT_OPTION_VALUE_NOT_FOUND_CODE,
		Message:    utils.PRODUCT_OPTION_VALUE_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrProductOptionValueInUse is returned when trying to delete an option value being used by variants
	ErrProductOptionValueInUse = &commonError.AppError{
		Code:       utils.PRODUCT_OPTION_VALUE_IN_USE_CODE,
		Message:    utils.PRODUCT_OPTION_VALUE_IN_USE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrProductOptionValueExists is returned when an option value already exists
	ErrProductOptionValueExists = &commonError.AppError{
		Code:       utils.PRODUCT_OPTION_VALUE_EXISTS_CODE,
		Message:    utils.PRODUCT_OPTION_VALUE_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrProductOptionMismatch is returned when option does not belong to the product
	ErrProductOptionMismatch = &commonError.AppError{
		Code:       utils.PRODUCT_OPTION_PRODUCT_MISMATCH_CODE,
		Message:    utils.PRODUCT_OPTION_PRODUCT_MISMATCH_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrProductOptionValueMismatch is returned when option value does not belong to the option
	ErrProductOptionValueMismatch = &commonError.AppError{
		Code:       utils.PRODUCT_OPTION_VALUE_OPTION_MISMATCH_CODE,
		Message:    utils.PRODUCT_OPTION_VALUE_OPTION_MISMATCH_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
