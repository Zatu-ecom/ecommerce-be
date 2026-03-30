package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// OrderHistory captures immutable audit entries for every order status transition.
type OrderHistory struct {
	ID              uint       `json:"id"              gorm:"primaryKey"`
	OrderID         uint       `json:"orderId"         gorm:"column:order_id;not null;index"`
	FromStatus      *string    `json:"fromStatus"      gorm:"column:from_status;size:32"`
	ToStatus        string     `json:"toStatus"        gorm:"column:to_status;size:32;not null"`
	ChangedByUserID *uint      `json:"changedByUserId" gorm:"column:changed_by_user_id"`
	ChangedByRole   *string    `json:"changedByRole"   gorm:"column:changed_by_role;size:32"`
	TransactionID   *string    `json:"transactionId"   gorm:"column:transaction_id;size:255"`
	FailureReason   *string    `json:"failureReason"   gorm:"column:failure_reason"`
	Note            *string    `json:"note"            gorm:"column:note"`
	Metadata        db.JSONMap `json:"metadata"        gorm:"column:metadata;type:jsonb;default:'{}'"`
	CreatedAt       time.Time  `json:"createdAt"       gorm:"column:created_at;autoCreateTime"`
}

