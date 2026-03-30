package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

// PromotionProductVariantScopeRepository defines the interface for promotion-product-variant scope operations
type PromotionProductVariantScopeRepository interface {
	AddPromotionProductVariants(
		ctx context.Context,
		variants []entity.PromotionProductVariant,
	) error
	DeletePromotionProductVariants(ctx context.Context, promotionID uint, variantIDs []uint) error
	DeletePromotionProductVariantByPromotionID(ctx context.Context, promotionID uint) error
	GetPromotionProductVariants(
		ctx context.Context,
		promotionID uint,
		variantIDs []uint,
		offset, limit int,
	) ([]entity.PromotionProductVariant, int64, error)
}

// PromotionProductVariantScopeRepositoryImpl implements the PromotionProductVariantRepository interface
type PromotionProductVariantScopeRepositoryImpl struct{}

// NewPromotionProductVariantScopeRepository creates a new instance of PromotionProductVariantRepository
func NewPromotionProductVariantScopeRepository() PromotionProductVariantScopeRepository {
	return &PromotionProductVariantScopeRepositoryImpl{}
}

// AddPromotionProductVariants bulk inserts promotion-product-variant mappings
func (r *PromotionProductVariantScopeRepositoryImpl) AddPromotionProductVariants(
	ctx context.Context,
	variants []entity.PromotionProductVariant,
) error {
	if len(variants) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&variants).Error
}

// DeletePromotionProductVariants deletes specific variant mappings from a promotion
func (r *PromotionProductVariantScopeRepositoryImpl) DeletePromotionProductVariants(
	ctx context.Context,
	promotionID uint,
	variantIDs []uint,
) error {
	if len(variantIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("promotion_id = ? AND variant_id IN ?", promotionID, variantIDs).
		Delete(&entity.PromotionProductVariant{}).Error
}

// DeletePromotionProductVariantByPromotionID removes all variant mappings for a promotion
func (r *PromotionProductVariantScopeRepositoryImpl) DeletePromotionProductVariantByPromotionID(
	ctx context.Context,
	promotionID uint,
) error {
	return db.DB(ctx).
		Where("promotion_id = ?", promotionID).
		Delete(&entity.PromotionProductVariant{}).Error
}

// GetPromotionProductVariants retrieves all variant mappings for a promotion with pagination
func (r *PromotionProductVariantScopeRepositoryImpl) GetPromotionProductVariants(
	ctx context.Context,
	promotionID uint,
	variantIDs []uint,
	offset, limit int,
) ([]entity.PromotionProductVariant, int64, error) {
	var variants []entity.PromotionProductVariant
	var total int64

	query := db.DB(ctx).
		Model(&entity.PromotionProductVariant{}).
		Where("promotion_id = ?", promotionID)

	if len(variantIDs) > 0 {
		query = query.Where("variant_id IN ?", variantIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Find(&variants).Error
	if err != nil {
		return nil, 0, err
	}

	return variants, total, nil
}
