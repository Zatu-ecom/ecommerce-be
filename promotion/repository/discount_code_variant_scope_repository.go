package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

// DiscountCodeVariantScopeRepository manages variant-level rows on discount_code_product
// (variant_id IS NOT NULL).
type DiscountCodeVariantScopeRepository interface {
	AddVariants(ctx context.Context, rows []entity.DiscountCodeProduct) error
	DeleteVariants(ctx context.Context, discountCodeID uint, variantIDs []uint) error
	DeleteAllVariants(ctx context.Context, discountCodeID uint) error
	GetVariants(
		ctx context.Context,
		discountCodeID uint,
		variantIDs []uint,
		offset, limit int,
	) ([]entity.DiscountCodeProduct, int64, error)
}

type DiscountCodeVariantScopeRepositoryImpl struct{}

func NewDiscountCodeVariantScopeRepository() DiscountCodeVariantScopeRepository {
	return &DiscountCodeVariantScopeRepositoryImpl{}
}

func (r *DiscountCodeVariantScopeRepositoryImpl) AddVariants(
	ctx context.Context,
	rows []entity.DiscountCodeProduct,
) error {
	if len(rows) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&rows).Error
}

func (r *DiscountCodeVariantScopeRepositoryImpl) DeleteVariants(
	ctx context.Context,
	discountCodeID uint,
	variantIDs []uint,
) error {
	if len(variantIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("discount_code_id = ? AND variant_id IN ?", discountCodeID, variantIDs).
		Delete(&entity.DiscountCodeProduct{}).Error
}

func (r *DiscountCodeVariantScopeRepositoryImpl) DeleteAllVariants(
	ctx context.Context,
	discountCodeID uint,
) error {
	return db.DB(ctx).
		Where("discount_code_id = ? AND variant_id IS NOT NULL", discountCodeID).
		Delete(&entity.DiscountCodeProduct{}).Error
}

func (r *DiscountCodeVariantScopeRepositoryImpl) GetVariants(
	ctx context.Context,
	discountCodeID uint,
	variantIDs []uint,
	offset, limit int,
) ([]entity.DiscountCodeProduct, int64, error) {
	var rows []entity.DiscountCodeProduct
	var total int64

	query := db.DB(ctx).Model(&entity.DiscountCodeProduct{}).
		Where("discount_code_id = ? AND variant_id IS NOT NULL", discountCodeID)

	if len(variantIDs) > 0 {
		query = query.Where("variant_id IN ?", variantIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
