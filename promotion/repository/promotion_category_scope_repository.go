package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

// PromotionCategoryScopeRepository defines the interface for promotion-category scope operations
type PromotionCategoryScopeRepository interface {
	AddPromotionCategories(ctx context.Context, categories []entity.PromotionCategory) error
	DeletePromotionCategories(ctx context.Context, promotionID uint, categoryIDs []uint) error
	DeletePromotionCategoryByPromotionID(ctx context.Context, promotionID uint) error
	GetPromotionCategories(
		ctx context.Context,
		promotionID uint,
		categoryIDs []uint,
		offset, limit int,
	) ([]entity.PromotionCategory, int64, error)
}

// PromotionCategoryScopeRepositoryImpl implements the PromotionCategoryRepository interface
type PromotionCategoryScopeRepositoryImpl struct{}

// NewPromotionCategoryScopeRepository creates a new instance of PromotionCategoryRepository
func NewPromotionCategoryScopeRepository() PromotionCategoryScopeRepository {
	return &PromotionCategoryScopeRepositoryImpl{}
}

// AddPromotionCategories bulk inserts promotion-category mappings
func (r *PromotionCategoryScopeRepositoryImpl) AddPromotionCategories(
	ctx context.Context,
	categories []entity.PromotionCategory,
) error {
	if len(categories) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&categories).Error
}

// DeletePromotionCategories deletes specific category mappings from a promotion
func (r *PromotionCategoryScopeRepositoryImpl) DeletePromotionCategories(
	ctx context.Context,
	promotionID uint,
	categoryIDs []uint,
) error {
	if len(categoryIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("promotion_id = ? AND category_id IN ?", promotionID, categoryIDs).
		Delete(&entity.PromotionCategory{}).Error
}

// DeletePromotionCategoryByPromotionID removes all category mappings for a promotion
func (r *PromotionCategoryScopeRepositoryImpl) DeletePromotionCategoryByPromotionID(
	ctx context.Context,
	promotionID uint,
) error {
	return db.DB(ctx).
		Where("promotion_id = ?", promotionID).
		Delete(&entity.PromotionCategory{}).Error
}

// GetPromotionCategories retrieves all category mappings for a promotion with pagination
func (r *PromotionCategoryScopeRepositoryImpl) GetPromotionCategories(
	ctx context.Context,
	promotionID uint,
	categoryIDs []uint,
	offset, limit int,
) ([]entity.PromotionCategory, int64, error) {
	var categories []entity.PromotionCategory
	var total int64

	query := db.DB(ctx).Model(&entity.PromotionCategory{}).Where("promotion_id = ?", promotionID)

	if len(categoryIDs) > 0 {
		query = query.Where("category_id IN ?", categoryIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Find(&categories).Error
	if err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}
