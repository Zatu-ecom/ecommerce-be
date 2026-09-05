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
	FindByCode(ctx context.Context, code string) (*entity.PaymentGateway, error)
	FindAllActive(ctx context.Context) ([]entity.PaymentGateway, error)
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

func (r *PaymentGatewayRepositoryImpl) FindByCode(
	ctx context.Context,
	code string,
) (*entity.PaymentGateway, error) {
	var paymentGateway entity.PaymentGateway
	err := db.DB(ctx).Where("code = ?", code).First(&paymentGateway).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paymenterrors.ErrorPaymentGatewayNotFound
		}
		return nil, err
	}

	return &paymentGateway, nil
}

func (r *PaymentGatewayRepositoryImpl) FindAllActive(
	ctx context.Context,
) ([]entity.PaymentGateway, error) {
	var gateways []entity.PaymentGateway
	err := db.DB(ctx).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&gateways).Error
	if err != nil {
		return nil, err
	}

	return gateways, nil
}
