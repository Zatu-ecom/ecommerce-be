package model

import (
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

// CheckoutPayload is the FE-facing, adapter-defined checkout session.
// Each provider fills Fields with what its SDK needs (Razorpay: keyId/orderId).
type CheckoutPayload struct {
	GatewayCode string         `json:"gatewayCode"`
	Fields      map[string]any `json:"fields"`
}

// InitiatePaymentOutput is the provider-agnostic result of starting a checkout session.
type InitiatePaymentOutput struct {
	GatewaySessionID string          `json:"gatewaySessionId"`
	Status           string          `json:"status"`
	AmountCents      int64           `json:"amountCents"`
	Currency         string          `json:"currency"`
	ExpiresAt        *time.Time      `json:"expiresAt,omitempty"`
	GatewayResponse  map[string]any  `json:"gatewayResponse,omitempty"`
	Checkout         CheckoutPayload `json:"checkout"`
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
