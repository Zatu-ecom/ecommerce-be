package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

type DiscountCodeCategoryScopeRepository interface {
	AddCategories(ctx context.Context, categories []entity.DiscountCodeCategory) error
	DeleteCategories(ctx context.Context, discountCodeID uint, categoryIDs []uint) error
	DeleteAllCategories(ctx context.Context, discountCodeID uint) error
	GetCategories(
		ctx context.Context,
		discountCodeID uint,
		categoryIDs []uint,
		offset, limit int,
	) ([]entity.DiscountCodeCategory, int64, error)
}

type DiscountCodeCategoryScopeRepositoryImpl struct{}

func NewDiscountCodeCategoryScopeRepository() DiscountCodeCategoryScopeRepository {
	return &DiscountCodeCategoryScopeRepositoryImpl{}
}

func (r *DiscountCodeCategoryScopeRepositoryImpl) AddCategories(
	ctx context.Context,
	categories []entity.DiscountCodeCategory,
) error {
	if len(categories) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&categories).Error
}

func (r *DiscountCodeCategoryScopeRepositoryImpl) DeleteCategories(
	ctx context.Context,
	discountCodeID uint,
	categoryIDs []uint,
) error {
	if len(categoryIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("discount_code_id = ? AND category_id IN ?", discountCodeID, categoryIDs).
		Delete(&entity.DiscountCodeCategory{}).Error
}

func (r *DiscountCodeCategoryScopeRepositoryImpl) DeleteAllCategories(
	ctx context.Context,
	discountCodeID uint,
) error {
	return db.DB(ctx).
		Where("discount_code_id = ?", discountCodeID).
		Delete(&entity.DiscountCodeCategory{}).Error
}

func (r *DiscountCodeCategoryScopeRepositoryImpl) GetCategories(
	ctx context.Context,
	discountCodeID uint,
	categoryIDs []uint,
	offset, limit int,
) ([]entity.DiscountCodeCategory, int64, error) {
	var categories []entity.DiscountCodeCategory
	var total int64

	query := db.DB(ctx).Model(&entity.DiscountCodeCategory{}).
		Where("discount_code_id = ?", discountCodeID)

	if len(categoryIDs) > 0 {
		query = query.Where("category_id IN ?", categoryIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset(offset).Limit(limit).Find(&categories).Error; err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}
