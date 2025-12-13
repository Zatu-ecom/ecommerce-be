package model

import "ecommerce-be/common"

// =========================================================================
// Endpoint: GET /api/inventory/locations/{locationId}/products
// Purpose: Shows products with inventory summary at a specific location
// =========================================================================

// ProductsAtLocationParams represents path and query parameters
type ProductsAtLocationParams struct {
	LocationID uint `uri:"locationId" binding:"required"`
	common.BaseListParams
}

// ProductsAtLocationFilter represents optional filter parameters
type ProductsAtLocationFilter struct {
	StockStatus  *StockStatus `form:"stockStatus"`  // Filter by stock status
	Search       *string      `form:"search"`       // Search by product name
	CategoryID   *uint        `form:"categoryId"`   // Filter by category
	LowStockOnly *bool        `form:"lowStockOnly"` // Filter to show only low stock products
}

// ProductInventorySummary represents inventory summary for a product at location
type ProductInventorySummary struct {
	ProductID         uint        `json:"productId"`
	ProductName       string      `json:"productName"`
	CategoryID        uint        `json:"categoryId"`
	BaseSKU           string      `json:"baseSku"`
	VariantCount      uint        `json:"variantCount"`
	TotalStock        int         `json:"totalStock"`        // Can be negative for backorder support
	TotalReserved     int         `json:"totalReserved"`
	TotalAvailable    int         `json:"totalAvailable"`
	LowStockVariants  uint        `json:"lowStockVariants"`  // Count of variants at low stock
	OutOfStockVariants uint       `json:"outOfStockVariants"` // Count of variants out of stock
	StockStatus       StockStatus `json:"stockStatus"`       // IN_STOCK, LOW_STOCK, OUT_OF_STOCK
}

// ProductAtLocationResponse represents full product inventory details
// Used in detailed view (API 3) or expanded view
type ProductAtLocationResponse struct {
	ProductInventorySummary
	Variants []InventoryResponse `json:"variants,omitempty"` // Reuse existing InventoryResponse for consistency
}

// ProductsAtLocationResponse represents the paginated response
type ProductsAtLocationResponse struct {
	LocationID   uint                      `json:"locationId"`
	LocationName string                    `json:"locationName"`
	Products     []ProductInventorySummary `json:"products"`
	Pagination   common.PaginationResponse `json:"pagination"`
}

// =========================================================================
// API 3: Get Variant Details with Inventory
// Endpoint: GET /api/inventory/locations/{locationId}/products/{productId}/variants
// Purpose: Shows all variants of a product with their inventory at a location
// =========================================================================

// VariantInventoryParams represents path parameters for API 3
type VariantInventoryParams struct {
	LocationID uint `uri:"locationId" binding:"required"`
	ProductID  uint `uri:"productId"  binding:"required"`
}

// VariantInventoryFilter represents query parameters for filtering variants (API 3)
// Embeds BaseListParams for sorting (pagination ignored for this API)
type VariantInventoryFilter struct {
	common.BaseListParams
	StockStatus string `form:"stockStatus" binding:"omitempty,oneof=all IN_STOCK LOW_STOCK OUT_OF_STOCK"`
	Search      string `form:"search"      binding:"omitempty,max=100"`
}

// SetDefaults sets default values for VariantInventoryFilter
func (f *VariantInventoryFilter) SetDefaults() {
	if f.StockStatus == "" {
		f.StockStatus = "all"
	}
	if f.SortBy == "" {
		f.SortBy = "sku"
	}
	if f.SortOrder == "" {
		f.SortOrder = "asc"
	}
}

// ValidSortFields returns valid sort fields for variant inventory
func (f *VariantInventoryFilter) ValidSortFields() []string {
	return []string{
		"sku",
		"variantName",
		"quantity",
		"reservedQuantity",
		"availableQuantity",
		"threshold",
	}
}

// IsValidSortBy checks if the sortBy field is valid
func (f *VariantInventoryFilter) IsValidSortBy() bool {
	for _, field := range f.ValidSortFields() {
		if f.SortBy == field {
			return true
		}
	}
	return false
}

// VariantWithInventory represents a variant with its inventory data at a location
type VariantWithInventory struct {
	VariantID   uint              `json:"variantId"`
	SKU         string            `json:"sku"`
	VariantName string            `json:"variantName,omitempty"` // Built from options if available
	Options     []VariantOption   `json:"options,omitempty"`     // e.g., [{name: "Color", value: "Black"}]
	Inventory   InventoryResponse `json:"inventory"`             // Reuse existing InventoryResponse
}

// VariantOption represents a single option value for a variant
type VariantOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// VariantInventoryResponse represents the response for API 3
type VariantInventoryResponse struct {
	ProductID    uint                    `json:"productId"`
	ProductName  string                  `json:"productName"`
	CategoryID   uint                    `json:"categoryId"`
	BaseSKU      string                  `json:"baseSku"`
	LocationID   uint                    `json:"locationId"`
	LocationName string                  `json:"locationName"`
	Variants     []VariantWithInventory  `json:"variants"`
	Summary      VariantInventorySummary `json:"summary"`
	Filters      VariantInventoryFilter  `json:"filters"`
}

// VariantInventorySummary represents aggregated inventory stats for all variants
type VariantInventorySummary struct {
	TotalVariants      uint        `json:"totalVariants"`
	TotalStock         int         `json:"totalStock"`
	TotalReserved      int         `json:"totalReserved"`
	TotalAvailable     int         `json:"totalAvailable"`
	LowStockVariants   uint        `json:"lowStockVariants"`
	OutOfStockVariants uint        `json:"outOfStockVariants"`
	StockStatus        StockStatus `json:"stockStatus"` // Overall status based on variants
}
