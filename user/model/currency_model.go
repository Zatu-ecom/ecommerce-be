package model

// ========================================
// BASE MODELS (for inheritance)
// ========================================

// CurrencyBase contains common fields used in create/update/response
// Changes here will reflect in all currency-related models
type CurrencyBase struct {
	Code          string `json:"code"                   binding:"required,len=3"`         // ISO 4217 code
	Name          string `json:"name"                   binding:"required,min=2,max=100"` // Currency name
	Symbol        string `json:"symbol"                 binding:"required,max=10"`        // Symbol ($, €, ₹)
	SymbolNative  string `json:"symbolNative,omitempty" binding:"omitempty,max=10"`       // Native symbol
	DecimalDigits int    `json:"decimalDigits"          binding:"omitempty,gte=0,lte=4"`  // Decimal places (0-4)
}

// ========================================
// REQUEST MODELS
// ========================================

// CurrencyCreateRequest - Admin creates a new currency
type CurrencyCreateRequest struct {
	CurrencyBase
	IsActive bool `json:"isActive"` // Default: true (handled in service)
}

// CurrencyUpdateRequest - Admin updates a currency (all fields optional)
type CurrencyUpdateRequest struct {
	Code          *string `json:"code"          binding:"omitempty,len=3"`
	Name          *string `json:"name"          binding:"omitempty,min=2,max=100"`
	Symbol        *string `json:"symbol"        binding:"omitempty,max=10"`
	SymbolNative  *string `json:"symbolNative"  binding:"omitempty,max=10"`
	DecimalDigits *int    `json:"decimalDigits" binding:"omitempty,gte=0,lte=4"`
	IsActive      *bool   `json:"isActive"`
}

// ========================================
// RESPONSE MODELS
// ========================================

// CurrencyResponse - Basic currency response (used in lists)
// Embeds CurrencyBase for inheritance - changes in base reflect here
type CurrencyResponse struct {
	CurrencyBase
	ID       uint `json:"id"`
	IsActive bool `json:"isActive"`
}

// CurrencyDetailResponse - Currency with countries (for detail view)
type CurrencyDetailResponse struct {
	CurrencyResponse
	Countries []CountryInCurrencyResponse `json:"countries,omitempty"`
	CreatedAt string                      `json:"createdAt,omitempty"`
	UpdatedAt string                      `json:"updatedAt,omitempty"`
}

// CountryInCurrencyResponse - Country info when embedded in currency response
type CountryInCurrencyResponse struct {
	ID        uint   `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	FlagEmoji string `json:"flagEmoji,omitempty"`
	IsPrimary bool   `json:"isPrimary"`
}

// ========================================
// LIST RESPONSE MODELS
// ========================================

// CurrencyListResponse - Paginated list of currencies
type CurrencyListResponse struct {
	Currencies []CurrencyResponse `json:"currencies"`
	Pagination PaginationResponse `json:"pagination,omitempty"`
}

// CurrencyListWithCountriesResponse - Currencies with their countries
type CurrencyListWithCountriesResponse struct {
	Currencies []CurrencyDetailResponse `json:"currencies"`
	Pagination PaginationResponse       `json:"pagination,omitempty"`
}

// ========================================
// QUERY MODELS
// ========================================

// CurrencyQueryParams - Query parameters for listing currencies
type CurrencyQueryParams struct {
	IsActive *bool `form:"isActive"` // Filter by active status (admin only)
	Page     int   `form:"page"`
	Limit    int   `form:"limit"`
}
