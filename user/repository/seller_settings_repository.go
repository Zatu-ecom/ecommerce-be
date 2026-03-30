package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/user/entity"
)

// SellerSettingsRepository defines the interface for seller settings data operations
type SellerSettingsRepository interface {
	Create(ctx context.Context, settings *entity.SellerSettings) error
	FindBySellerID(ctx context.Context, sellerID uint) (*entity.SellerSettings, error)
	Update(ctx context.Context, settings *entity.SellerSettings) error
	ExistsBySellerID(ctx context.Context, sellerID uint) (bool, error)
}

// SellerSettingsRepositoryImpl implements the SellerSettingsRepository interface
type SellerSettingsRepositoryImpl struct{}

// NewSellerSettingsRepository creates a new instance of SellerSettingsRepository
func NewSellerSettingsRepository() SellerSettingsRepository {
	return &SellerSettingsRepositoryImpl{}
}

// Create creates new seller settings in the database
func (r *SellerSettingsRepositoryImpl) Create(
	ctx context.Context,
	settings *entity.SellerSettings,
) error {
	return db.DB(ctx).Create(settings).Error
}

// FindBySellerID retrieves seller settings by seller ID
func (r *SellerSettingsRepositoryImpl) FindBySellerID(
	ctx context.Context,
	sellerID uint,
) (*entity.SellerSettings, error) {
	var settings entity.SellerSettings
	err := db.DB(ctx).Where("seller_id = ?", sellerID).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// Update updates existing seller settings
func (r *SellerSettingsRepositoryImpl) Update(
	ctx context.Context,
	settings *entity.SellerSettings,
) error {
	return db.DB(ctx).Save(settings).Error
}

// ExistsBySellerID checks if seller settings exist for a given seller ID
func (r *SellerSettingsRepositoryImpl) ExistsBySellerID(
	ctx context.Context,
	sellerID uint,
) (bool, error) {
	var count int64
	err := db.DB(ctx).
		Model(&entity.SellerSettings{}).
		Where("seller_id = ?", sellerID).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
