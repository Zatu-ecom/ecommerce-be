package model

import "ecommerce-be/common"

// StockStatus represents the inventory status at location/product/variant level
type StockStatus string

const (
	StockStatusInStock    StockStatus = "IN_STOCK"     // Healthy stock levels
	StockStatusLowStock   StockStatus = "LOW_STOCK"    // At least one item below threshold
	StockStatusOutOfStock StockStatus = "OUT_OF_STOCK" // At least one item has zero stock
)

// CalculateStockStatus determines stock status for a single inventory record
func CalculateStockStatus(quantity, threshold int) StockStatus {
	available := quantity // Available = Quantity - Reserved (but for status, we use raw quantity)
	if available <= 0 {
		return StockStatusOutOfStock
	}
	if quantity <= threshold {
		return StockStatusLowStock
	}
	return StockStatusInStock
}

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
