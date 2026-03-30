package factory

import (
	"context"

	paymenterrors "ecommerce-be/payment/error"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

type PaymentGatewayFactory struct {
	paymentGatewayRepository repository.PaymentGatewayRepository
	cashfreeGateway          *gateway.CashfreeGateway
}

func NewPaymentGatewayFactory(
	cashfreeGateway *gateway.CashfreeGateway,
) *PaymentGatewayFactory {
	return &PaymentGatewayFactory{
		cashfreeGateway: cashfreeGateway,
	}
}

func (f *PaymentGatewayFactory) GetPaymentGateway(
	ctx context.Context,
	gatewayID uint,
) (gateway.PaymentGateway, error) {
	gateway, err := f.paymentGatewayRepository.FindById(ctx, gatewayID)
	if err != nil {
		return nil, err
	}

	if !gateway.IsActive {
		return nil, paymenterrors.ErrorPaymentGatewayNotActive
	}

	return f.getGatewayByCode(gateway.Code)
}

func (f *PaymentGatewayFactory) getGatewayByCode(
	code string,
) (gateway.PaymentGateway, error) {
	switch code {
	case f.cashfreeGateway.Code:
		return f.cashfreeGateway, nil
	default:
		return nil, paymenterrors.ErrorPaymentGatewayNotFound
	}
}
