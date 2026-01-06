package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/user/entity"
)

// SellerProfileRepository defines the interface for seller profile data operations
type SellerProfileRepository interface {
	Create(ctx context.Context, profile *entity.SellerProfile) error
	FindByUserID(ctx context.Context, userID uint) (*entity.SellerProfile, error)
	Update(ctx context.Context, profile *entity.SellerProfile) error
	ExistsByTaxID(ctx context.Context, taxID string) (bool, error)
	ExistsByTaxIDExcluding(ctx context.Context, taxID string, excludeUserID uint) (bool, error)
}

// SellerProfileRepositoryImpl implements the SellerProfileRepository interface
type SellerProfileRepositoryImpl struct{}

// NewSellerProfileRepository creates a new instance of SellerProfileRepository
func NewSellerProfileRepository() SellerProfileRepository {
	return &SellerProfileRepositoryImpl{}
}

// Create creates a new seller profile in the database
func (r *SellerProfileRepositoryImpl) Create(ctx context.Context, profile *entity.SellerProfile) error {
	return db.DB(ctx).Create(profile).Error
}

// FindByUserID retrieves a seller profile by user ID
func (r *SellerProfileRepositoryImpl) FindByUserID(ctx context.Context, userID uint) (*entity.SellerProfile, error) {
	var profile entity.SellerProfile
	err := db.DB(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// Update updates an existing seller profile
func (r *SellerProfileRepositoryImpl) Update(ctx context.Context, profile *entity.SellerProfile) error {
	return db.DB(ctx).Save(profile).Error
}

// ExistsByTaxID checks if a seller profile exists with the given tax ID
func (r *SellerProfileRepositoryImpl) ExistsByTaxID(ctx context.Context, taxID string) (bool, error) {
	if taxID == "" {
		return false, nil
	}
	var count int64
	if err := db.DB(ctx).Model(&entity.SellerProfile{}).Where("tax_id = ?", taxID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByTaxIDExcluding checks if a seller profile exists with the given tax ID, excluding a specific user
func (r *SellerProfileRepositoryImpl) ExistsByTaxIDExcluding(ctx context.Context, taxID string, excludeUserID uint) (bool, error) {
	if taxID == "" {
		return false, nil
	}
	var count int64
	if err := db.DB(ctx).Model(&entity.SellerProfile{}).
		Where("tax_id = ? AND user_id != ?", taxID, excludeUserID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
