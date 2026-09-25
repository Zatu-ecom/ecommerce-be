package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

// DiscountCodeProductScopeRepository manages product-level rows on discount_code_product
// (variant_id IS NULL).
type DiscountCodeProductScopeRepository interface {
	AddProducts(ctx context.Context, products []entity.DiscountCodeProduct) error
	DeleteProducts(ctx context.Context, discountCodeID uint, productIDs []uint) error
	DeleteAllProducts(ctx context.Context, discountCodeID uint) error
	GetProducts(
		ctx context.Context,
		discountCodeID uint,
		productIDs []uint,
		offset, limit int,
	) ([]entity.DiscountCodeProduct, int64, error)
}

type DiscountCodeProductScopeRepositoryImpl struct{}

func NewDiscountCodeProductScopeRepository() DiscountCodeProductScopeRepository {
	return &DiscountCodeProductScopeRepositoryImpl{}
}

func (r *DiscountCodeProductScopeRepositoryImpl) AddProducts(
	ctx context.Context,
	products []entity.DiscountCodeProduct,
) error {
	if len(products) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&products).Error
}

func (r *DiscountCodeProductScopeRepositoryImpl) DeleteProducts(
	ctx context.Context,
	discountCodeID uint,
	productIDs []uint,
) error {
	if len(productIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("discount_code_id = ? AND product_id IN ? AND variant_id IS NULL", discountCodeID, productIDs).
		Delete(&entity.DiscountCodeProduct{}).Error
}

func (r *DiscountCodeProductScopeRepositoryImpl) DeleteAllProducts(
	ctx context.Context,
	discountCodeID uint,
) error {
	return db.DB(ctx).
		Where("discount_code_id = ? AND variant_id IS NULL", discountCodeID).
		Delete(&entity.DiscountCodeProduct{}).Error
}

func (r *DiscountCodeProductScopeRepositoryImpl) GetProducts(
	ctx context.Context,
	discountCodeID uint,
	productIDs []uint,
	offset, limit int,
) ([]entity.DiscountCodeProduct, int64, error) {
	var products []entity.DiscountCodeProduct
	var total int64

	query := db.DB(ctx).Model(&entity.DiscountCodeProduct{}).
		Where("discount_code_id = ? AND variant_id IS NULL", discountCodeID)

	if len(productIDs) > 0 {
		query = query.Where("product_id IN ?", productIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}
