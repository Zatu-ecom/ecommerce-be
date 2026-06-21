package model

// ============================================================================
// SELLER REGISTRATION REQUEST MODELS
// ============================================================================

// SellerRegisterRequest represents the complete seller registration request
// Contains user data, seller profile, and optional settings
// Reuses existing models for maintainability
type SellerRegisterRequest struct {
	// User basic information (reuses CreateUserRequest with ConfirmPassword)
	User UserRegisterRequest `json:"user" binding:"required"`

	// Seller business profile (reuses SellerProfileCreateRequest)
	Profile SellerProfileCreateRequest `json:"profile" binding:"required"`

	// Seller settings (optional - reuses SellerSettingsCreateRequest)
	// If settings are not provided during registration, seller must complete onboarding
	Settings *SellerSettingsCreateRequest `json:"settings" binding:"omitempty"`
}

// ============================================================================
// SELLER REGISTRATION RESPONSE MODELS
// ============================================================================

// SellerRegisterResponse represents the response after successful seller registration
// Reuses existing response models for consistency
type SellerRegisterResponse struct {
	User      UserResponse            `json:"user"`
	Profile   SellerProfileResponse   `json:"profile"`
	Settings  *SellerSettingsResponse `json:"settings,omitempty"`
	Token     string                  `json:"token"`
	ExpiresIn string                  `json:"expiresIn"`

	// Indicates if seller needs to complete onboarding (settings not provided)
	RequiresOnboarding bool `json:"requiresOnboarding"`
}
