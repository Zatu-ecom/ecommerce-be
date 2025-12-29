package model

// ========================================
// BASE MODELS (for inheritance)
// ========================================

// SellerSettingsBase contains common fields for seller settings
// Changes here will reflect in create/update/response models
type SellerSettingsBase struct {
	BusinessCountryID            uint `json:"businessCountryId"`            // Country where business is registered
	BaseCurrencyID               uint `json:"baseCurrencyId"`               // Prices stored in this currency
	SettlementCurrencyID         uint `json:"settlementCurrencyId"`         // Payouts in this currency
	DisplayPricesInBuyerCurrency bool `json:"displayPricesInBuyerCurrency"` // Convert prices for buyers
}

// ========================================
// REQUEST MODELS
// ========================================

// SellerSettingsCreateRequest - Seller creates their settings (onboarding)
type SellerSettingsCreateRequest struct {
	BusinessCountryID            uint  `json:"businessCountryId"            binding:"required"`
	BaseCurrencyID               uint  `json:"baseCurrencyId"               binding:"required"`
	SettlementCurrencyID         *uint `json:"settlementCurrencyId"`                           // Optional, defaults to BaseCurrencyID
	DisplayPricesInBuyerCurrency *bool `json:"displayPricesInBuyerCurrency"`                   // Optional, defaults to false
}

// SellerSettingsUpdateRequest - Seller updates their settings (all fields optional)
type SellerSettingsUpdateRequest struct {
	BusinessCountryID            *uint `json:"businessCountryId"`
	BaseCurrencyID               *uint `json:"baseCurrencyId"`
	SettlementCurrencyID         *uint `json:"settlementCurrencyId"`
	DisplayPricesInBuyerCurrency *bool `json:"displayPricesInBuyerCurrency"`
}

// ========================================
// RESPONSE MODELS
// ========================================

// SellerSettingsResponse - Seller settings response
type SellerSettingsResponse struct {
	ID                           uint             `json:"id"`
	SellerID                     uint             `json:"sellerId"`
	BusinessCountryID            uint             `json:"businessCountryId"`
	BaseCurrencyID               uint             `json:"baseCurrencyId"`
	SettlementCurrencyID         uint             `json:"settlementCurrencyId"`
	DisplayPricesInBuyerCurrency bool             `json:"displayPricesInBuyerCurrency"`
	CreatedAt                    string           `json:"createdAt"`
	UpdatedAt                    string           `json:"updatedAt"`
}

// SellerSettingsDetailResponse - Seller settings with expanded country/currency info
type SellerSettingsDetailResponse struct {
	ID                           uint             `json:"id"`
	SellerID                     uint             `json:"sellerId"`
	BusinessCountry              CountryResponse  `json:"businessCountry"`
	BaseCurrency                 CurrencyResponse `json:"baseCurrency"`
	SettlementCurrency           CurrencyResponse `json:"settlementCurrency"`
	DisplayPricesInBuyerCurrency bool             `json:"displayPricesInBuyerCurrency"`
	CreatedAt                    string           `json:"createdAt"`
	UpdatedAt                    string           `json:"updatedAt"`
}
