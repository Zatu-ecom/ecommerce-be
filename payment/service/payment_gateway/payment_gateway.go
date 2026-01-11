package gateway

import (
	"context"

	"ecommerce-be/payment/entity"
)

type RefundType string

const (
	REFUND_TYPE_PARTIAL RefundType = "partial"
	REFUND_TYPE_FULL    RefundType = "full"
)

type PaymentGateway interface {
	CreatePayment(
		ctx context.Context,
		amount int64,
		currency string,
		paymentGatewayConfig entity.PaymentGatewayConfig,
	) (string, error)

	RefundPayment(
		ctx context.Context,
		refundType RefundType,
		amount int64,
		currency string,
		transactionID string,
		paymentGatewayConfig entity.PaymentGatewayConfig,
	) (string, error)

	CancelPayment(
		ctx context.Context,
		transactionID string,
		paymentGatewayConfig entity.PaymentGatewayConfig,
	) (string, error)

	GetPaymentStatus(
		ctx context.Context,
		transactionID string,
		paymentGatewayConfig entity.PaymentGatewayConfig,
	) (string, error)
}
