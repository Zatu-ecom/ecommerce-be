package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"

	"gorm.io/gorm"
)

// CourierProviderRepository reads the courier catalog and credential forms.
// Catalog rows are seed-managed; no write methods by design.
type CourierProviderRepository interface {
	FindByCode(ctx context.Context, code string) (*entity.CourierProvider, error)
	FindAllActive(ctx context.Context) ([]entity.CourierProvider, error)
	FindFields(ctx context.Context, providerCode string) ([]entity.CourierProviderField, error)
}

type CourierProviderRepositoryImpl struct{}

func NewCourierProviderRepository() CourierProviderRepository {
	return &CourierProviderRepositoryImpl{}
}

func (r *CourierProviderRepositoryImpl) FindByCode(
	ctx context.Context,
	code string,
) (*entity.CourierProvider, error) {
	var provider entity.CourierProvider
	err := db.DB(ctx).Where("code = ?", code).First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorProviderNotSupported
		}
		return nil, err
	}
	return &provider, nil
}

func (r *CourierProviderRepositoryImpl) FindAllActive(
	ctx context.Context,
) ([]entity.CourierProvider, error) {
	var providers []entity.CourierProvider
	err := db.DB(ctx).
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

func (r *CourierProviderRepositoryImpl) FindFields(
	ctx context.Context,
	providerCode string,
) ([]entity.CourierProviderField, error) {
	var fields []entity.CourierProviderField
	err := db.DB(ctx).
		Where("provider_code = ?", providerCode).
		Order("display_order ASC, id ASC").
		Find(&fields).Error
	if err != nil {
		return nil, err
	}
	return fields, nil
}
