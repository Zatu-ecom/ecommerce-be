package model

import (
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
)

// ============================================================================
// List Users Filter
// ============================================================================

// ListUsersQueryParams represents raw query parameters (auto-bound by Gin)
type ListUsersQueryParams struct {
	common.BaseListParams
	IDs         string `form:"ids"`
	Emails      string `form:"emails"`
	Phones      string `form:"phones"`
	RoleIDs     string `form:"roleIds"`
	SellerIDs   string `form:"sellerIds"`
	Name        string `form:"name"`
	IsActive    *bool  `form:"isActive"`
	CreatedFrom string `form:"createdFrom"`
	CreatedTo   string `form:"createdTo"`
}

// ListUsersFilter contains parsed filter parameters for listing users
type ListUsersFilter struct {
	common.BaseListParams
	// Multiple ID/value support (parsed from comma-separated strings)
	IDs       []uint
	Emails    []string
	Phones    []string
	RoleIDs   []uint
	SellerIDs []uint // Admin only

	// Search (partial match)
	Name *string

	// Status
	IsActive *bool

	// Date range
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// ToFilter converts query params to ListUsersFilter with parsing
func (p *ListUsersQueryParams) ToFilter() ListUsersFilter {
	filter := ListUsersFilter{
		BaseListParams: p.BaseListParams,
		IsActive:       p.IsActive,
	}

	// Parse name
	if p.Name != "" {
		filter.Name = &p.Name
	}

	// Parse comma-separated values using helper
	filter.IDs = helper.ParseCommaSeparated[uint](p.IDs)
	filter.Emails = helper.ParseCommaSeparated[string](p.Emails)
	filter.Phones = helper.ParseCommaSeparated[string](p.Phones)
	filter.RoleIDs = helper.ParseCommaSeparated[uint](p.RoleIDs)
	filter.SellerIDs = helper.ParseCommaSeparated[uint](p.SellerIDs)

	// Parse dates
	if p.CreatedFrom != "" {
		if t, err := time.Parse(time.RFC3339, p.CreatedFrom); err == nil {
			filter.CreatedFrom = &t
		}
	}
	if p.CreatedTo != "" {
		if t, err := time.Parse(time.RFC3339, p.CreatedTo); err == nil {
			filter.CreatedTo = &t
		}
	}

	return filter
}

// ============================================================================
// List Users Response
// ============================================================================

// UserListResponse represents a single user in list response
type UserListResponse struct {
	ID        uint         `json:"id"`
	FirstName string       `json:"firstName"`
	LastName  string       `json:"lastName"`
	Name      string       `json:"name"` // Computed: FirstName + " " + LastName
	Email     string       `json:"email"`
	Phone     string       `json:"phone,omitempty"`
	Role      RoleResponse `json:"role"`
	SellerID  uint         `json:"sellerId,omitempty"`
	IsActive  bool         `json:"isActive"`
	CreatedAt string       `json:"createdAt"`
}

// RoleResponse represents role info in user response
type RoleResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// ListUsersResponse represents paginated list of users
type ListUsersResponse struct {
	Users      []UserListResponse       `json:"users"`
	Pagination common.PaginationResponse `json:"pagination"`
}

// ============================================================================
// Cross-Module Response (Minimal)
// ============================================================================

// UserBasicInfo is a minimal user response for cross-module use
type UserBasicInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"` // FirstName + " " + LastName
}
