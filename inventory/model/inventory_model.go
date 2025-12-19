package model

import (
	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/inventory/entity"
)

// ManageInventoryRequest represents the request body for managing inventory (create/update)
type ManageInventoryRequest struct {
	VariantID       uint                   `json:"variantId"       binding:"required"`
	LocationID      uint                   `json:"locationId"      binding:"required"`
	Quantity        int                    `json:"quantity"        binding:"required,gt=0"`
	TransactionType entity.TransactionType `json:"transactionType" binding:"required"` // Validated by validator

	// Optional: Only required for ADJUSTMENT type
	Direction *entity.AdjustmentType `json:"direction" binding:"omitempty"` // Validated by validator

	// Optional: Update threshold (for backorder limits)
	Threshold *int `json:"threshold" binding:"omitempty"`

	// Reference ID for tracking source (Order ID, PO Number, Transfer ID, etc.)
	// Required for: RESERVED, RELEASED, SALE, PURCHASE, RETURN, TRANSFER_IN, TRANSFER_OUT
	// Optional for: ADJUSTMENT, DAMAGE, REFRESH
	Reference *string `json:"reference" binding:"omitempty,min=1,max=100"`

	Reason string  `json:"reason" binding:"required,min=5,max=500"`
	Note   *string `json:"note"   binding:"omitempty,max=1000"`
}

// InventoryResponse represents inventory data in API response
type InventoryResponse struct {
	ID                uint        `json:"id"`
	VariantID         uint        `json:"variantId"`
	LocationID        uint        `json:"locationId"`
	Quantity          int         `json:"quantity"`
	ReservedQuantity  int         `json:"reservedQuantity"`
	Threshold         int         `json:"threshold"`
	AvailableQuantity int         `json:"availableQuantity"`     // Computed: Quantity - ReservedQuantity
	StockStatus       StockStatus `json:"stockStatus"`           // IN_STOCK, LOW_STOCK, OUT_OF_STOCK
	BinLocation       string      `json:"binLocation,omitempty"` // Specific bin/shelf location within warehouse
}

// ManageInventoryResponse represents the response after managing inventory
type ManageInventoryResponse struct {
	InventoryID       uint `json:"inventoryId"`
	PreviousQuantity  int  `json:"previousQuantity"`
	NewQuantity       int  `json:"newQuantity"`
	QuantityChanged   int  `json:"quantityChanged"`
	AvailableQuantity int  `json:"availableQuantity"`
	Threshold         int  `json:"threshold"`
	TransactionID     uint `json:"transactionId"`
}

// BulkManageInventoryRequest represents bulk inventory management request
type BulkManageInventoryRequest struct {
	Items []ManageInventoryRequest `json:"items" binding:"required,min=1,max=100,dive"`
}

// BulkManageInventoryResponse represents bulk inventory management response
type BulkManageInventoryResponse struct {
	SuccessCount int                       `json:"successCount"`
	FailureCount int                       `json:"failureCount"`
	Results      []BulkInventoryItemResult `json:"results"`
}

// BulkInventoryItemResult represents individual item result in bulk operation
type BulkInventoryItemResult struct {
	VariantID  uint                     `json:"variantId"`
	LocationID uint                     `json:"locationId"`
	Success    bool                     `json:"success"`
	Response   *ManageInventoryResponse `json:"response,omitempty"`
	Error      string                   `json:"error,omitempty"`
}

// InventoryDetailResponse represents detailed inventory with location info
type InventoryDetailResponse struct {
	InventoryResponse
	LocationName string `json:"locationName"`
	LocationType string `json:"locationType"`
}

// Get Inventories Filters Params //
type GetInventoriesBase struct {
	common.BaseListParams
	MinQuantity *int `form:"minQuantity" binding:"omitempty"`
	MaxQuantity *int `form:"maxQuantity" binding:"omitempty"`
}

type GetInventoriesFilter struct {
	GetInventoriesBase
	IDs         []uint
	VariantIDs  []uint
	LocationIDs []uint
}

type GetInventoriesParam struct {
	GetInventoriesBase
	VariantIDs  *string `form:"variantIds"  binding:"omitempty,dive,gt=0"`
	LocationIDs *string `form:"locationIds" binding:"omitempty,dive,gt=0"`
	IDs         *string `form:"ids"         binding:"omitempty,dive,gt=0"`
}

func (f *GetInventoriesParam) ToFilter() GetInventoriesFilter {
	filter := GetInventoriesFilter{
		GetInventoriesBase: f.GetInventoriesBase,
	}

	if f.VariantIDs != nil {
		filter.VariantIDs = helper.ParseCommaSeparatedPtr[uint](f.VariantIDs)
	}

	if f.LocationIDs != nil {
		filter.LocationIDs = helper.ParseCommaSeparatedPtr[uint](f.LocationIDs)
	}

	if f.IDs != nil {
		filter.IDs = helper.ParseCommaSeparatedPtr[uint](f.IDs)
	}

	return filter
}

type InventoryResponseWithPagination struct {
	common.PaginationResponse
	Inventories []InventoryResponse `json:"inventories"`
}
