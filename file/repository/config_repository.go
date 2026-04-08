package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/file/entity"
)

type ConfigRepository interface {
	GetProviders(ctx context.Context) ([]entity.StorageProvider, error)
	CreateConfig(ctx context.Context, config *entity.StorageConfig) error
	UpdateConfig(ctx context.Context, config *entity.StorageConfig) error
	GetConfigByID(ctx context.Context, id uint) (*entity.StorageConfig, error)
	ClearDefaultConfigs(ctx context.Context) error
}

type configRepository struct{}

func NewConfigRepository() ConfigRepository {
	return &configRepository{}
}

func (r *configRepository) GetProviders(ctx context.Context) ([]entity.StorageProvider, error) {
	var providers []entity.StorageProvider
	err := db.DB(ctx).Where("is_active = ?", true).Find(&providers).Error
	return providers, err
}

func (r *configRepository) CreateConfig(
	ctx context.Context,
	config *entity.StorageConfig,
) error {
	return db.DB(ctx).Create(config).Error
}

func (r *configRepository) UpdateConfig(
	ctx context.Context,
	config *entity.StorageConfig,
) error {
	return db.DB(ctx).Save(config).Error
}

func (r *configRepository) GetConfigByID(
	ctx context.Context,
	id uint,
) (*entity.StorageConfig, error) {
	var config entity.StorageConfig
	err := db.DB(ctx).First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *configRepository) ClearDefaultConfigs(ctx context.Context) error {
	return db.DB(ctx).
		Model(&entity.StorageConfig{}).
		Where("owner_type = ?", entity.OwnerTypePlatform).
		Update("is_default", false).
		Error
}
