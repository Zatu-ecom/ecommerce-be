package error

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/user/utils/constant"
)

var (
	// ErrCountryCurrencyNotFound is returned when a country-currency mapping is not found
	ErrCountryCurrencyNotFound = &commonerrors.AppError{
		Code:       constant.COUNTRY_CURRENCY_NOT_FOUND_CODE,
		Message:    constant.COUNTRY_CURRENCY_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrCountryCurrencyExists is returned when trying to add a duplicate mapping
	ErrCountryCurrencyExists = &commonerrors.AppError{
		Code:       constant.COUNTRY_CURRENCY_EXISTS_CODE,
		Message:    constant.COUNTRY_CURRENCY_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrPrimaryCurrencyRequired is returned when trying to remove the primary currency
	ErrPrimaryCurrencyRequired = &commonerrors.AppError{
		Code:       constant.PRIMARY_CURRENCY_REQUIRED_CODE,
		Message:    constant.PRIMARY_CURRENCY_REQUIRED_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrMultiplePrimaryCurrencies is returned when multiple currencies are marked as primary
	ErrMultiplePrimaryCurrencies = &commonerrors.AppError{
		Code:       constant.MULTIPLE_PRIMARY_CURRENCIES_CODE,
		Message:    constant.MULTIPLE_PRIMARY_CURRENCIES_MSG,
		StatusCode: http.StatusConflict,
	}
)
