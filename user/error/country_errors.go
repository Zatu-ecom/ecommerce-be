package error

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/user/utils/constant"
)

var (
	// ErrCountryNotFound is returned when a country is not found
	ErrCountryNotFound = &commonerrors.AppError{
		Code:       constant.COUNTRY_NOT_FOUND_CODE,
		Message:    constant.COUNTRY_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrDuplicateCountryCode is returned when a country code already exists
	ErrDuplicateCountryCode = &commonerrors.AppError{
		Code:       constant.DUPLICATE_COUNTRY_CODE_CODE,
		Message:    constant.DUPLICATE_COUNTRY_CODE_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrCountryInactive is returned when trying to use an inactive country
	ErrCountryInactive = &commonerrors.AppError{
		Code:       constant.COUNTRY_INACTIVE_CODE,
		Message:    constant.COUNTRY_INACTIVE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrCountryHasReferences is returned when trying to delete a country that has references
	ErrCountryHasReferences = &commonerrors.AppError{
		Code:       constant.COUNTRY_HAS_REFERENCES_CODE,
		Message:    constant.COUNTRY_HAS_REFERENCES_MSG,
		StatusCode: http.StatusConflict,
	}
)
