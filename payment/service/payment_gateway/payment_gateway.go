package gateway

import (
	"context"

	paymentModel "ecommerce-be/payment/model"
)

// RefundType distinguishes partial vs full refund requests.
type RefundType string

const (
	REFUND_TYPE_PARTIAL RefundType = "partial"
	REFUND_TYPE_FULL    RefundType = "full"
)

// PaymentGateway is the gateway-agnostic contract every provider adapter implements.
// The payment service never references a concrete gateway type — it only depends on
// this interface and the gateway `code`. Adding a new provider is purely additive:
// a new adapter + one factory registry entry.
type PaymentGateway interface {
	// Code returns the provider code (e.g. "razorpay").
	Code() string

	// InitiatePayment starts a provider checkout session.
	InitiatePayment(
		ctx context.Context,
		in paymentModel.InitiatePaymentInput,
	) (*paymentModel.InitiatePaymentOutput, error)

	// FetchPayment returns the current provider status of a captured payment.
	FetchPayment(
		ctx context.Context,
		gatewayPaymentID string,
		credentials map[string]any,
	) (*paymentModel.PaymentStatusOutput, error)

	// Refund creates a full or partial refund for a captured payment.
	Refund(
		ctx context.Context,
		refundType RefundType,
		in paymentModel.RefundInput,
	) (*paymentModel.RefundOutput, error)

	// VerifyWebhook validates the authenticity of an inbound webhook using the raw body.
	VerifyWebhook(rawBody []byte, signature string, credentials map[string]any) (bool, error)

	// ParseWebhook normalizes an inbound raw webhook body into a gateway-agnostic event.
	ParseWebhook(rawBody []byte) (*paymentModel.WebhookEvent, error)
}
