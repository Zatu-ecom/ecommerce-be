package entity

import (
	"ecommerce-be/common/db"
	"time"
)

type TransferStatus string

const (
	TrfPending   TransferStatus = "PENDING"   // Draft
	TrfShipped   TransferStatus = "SHIPPED"   // Left source location (In-Transit)
	TrfReceived  TransferStatus = "RECEIVED"  // Arrived at destination
	TrfCancelled TransferStatus = "CANCELLED"
)

type StockTransfer struct {
	db.BaseEntity
	ReferenceNumber string `json:"referenceNumber" gorm:"column:reference_number;uniqueIndex"`

	FromLocationID uint `json:"fromLocationId" gorm:"column:from_location_id;not null"`
	ToLocationID   uint `json:"toLocationId"   gorm:"column:to_location_id;not null"`

	Status TransferStatus `json:"status" gorm:"column:status;default:'PENDING'"`

	// Audit
	RequestedBy uint       `json:"requestedBy" gorm:"column:requested_by"`
	ShippedAt   *time.Time `json:"shippedAt"   gorm:"column:shipped_at"`
	ReceivedAt  *time.Time `json:"receivedAt"  gorm:"column:received_at"`

	Items []StockTransferItem `json:"items" gorm:"foreignKey:StockTransferID"`
}

type StockTransferItem struct {
	db.BaseEntity
	StockTransferID uint `json:"stockTransferId" gorm:"column:stock_transfer_id;not null"`
	VariantID       uint `json:"variantId"       gorm:"column:variant_id;not null"`
	Quantity        int  `json:"quantity"        gorm:"column:quantity;not null"`
}
