package model

import (
	"ecommerce-be/payment/entity"
)

// InitiatePaymentRequest is the customer request to start a payment for an order.
type InitiatePaymentRequest struct {
	OrderID           uint                     `json:"orderId" binding:"required,gt=0"`
	PaymentMethodType entity.PaymentMethodType `json:"paymentMethodType"`
}

// RefundRequest is the seller request to refund a completed payment.
type RefundRequest struct {
	TransactionID string              `json:"transactionId" binding:"required"`
	AmountCents   int64               `json:"amountCents"   binding:"required,gt=0"`
	Reason        entity.RefundReason `json:"reason"`
	Notes         string              `json:"notes"`
}

// ConfigureGatewayRequest is the seller request to save/update a gateway config.
// Environment is required: credentials are stored per environment, never mixed.
type ConfigureGatewayRequest struct {
	Credentials map[string]any            `json:"credentials" binding:"required"`
	Environment entity.GatewayEnvironment `json:"environment" binding:"required,oneof=sandbox production"`
	IsActive    *bool                     `json:"isActive"`
	Priority    int                       `json:"priority"`
}

// TestGatewayRequest probes credentials without persisting anything.
// Credentials are optional: when omitted, the saved row for the environment
// is tested. Provided values are merged over the saved row (empty secrets
// keep stored values) for the probe only.
type TestGatewayRequest struct {
	Environment entity.GatewayEnvironment `json:"environment" binding:"required,oneof=sandbox production"`
	Credentials map[string]any            `json:"credentials"`
}
