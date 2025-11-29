package errors

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/inventory/utils/constant"
)

var (
	// ErrLocationNotFound is returned when a location is not found
	ErrLocationNotFound = &commonerrors.AppError{
		Code:       constant.LOCATION_NOT_FOUND_CODE,
		Message:    constant.LOCATION_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrDuplicateLocationName is returned when a location name already exists for the seller
	ErrDuplicateLocationName = &commonerrors.AppError{
		Code:       constant.DUPLICATE_LOCATION_NAME_CODE,
		Message:    constant.DUPLICATE_LOCATION_NAME_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrInvalidLocationType is returned when an invalid location type is provided
	ErrInvalidLocationType = &commonerrors.AppError{
		Code:       constant.INVALID_LOCATION_TYPE_CODE,
		Message:    constant.INVALID_LOCATION_TYPE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrLocationInactive is returned when trying to use an inactive location
	ErrLocationInactive = &commonerrors.AppError{
		Code:       constant.LOCATION_INACTIVE_CODE,
		Message:    constant.LOCATION_INACTIVE_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrUnauthorizedLocationAccess is returned when a seller tries to access another seller's location
	ErrUnauthorizedLocationAccess = &commonerrors.AppError{
		Code:       constant.UNAUTHORIZED_LOCATION_ACCESS_CODE,
		Message:    constant.UNAUTHORIZED_LOCATION_ACCESS_MSG,
		StatusCode: http.StatusForbidden,
	}
)
