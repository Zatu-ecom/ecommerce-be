package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	productError "ecommerce-be/product/error"

	"gorm.io/gorm"
)

// VariantMediaRepository defines data-access operations for the variant_media table.
// All queries are scoped to a single variant; no direct File-module table access.
type VariantMediaRepository interface {
	// Create inserts a new variant-media association row.
	Create(ctx context.Context, media *entity.VariantMedia) error

	// FindByVariantAndFile returns the link for a specific variant+file pair,
	// or ErrVariantMediaNotFound when no row exists.
	FindByVariantAndFile(ctx context.Context, variantID uint, fileID string) (*entity.VariantMedia, error)

	// FindByVariantIDs returns all media rows for the supplied variant IDs,
	// ordered by (variant_id, display_order ASC, id ASC) for stable presentation.
	FindByVariantIDs(ctx context.Context, variantIDs []uint) ([]entity.VariantMedia, error)

	// UpdateMetadata patches is_primary and/or display_order for the given row ID.
	UpdateMetadata(ctx context.Context, id uint, isPrimary *bool, displayOrder *int) error

	// UnsetPrimary sets is_primary = false for every media row of a variant.
	UnsetPrimary(ctx context.Context, variantID uint) error

	// PromoteFallbackPrimary promotes the lowest-order remaining item to primary
	// after the current primary has been removed. No-ops when no rows remain.
	PromoteFallbackPrimary(ctx context.Context, variantID uint) error

	// Delete removes a single variant-media row by its internal ID.
	Delete(ctx context.Context, id uint) error
}

type variantMediaRepository struct{}

// NewVariantMediaRepository returns a GORM-backed VariantMediaRepository.
func NewVariantMediaRepository() VariantMediaRepository {
	return &variantMediaRepository{}
}

func (r *variantMediaRepository) Create(ctx context.Context, media *entity.VariantMedia) error {
	return db.GetDB().WithContext(ctx).Create(media).Error
}

func (r *variantMediaRepository) FindByVariantAndFile(
	ctx context.Context,
	variantID uint,
	fileID string,
) (*entity.VariantMedia, error) {
	var media entity.VariantMedia
	result := db.GetDB().WithContext(ctx).
		Where("variant_id = ? AND file_id = ?", variantID, fileID).
		First(&media)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, productError.ErrVariantMediaNotFound
		}
		return nil, result.Error
	}
	return &media, nil
}

func (r *variantMediaRepository) FindByVariantIDs(
	ctx context.Context,
	variantIDs []uint,
) ([]entity.VariantMedia, error) {
	if len(variantIDs) == 0 {
		return nil, nil
	}
	var items []entity.VariantMedia
	result := db.GetDB().WithContext(ctx).
		Where("variant_id IN ?", variantIDs).
		Order("variant_id ASC, display_order ASC, id ASC").
		Find(&items)
	return items, result.Error
}

func (r *variantMediaRepository) UpdateMetadata(
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
		Model(&entity.VariantMedia{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *variantMediaRepository) UnsetPrimary(ctx context.Context, variantID uint) error {
	return db.GetDB().WithContext(ctx).
		Model(&entity.VariantMedia{}).
		Where("variant_id = ? AND is_primary = true", variantID).
		Update("is_primary", false).Error
}

func (r *variantMediaRepository) PromoteFallbackPrimary(ctx context.Context, variantID uint) error {
	var candidate entity.VariantMedia
	result := db.GetDB().WithContext(ctx).
		Where("variant_id = ?", variantID).
		Order("display_order ASC, id ASC").
		First(&candidate)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return result.Error
	}
	return db.GetDB().WithContext(ctx).
		Model(&entity.VariantMedia{}).
		Where("id = ?", candidate.ID).
		Update("is_primary", true).Error
}

func (r *variantMediaRepository) Delete(ctx context.Context, id uint) error {
	return db.GetDB().WithContext(ctx).
		Delete(&entity.VariantMedia{}, id).Error
}
