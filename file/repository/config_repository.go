package repository

import (
	"context"
	"strings"

	"ecommerce-be/common/db"
	"ecommerce-be/file/entity"
	"ecommerce-be/file/model"

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
	ListConfigs(ctx context.Context, filter model.ListStorageConfigFilter) ([]entity.StorageConfig, int64, error)
	ActivateConfig(ctx context.Context, configID uint, ownerType entity.OwnerType, ownerID *uint) error
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

func (r *configRepository) ListConfigs(
	ctx context.Context,
	filter model.ListStorageConfigFilter,
) ([]entity.StorageConfig, int64, error) {
	query := db.DB(ctx).Model(&entity.StorageConfig{}).
		Where("owner_type = ?", filter.OwnerType)

	// Seller scope — constrain to the owner's ID
	if filter.OwnerType == entity.OwnerTypeSeller && filter.OwnerID != nil {
		query = query.Where("owner_id = ?", *filter.OwnerID)
	}

	// Multi-value filters
	if len(filter.IDs) > 0 {
		query = query.Where("id IN ?", filter.IDs)
	}
	if len(filter.ProviderIDs) > 0 {
		query = query.Where("provider_id IN ?", filter.ProviderIDs)
	}
	if len(filter.ValidationStatuses) > 0 {
		query = query.Where("validation_status IN ?", filter.ValidationStatuses)
	}

	// Single-value filters
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if filter.AdapterType != nil && *filter.AdapterType != "" {
		query = query.Joins("JOIN storage_provider ON storage_provider.id = storage_config.provider_id").
			Where("storage_provider.adapter_type = ?", *filter.AdapterType)
	}
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + strings.ToLower(*filter.Search) + "%"
		query = query.Where("LOWER(display_name) LIKE ?", like)
	}

	// Count total before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Resolve sort column (allowlist)
	sortBy := resolveSortColumn(filter.SortBy)
	sortOrder := "DESC"
	if strings.ToLower(filter.SortOrder) == "asc" {
		sortOrder = "ASC"
	}

	offset := (filter.Page - 1) * filter.PageSize

	var configs []entity.StorageConfig
	err := query.
		Order(sortBy + " " + sortOrder).
		Offset(offset).
		Limit(filter.PageSize).
		Find(&configs).
		Error

	return configs, total, err
}

// resolveSortColumn maps client-provided sort-by values to safe DB column names.
func resolveSortColumn(sortBy string) string {
	switch sortBy {
	case "displayName":
		return "display_name"
	case "validationStatus":
		return "validation_status"
	case "updatedAt":
		return "updated_at"
	default:
		return "created_at"
	}
}

func (r *configRepository) ActivateConfig(
	ctx context.Context,
	configID uint,
	ownerType entity.OwnerType,
	ownerID *uint,
) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// Deactivate all configs in the same scope
		deactivate := tx.Model(&entity.StorageConfig{}).
			Where("owner_type = ? AND id <> ?", ownerType, configID)
		if ownerType == entity.OwnerTypeSeller && ownerID != nil {
			deactivate = deactivate.Where("owner_id = ?", *ownerID)
		}
		if err := deactivate.Update("is_active", false).Error; err != nil {
			return err
		}

		// Activate the target config
		return tx.Model(&entity.StorageConfig{}).
			Where("id = ? AND owner_type = ?", configID, ownerType).
			Update("is_active", true).
			Error
	})
}
