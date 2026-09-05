package model

import (
	"context"
	"time"
)

// ─── Payment Gateway DTOs (provider-agnostic) ─────────────────────────────────

// InitiatePaymentInput is the provider-agnostic input for starting a checkout session.
type InitiatePaymentInput struct {
	TransactionID  string
	AmountCents    int64
	Currency       string
	PaymentMethod  string
	GatewayOrderID string // optional provider reference for re-initiation
	CustomerID     uint
	SellerID       uint
	ReferenceType  string
	ReferenceID    uint
	Credentials    map[string]any // decrypted credentials
}

// InitiatePaymentOutput is the provider-agnostic result of starting a checkout session.
type InitiatePaymentOutput struct {
	GatewaySessionID string         `json:"gatewaySessionId"`
	Status           string         `json:"status"`
	AmountCents      int64          `json:"amountCents"`
	Currency         string         `json:"currency"`
	ExpiresAt        *time.Time     `json:"expiresAt,omitempty"`
	GatewayResponse  map[string]any `json:"gatewayResponse,omitempty"`
}

// RefundInput is the provider-agnostic input for creating a refund.
type RefundInput struct {
	GatewayPaymentID string
	RefundID         string
	AmountCents      int64
	Currency         string
	Notes            string
	Credentials      map[string]any // decrypted credentials
}

// RefundOutput is the provider-agnostic result of creating a refund.
type RefundOutput struct {
	GatewayRefundID string         `json:"gatewayRefundId"`
	Status          string         `json:"status"`
	AmountCents     int64          `json:"amountCents"`
	Currency        string         `json:"currency"`
	GatewayResponse map[string]any `json:"gatewayResponse,omitempty"`
}

// PaymentStatusOutput is the provider-agnostic status of a payment.
type PaymentStatusOutput struct {
	GatewayPaymentID string         `json:"gatewayPaymentId"`
	Status           string         `json:"status"`
	AmountCents      int64          `json:"amountCents"`
	Currency         string         `json:"currency"`
	GatewayFeeCents  *int64         `json:"gatewayFeeCents,omitempty"`
	PaymentMethod    string         `json:"paymentMethod,omitempty"`
	GatewayResponse  map[string]any `json:"gatewayResponse,omitempty"`
}

// WebhookEvent is the normalized, provider-agnostic representation of a webhook.
type WebhookEvent struct {
	GatewayCode    string         `json:"gatewayCode"`
	EventID        string         `json:"eventId"`
	EventType      string         `json:"eventType"`
	TransactionID  string         `json:"transactionId,omitempty"`
	SessionID      string         `json:"sessionId,omitempty"` // gateway order/intent id (Razorpay order_id)
	PaymentID      string         `json:"paymentId,omitempty"`
	RefundID       string         `json:"refundId,omitempty"`
	AmountCents    int64          `json:"amountCents,omitempty"`
	Currency       string         `json:"currency,omitempty"`
	FeeCents       *int64         `json:"feeCents,omitempty"`
	FailureCode    string         `json:"failureCode,omitempty"`
	FailureMessage string         `json:"failureMessage,omitempty"`
	Payload        map[string]any `json:"payload"`
}

// RazorpayCredentials holds the decrypted credentials for a Razorpay config.
type RazorpayCredentials struct {
	KeyID         string `json:"key_id" validate:"required"`
	KeySecret     string `json:"key_secret" validate:"required"`
	WebhookSecret string `json:"webhook_secret" validate:"required"`
	AccountID     string `json:"account_id,omitempty"`
}

// VerifyWebhookFunc is a helper type alias for the signature-verification contract.
type VerifyWebhookFunc func(ctx context.Context, rawBody []byte, signature string, credentials map[string]any) (bool, error)
