package model

// ========================================
// BASE MODELS (for inheritance)
// ========================================

// CountryBase contains common fields used in create/update/response
// Changes here will reflect in all country-related models
type CountryBase struct {
	Code       string `json:"code"                 binding:"required,len=2"`         // ISO 3166-1 alpha-2
	CodeAlpha3 string `json:"codeAlpha3,omitempty" binding:"omitempty,len=3"`        // ISO 3166-1 alpha-3
	Name       string `json:"name"                 binding:"required,min=2,max=100"` // Country name
	NativeName string `json:"nativeName,omitempty" binding:"omitempty,max=100"`      // Native name
	PhoneCode  string `json:"phoneCode,omitempty"  binding:"omitempty,max=10"`       // Phone code
	Region     string `json:"region,omitempty"     binding:"omitempty,max=50"`       // Region/Continent
	FlagEmoji  string `json:"flagEmoji,omitempty"  binding:"omitempty,max=10"`       // Flag emoji
}

// ========================================
// REQUEST MODELS
// ========================================

// CountryCreateRequest - Admin creates a new country
type CountryCreateRequest struct {
	CountryBase
	IsActive bool `json:"isActive"` // Default: true (handled in service)
}

// CountryUpdateRequest - Admin updates a country (all fields optional)
type CountryUpdateRequest struct {
	Code       *string `json:"code"       binding:"omitempty,len=2"`
	CodeAlpha3 *string `json:"codeAlpha3" binding:"omitempty,len=3"`
	Name       *string `json:"name"       binding:"omitempty,min=2,max=100"`
	NativeName *string `json:"nativeName" binding:"omitempty,max=100"`
	PhoneCode  *string `json:"phoneCode"  binding:"omitempty,max=10"`
	Region     *string `json:"region"     binding:"omitempty,max=50"`
	FlagEmoji  *string `json:"flagEmoji"  binding:"omitempty,max=10"`
	IsActive   *bool   `json:"isActive"`
}

// ========================================
// RESPONSE MODELS
// ========================================

// CountryResponse - Basic country response (used in lists)
// Embeds CountryBase for inheritance - changes in base reflect here
type CountryResponse struct {
	ID uint `json:"id"`
	CountryBase
	IsActive bool `json:"isActive"`
}

// CountryDetailResponse - Country with currencies (for detail view)
type CountryDetailResponse struct {
	CountryResponse
	Currencies []CurrencyInCountryResponse `json:"currencies,omitempty"`
	CreatedAt  string                      `json:"createdAt,omitempty"`
	UpdatedAt  string                      `json:"updatedAt,omitempty"`
}

// CurrencyInCountryResponse - Currency info when embedded in country response
type CurrencyInCountryResponse struct {
	ID        uint   `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	IsPrimary bool   `json:"isPrimary"`
}

// ========================================
// LIST RESPONSE MODELS
// ========================================

// CountryListResponse - Paginated list of countries
type CountryListResponse struct {
	Countries  []CountryResponse  `json:"countries"`
	Pagination PaginationResponse `json:"pagination,omitempty"`
}

// CountryListWithCurrenciesResponse - Countries with their currencies
type CountryListWithCurrenciesResponse struct {
	Countries  []CountryDetailResponse `json:"countries"`
	Pagination PaginationResponse      `json:"pagination,omitempty"`
}

// ========================================
// QUERY MODELS
// ========================================

// CountryQueryParams - Query parameters for listing countries
type CountryQueryParams struct {
	Search   string `form:"search"`   // Search by name or code (case-insensitive)
	Region   string `form:"region"`   // Filter by region
	IsActive *bool  `form:"isActive"` // Filter by active status (admin only)
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
}
