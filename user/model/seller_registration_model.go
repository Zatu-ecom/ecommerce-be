package model

// ============================================================================
// SELLER REGISTRATION REQUEST MODELS
// ============================================================================

// SellerRegisterRequest represents the complete seller registration request
// Contains user data, seller profile, and optional settings
// Reuses existing models for maintainability
type SellerRegisterRequest struct {
	// User basic information (reuses CreateUserRequest with ConfirmPassword)
	User UserRegisterRequest `json:"user" binding:"required,dive"`

	// Seller business profile (reuses SellerProfileCreateRequest)
	Profile SellerProfileCreateRequest `json:"profile" binding:"required,dive"`

	// Seller settings (optional - reuses SellerSettingsCreateRequest)
	// If settings are not provided during registration, seller must complete onboarding
	Settings *SellerSettingsCreateRequest `json:"settings" binding:"omitempty,dive"`
}

// ============================================================================
// SELLER REGISTRATION RESPONSE MODELS
// ============================================================================

// SellerRegisterResponse represents the response after successful seller registration
// Reuses existing response models for consistency
type SellerRegisterResponse struct {
	User      UserResponse            `json:"user"`               // Reuses UserResponse
	Profile   SellerProfileResponse   `json:"profile"`            // Reuses SellerProfileResponse
	Settings  *SellerSettingsResponse `json:"settings,omitempty"` // Null if settings not provided during registration
	Token     string                  `json:"token"`
	ExpiresIn string                  `json:"expiresIn"`

	// Indicates if seller needs to complete onboarding (settings not provided)
	RequiresOnboarding bool `json:"requiresOnboarding"`
}

// SellerProfileResponse contains profile data in the response
type SellerProfileResponse struct {
	UserID       uint   `json:"userId"`
	BusinessName string `json:"businessName"`
	BusinessLogo string `json:"businessLogo"`
	TaxID        string `json:"taxId,omitempty"`
	IsVerified   bool   `json:"isVerified"`
	CreatedAt    string `json:"createdAt"`
}
