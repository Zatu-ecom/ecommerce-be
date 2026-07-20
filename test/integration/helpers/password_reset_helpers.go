package helpers

// ForgotPasswordRequest represents the forgot-password request body
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest represents the reset-password request body
type ResetPasswordRequest struct {
	Token           string `json:"token"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

// BuildForgotPasswordRequest creates a forgot-password request body
func BuildForgotPasswordRequest(email string) map[string]any {
	return map[string]any{
		"email": email,
	}
}

// BuildResetPasswordRequest creates a reset-password request body
func BuildResetPasswordRequest(token, newPassword, confirmPassword string) map[string]any {
	return map[string]any{
		"token":           token,
		"newPassword":     newPassword,
		"confirmPassword": confirmPassword,
	}
}
