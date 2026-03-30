package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"

	"gorm.io/gorm"
)

type PaymentGatewayRepository interface {
	FindById(ctx context.Context, id uint) (*entity.PaymentGateway, error)
}

type PaymentGatewayRepositoryImpl struct{}

func NewPaymentGatewayRepository() PaymentGatewayRepository {
	return &PaymentGatewayRepositoryImpl{}
}

func (r *PaymentGatewayRepositoryImpl) FindById(
	ctx context.Context,
	id uint,
) (*entity.PaymentGateway, error) {
	var paymentGateway entity.PaymentGateway
	err := db.DB(ctx).Where("id = ?", id).First(&paymentGateway).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paymenterrors.ErrorPaymentGatewayNotFound
		}
		return nil, err
	}

	return &paymentGateway, nil
}
