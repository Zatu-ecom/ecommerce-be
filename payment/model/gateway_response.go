package model

import "time"

// CreatePaymentResponse represents the response from creating a payment
type CreatePaymentResponse struct {
	GatewayTransactionID string    `json:"gatewayTransactionId"` // Gateway's transaction ID
	PaymentURL           string    `json:"paymentUrl"`           // URL to redirect user for payment
	Status               string    `json:"status"`               // 'pending', 'initiated', 'completed'
	Amount               int64     `json:"amount"`               // Amount in cents
	Currency             string    `json:"currency"`             // Currency code (INR, USD, etc.)
	ExpiresAt            time.Time `json:"expiresAt"`            // When payment link expires
	GatewayResponse      any       `json:"gatewayResponse"`      // Full gateway response for debugging
}

// RefundPaymentResponse represents the response from a refund request
type RefundPaymentResponse struct {
	GatewayRefundID string    `json:"gatewayRefundId"` // Gateway's refund ID
	Status          string    `json:"status"`          // 'pending', 'processing', 'completed', 'failed'
	Amount          int64     `json:"amount"`          // Refund amount in cents
	Currency        string    `json:"currency"`        // Currency code
	ProcessedAt     time.Time `json:"processedAt"`     // When refund was processed
	GatewayResponse any       `json:"gatewayResponse"` // Full gateway response for debugging
}

// CancelPaymentResponse represents the response from canceling a payment
type CancelPaymentResponse struct {
	GatewayTransactionID string    `json:"gatewayTransactionId"` // Gateway's transaction ID
	Status               string    `json:"status"`               // 'cancelled'
	CancelledAt          time.Time `json:"cancelledAt"`          // When payment was cancelled
	GatewayResponse      any       `json:"gatewayResponse"`      // Full gateway response for debugging
}

// PaymentStatusResponse represents the current status of a payment
type PaymentStatusResponse struct {
	GatewayTransactionID string    `json:"gatewayTransactionId"` // Gateway's transaction ID
	Status               string    `json:"status"`               // Current status
	Amount               int64     `json:"amount"`               // Amount in cents
	Currency             string    `json:"currency"`             // Currency code
	GatewayFeeCents      int64     `json:"gatewayFeeCents"`      // Gateway fee in cents
	CompletedAt          time.Time `json:"completedAt"`          // When payment completed
	GatewayResponse      any       `json:"gatewayResponse"`      // Full gateway response for debugging
}
