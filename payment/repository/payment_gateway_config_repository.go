package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/payment/entity"

	"gorm.io/gorm"
)

type PaymentGatewayConfigRepository interface {
	// FindBySellerGatewayAndEnvironment loads one environment row.
	// Returns (nil, nil) when the seller never configured that environment.
	// Callers must always scope by environment — never First() without it.
	FindBySellerGatewayAndEnvironment(
		ctx context.Context,
		sellerID, gatewayID uint,
		environment entity.GatewayEnvironment,
	) (*entity.PaymentGatewayConfig, error)
	// FindAllBySellerAndGateway loads both environment rows (sandbox/production).
	FindAllBySellerAndGateway(
		ctx context.Context,
		sellerID, gatewayID uint,
	) ([]entity.PaymentGatewayConfig, error)
	FindActiveBySeller(
		ctx context.Context,
		sellerID uint,
	) ([]entity.PaymentGatewayConfig, error)
	// FindActiveBySellerAndEnvironment loads the seller's active configs for one
	// environment, highest priority first, with gateways preloaded. This is the
	// initiate-time selection set: no environment fallback, ever.
	FindActiveBySellerAndEnvironment(
		ctx context.Context,
		sellerID uint,
		environment entity.GatewayEnvironment,
	) ([]entity.PaymentGatewayConfig, error)
	FindAllForGateway(ctx context.Context, gatewayID uint) ([]entity.PaymentGatewayConfig, error)
	UpsertConfig(ctx context.Context, config *entity.PaymentGatewayConfig) error
	DeactivateConfig(ctx context.Context, id uint) error
}

type PaymentGatewayConfigRepositoryImpl struct{}

func NewPaymentGatewayConfigRepository() PaymentGatewayConfigRepository {
	return &PaymentGatewayConfigRepositoryImpl{}
}

func (r *PaymentGatewayConfigRepositoryImpl) FindBySellerGatewayAndEnvironment(
	ctx context.Context,
	sellerID, gatewayID uint,
	environment entity.GatewayEnvironment,
) (*entity.PaymentGatewayConfig, error) {
	var config entity.PaymentGatewayConfig
	err := db.DB(ctx).
		Preload("Gateway").
		Where("seller_id = ? AND gateway_id = ? AND environment = ?", sellerID, gatewayID, environment).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &config, nil
}

func (r *PaymentGatewayConfigRepositoryImpl) FindAllBySellerAndGateway(
	ctx context.Context,
	sellerID, gatewayID uint,
) ([]entity.PaymentGatewayConfig, error) {
	var configs []entity.PaymentGatewayConfig
	err := db.DB(ctx).
		Preload("Gateway").
		Where("seller_id = ? AND gateway_id = ?", sellerID, gatewayID).
		Order("environment ASC").
		Find(&configs).Error
	if err != nil {
		return nil, err
	}

	return configs, nil
}

func (r *PaymentGatewayConfigRepositoryImpl) FindActiveBySeller(
	ctx context.Context,
	sellerID uint,
) ([]entity.PaymentGatewayConfig, error) {
	var configs []entity.PaymentGatewayConfig
	err := db.DB(ctx).
		Preload("Gateway").
		Where("seller_id = ? AND is_active = ?", sellerID, true).
		Order("priority DESC").
		Find(&configs).Error
	if err != nil {
		return nil, err
	}

	return configs, nil
}

// FindAllForGateway returns all configs for a gateway (used to resolve the webhook
// secret). Ordered by seller_id so behavior is deterministic in tests.
func (r *PaymentGatewayConfigRepositoryImpl) FindActiveBySellerAndEnvironment(
	ctx context.Context,
	sellerID uint,
	environment entity.GatewayEnvironment,
) ([]entity.PaymentGatewayConfig, error) {
	var configs []entity.PaymentGatewayConfig
	err := db.DB(ctx).
		Preload("Gateway").
		Where("seller_id = ? AND environment = ? AND is_active = ?", sellerID, environment, true).
		Order("priority DESC").
		Find(&configs).Error
	if err != nil {
		return nil, err
	}

	return configs, nil
}

func (r *PaymentGatewayConfigRepositoryImpl) FindAllForGateway(ctx context.Context,
	gatewayID uint,
) ([]entity.PaymentGatewayConfig, error) {
	var configs []entity.PaymentGatewayConfig
	err := db.DB(ctx).
		Where("gateway_id = ? AND is_active = ?", gatewayID, true).
		Order("seller_id ASC").
		Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}

// UpsertConfig inserts or updates the seller's config for a gateway.
// The (seller_id, gateway_id, environment) unique constraint drives the conflict target.
func (r *PaymentGatewayConfigRepositoryImpl) UpsertConfig(
	ctx context.Context,
	config *entity.PaymentGatewayConfig,
) error {
	return db.DB(ctx).
		Clauses(gormOnConflictConfig()).
		Create(config).Error
}

func (r *PaymentGatewayConfigRepositoryImpl) DeactivateConfig(
	ctx context.Context,
	id uint,
) error {
	return db.DB(ctx).
		Model(&entity.PaymentGatewayConfig{}).
		Where("id = ?", id).
		Update("is_active", false).
		Error
}
