package error

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/user/utils/constant"
)

var (
	// ErrCurrencyNotFound is returned when a currency is not found
	ErrCurrencyNotFound = &commonerrors.AppError{
		Code:       constant.CURRENCY_NOT_FOUND_CODE,
		Message:    constant.CURRENCY_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrDuplicateCurrencyCode is returned when a currency code already exists
	ErrDuplicateCurrencyCode = &commonerrors.AppError{
		Code:       constant.DUPLICATE_CURRENCY_CODE_CODE,
		Message:    constant.DUPLICATE_CURRENCY_CODE_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrCurrencyInactive is returned when trying to use an inactive currency
	ErrCurrencyInactive = &commonerrors.AppError{
		Code:       constant.CURRENCY_INACTIVE_CODE,
		Message:    constant.CURRENCY_INACTIVE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrCurrencyHasReferences is returned when trying to delete a currency that has references
	ErrCurrencyHasReferences = &commonerrors.AppError{
		Code:       constant.CURRENCY_HAS_REFERENCES_CODE,
		Message:    constant.CURRENCY_HAS_REFERENCES_MSG,
		StatusCode: http.StatusConflict,
	}
)
