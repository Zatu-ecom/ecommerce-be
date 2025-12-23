package model

import (
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/inventory/entity"
)

// ============================================================================
// Transaction Creation
// ============================================================================

// CreateTransactionParams contains all parameters needed to create an inventory transaction
type CreateTransactionParams struct {
	// Inventory ID the transaction belongs to
	InventoryID uint

	// Type of transaction
	TransactionType entity.TransactionType

	// Quantity changed (positive for add, negative for remove)
	QuantityChange int

	// Snapshot quantities for audit trail
	BeforeQuantity int
	AfterQuantity  int

	// Who performed this transaction
	PerformedBy uint

	// Reference information (Order ID, PO Number, etc.)
	Reference     *string
	ReferenceType string

	// Reason and notes
	Reason string
	Note   *string
}

// ============================================================================
// List Transactions Filter
// ============================================================================

// ListTransactionsQueryParams represents raw query parameters (auto-bound by Gin)
type ListTransactionsQueryParams struct {
	common.BaseListParams
	InventoryIDs  string `form:"inventoryIds"`
	VariantIDs    string `form:"variantIds"`
	LocationIDs   string `form:"locationIds"`
	Types         string `form:"types"`
	ReferenceID   string `form:"referenceId"`
	ReferenceType string `form:"referenceType"`
	PerformedBy   *uint  `form:"performedBy"`
	CreatedFrom   string `form:"createdFrom"`
	CreatedTo     string `form:"createdTo"`
}

// ListTransactionsFilter contains parsed filter parameters for listing transactions
type ListTransactionsFilter struct {
	common.BaseListParams
	// Multiple ID support (parsed from comma-separated strings)
	InventoryIDs []uint
	VariantIDs   []uint
	LocationIDs  []uint
	Types        []entity.TransactionType

	// Single value filters
	ReferenceID   *string
	ReferenceType *string
	PerformedBy   *uint

	// Date range
	CreatedFrom *time.Time
	CreatedTo   *time.Time

	// Seller isolation (set by service layer)
	SellerID uint
}

// ToFilter converts query params to ListTransactionsFilter with parsing
func (p *ListTransactionsQueryParams) ToFilter() ListTransactionsFilter {
	filter := ListTransactionsFilter{
		BaseListParams: p.BaseListParams,
		PerformedBy:    p.PerformedBy,
	}

	// Parse reference filters
	if p.ReferenceID != "" {
		filter.ReferenceID = &p.ReferenceID
	}
	if p.ReferenceType != "" {
		filter.ReferenceType = &p.ReferenceType
	}

	// Parse comma-separated values
	filter.InventoryIDs = helper.ParseCommaSeparated[uint](p.InventoryIDs)
	filter.VariantIDs = helper.ParseCommaSeparated[uint](p.VariantIDs)
	filter.LocationIDs = helper.ParseCommaSeparated[uint](p.LocationIDs)

	// Parse transaction types
	typeStrings := helper.ParseCommaSeparated[string](p.Types)
	for _, ts := range typeStrings {
		if tt, err := entity.ParseTransactionType(ts); err == nil {
			filter.Types = append(filter.Types, tt)
		}
	}

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
// List Transactions Response
// ============================================================================

// TransactionResponse represents a single transaction in list response
type TransactionResponse struct {
	ID              uint                    `json:"id"`
	InventoryID     uint                    `json:"inventoryId"`
	VariantID       uint                    `json:"variantId"`
	LocationID      uint                    `json:"locationId"`
	LocationName    string                  `json:"locationName"`
	Type            entity.TransactionType  `json:"type"`
	Quantity        int                     `json:"quantity"`
	BeforeQuantity  int                     `json:"beforeQuantity"`
	AfterQuantity   int                     `json:"afterQuantity"`
	PerformedBy     uint                    `json:"performedBy"`
	PerformedByName string                  `json:"performedByName"`
	ReferenceID     *string                 `json:"referenceId,omitempty"`
	ReferenceType   *string                 `json:"referenceType,omitempty"`
	Reason          string                  `json:"reason"`
	Note            *string                 `json:"note,omitempty"`
	CreatedAt       string                  `json:"createdAt"`
}

// ListTransactionsResponse contains the paginated list of transactions
type ListTransactionsResponse struct {
	Transactions []TransactionResponse     `json:"transactions"`
	Pagination   common.PaginationResponse `json:"pagination"`
}
