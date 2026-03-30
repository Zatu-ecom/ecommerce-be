package gateway

import (
	"context"

	"ecommerce-be/payment/entity"
)

type CashfreeGateway struct {
	Code string
}

func NewCashfreeGateway() *CashfreeGateway {
	return &CashfreeGateway{
		Code: "cashfree",
	}
}

func (c *CashfreeGateway) CreatePayment(
	ctx context.Context,
	amount int64,
	currency string,
	paymentGatewayConfig entity.PaymentGatewayConfig,
) (string, error) {
	return "", nil
}

func (c *CashfreeGateway) RefundPayment(
	ctx context.Context,
	refundType RefundType,
	amount int64,
	currency string,
	transactionID string,
	paymentGatewayConfig entity.PaymentGatewayConfig,
) (string, error) {
	return "", nil
}

func (c *CashfreeGateway) CancelPayment(
	ctx context.Context,
	transactionID string,
	paymentGatewayConfig entity.PaymentGatewayConfig,
) (string, error) {
	return "", nil
}

func (c *CashfreeGateway) GetPaymentStatus(
	ctx context.Context,
	transactionID string,
	paymentGatewayConfig entity.PaymentGatewayConfig,
) (string, error) {
	return "", nil
}
