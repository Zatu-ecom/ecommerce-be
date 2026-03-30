package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

// PromotionProductScopeRepository defines the interface for promotion-product scope operations
type PromotionProductScopeRepository interface {
	AddPromotionProducts(ctx context.Context, promotionProducts []entity.PromotionProduct) error
	DeletePromotionProducts(ctx context.Context, promotionID uint, productIDs []uint) error
	DeletePromotionProductByPromotionID(ctx context.Context, promotionID uint) error
	GetPromotionProducts(
		ctx context.Context,
		promotionID uint,
		productIDs []uint,
		offset, limit int,
	) ([]entity.PromotionProduct, int64, error)
}

// PromotionProductScopeRepositoryImpl implements the PromotionProductRepository interface
type PromotionProductScopeRepositoryImpl struct{}

// NewPromotionProductScopeRepository creates a new instance of PromotionProductRepository
func NewPromotionProductScopeRepository() PromotionProductScopeRepository {
	return &PromotionProductScopeRepositoryImpl{}
}

// AddPromotionProducts bulk inserts promotion-product mappings
func (r *PromotionProductScopeRepositoryImpl) AddPromotionProducts(
	ctx context.Context,
	promotionProducts []entity.PromotionProduct,
) error {
	if len(promotionProducts) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&promotionProducts).Error
}

// DeletePromotionProducts deletes specific product mappings from a promotion
func (r *PromotionProductScopeRepositoryImpl) DeletePromotionProducts(
	ctx context.Context,
	promotionID uint,
	productIDs []uint,
) error {
	if len(productIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("promotion_id = ? AND product_id IN ?", promotionID, productIDs).
		Delete(&entity.PromotionProduct{}).Error
}

// DeletePromotionProductByPromotionID removes all product mappings for a promotion
func (r *PromotionProductScopeRepositoryImpl) DeletePromotionProductByPromotionID(
	ctx context.Context,
	promotionID uint,
) error {
	return db.DB(ctx).
		Where("promotion_id = ?", promotionID).
		Delete(&entity.PromotionProduct{}).Error
}

// GetPromotionProducts retrieves all product mappings for a promotion with pagination
func (r *PromotionProductScopeRepositoryImpl) GetPromotionProducts(
	ctx context.Context,
	promotionID uint,
	productIDs []uint,
	offset, limit int,
) ([]entity.PromotionProduct, int64, error) {
	var products []entity.PromotionProduct
	var total int64

	query := db.DB(ctx).Model(&entity.PromotionProduct{}).Where("promotion_id = ?", promotionID)

	if len(productIDs) > 0 {
		query = query.Where("product_id IN ?", productIDs)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Offset(offset).Limit(limit).Find(&products).Error
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
