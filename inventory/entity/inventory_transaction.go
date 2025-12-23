package entity

import (
	"fmt"
	"strings"

	"ecommerce-be/common/db"
)

// AdjustmentType represents the direction of manual adjustment
type AdjustmentType string

const (
	ADJ_ADD    AdjustmentType = "ADD"    // Increase stock
	ADJ_REMOVE AdjustmentType = "REMOVE" // Decrease stock
)

// ValidAdjustmentTypes returns all valid adjustment type values
func ValidAdjustmentTypes() []AdjustmentType {
	return []AdjustmentType{
		ADJ_ADD,
		ADJ_REMOVE,
	}
}

// String returns the string representation
func (at AdjustmentType) String() string {
	return string(at)
}

// IsValid checks if the adjustment type is valid
func (at AdjustmentType) IsValid() bool {
	for _, valid := range ValidAdjustmentTypes() {
		if at == valid {
			return true
		}
	}
	return false
}

// ParseAdjustmentType converts string to AdjustmentType with case-insensitive matching
func ParseAdjustmentType(s string) (AdjustmentType, error) {
	upperStr := strings.ToUpper(strings.TrimSpace(s))
	for _, valid := range ValidAdjustmentTypes() {
		if strings.ToUpper(valid.String()) == upperStr {
			return valid, nil
		}
	}
	return "", fmt.Errorf("invalid adjustment type: %s. Valid types: %v", s, ValidAdjustmentTypes())
}

type TransactionType string

const (
	// Stock Increases
	TXN_PURCHASE    TransactionType = "PURCHASE"    // Supplier delivery
	TXN_RETURN      TransactionType = "RETURN"      // Customer return
	TXN_TRANSFER_IN TransactionType = "TRANSFER_IN" // From another location

	// Stock Decreases
	TXN_SALE         TransactionType = "OUTBOUND"     // Order fulfilled (shipped)
	TXN_TRANSFER_OUT TransactionType = "TRANSFER_OUT" // To another location
	TXN_DAMAGE       TransactionType = "DAMAGE"       // Damaged/Lost items

	// Reservation Management (NEW)
	TXN_RESERVED TransactionType = "RESERVED" // Order placed (lock stock)
	TXN_RELEASED TransactionType = "RELEASED" // Order cancelled (unlock)

	// Manual Operations
	TXN_ADJUSTMENT TransactionType = "ADJUSTMENT" // Manual correction
	TXN_REFRESH    TransactionType = "REFRESH"    // Physical count
)

// ValidTransactionTypes returns all valid transaction type values
func ValidTransactionTypes() []TransactionType {
	return []TransactionType{
		TXN_PURCHASE,
		TXN_RETURN,
		TXN_TRANSFER_IN,
		TXN_SALE,
		TXN_TRANSFER_OUT,
		TXN_DAMAGE,
		TXN_RESERVED,
		TXN_RELEASED,
		TXN_ADJUSTMENT,
		TXN_REFRESH,
	}
}

// UpdatesReservedQuantity returns true if transaction type updates reserved quantity
func (tt TransactionType) UpdatesReservedQuantity() bool {
	return tt == TXN_RESERVED || tt == TXN_RELEASED
}

// UpdatesQuantity returns true if transaction type updates regular quantity
func (tt TransactionType) UpdatesQuantity() bool {
	return !tt.UpdatesReservedQuantity()
}

// RequiresReference returns true if transaction type requires a reference ID
func (tt TransactionType) RequiresReference() bool {
	// System operations always need reference (Order ID, PO Number, Transfer ID, etc.)
	return tt == TXN_RESERVED ||
		tt == TXN_RELEASED ||
		tt == TXN_SALE ||
		tt == TXN_PURCHASE ||
		tt == TXN_RETURN ||
		tt == TXN_TRANSFER_IN ||
		tt == TXN_TRANSFER_OUT
}

// ValidManualTransactionTypes returns transaction types allowed for adjust API
// Note: RESERVED and RELEASED will be handled by order service via this API
func ValidManualTransactionTypes() []TransactionType {
	return []TransactionType{
		// Manual adjustments
		TXN_ADJUSTMENT,
		TXN_DAMAGE,
		TXN_REFRESH,
		// Reservation management (called by order service)
		TXN_RESERVED,
		TXN_RELEASED,
		// Stock movements (called by internal services)
		TXN_PURCHASE,
		TXN_RETURN,
		TXN_SALE,
		TXN_TRANSFER_IN,
		TXN_TRANSFER_OUT,
	}
}

// String returns the string representation
func (tt TransactionType) String() string {
	return string(tt)
}

// IsValid checks if the transaction type is valid
func (tt TransactionType) IsValid() bool {
	for _, valid := range ValidTransactionTypes() {
		if tt == valid {
			return true
		}
	}
	return false
}

// IsManualType checks if the transaction type is allowed for manual adjustments
func (tt TransactionType) IsManualType() bool {
	for _, valid := range ValidManualTransactionTypes() {
		if tt == valid {
			return true
		}
	}
	return false
}

// RequiresDirection checks if this transaction type requires an adjustment direction
func (tt TransactionType) RequiresDirection() bool {
	return tt == TXN_ADJUSTMENT
}

// ParseTransactionType converts string to TransactionType with case-insensitive matching
func ParseTransactionType(s string) (TransactionType, error) {
	upperStr := strings.ToUpper(strings.TrimSpace(s))
	for _, valid := range ValidTransactionTypes() {
		if strings.ToUpper(valid.String()) == upperStr {
			return valid, nil
		}
	}
	return "", fmt.Errorf("invalid transaction type: %s", s)
}

type InventoryTransaction struct {
	db.BaseEntity
	InventoryID uint `json:"inventoryId" gorm:"column:inventory_id;not null;index"`

	// Type of movement
	Type TransactionType `json:"type" gorm:"column:type;type:varchar(20);not null;index"`

	// Quantity changed (positive for add, negative for remove)
	Quantity int `json:"quantity" gorm:"column:quantity;not null"`

	// Snapshots for audit trail
	BeforeQuantity int `json:"beforeQuantity" gorm:"column:before_quantity;not null"`
	AfterQuantity  int `json:"afterQuantity"  gorm:"column:after_quantity;not null"`

	// Audit: Who performed this transaction
	PerformedBy uint `json:"performedBy" gorm:"column:performed_by;not null;index"`

	// What caused this? (Order ID, PO Number, Return ID, etc.)
	ReferenceID   *string `json:"referenceId,omitempty"   gorm:"column:reference_id;index"`
	ReferenceType *string `json:"referenceType,omitempty" gorm:"column:reference_type;type:varchar(50)"` // e.g., "ORDER", "ADMIN_ADJUSTMENT"

	// Reason for the transaction (required for manual adjustments)
	Reason string `json:"reason" gorm:"column:reason;type:text"`

	// Additional notes
	Note *string `json:"note,omitempty" gorm:"column:note;type:text"`
}
