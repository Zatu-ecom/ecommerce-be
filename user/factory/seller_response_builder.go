package factory

import (
	"time"

	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
)

/***********************************************
 *      Seller Response Builders               *
 ***********************************************/

// BuildUserResponseForSeller creates UserResponse from User entity for seller registration
// Reuses UserResponse model for consistency
func BuildUserResponseForSeller(user *entity.User) model.UserResponse {
	return model.UserResponse{
		ID:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Phone:       user.Phone,
		DateOfBirth: user.DateOfBirth,
		Gender:      user.Gender,
		IsActive:    user.IsActive,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
	}
}

// BuildSellerProfileResponse creates SellerProfileResponse from SellerProfile entity
// Used by seller registration and seller profile endpoints
func BuildSellerProfileResponse(profile *entity.SellerProfile) model.SellerProfileResponse {
	return model.SellerProfileResponse{
		UserID:       profile.UserID,
		BusinessName: profile.BusinessName,
		BusinessLogo: profile.BusinessLogo,
		TaxID:        profile.TaxID,
		IsVerified:   profile.IsVerified,
		CreatedAt:    profile.CreatedAt.Format(time.RFC3339),
	}
}

// BuildSellerProfileResponsePtr creates *SellerProfileResponse from SellerProfile entity
// Used when returning a pointer is required
func BuildSellerProfileResponsePtr(profile *entity.SellerProfile) *model.SellerProfileResponse {
	if profile == nil {
		return nil
	}
	resp := BuildSellerProfileResponse(profile)
	return &resp
}

// BuildSellerFullProfileResponse creates the complete seller profile response
// Combines user, profile, and settings information
// Reuses existing response models for consistency
func BuildSellerFullProfileResponse(
	user model.UserResponse,
	profile *entity.SellerProfile,
	settings *model.SellerSettingsResponse,
	addresses []model.AddressResponse,
) *model.SellerFullProfileResponse {
	return &model.SellerFullProfileResponse{
		User:      user,
		Profile:   BuildSellerProfileResponse(profile),
		Settings:  settings,
		Addresses: addresses,
	}
}

// BuildSellerSettingsResponse creates SellerSettingsResponse from SellerSettings entity
// Used by seller registration and seller settings endpoints
func BuildSellerSettingsResponse(settings *entity.SellerSettings) *model.SellerSettingsResponse {
	if settings == nil {
		return nil
	}
	return &model.SellerSettingsResponse{
		ID:                           settings.ID,
		SellerID:                     settings.SellerID,
		BusinessCountryID:            settings.BusinessCountryID,
		BaseCurrencyID:               settings.BaseCurrencyID,
		SettlementCurrencyID:         settings.SettlementCurrencyID,
		DisplayPricesInBuyerCurrency: settings.DisplayPricesInBuyerCurrency,
		CreatedAt:                    settings.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                    settings.UpdatedAt.Format(time.RFC3339),
	}
}

// BuildSellerRegisterResponse creates the full seller registration response
// Combines user, profile, settings, and auth information
// Reuses existing response models for consistency
func BuildSellerRegisterResponse(
	user *entity.User,
	profile *entity.SellerProfile,
	settings *model.SellerSettingsResponse,
	token string,
	expiresIn string,
	requiresOnboarding bool,
) *model.SellerRegisterResponse {
	return &model.SellerRegisterResponse{
		User:               BuildUserResponseForSeller(user),
		Profile:            BuildSellerProfileResponse(profile),
		Settings:           settings,
		Token:              token,
		ExpiresIn:          expiresIn,
		RequiresOnboarding: requiresOnboarding,
	}
}

/***********************************************
 *      Seller Settings Entity Builder         *
 ***********************************************/

// BuildSellerSettingsEntity creates SellerSettings entity from create request
// Handles default values for optional fields
func BuildSellerSettingsEntity(
	sellerID uint,
	req *model.SellerSettingsCreateRequest,
	now time.Time,
) *entity.SellerSettings {
	settings := &entity.SellerSettings{
		SellerID:                     sellerID,
		BusinessCountryID:            req.BusinessCountryID,
		BaseCurrencyID:               req.BaseCurrencyID,
		DisplayPricesInBuyerCurrency: false,
	}
	settings.CreatedAt = now
	settings.UpdatedAt = now

	// Set settlement currency (defaults to base currency if not provided)
	if req.SettlementCurrencyID != nil {
		settings.SettlementCurrencyID = *req.SettlementCurrencyID
	} else {
		settings.SettlementCurrencyID = req.BaseCurrencyID
	}

	// Set display preference if provided
	if req.DisplayPricesInBuyerCurrency != nil {
		settings.DisplayPricesInBuyerCurrency = *req.DisplayPricesInBuyerCurrency
	}

	return settings
}
