package model

import (
	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/inventory/entity"
)

// ===========================================================================
// Request Models
// ===========================================================================

// LocationCreateRequest represents the request body for creating a location
type LocationCreateRequest struct {
	Name     string              `json:"name"     binding:"required,min=3,max=255"`
	Type     entity.LocationType `json:"type"     binding:"required"` // Validated by validator.ValidateLocationType
	Priority int                 `json:"priority" binding:"omitempty,gte=0"`
	Address  AddressRequest      `json:"address"  binding:"required"`
}

// LocationUpdateRequest represents the request body for updating a location
type LocationUpdateRequest struct {
	Name     *string               `json:"name"     binding:"omitempty,min=3,max=255"`
	Priority *int                  `json:"priority" binding:"omitempty,gte=0"`
	Address  *AddressUpdateRequest `json:"address"  binding:"omitempty"`
	Type     *entity.LocationType  `json:"type"     binding:"omitempty"` // Validated by validator.ValidateLocationType
	IsActive *bool                 `json:"isActive" binding:"omitempty"`
}

// AddressRequest represents the address information in a request
type AddressRequest struct {
	Address   string   `json:"address"   binding:"required,min=5,max=500"`
	Landmark  string   `json:"landmark"  binding:"omitempty,max=255"`
	City      string   `json:"city"      binding:"required,min=2,max=100"`
	State     string   `json:"state"     binding:"required,min=2,max=100"`
	ZipCode   string   `json:"zipCode"   binding:"required,max=20"`
	CountryID uint     `json:"countryId" binding:"required"`
	Latitude  *float64 `json:"latitude"  binding:"omitempty"`
	Longitude *float64 `json:"longitude" binding:"omitempty"`
}

// AddressUpdateRequest represents the address information in an update request
type AddressUpdateRequest struct {
	Address   *string  `json:"address"   binding:"omitempty,min=5,max=500"`
	Landmark  *string  `json:"landmark"  binding:"omitempty,max=255"`
	City      *string  `json:"city"      binding:"omitempty,min=2,max=100"`
	State     *string  `json:"state"     binding:"omitempty,min=2,max=100"`
	ZipCode   *string  `json:"zipCode"   binding:"omitempty,max=20"`
	CountryID *uint    `json:"countryId"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// Locations filter API paramters

type LocationsFilter struct {
	LocationsFilterBase
	LocationTypes []string
}

type LocationsParam struct {
	LocationsFilterBase
	LocationTypes *string `form:"locationTypes"`
}

type LocationsFilterBase struct {
	common.BaseListParams
	IsActive *bool `form:"isActive"`
}

func (p *LocationsParam) ToLocationSummaryFilter() LocationsFilter {
	return LocationsFilter{
		LocationsFilterBase: p.LocationsFilterBase,
		LocationTypes:       helper.ParseCommaSeparatedPtr[string](p.LocationTypes),
	}
}

/// ===========================================================================
// Response Models
// ============================================================================

// LocationResponse represents the location data returned in API responses
type LocationResponse struct {
	ID       uint                `json:"id"`
	Name     string              `json:"name"`
	Type     entity.LocationType `json:"type"`
	IsActive bool                `json:"isActive"`
	Priority int                 `json:"priority"`
	Address  AddressResponse     `json:"address,omitempty"`
}

// AddressResponse represents the address data in a response
type AddressResponse struct {
	ID        uint     `json:"id,omitempty"`
	Address   string   `json:"address,omitempty"`
	Landmark  string   `json:"landmark,omitempty"`
	City      string   `json:"city,omitempty"`
	State     string   `json:"state,omitempty"`
	ZipCode   string   `json:"zipCode,omitempty"`
	CountryID uint     `json:"countryId"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// LocationsResponse represents the paginated response for getting all locations
type LocationsResponse struct {
	Locations  []LocationResponse        `json:"locations"`
	Pagination common.PaginationResponse `json:"pagination"`
}
