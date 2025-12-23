package entity

import (
	"time"

	"ecommerce-be/common/db"
)

type ReservationStatus string

const (
	ResPending   ReservationStatus = "PENDING"
	ResExpired   ReservationStatus = "EXPIRED"   // Released back to stock
	ResConfirmed ReservationStatus = "CONFIRMED" // Converted to order
	ResCancelled ReservationStatus = "CANCELLED" // Released back to stock
	ResFulfilled ReservationStatus = "FULFILLED" // Order has been fulfilled
)

type InventoryReservation struct {
	db.BaseEntity
	InventoryID uint `json:"inventoryId" gorm:"column:inventory_id;not null;index"`

	// Which checkout session/cart/order is this for?
	ReferenceID uint `json:"referenceId" gorm:"column:reference_id;not null;index"`

	Quantity uint `json:"quantity" gorm:"column:quantity;not null"`

	// When does this reservation die?
	ExpiresAt time.Time `json:"expiresAt" gorm:"column:expires_at;not null;index"`

	Status ReservationStatus `json:"status" gorm:"column:status;default:'PENDING'"`
}
