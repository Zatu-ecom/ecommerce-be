package model

import "ecommerce-be/user/entity"

// AddressRequest represents the request body for adding a new address
type AddressRequest struct {
	Type      entity.AddressType `json:"type"      binding:"omitempty"`
	Address   string             `json:"address"   binding:"required,min=5,max=500"`
	Landmark  string             `json:"landmark"  binding:"omitempty,max=255"`
	City      string             `json:"city"      binding:"required,min=2,max=100"`
	State     string             `json:"state"     binding:"required,min=2,max=100"`
	ZipCode   string             `json:"zipCode"   binding:"required,max=20"`
	CountryID uint               `json:"countryId" binding:"required"`
	Latitude  *float64           `json:"latitude"  binding:"omitempty"`
	Longitude *float64           `json:"longitude" binding:"omitempty"`
	IsDefault bool               `json:"isDefault"`
}

// AddressUpdateRequest represents the request body for updating an existing address
// Uses pointers to distinguish between null (don't update) and empty (set to empty)
type AddressUpdateRequest struct {
	Type      *entity.AddressType `json:"type"      binding:"omitempty"`
	Address   *string             `json:"address"   binding:"omitempty,min=5,max=500"`
	Landmark  *string             `json:"landmark"  binding:"omitempty,max=255"`
	City      *string             `json:"city"      binding:"omitempty,min=2,max=100"`
	State     *string             `json:"state"     binding:"omitempty,min=2,max=100"`
	ZipCode   *string             `json:"zipCode"   binding:"omitempty,max=20"`
	CountryID *uint               `json:"countryId"`
	Latitude  *float64            `json:"latitude"`
	Longitude *float64            `json:"longitude"`
	IsDefault *bool               `json:"isDefault"`
}

// AddressResponse represents the address data returned in API responses
type AddressResponse struct {
	ID        uint               `json:"id,omitempty"`
	Type      entity.AddressType `json:"type"`
	Address   string             `json:"address,omitempty"`
	Landmark  string             `json:"landmark,omitempty"`
	City      string             `json:"city,omitempty"`
	State     string             `json:"state,omitempty"`
	ZipCode   string             `json:"zipCode,omitempty"`
	CountryID uint               `json:"countryId"`
	Country   *CountryResponse   `json:"country,omitempty"` // Expanded country info
	Latitude  *float64           `json:"latitude,omitempty"`
	Longitude *float64           `json:"longitude,omitempty"`
	IsDefault bool               `json:"isDefault,omitempty"`
}
