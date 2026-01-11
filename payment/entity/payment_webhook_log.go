package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
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
type WebhookPayload map[string]interface{}

// Scan implements sql.Scanner interface
func (p *WebhookPayload) Scan(value interface{}) error {
	if value == nil {
		*p = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal WebhookPayload value: %v", value)
	}
	return json.Unmarshal(bytes, p)
}

// Value implements driver.Valuer interface
func (p WebhookPayload) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}

// WebhookHeaders represents the webhook headers
type WebhookHeaders map[string]interface{}

// Scan implements sql.Scanner interface
func (h *WebhookHeaders) Scan(value interface{}) error {
	if value == nil {
		*h = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal WebhookHeaders value: %v", value)
	}
	return json.Unmarshal(bytes, h)
}

// Value implements driver.Valuer interface
func (h WebhookHeaders) Value() (driver.Value, error) {
	if h == nil {
		return nil, nil
	}
	return json.Marshal(h)
}

// PaymentWebhookLog represents a webhook event log
type PaymentWebhookLog struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	GatewayID     *uint           `json:"gatewayId" gorm:"column:gateway_id;index"`
	EventType     string          `json:"eventType" gorm:"column:event_type;size:100;not null"`
	EventID       string          `json:"eventId" gorm:"column:event_id;size:255;index"`
	Payload       WebhookPayload  `json:"payload" gorm:"column:payload;type:jsonb;not null"`
	Headers       WebhookHeaders  `json:"headers" gorm:"column:headers;type:jsonb"`
	Status        WebhookStatus   `json:"status" gorm:"column:status;size:30;not null;index"`
	ErrorMessage  string          `json:"errorMessage" gorm:"column:error_message;type:text"`
	ProcessedAt   *time.Time      `json:"processedAt" gorm:"column:processed_at"`
	TransactionID *uint           `json:"transactionId" gorm:"column:transaction_id"`
	RefundID      *uint           `json:"refundId" gorm:"column:refund_id"`
	IPAddress     string          `json:"ipAddress" gorm:"column:ip_address;size:50"`
	CreatedAt     time.Time       `json:"createdAt" gorm:"column:created_at;autoCreateTime;index"`

	// Relationships
	Gateway     *PaymentGateway     `json:"gateway,omitempty" gorm:"foreignKey:GatewayID"`
	Transaction *PaymentTransaction `json:"transaction,omitempty" gorm:"foreignKey:TransactionID"`
	Refund      *PaymentRefund      `json:"refund,omitempty" gorm:"foreignKey:RefundID"`
}

func (PaymentWebhookLog) TableName() string {
	return "payment_webhook_log"
}
