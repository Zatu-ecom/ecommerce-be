package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"

	"gorm.io/gorm"
)

// CourierProviderConfigRepository persists hybrid credentials: seller rows
// override the NULL-seller platform default. Callers resolve via
// FindSellerConfig with platform fallback (see provider_config_service).
type CourierProviderConfigRepository interface {
	Create(ctx context.Context, config *entity.CourierProviderConfig) error
	Save(ctx context.Context, config *entity.CourierProviderConfig) error
	FindByID(ctx context.Context, id uint) (*entity.CourierProviderConfig, error)
	// FindSellerConfig returns the seller's active row for (code, env),
	// or a not-configured error when absent. No fallback here.
	FindSellerConfig(
		ctx context.Context,
		sellerID uint,
		providerCode, environment string,
	) (*entity.CourierProviderConfig, error)
	// FindPlatformDefault returns the NULL-seller row for (code, env).
	FindPlatformDefault(
		ctx context.Context,
		providerCode, environment string,
	) (*entity.CourierProviderConfig, error)
	// FindSellerRow returns the seller's row for (code, env) in ANY active
	// state (configure-after-deactivate path). Absent row is an error.
	FindSellerRow(
		ctx context.Context,
		sellerID uint,
		providerCode, environment string,
	) (*entity.CourierProviderConfig, error)
	// FindBySeller returns all of the seller's config rows (list overlay).
	FindBySeller(ctx context.Context, sellerID uint) ([]entity.CourierProviderConfig, error)
}

type CourierProviderConfigRepositoryImpl struct{}

func NewCourierProviderConfigRepository() CourierProviderConfigRepository {
	return &CourierProviderConfigRepositoryImpl{}
}

func (r *CourierProviderConfigRepositoryImpl) Create(
	ctx context.Context,
	config *entity.CourierProviderConfig,
) error {
	return db.DB(ctx).Create(config).Error
}

func (r *CourierProviderConfigRepositoryImpl) Save(
	ctx context.Context,
	config *entity.CourierProviderConfig,
) error {
	return db.DB(ctx).Save(config).Error
}

func (r *CourierProviderConfigRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.CourierProviderConfig, error) {
	var config entity.CourierProviderConfig
	err := db.DB(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorProviderNotConfigured
		}
		return nil, err
	}
	return &config, nil
}

func (r *CourierProviderConfigRepositoryImpl) FindSellerConfig(
	ctx context.Context,
	sellerID uint,
	providerCode, environment string,
) (*entity.CourierProviderConfig, error) {
	var config entity.CourierProviderConfig
	err := db.DB(ctx).
		Where("seller_id = ? AND provider_code = ? AND environment = ? AND is_active = ?",
			sellerID, providerCode, environment, true).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorProviderNotConfigured
		}
		return nil, err
	}
	return &config, nil
}

func (r *CourierProviderConfigRepositoryImpl) FindPlatformDefault(
	ctx context.Context,
	providerCode, environment string,
) (*entity.CourierProviderConfig, error) {
	var config entity.CourierProviderConfig
	err := db.DB(ctx).
		Where("seller_id IS NULL AND provider_code = ? AND environment = ? AND is_active = ?",
			providerCode, environment, true).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorProviderNotConfigured
		}
		return nil, err
	}
	return &config, nil
}

func (r *CourierProviderConfigRepositoryImpl) FindSellerRow(
	ctx context.Context,
	sellerID uint,
	providerCode, environment string,
) (*entity.CourierProviderConfig, error) {
	var config entity.CourierProviderConfig
	err := db.DB(ctx).
		Where("seller_id = ? AND provider_code = ? AND environment = ?",
			sellerID, providerCode, environment).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorProviderNotConfigured
		}
		return nil, err
	}
	return &config, nil
}

func (r *CourierProviderConfigRepositoryImpl) FindBySeller(
	ctx context.Context,
	sellerID uint,
) ([]entity.CourierProviderConfig, error) {
	var configs []entity.CourierProviderConfig
	err := db.DB(ctx).
		Where("seller_id = ?", sellerID).
		Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}
