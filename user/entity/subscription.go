package entity

import (
	"time"

	"ecommerce-be/common/db"
)

type Subscription struct {
	db.BaseEntity
	SellerID             uint               `json:"sellerId"                       gorm:"not null;index"`                 // Foreign Key to Seller
	PlanID               uint               `json:"planId"                         gorm:"not null;index"`                 // Foreign Key to Plan
	Status               SubscriptionStatus `json:"status"                         gorm:"not null;default:pending;index"` // e.g., pending, active, trialing, expired, cancelled
	StartDate            time.Time          `json:"startDate"                      gorm:"not null"`                       // When this subscription period started.
	EndDate              time.Time          `json:"endDate"                        gorm:"not null"`                       // When this subscription period ends.
	PaymentTransactionID string             `json:"paymentTransactionId,omitempty"`                                       // The transaction ID from your payment provider.
}

type SubscriptionStatus string

const (
	SUBSCRIPTION_STATUS_PENDING   SubscriptionStatus = "pending"
	SUBSCRIPTION_STATUS_ACTIVE    SubscriptionStatus = "active"
	SUBSCRIPTION_STATUS_TRIALING  SubscriptionStatus = "trialing"
	SUBSCRIPTION_STATUS_EXPIRED   SubscriptionStatus = "expired"
	SUBSCRIPTION_STATUS_CANCELLED SubscriptionStatus = "cancelled"
)
