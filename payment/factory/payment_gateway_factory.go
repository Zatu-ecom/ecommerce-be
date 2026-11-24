package factory

import (
	"context"

	paymenterrors "ecommerce-be/payment/error"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

// PaymentGatewayFactory selects provider adapters by payment_gateway.code.
// Adding a new gateway is purely additive: new adapter + one registry entry.
type PaymentGatewayFactory struct {
	paymentGatewayRepository repository.PaymentGatewayRepository
	registry                 map[string]gateway.PaymentGateway
}

func NewPaymentGatewayFactory(
	paymentGatewayRepository repository.PaymentGatewayRepository,
	adapters ...gateway.PaymentGateway,
) *PaymentGatewayFactory {
	registry := make(map[string]gateway.PaymentGateway, len(adapters))
	for _, adapter := range adapters {
		if adapter != nil {
			registry[adapter.Code()] = adapter
		}
	}
	return &PaymentGatewayFactory{
		paymentGatewayRepository: paymentGatewayRepository,
		registry:                 registry,
	}
}

func (f *PaymentGatewayFactory) GetPaymentGateway(
	ctx context.Context,
	gatewayID uint,
) (gateway.PaymentGateway, error) {
	gatewayEntity, err := f.paymentGatewayRepository.FindById(ctx, gatewayID)
	if err != nil {
		return nil, err
	}

	if !gatewayEntity.IsActive {
		return nil, paymenterrors.ErrorPaymentGatewayNotActive
	}

	return f.getGatewayByCode(gatewayEntity.Code)
}

func (f *PaymentGatewayFactory) GetPaymentGatewayByCode(
	code string,
) (gateway.PaymentGateway, error) {
	return f.getGatewayByCode(code)
}

func (f *PaymentGatewayFactory) getGatewayByCode(
	code string,
) (gateway.PaymentGateway, error) {
	if adapter, ok := f.registry[code]; ok {
		return adapter, nil
	}
	return nil, paymenterrors.ErrorPaymentGatewayNotSupported
}
