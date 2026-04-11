package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/file/entity"

	"gorm.io/gorm"
)

type ConfigRepository interface {
	GetProviders(ctx context.Context) ([]entity.StorageProvider, error)
	GetActiveProviderByID(ctx context.Context, id uint) (*entity.StorageProvider, error)
	GetConfigByID(ctx context.Context, id uint) (*entity.StorageConfig, error)
	GetSellerOwnedConfigByID(
		ctx context.Context,
		id uint,
		sellerID uint,
	) (*entity.StorageConfig, error)
	GetPlatformConfigByID(ctx context.Context, id uint) (*entity.StorageConfig, error)
	SaveConfig(ctx context.Context, config *entity.StorageConfig, clearPlatformDefaults bool) error
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

func (r *configRepository) GetActiveProviderByID(
	ctx context.Context,
	id uint,
) (*entity.StorageProvider, error) {
	var provider entity.StorageProvider
	err := db.DB(ctx).
		Where("id = ? AND is_active = ?", id, true).
		First(&provider).
		Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
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

func (r *configRepository) GetSellerOwnedConfigByID(
	ctx context.Context,
	id uint,
	sellerID uint,
) (*entity.StorageConfig, error) {
	var cfg entity.StorageConfig
	err := db.DB(ctx).
		Where(
			"id = ? AND owner_type = ? AND owner_id = ?",
			id,
			entity.OwnerTypeSeller,
			sellerID,
		).
		First(&cfg).
		Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *configRepository) GetPlatformConfigByID(
	ctx context.Context,
	id uint,
) (*entity.StorageConfig, error) {
	var cfg entity.StorageConfig
	err := db.DB(ctx).
		Where("id = ? AND owner_type = ?", id, entity.OwnerTypePlatform).
		First(&cfg).
		Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *configRepository) SaveConfig(
	ctx context.Context,
	config *entity.StorageConfig,
	clearPlatformDefaults bool,
) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if clearPlatformDefaults {
			if err := tx.
				Model(&entity.StorageConfig{}).
				Where("owner_type = ? AND id <> ?", entity.OwnerTypePlatform, config.ID).
				Update("is_default", false).
				Error; err != nil {
				return err
			}
		}

		if config.ID == 0 {
			return tx.Create(config).Error
		}

		return tx.Save(config).Error
	})
}
