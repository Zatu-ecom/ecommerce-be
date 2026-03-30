package model

// ============================================================================
// SELLER PROFILE REQUEST MODELS
// ============================================================================

// SellerProfileCreateRequest contains business profile data for creation
// Used in seller registration and standalone profile creation
type SellerProfileCreateRequest struct {
	BusinessName string `json:"businessName" binding:"required,min=2,max=200"`
	BusinessLogo string `json:"businessLogo" binding:"omitempty,url,max=500"`
	TaxID        string `json:"taxId"        binding:"omitempty,max=50"`
}

// SellerProfileUpdateRequest contains fields for updating seller profile
// Uses pointers to distinguish between null (don't update) and empty (set to empty)
type SellerProfileUpdateRequest struct {
	BusinessName *string `json:"businessName" binding:"omitempty,min=2,max=200"`
	BusinessLogo *string `json:"businessLogo" binding:"omitempty,url,max=500"`
	TaxID        *string `json:"taxId"        binding:"omitempty,max=50"`
}

// ============================================================================
// SELLER PROFILE RESPONSE MODELS
// ============================================================================

// SellerFullProfileResponse represents the complete seller profile
// Combines user info, business profile, and settings
// Reuses existing response models for consistency
type SellerFullProfileResponse struct {
	User      UserResponse            `json:"user"`               // Reuses UserResponse
	Profile   SellerProfileResponse   `json:"profile"`            // Reuses SellerProfileResponse
	Settings  *SellerSettingsResponse `json:"settings,omitempty"` // Null if settings not configured
	Addresses []AddressResponse       `json:"addresses"`
}
