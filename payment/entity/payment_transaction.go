package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// TransactionStatus represents the status of a payment transaction
type TransactionStatus string

const (
	TransactionStatusPending           TransactionStatus = "pending"
	TransactionStatusCompleted         TransactionStatus = "completed"
	TransactionStatusFailed            TransactionStatus = "failed"
	TransactionStatusRefunded          TransactionStatus = "refunded"
	TransactionStatusPartiallyRefunded TransactionStatus = "partially_refunded"
)

// ReferenceType identifies the polymorphic owner of a payment transaction.
type ReferenceType string

const (
	ReferenceTypeOrder        ReferenceType = "order"
	ReferenceTypeSubscription ReferenceType = "subscription"
)

// PaymentTransactionMetadata represents additional transaction metadata
type PaymentTransactionMetadata = db.JSONMap

// PaymentTransaction represents a payment transaction
type PaymentTransaction struct {
	db.BaseEntity
	TransactionID        string                     `json:"transactionId"        gorm:"column:transaction_id;size:50;not null;uniqueIndex"`
	UserID               uint                       `json:"userId"               gorm:"column:user_id;not null;index"`
	SellerID             uint                       `json:"sellerId"             gorm:"column:seller_id;not null;index"`
	GatewayID            *uint                      `json:"gatewayId"            gorm:"column:gateway_id;index"`
	ReferenceType        ReferenceType              `json:"referenceType"        gorm:"column:reference_type;size:20;index"`
	ReferenceID          uint                       `json:"referenceId"          gorm:"column:reference_id;index"`
	GatewaySessionID     string                     `json:"gatewaySessionId"     gorm:"column:gateway_session_id;size:255;index"`
	GatewayPaymentID     string                     `json:"gatewayPaymentId"     gorm:"column:gateway_payment_id;size:255;index"`
	Currency             string                     `json:"currency"             gorm:"column:currency;size:3;not null"`
	AmountCents          int64                      `json:"amountCents"          gorm:"column:amount_cents;not null"`
	GatewayFeeCents      *int64                     `json:"gatewayFeeCents"      gorm:"column:gateway_fee_cents"`
	Status               TransactionStatus          `json:"status"               gorm:"column:status;size:30;not null;index"`
	FailureCode          string                     `json:"failureCode"          gorm:"column:failure_code;size:100"`
	FailureMessage       string                     `json:"failureMessage"       gorm:"column:failure_message;type:text"`
	PaymentMethodType    PaymentMethodType          `json:"paymentMethodType"    gorm:"column:payment_method_type;size:50"`
	PaymentMethodDetails PaymentTransactionMetadata `json:"paymentMethodDetails" gorm:"column:payment_method_details;type:jsonb"`
	CompletedAt          *time.Time                 `json:"completedAt"          gorm:"column:completed_at"`

	// Relationships
	Gateway *PaymentGateway `json:"gateway,omitempty" gorm:"foreignKey:GatewayID"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transaction"
}
