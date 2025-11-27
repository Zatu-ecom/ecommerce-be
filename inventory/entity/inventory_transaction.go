package entity

import "ecommerce-be/common/db"

type TransactionType string

const (
	TxnInbound    TransactionType = "INBOUND"    // Purchase Order / Restock
	TxnOutbound   TransactionType = "OUTBOUND"   // Order Shipped
	TxnReserved   TransactionType = "RESERVED"   // Order Placed (locks stock)
	TxnReleased   TransactionType = "RELEASED"   // Order Cancelled (unlocks stock)
	TxnAdjustment TransactionType = "ADJUSTMENT" // Stock Count / Damaged / Lost
	TxnReturn     TransactionType = "RETURN"     // Customer Return
)

type InventoryTransaction struct {
	db.BaseEntity
	InventoryID uint `json:"inventoryId" gorm:"column:inventory_id;not null;index"`

	// Type of movement
	Type TransactionType `json:"type" gorm:"column:type;type:varchar(20);not null"`

	// Quantity changed (positive for add, negative for remove)
	Quantity int `json:"quantity" gorm:"column:quantity;not null"`

	// Snapshots for audit safety
	BeforeQuantity int `json:"beforeQuantity" gorm:"column:before_quantity;not null"`
	AfterQuantity  int `json:"afterQuantity"  gorm:"column:after_quantity;not null"`

	// What caused this? (Order ID, PO Number, Return ID)
	ReferenceID   string `json:"referenceId"   gorm:"column:reference_id;index"`
	ReferenceType string `json:"referenceType" gorm:"column:reference_type;type:varchar(50)"` // e.g., "ORDER", "ADMIN_ADJUSTMENT"

	Note string `json:"note" gorm:"column:note;type:text"`
}
