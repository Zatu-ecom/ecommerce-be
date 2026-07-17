package model

// ForgotPasswordRequest is the request body for the forgot-password endpoint
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest is the request body for the reset-password endpoint
type ResetPasswordRequest struct {
	Token           string `json:"token"           binding:"required"`
	NewPassword     string `json:"newPassword"     binding:"required,min=6"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,min=6"`
}
