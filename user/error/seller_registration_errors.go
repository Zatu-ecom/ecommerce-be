package error

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/user/utils/constant"
)

// ========================================
// SELLER REGISTRATION ERRORS
// ========================================
var (
	// ErrEmailAlreadyExists is returned when the email is already registered
	ErrEmailAlreadyExists = &commonerrors.AppError{
		Code:       constant.EMAIL_ALREADY_EXISTS_CODE,
		Message:    constant.EMAIL_ALREADY_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrTaxIDAlreadyExists is returned when the tax ID is already registered
	ErrTaxIDAlreadyExists = &commonerrors.AppError{
		Code:       constant.TAX_ID_ALREADY_EXISTS_CODE,
		Message:    constant.TAX_ID_ALREADY_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrTaxIDCheckFailed is returned when tax ID validation fails
	ErrTaxIDCheckFailed = &commonerrors.AppError{
		Code:       constant.TAX_ID_CHECK_FAILED_CODE,
		Message:    constant.TAX_ID_CHECK_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}

	// ErrUserCreateFailed is returned when user creation fails
	ErrUserCreateFailed = &commonerrors.AppError{
		Code:       constant.USER_CREATE_FAILED_CODE,
		Message:    constant.USER_CREATE_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}

	// ErrSellerIDUpdateFailed is returned when seller ID update fails
	ErrSellerIDUpdateFailed = &commonerrors.AppError{
		Code:       constant.SELLER_ID_UPDATE_FAILED_CODE,
		Message:    constant.SELLER_ID_UPDATE_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}

	// ErrProfileCreateFailed is returned when seller profile creation fails
	ErrProfileCreateFailed = &commonerrors.AppError{
		Code:       constant.PROFILE_CREATE_FAILED_CODE,
		Message:    constant.PROFILE_CREATE_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}

	// ErrSettingsCreateFailed is returned when seller settings creation fails
	ErrSettingsCreateFailed = &commonerrors.AppError{
		Code:       constant.SETTINGS_CREATE_FAILED_CODE,
		Message:    constant.SETTINGS_CREATE_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}

	// ErrTokenGenerationFailed is returned when JWT token generation fails
	ErrTokenGenerationFailed = &commonerrors.AppError{
		Code:       constant.TOKEN_GENERATION_FAILED_CODE,
		Message:    constant.TOKEN_GENERATION_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}
)

// ========================================
// SELLER PROFILE ERRORS
// ========================================
var (
	// ErrSellerProfileNotFound is returned when seller profile is not found
	ErrSellerProfileNotFound = &commonerrors.AppError{
		Code:       constant.SELLER_PROFILE_NOT_FOUND_CODE,
		Message:    constant.SELLER_PROFILE_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrSellerProfileExists is returned when seller profile already exists
	ErrSellerProfileExists = &commonerrors.AppError{
		Code:       constant.SELLER_PROFILE_EXISTS_CODE,
		Message:    constant.SELLER_PROFILE_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrProfileUpdateFailed is returned when seller profile update fails
	ErrProfileUpdateFailed = &commonerrors.AppError{
		Code:       constant.SELLER_PROFILE_UPDATE_FAILED_CODE,
		Message:    constant.SELLER_PROFILE_UPDATE_FAILED_MSG,
		StatusCode: http.StatusInternalServerError,
	}
)
