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
	Street  string `json:"street"  binding:"required,min=5"`
	City    string `json:"city"    binding:"required,min=2"`
	State   string `json:"state"   binding:"required,min=2"`
	ZipCode string `json:"zipCode" binding:"required"`
	Country string `json:"country" binding:"required,min=2"`
}

// AddressRequest represents the address information in a request
type AddressUpdateRequest struct {
	Street  *string `json:"street"  binding:"omitempty,min=5"`
	City    *string `json:"city"    binding:"omitempty,min=2"`
	State   *string `json:"state"   binding:"omitempty,min=2"`
	ZipCode *string `json:"zipCode" binding:"omitempty"`
	Country *string `json:"country" binding:"omitempty,min=2"`
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
	ID      uint   `json:"id,omitempty"`
	Street  string `json:"street,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	ZipCode string `json:"zipCode,omitempty"`
	Country string `json:"country,omitempty"`
}

// LocationsResponse represents the paginated response for getting all locations
type LocationsResponse struct {
	Locations  []LocationResponse       `json:"locations"`
	Pagination common.PaginationResponse `json:"pagination"`
}
