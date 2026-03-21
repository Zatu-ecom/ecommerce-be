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

// TransactionMetadata represents additional transaction metadata
type TransactionMetadata = db.JSONMap

// PaymentTransaction represents a payment transaction
type PaymentTransaction struct {
	db.BaseEntity
	TransactionID        string              `json:"transactionId"        gorm:"column:transaction_id;size:50;not null;uniqueIndex"`
	UserID               uint                `json:"userId"               gorm:"column:user_id;not null;index"`
	SellerID             uint                `json:"sellerId"             gorm:"column:seller_id;not null;index"`
	GatewayID            *uint               `json:"gatewayId"            gorm:"column:gateway_id;index"`
	GatewayTransactionID string              `json:"gatewayTransactionId" gorm:"column:gateway_transaction_id;size:255;index"`
	Currency             string              `json:"currency"             gorm:"column:currency;size:3;not null"`
	AmountCents          int64               `json:"amountCents"          gorm:"column:amount_cents;not null"`
	GatewayFeeCents      int64               `json:"gatewayFeeCents"      gorm:"column:gateway_fee_cents;not null"`
	Status               TransactionStatus   `json:"status"               gorm:"column:status;size:30;not null;index"`
	FailureCode          string              `json:"failureCode"          gorm:"column:failure_code;size:100"`
	FailureMessage       string              `json:"failureMessage"       gorm:"column:failure_message;type:text"`
	PaymentMethodType    PaymentMethodType   `json:"paymentMethodType"    gorm:"column:payment_method_type;size:50"`
	CompletedAt          *time.Time          `json:"completedAt"          gorm:"column:completed_at"`
	Metadata             TransactionMetadata `json:"metadata"             gorm:"column:metadata;type:jsonb"`

	// Relationships
	Gateway *PaymentGateway `json:"gateway,omitempty" gorm:"foreignKey:GatewayID"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transaction"
}
