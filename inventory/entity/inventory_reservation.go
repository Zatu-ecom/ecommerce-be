package entity

import (
	"ecommerce-be/common/db"
	"time"
)

type ReservationStatus string

const (
	ResPending   ReservationStatus = "PENDING"
	ResConfirmed ReservationStatus = "CONFIRMED" // Converted to order
	ResExpired   ReservationStatus = "EXPIRED"   // Released back to stock
)

type InventoryReservation struct {
	db.BaseEntity
	InventoryID uint `json:"inventoryId" gorm:"column:inventory_id;not null;index"`

	// Which checkout session/cart/order is this for?
	ReferenceID string `json:"referenceId" gorm:"column:reference_id;not null;index"`

	Quantity int `json:"quantity" gorm:"column:quantity;not null"`

	// When does this reservation die?
	ExpiresAt time.Time `json:"expiresAt" gorm:"column:expires_at;not null;index"`

	Status ReservationStatus `json:"status" gorm:"column:status;default:'PENDING'"`
}
