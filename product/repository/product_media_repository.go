package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	productError "ecommerce-be/product/error"

	"gorm.io/gorm"
)

// ProductMediaRepository defines data-access operations for the product_media table.
// All queries are scoped to a single product; no direct File-module table access.
type ProductMediaRepository interface {
	// Create inserts a new product-media association row.
	Create(ctx context.Context, media *entity.ProductMedia) error

	// FindByProductAndFile returns the link for a specific product+file pair,
	// or ErrProductMediaNotFound when no row exists.
	FindByProductAndFile(ctx context.Context, productID uint, fileID string) (*entity.ProductMedia, error)

	// FindByProductIDs returns all media rows for the supplied product IDs,
	// ordered by (product_id, display_order ASC, id ASC) for stable presentation.
	FindByProductIDs(ctx context.Context, productIDs []uint) ([]entity.ProductMedia, error)

	// UpdateMetadata patches is_primary and/or display_order for the given row ID.
	// Only non-nil pointer fields are written.
	UpdateMetadata(ctx context.Context, id uint, isPrimary *bool, displayOrder *int) error

	// UnsetPrimary sets is_primary = false for every media row of a product.
	// Used before promoting a new primary item.
	UnsetPrimary(ctx context.Context, productID uint) error

	// PromoteFallbackPrimary promotes the media row with the lowest display_order
	// (then lowest id) to primary, after the current primary has been removed.
	// No-ops gracefully when no rows remain.
	PromoteFallbackPrimary(ctx context.Context, productID uint) error

	// Delete removes a single product-media row by its internal ID.
	Delete(ctx context.Context, id uint) error
}

type productMediaRepository struct{}

// NewProductMediaRepository returns a GORM-backed ProductMediaRepository.
func NewProductMediaRepository() ProductMediaRepository {
	return &productMediaRepository{}
}

func (r *productMediaRepository) Create(ctx context.Context, media *entity.ProductMedia) error {
	result := db.GetDB().WithContext(ctx).Create(media)
	return result.Error
}

func (r *productMediaRepository) FindByProductAndFile(
	ctx context.Context,
	productID uint,
	fileID string,
) (*entity.ProductMedia, error) {
	var media entity.ProductMedia
	result := db.GetDB().WithContext(ctx).
		Where("product_id = ? AND file_id = ?", productID, fileID).
		First(&media)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, productError.ErrProductMediaNotFound
		}
		return nil, result.Error
	}
	return &media, nil
}

func (r *productMediaRepository) FindByProductIDs(
	ctx context.Context,
	productIDs []uint,
) ([]entity.ProductMedia, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}
	var items []entity.ProductMedia
	result := db.GetDB().WithContext(ctx).
		Where("product_id IN ?", productIDs).
		Order("product_id ASC, display_order ASC, id ASC").
		Find(&items)
	return items, result.Error
}

func (r *productMediaRepository) UpdateMetadata(
	ctx context.Context,
	id uint,
	isPrimary *bool,
	displayOrder *int,
) error {
	updates := map[string]any{}
	if isPrimary != nil {
		updates["is_primary"] = *isPrimary
	}
	if displayOrder != nil {
		updates["display_order"] = *displayOrder
	}
	if len(updates) == 0 {
		return nil
	}
	return db.GetDB().WithContext(ctx).
		Model(&entity.ProductMedia{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *productMediaRepository) UnsetPrimary(ctx context.Context, productID uint) error {
	return db.GetDB().WithContext(ctx).
		Model(&entity.ProductMedia{}).
		Where("product_id = ? AND is_primary = true", productID).
		Update("is_primary", false).Error
}

func (r *productMediaRepository) PromoteFallbackPrimary(ctx context.Context, productID uint) error {
	var candidate entity.ProductMedia
	result := db.GetDB().WithContext(ctx).
		Where("product_id = ?", productID).
		Order("display_order ASC, id ASC").
		First(&candidate)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return result.Error
	}
	return db.GetDB().WithContext(ctx).
		Model(&entity.ProductMedia{}).
		Where("id = ?", candidate.ID).
		Update("is_primary", true).Error
}

func (r *productMediaRepository) Delete(ctx context.Context, id uint) error {
	return db.GetDB().WithContext(ctx).
		Delete(&entity.ProductMedia{}, id).Error
}
