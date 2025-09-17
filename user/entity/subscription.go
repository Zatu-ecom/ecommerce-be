package entity

import (
	"time"

	"ecommerce-be/common/db"
)

type Subscription struct {
	db.BaseEntity
	SellerID             uint      `json:"sellerId"                       gorm:"not null;index"`                 // Foreign Key to Seller
	PlanID               uint      `json:"planId"                         gorm:"not null;index"`                 // Foreign Key to Plan
	Status               string    `json:"status"                         gorm:"not null;default:pending;index"` // e.g., pending, active, trialing, expired, cancelled
	StartDate            time.Time `json:"startDate"                      gorm:"not null"`                       // When this subscription period started.
	EndDate              time.Time `json:"endDate"                        gorm:"not null"`                       // When this subscription period ends.
	PaymentTransactionID string    `json:"paymentTransactionId,omitempty"`                                       // The transaction ID from your payment provider.
}
