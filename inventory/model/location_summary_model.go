package model

import "ecommerce-be/common"

// StockStatus represents the inventory status at location/product/variant level
type StockStatus string

const (
	StockStatusInStock    StockStatus = "IN_STOCK"     // Healthy stock levels
	StockStatusLowStock   StockStatus = "LOW_STOCK"    // At least one item below threshold
	StockStatusOutOfStock StockStatus = "OUT_OF_STOCK" // At least one item has zero stock
)

type InventorySummary struct {
	ProductCount      uint        `json:"productCount"`
	VariantCount      uint        `json:"variantCount"`
	TotalStock        uint        `json:"totalStock"`
	TotalReserved     uint        `json:"totalReserved"`
	TotalAvailable    uint        `json:"totalAvailable"`
	LowStockCount     uint        `json:"lowStockCount"`
	OutOfStockCount   uint        `json:"outOfStockCount"`
	AverageStockValue float64     `json:"averageStockValue"`
	StockStatus       StockStatus `json:"stockStatus"`
}

type LocationSummaryResponse struct {
	LocationResponse
	InventorySummary
}

// LocationsSummaryResponse represents the paginated response for location summaries
type LocationsSummaryResponse struct {
	Locations  []LocationSummaryResponse `json:"locations"`
	Pagination common.PaginationResponse `json:"pagination"`
}
