package constant

// ========================================
// PASSWORD RESET ERROR CODES
// ========================================
const (
	INVALID_RESET_TOKEN_CODE = "INVALID_RESET_TOKEN"
	EXPIRED_RESET_TOKEN_CODE = "EXPIRED_RESET_TOKEN"
	USED_RESET_TOKEN_CODE    = "USED_RESET_TOKEN"
)

// ========================================
// PASSWORD RESET ERROR MESSAGES
// ========================================
const (
	INVALID_RESET_TOKEN_MSG = "Invalid or malformed reset token"
	EXPIRED_RESET_TOKEN_MSG = "Reset token has expired"
	USED_RESET_TOKEN_MSG    = "Reset token has already been used"
)

// ========================================
// PASSWORD RESET SUCCESS MESSAGES
// ========================================
const (
	FORGOT_PASSWORD_SUCCESS_MSG = "If the email exists, a password reset link will be sent"
	RESET_PASSWORD_SUCCESS_MSG  = "Password has been reset successfully"
)

// ========================================
// PASSWORD RESET CONTEXT KEYS
// ========================================
const (
	RESET_TOKEN_KEY = "resetToken"
)
