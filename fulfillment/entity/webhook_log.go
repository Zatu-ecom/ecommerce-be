package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// ============================================================================
// FulfillmentWebhookLog — verified-only audit + idempotency (032, T7).
// Persisted ONLY after signature verification: unverifiable bodies are
// dropped without storage. (provider_code, event_id) dedupes replays.
// ============================================================================

type FulfillmentWebhookLog struct {
	ID        uint      `json:"id"        gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`

	ProviderCode string  `json:"providerCode" gorm:"column:provider_code;size:50;not null"`
	EventID      string  `json:"eventId"      gorm:"column:event_id;not null"`
	AWB          *string `json:"awb"          gorm:"column:awb"`

	// Same vocabulary as status, or 'ignore'.
	Action string `json:"action" gorm:"column:action;size:50;not null"`
	// 'received', 'applied', 'failed', 'ignored'.
	Status       string  `json:"status"       gorm:"column:status;size:30;not null"`
	ErrorMessage *string `json:"errorMessage" gorm:"column:error_message"`

	// Sanitized: phones/addresses stripped. Secrets removed from headers.
	Payload db.JSONMap `json:"payload" gorm:"column:payload;type:jsonb;not null"`
	Headers db.JSONMap `json:"headers" gorm:"column:headers;type:jsonb"`

	ShipmentID  *uint      `json:"shipmentId"  gorm:"column:shipment_id"`
	IPAddress   *string    `json:"ipAddress"   gorm:"column:ip_address;size:50"`
	ProcessedAt *time.Time `json:"processedAt" gorm:"column:processed_at"`
}

// TableName returns the singular table name.
func (FulfillmentWebhookLog) TableName() string {
	return "fulfillment_webhook_log"
}
