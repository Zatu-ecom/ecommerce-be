package model

// AddressRequest represents the request body for adding a new address
type AddressRequest struct {
	Street    string `json:"street"    binding:"required"`
	City      string `json:"city"      binding:"required"`
	State     string `json:"state"     binding:"required"`
	ZipCode   string `json:"zipCode"   binding:"required"`
	CountryID uint   `json:"countryId" binding:"required"`
	IsDefault bool   `json:"isDefault"`
}

// AddressUpdateRequest represents the request body for updating an existing address
// Uses pointers to distinguish between null (don't update) and empty (set to empty)
type AddressUpdateRequest struct {
	Street    *string `json:"street"    binding:"omitempty,min=1"`
	City      *string `json:"city"      binding:"omitempty,min=1"`
	State     *string `json:"state"     binding:"omitempty,min=1"`
	ZipCode   *string `json:"zipCode"   binding:"omitempty,min=1"`
	CountryID *uint   `json:"countryId"`
	IsDefault *bool   `json:"isDefault"`
}

// AddressResponse represents the address data returned in API responses
type AddressResponse struct {
	ID        uint             `json:"id,omitempty"`
	Street    string           `json:"street,omitempty"`
	City      string           `json:"city,omitempty"`
	State     string           `json:"state,omitempty"`
	ZipCode   string           `json:"zipCode,omitempty"`
	CountryID uint             `json:"countryId"`
	Country   *CountryResponse `json:"country,omitempty"` // Expanded country info
	IsDefault bool             `json:"isDefault,omitempty"`
}
