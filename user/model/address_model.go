package model

// AddressRequest represents the request body for adding a new address
type AddressRequest struct {
	Street    string `json:"street"    binding:"required"`
	City      string `json:"city"      binding:"required"`
	State     string `json:"state"     binding:"required"`
	ZipCode   string `json:"zipCode"   binding:"required"`
	Country   string `json:"country"   binding:"required"`
	IsDefault bool   `json:"isDefault"`
}

// AddressUpdateRequest represents the request body for updating an existing address
// Uses pointers to distinguish between null (don't update) and empty (set to empty)
type AddressUpdateRequest struct {
	Street    *string `json:"street"    binding:"omitempty,min=1"`
	City      *string `json:"city"      binding:"omitempty,min=1"`
	State     *string `json:"state"     binding:"omitempty,min=1"`
	ZipCode   *string `json:"zipCode"   binding:"omitempty,min=1"`
	Country   *string `json:"country"   binding:"omitempty,min=1"`
	IsDefault *bool   `json:"isDefault"`
}

// AddressResponse represents the address data returned in API responses
type AddressResponse struct {
	ID        uint   `json:"id,omitempty"`
	Street    string `json:"street,omitempty"`
	City      string `json:"city,omitempty"`
	State     string `json:"state,omitempty"`
	ZipCode   string `json:"zipCode,omitempty"`
	Country   string `json:"country,omitempty"`
	IsDefault bool   `json:"isDefault,omitempty"`
}
