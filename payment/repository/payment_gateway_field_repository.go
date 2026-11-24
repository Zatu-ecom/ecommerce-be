package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/payment/entity"
)

type PaymentGatewayFieldRepository interface {
	FindByGatewayID(ctx context.Context, gatewayID uint) ([]entity.PaymentGatewayField, error)
}

type PaymentGatewayFieldRepositoryImpl struct{}

func NewPaymentGatewayFieldRepository() PaymentGatewayFieldRepository {
	return &PaymentGatewayFieldRepositoryImpl{}
}

func (r *PaymentGatewayFieldRepositoryImpl) FindByGatewayID(
	ctx context.Context,
	gatewayID uint,
) ([]entity.PaymentGatewayField, error) {
	var fields []entity.PaymentGatewayField
	err := db.DB(ctx).
		Where("gateway_id = ?", gatewayID).
		Order("display_order ASC").
		Find(&fields).Error
	if err != nil {
		return nil, err
	}

	return fields, nil
}
