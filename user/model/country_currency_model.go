package model

// ========================================
// REQUEST MODELS
// ========================================

// CountryCurrencyCreateRequest - Admin adds a currency to a country
type CountryCurrencyCreateRequest struct {
	CurrencyID uint `json:"currencyId" binding:"required"`
	IsPrimary  bool `json:"isPrimary"` // Default: false
}

// CountryCurrencyUpdateRequest - Admin updates a country-currency mapping
type CountryCurrencyUpdateRequest struct {
	IsPrimary *bool `json:"isPrimary" binding:"required"`
}

// CountryCurrencyBulkRequest - Admin adds multiple currencies to a country at once
type CountryCurrencyBulkRequest struct {
	Currencies []CountryCurrencyItem `json:"currencies" binding:"required,min=1,dive"`
}

// CountryCurrencyItem - Single item in bulk request
type CountryCurrencyItem struct {
	CurrencyID uint `json:"currencyId" binding:"required"`
	IsPrimary  bool `json:"isPrimary"`
}

// ========================================
// RESPONSE MODELS
// ========================================

// CountryCurrencyResponse - Country-Currency mapping response
type CountryCurrencyResponse struct {
	ID        uint             `json:"id"`
	Country   CountryResponse  `json:"country"`
	Currency  CurrencyResponse `json:"currency"`
	IsPrimary bool             `json:"isPrimary"`
	CreatedAt string           `json:"createdAt,omitempty"`
}

// CountryCurrencySimpleResponse - Simplified response for mapping operations
type CountryCurrencySimpleResponse struct {
	ID         uint   `json:"id"`
	CountryID  uint   `json:"countryId"`
	CurrencyID uint   `json:"currencyId"`
	IsPrimary  bool   `json:"isPrimary"`
	CreatedAt  string `json:"createdAt,omitempty"`
}

// ========================================
// LIST RESPONSE MODELS
// ========================================

// CountryCurrencyListResponse - List of currencies for a country
type CountryCurrencyListResponse struct {
	CountryID  uint                            `json:"countryId"`
	Currencies []CurrencyWithPrimaryResponse   `json:"currencies"`
}

// CurrencyWithPrimaryResponse - Currency with isPrimary flag
type CurrencyWithPrimaryResponse struct {
	CurrencyResponse
	MappingID uint `json:"mappingId"` // ID of the country_currency record
	IsPrimary bool `json:"isPrimary"`
}
