package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// WebhookStatus represents the processing status of a webhook
type WebhookStatus string

const (
	WebhookStatusReceived  WebhookStatus = "received"
	WebhookStatusProcessed WebhookStatus = "processed"
	WebhookStatusFailed    WebhookStatus = "failed"
	WebhookStatusIgnored   WebhookStatus = "ignored"
)

// WebhookPayload represents the webhook payload
type WebhookPayload = db.JSONMap

// WebhookHeaders represents the webhook headers
type WebhookHeaders = db.JSONMap

// PaymentWebhookLog represents a webhook event log
type PaymentWebhookLog struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	GatewayID     *uint          `json:"gatewayId" gorm:"column:gateway_id;index"`
	EventType     string         `json:"eventType" gorm:"column:event_type;size:100;not null"`
	EventID       string         `json:"eventId" gorm:"column:event_id;size:255;index"`
	Payload       WebhookPayload `json:"payload" gorm:"column:payload;type:jsonb;not null"`
	Headers       WebhookHeaders `json:"headers" gorm:"column:headers;type:jsonb"`
	Status        WebhookStatus  `json:"status" gorm:"column:status;size:30;not null;index"`
	ErrorMessage  string         `json:"errorMessage" gorm:"column:error_message;type:text"`
	ProcessedAt   *time.Time     `json:"processedAt" gorm:"column:processed_at"`
	TransactionID *uint          `json:"transactionId" gorm:"column:transaction_id"`
	RefundID      *uint          `json:"refundId" gorm:"column:refund_id"`
	IPAddress     string         `json:"ipAddress" gorm:"column:ip_address;size:50"`
	CreatedAt     time.Time      `json:"createdAt" gorm:"column:created_at;autoCreateTime;index"`

	// Relationships
	Gateway     *PaymentGateway     `json:"gateway,omitempty" gorm:"foreignKey:GatewayID"`
	Transaction *PaymentTransaction `json:"transaction,omitempty" gorm:"foreignKey:TransactionID"`
	Refund      *PaymentRefund      `json:"refund,omitempty" gorm:"foreignKey:RefundID"`
}

func (PaymentWebhookLog) TableName() string {
	return "payment_webhook_log"
}
