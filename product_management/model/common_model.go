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
	DataType      string   `json:"dataType"`
	AllowedValues []string `json:"allowedValues"`
	Count         int      `json:"count"`
}

// ProductFilters represents available filters for product search
type ProductFilters struct {
	Categories  []CategoryFilter   `json:"categories"`
	Brands      []BrandFilter      `json:"brands"`
	PriceRanges []PriceRangeFilter `json:"priceRanges"`
	Attributes  []AttributeFilter  `json:"attributes"`
}

// CategoryFilter represents category filter option
type CategoryFilter struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	ProductCount int    `json:"productCount"`
}

// BrandFilter represents brand filter option
type BrandFilter struct {
	Brand        string `json:"brand"`
	ProductCount int    `json:"productCount"`
}

// PriceRangeFilter represents price range filter option
type PriceRangeFilter struct {
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	ProductCount int     `json:"productCount"`
}
