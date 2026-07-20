package error

import (
	"net/http"

	commonerrors "ecommerce-be/common/error"
	"ecommerce-be/user/utils/constant"
)

// ========================================
// PASSWORD RESET ERRORS
// ========================================
var (
	// ErrInvalidResetToken is returned when the reset token is invalid or malformed
	ErrInvalidResetToken = &commonerrors.AppError{
		Code:       constant.INVALID_RESET_TOKEN_CODE,
		Message:    constant.INVALID_RESET_TOKEN_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrExpiredResetToken is returned when the reset token has expired
	ErrExpiredResetToken = &commonerrors.AppError{
		Code:       constant.EXPIRED_RESET_TOKEN_CODE,
		Message:    constant.EXPIRED_RESET_TOKEN_MSG,
		StatusCode: http.StatusBadRequest,
	}

	// ErrUsedResetToken is returned when the reset token has already been used
	ErrUsedResetToken = &commonerrors.AppError{
		Code:       constant.USED_RESET_TOKEN_CODE,
		Message:    constant.USED_RESET_TOKEN_MSG,
		StatusCode: http.StatusBadRequest,
	}
)
