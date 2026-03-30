package error

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/user/utils/constant"
)

var (
	// ErrSellerSettingsNotFound is returned when seller settings are not found
	ErrSellerSettingsNotFound = &commonerrors.AppError{
		Code:       constant.SELLER_SETTINGS_NOT_FOUND_CODE,
		Message:    constant.SELLER_SETTINGS_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrSellerSettingsExists is returned when seller settings already exist
	ErrSellerSettingsExists = &commonerrors.AppError{
		Code:       constant.SELLER_SETTINGS_EXISTS_CODE,
		Message:    constant.SELLER_SETTINGS_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}
)
