package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// ============================================================================
// FulfillmentShipmentEvent — immutable append-only ledger (032, T6).
// event_type uses the shipment status vocabulary, or 'ignore'.
// Rows are never updated or deleted (no updated_at by design).
// ============================================================================

type FulfillmentShipmentEvent struct {
	ID         uint      `json:"id"        gorm:"primaryKey"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	ShipmentID uint      `json:"shipmentId" gorm:"column:shipment_id;not null;index"`

	EventType  string  `json:"eventType"  gorm:"column:event_type;size:50;not null"`
	FromStatus *string `json:"fromStatus" gorm:"column:from_status;size:32"`
	ToStatus   string  `json:"toStatus"   gorm:"column:to_status;size:32;not null"`

	// Raw provider label, audit only. Logic switches on event_type.
	ProviderEvent *string `json:"providerEvent" gorm:"column:provider_event;size:100"`

	ProviderRequest  db.JSONMap `json:"providerRequest"  gorm:"column:provider_request;type:jsonb"`
	ProviderResponse db.JSONMap `json:"providerResponse" gorm:"column:provider_response;type:jsonb"`

	FailureCode    *string `json:"failureCode"    gorm:"column:failure_code;size:100"`
	FailureMessage *string `json:"failureMessage" gorm:"column:failure_message"`

	Source    string  `json:"source"    gorm:"column:source;size:20;not null"`
	ActorID   *uint   `json:"actorId"   gorm:"column:actor_id"`
	ActorType *string `json:"actorType" gorm:"column:actor_type;size:20"`
}

// TableName returns the singular table name.
func (FulfillmentShipmentEvent) TableName() string {
	return "fulfillment_shipment_event"
}
