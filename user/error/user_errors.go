package error

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/user/utils/constant"
)

// ========================================
// USER ERRORS (Common across user module)
// ========================================
var (
	// ErrPasswordMismatch is returned when password and confirm password do not match
	ErrPasswordMismatch = &commonerrors.AppError{
		Code:       constant.PASSWORD_MISMATCH_CODE,
		Message:    constant.PASSWORD_MISMATCH_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrUserExists is returned when user with same email already exists
	ErrUserExists = &commonerrors.AppError{
		Code:       constant.USER_EXISTS_CODE,
		Message:    constant.USER_EXISTS_MSG,
		StatusCode: http.StatusConflict,
	}

	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = &commonerrors.AppError{
		Code:       constant.USER_NOT_FOUND_CODE,
		Message:    constant.USER_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}

	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = &commonerrors.AppError{
		Code:       constant.INVALID_CREDENTIALS_CODE,
		Message:    constant.INVALID_CREDENTIALS_MSG,
		StatusCode: http.StatusUnauthorized,
	}

	// ErrAccountDeactivated is returned when account is deactivated
	ErrAccountDeactivated = &commonerrors.AppError{
		Code:       constant.ACCOUNT_DEACTIVATED_CODE,
		Message:    constant.ACCOUNT_DEACTIVATED_MSG,
		StatusCode: http.StatusForbidden,
	}

	// ErrInvalidCurrentPassword is returned when current password is incorrect
	ErrInvalidCurrentPassword = &commonerrors.AppError{
		Code:       constant.INVALID_CURRENT_PASSWORD_CODE,
		Message:    constant.INVALID_CURRENT_PASSWORD_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
