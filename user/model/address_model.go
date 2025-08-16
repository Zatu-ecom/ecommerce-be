package model

// AddressRequest represents the request body for adding/updating an address
type AddressRequest struct {
	Street    string `json:"street" binding:"required"`
	City      string `json:"city" binding:"required"`
	State     string `json:"state" binding:"required"`
	ZipCode   string `json:"zipCode" binding:"required"`
	Country   string `json:"country" binding:"required"`
	IsDefault bool   `json:"isDefault"`
}

// AddressResponse represents the address data returned in API responses
type AddressResponse struct {
	ID        uint   `json:"id"`
	Street    string `json:"street"`
	City      string `json:"city"`
	State     string `json:"state"`
	ZipCode   string `json:"zipCode"`
	Country   string `json:"country"`
	IsDefault bool   `json:"isDefault"`
}
