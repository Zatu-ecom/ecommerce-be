package factory

import (
	"time"

	"ecommerce-be/common/filegateway"
	"ecommerce-be/user/entity"
	"ecommerce-be/user/model"
)

/***********************************************
 *      Seller Response Builders               *
 ***********************************************/

// BuildUserResponseForSeller creates UserResponse from User entity for seller registration
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
func BuildSellerProfileResponse(
	profile *entity.SellerProfile,
	logo *filegateway.FileAssetResponse,
) model.SellerProfileResponse {
	return model.SellerProfileResponse{
		UserID:       profile.UserID,
		BusinessName: profile.BusinessName,
		BusinessLogo: logo,
		TaxID:        profile.TaxID,
		IsVerified:   profile.IsVerified,
		CreatedAt:    profile.CreatedAt.Format(time.RFC3339),
	}
}

// BuildSellerProfileResponsePtr creates *SellerProfileResponse from SellerProfile entity
func BuildSellerProfileResponsePtr(
	profile *entity.SellerProfile,
	logo *filegateway.FileAssetResponse,
) *model.SellerProfileResponse {
	if profile == nil {
		return nil
	}
	resp := BuildSellerProfileResponse(profile, logo)
	return &resp
}

// BuildSellerFullProfileResponse creates the complete seller profile response
func BuildSellerFullProfileResponse(
	user model.UserResponse,
	profile *entity.SellerProfile,
	settings *model.SellerSettingsResponse,
	addresses []model.AddressResponse,
	logo *filegateway.FileAssetResponse,
) *model.SellerFullProfileResponse {
	return &model.SellerFullProfileResponse{
		User:      user,
		Profile:   BuildSellerProfileResponse(profile, logo),
		Settings:  settings,
		Addresses: addresses,
	}
}

// BuildSellerLoginProfileResponse creates the seller-specific payload for auth/login responses.
func BuildSellerLoginProfileResponse(
	profile *entity.SellerProfile,
	settings *model.SellerSettingsResponse,
	addresses []model.AddressResponse,
	logo *filegateway.FileAssetResponse,
) *model.SellerLoginProfileResponse {
	if profile == nil {
		return nil
	}

	return &model.SellerLoginProfileResponse{
		Profile:   BuildSellerProfileResponse(profile, logo),
		Settings:  settings,
		Addresses: addresses,
	}
}

// BuildSellerSettingsResponse creates SellerSettingsResponse from SellerSettings entity
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
func BuildSellerRegisterResponse(
	user *entity.User,
	profile *entity.SellerProfile,
	settings *model.SellerSettingsResponse,
	token string,
	expiresIn string,
	requiresOnboarding bool,
	logo *filegateway.FileAssetResponse,
) *model.SellerRegisterResponse {
	return &model.SellerRegisterResponse{
		User:               BuildUserResponseForSeller(user),
		Profile:            BuildSellerProfileResponse(profile, logo),
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

	if req.SettlementCurrencyID != nil {
		settings.SettlementCurrencyID = *req.SettlementCurrencyID
	} else {
		settings.SettlementCurrencyID = req.BaseCurrencyID
	}

	if req.DisplayPricesInBuyerCurrency != nil {
		settings.DisplayPricesInBuyerCurrency = *req.DisplayPricesInBuyerCurrency
	}

	return settings
}
