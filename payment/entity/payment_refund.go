package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// RefundStatus represents the status of a refund
type RefundStatus string

const (
	RefundStatusPending    RefundStatus = "pending"
	RefundStatusProcessing RefundStatus = "processing"
	RefundStatusCompleted  RefundStatus = "completed"
	RefundStatusFailed     RefundStatus = "failed"
)

// RefundReason represents the reason for a refund
type RefundReason string

const (
	RefundReasonCustomerRequest RefundReason = "customer_request"
	RefundReasonOrderCancelled  RefundReason = "order_cancelled"
	RefundReasonDefective       RefundReason = "defective"
	RefundReasonDuplicate       RefundReason = "duplicate"
)

// InitiatedByType represents who initiated the refund
type InitiatedByType string

const (
	InitiatedByCustomer InitiatedByType = "customer"
	InitiatedBySeller   InitiatedByType = "seller"
	InitiatedByAdmin    InitiatedByType = "admin"
	InitiatedBySystem   InitiatedByType = "system"
)

// RefundMetadata represents additional refund metadata
type RefundMetadata = db.JSONMap

// PaymentRefund represents a refund transaction
type PaymentRefund struct {
	db.BaseEntity
	RefundID        string          `json:"refundId"        gorm:"column:refund_id;size:50;not null;uniqueIndex"`
	TransactionID   uint            `json:"transactionId"   gorm:"column:transaction_id;not null;index"`
	GatewayRefundID string          `json:"gatewayRefundId" gorm:"column:gateway_refund_id;size:255;index"`
	Currency        string          `json:"currency"        gorm:"column:currency;size:3;not null"`
	AmountCents     int64           `json:"amountCents"     gorm:"column:amount_cents;not null"`
	Status          RefundStatus    `json:"status"          gorm:"column:status;size:30;not null;index"`
	FailureReason   string          `json:"failureReason"   gorm:"column:failure_reason;type:text"`
	Reason          RefundReason    `json:"reason"          gorm:"column:reason;size:100"`
	Notes           string          `json:"notes"           gorm:"column:notes;type:text"`
	InitiatedBy     *uint           `json:"initiatedBy"     gorm:"column:initiated_by"`
	InitiatedByType InitiatedByType `json:"initiatedByType" gorm:"column:initiated_by_type;size:20"`
	CompletedAt     *time.Time      `json:"completedAt"     gorm:"column:completed_at"`
	Metadata        RefundMetadata  `json:"metadata"        gorm:"column:metadata;type:jsonb"`

	// Relationships
	Transaction *PaymentTransaction `json:"transaction,omitempty" gorm:"foreignKey:TransactionID"`
}

func (PaymentRefund) TableName() string {
	return "payment_refund"
}
