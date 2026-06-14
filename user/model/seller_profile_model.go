package model

import "ecommerce-be/common/filegateway"

// ============================================================================
// SELLER PROFILE REQUEST MODELS
// ============================================================================

// SellerProfileCreateRequest contains business profile data for creation
// Used in seller registration and standalone profile creation
type SellerProfileCreateRequest struct {
	BusinessName       string `json:"businessName"       binding:"required,min=2,max=200"`
	BusinessLogoFileID string `json:"businessLogoFileId" binding:"required"`
	TaxID              string `json:"taxId"              binding:"omitempty,max=50"`
}

// SellerProfileUpdateRequest contains fields for updating seller profile
// Uses pointers to distinguish between null (don't update) and empty (set to empty)
type SellerProfileUpdateRequest struct {
	BusinessName       *string `json:"businessName"       binding:"omitempty,min=2,max=200"`
	BusinessLogoFileID *string `json:"businessLogoFileId" binding:"omitempty"`
	TaxID              *string `json:"taxId"              binding:"omitempty,max=50"`
}

// ============================================================================
// SELLER PROFILE RESPONSE MODELS
// ============================================================================

// SellerFullProfileResponse represents the complete seller profile
// Combines user info, business profile, and settings
// Reuses existing response models for consistency
type SellerFullProfileResponse struct {
	User      UserResponse            `json:"user"`
	Profile   SellerProfileResponse   `json:"profile"`
	Settings  *SellerSettingsResponse `json:"settings,omitempty"`
	Addresses []AddressResponse       `json:"addresses"`
}

// SellerLoginProfileResponse represents seller-specific profile data returned with login.
type SellerLoginProfileResponse struct {
	Profile   SellerProfileResponse   `json:"profile"`
	Settings  *SellerSettingsResponse `json:"settings,omitempty"`
	Addresses []AddressResponse       `json:"addresses"`
}

// SellerProfileResponse contains profile data in the response
type SellerProfileResponse struct {
	UserID       uint                           `json:"userId"`
	BusinessName string                         `json:"businessName"`
	BusinessLogo *filegateway.FileAssetResponse `json:"businessLogo,omitempty"`
	TaxID        string                         `json:"taxId,omitempty"`
	IsVerified   bool                           `json:"isVerified"`
	CreatedAt    string                         `json:"createdAt"`
}
