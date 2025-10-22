package model

import (
	"ecommerce-be/common"
)

// PaginationResponse represents pagination information in API responses
// Using common.PaginationResponse instead of local definition
type PaginationResponse = common.PaginationResponse

// SearchQuery represents search parameters
type SearchQuery struct {
	Query     string                 `json:"query"`
	Filters   map[string]interface{} `json:"filters"`
	Page      int                    `json:"page"`
	Limit     int                    `json:"limit"`
	SortBy    string                 `json:"sortBy"`
	SortOrder string                 `json:"sortOrder"`
}

// FilterOption represents a generic filter option
type FilterOption struct {
	ID    interface{} `json:"id"`
	Value string      `json:"value"`
	Count int         `json:"count"`
}

// AttributeFilter represents an attribute filter option
type AttributeFilter struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	AllowedValues []string `json:"allowedValues"`
	ProductCount  uint     `json:"productCount"`
}

// ProductFilters represents available filters for product search
type ProductFilters struct {
	Categories   []CategoryFilter      `json:"categories"`
	Brands       []BrandFilter         `json:"brands"`
	Attributes   []AttributeFilter     `json:"attributes"`
	PriceRange   *PriceRangeFilter     `json:"priceRange,omitempty"`   // Price range from variants
	VariantTypes []VariantTypeFilter   `json:"variantTypes,omitempty"` // Available variant options (Color, Size, etc.)
	StockStatus  *StockStatusFilter    `json:"stockStatus,omitempty"`  // Stock availability
}

// CategoryFilter represents category filter option
type CategoryFilter struct {
	ID           uint             `json:"id"`
	Name         string           `json:"name"`
	ProductCount uint             `json:"productCount"`
	Children     []CategoryFilter `json:"children"`
}

// BrandFilter represents brand filter option
type BrandFilter struct {
	Brand        string `json:"brand"`
	ProductCount uint   `json:"productCount"`
}

// PriceRangeFilter represents price range filter option
type PriceRangeFilter struct {
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	ProductCount uint    `json:"productCount"`
}

// VariantTypeFilter represents variant option types (e.g., Color, Size)
type VariantTypeFilter struct {
	Name         string                `json:"name"`        // Option name (e.g., "Color", "Size")
	DisplayName  string                `json:"displayName"` // Display name
	Values       []VariantOptionFilter `json:"values"`      // Available values for this option
	ProductCount uint                  `json:"productCount"`
}

// VariantOptionFilter represents individual variant option values
type VariantOptionFilter struct {
	Value        string `json:"value"`        // Option value (e.g., "Red", "Large")
	DisplayName  string `json:"displayName"`  // Display name
	ColorCode    string `json:"colorCode,omitempty"` // For color options
	ProductCount uint   `json:"productCount"` // Number of products with this option
}

// StockStatusFilter represents stock availability filter
type StockStatusFilter struct {
	InStock      uint `json:"inStock"`      // Products with stock
	OutOfStock   uint `json:"outOfStock"`   // Products without stock
	TotalProducts uint `json:"totalProducts"`
}
