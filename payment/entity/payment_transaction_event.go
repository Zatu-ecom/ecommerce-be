package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// EventType represents the type of payment transaction event
type EventType string

const (
	EventTypeInitiated             EventType = "initiated"
	EventTypeGatewaySessionCreated EventType = "gateway_session_created"
	EventTypeAuthorized            EventType = "authorized"
	EventTypeCaptured              EventType = "captured"
	EventTypeCompleted             EventType = "completed"
	EventTypeFailed                EventType = "failed"
	EventTypeRefundInitiated       EventType = "refund_initiated"
	EventTypeRefundCompleted       EventType = "refund_completed"
	EventTypeRefundFailed          EventType = "refund_failed"
)

// EventSource represents who/what triggered the event
type EventSource string

const (
	EventSourceAPI     EventSource = "api"
	EventSourceWebhook EventSource = "webhook"
	EventSourceSystem  EventSource = "system"
	EventSourceAdmin   EventSource = "admin"
)

// ActorType represents the type of actor that triggered the event
type ActorType string

const (
	ActorTypeCustomer ActorType = "customer"
	ActorTypeSeller   ActorType = "seller"
	ActorTypeAdmin    ActorType = "admin"
	ActorTypeSystem   ActorType = "system"
)

// EventMetadata represents additional event metadata
type EventMetadata = db.JSONMap

// PaymentTransactionEvent is an append-only ledger of every payment state change.
// It has no UpdatedAt by design: rows are never mutated.
type PaymentTransactionEvent struct {
	ID              uint              `json:"id" gorm:"primaryKey"`
	TransactionID   uint              `json:"transactionId"   gorm:"column:transaction_id;not null;index"`
	EventType       EventType         `json:"eventType"       gorm:"column:event_type;size:50;not null;index"`
	FromStatus      TransactionStatus `json:"fromStatus"   gorm:"column:from_status;size:30"`
	ToStatus        TransactionStatus `json:"toStatus"     gorm:"column:to_status;size:30;not null"`
	GatewayEventID  string            `json:"gatewayEventId"  gorm:"column:gateway_event_id;size:255"`
	GatewayRequest  db.JSONMap        `json:"gatewayRequest"  gorm:"column:gateway_request;type:jsonb"`
	GatewayResponse db.JSONMap        `json:"gatewayResponse" gorm:"column:gateway_response;type:jsonb"`
	FailureCode     string            `json:"failureCode"     gorm:"column:failure_code;size:100"`
	FailureMessage  string            `json:"failureMessage"  gorm:"column:failure_message;type:text"`
	Source          EventSource       `json:"source"          gorm:"column:source;size:20;not null"`
	ActorID         *uint             `json:"actorId"         gorm:"column:actor_id"`
	ActorType       ActorType         `json:"actorType"       gorm:"column:actor_type;size:20"`
	CreatedAt       time.Time         `json:"createdAt"       gorm:"column:created_at;autoCreateTime;index"`

	// Relationship
	Transaction *PaymentTransaction `json:"transaction,omitempty" gorm:"foreignKey:TransactionID"`
}

func (PaymentTransactionEvent) TableName() string {
	return "payment_transaction_event"
}
