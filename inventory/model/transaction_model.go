package model

import "ecommerce-be/inventory/entity"

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

// ListTransactionsFilter contains filter parameters for listing transactions
// This will be expanded when ListTransactions is fully implemented
type ListTransactionsFilter struct {
	InventoryID   *uint
	ReferenceID   *string
	ReferenceType *string
	Type          *entity.TransactionType
	PerformedBy   *uint

	// Pagination
	Page     int
	PageSize int
}

// ListTransactionsResponse contains the paginated list of transactions
type ListTransactionsResponse struct {
	Transactions []entity.InventoryTransaction `json:"transactions"`
	Total        int64                         `json:"total"`
	Page         int                           `json:"page"`
	PageSize     int                           `json:"pageSize"`
}
