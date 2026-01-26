package repository

import (
	"context"

	"gorm.io/gorm"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
)

// ============================================================================
// Wishlist Item Repository Interface
// ============================================================================

// WishlistItemRepository defines the interface for wishlist item data operations
type WishlistItemRepository interface {
	// Create creates a new wishlist item
	Create(ctx context.Context, item *entity.WishlistItem) error

	// FindByID finds a wishlist item by ID
	FindByID(ctx context.Context, id uint) (*entity.WishlistItem, error)

	// FindByWishlistIDAndVariantID finds item by wishlist and variant
	FindByWishlistIDAndVariantID(
		ctx context.Context,
		wishlistID, variantID uint,
	) (*entity.WishlistItem, error)

	// Update updates a wishlist item
	Update(ctx context.Context, item *entity.WishlistItem) error

	// Delete deletes a wishlist item (soft delete)
	Delete(ctx context.Context, id uint) error

	// ExistsByWishlistIDAndVariantID checks if item exists in wishlist
	ExistsByWishlistIDAndVariantID(ctx context.Context, wishlistID, variantID uint) (bool, error)

	// IsVariantInUserWishlist checks if a variant is in any of user's wishlists
	IsVariantInUserWishlist(ctx context.Context, variantID, userID uint) (bool, error)

	// AreVariantsInUserWishlist checks if multiple variants are in any of user's wishlists
	// Returns a map of variantID -> isWishlisted for efficient batch lookup
	AreVariantsInUserWishlist(
		ctx context.Context,
		variantIDs []uint,
		userID uint,
	) (map[uint]bool, error)

	// CountByWishlistID counts items in a wishlist
	CountByWishlistID(ctx context.Context, wishlistID uint) (int64, error)

	// DeleteOldestByWishlistID deletes the oldest item in a wishlist
	DeleteOldestByWishlistID(ctx context.Context, wishlistID uint) error

	// FindVariantIDsByWishlistID finds all variant IDs for a wishlist with pagination
	// Returns variant IDs ordered by created_at DESC (newest first)
	FindVariantIDsByWishlistID(
		ctx context.Context,
		wishlistID uint,
		page, limit int,
	) ([]uint, int64, error)
}

// ============================================================================
// Wishlist Item Repository Implementation
// ============================================================================

type wishlistItemRepositoryImpl struct{}

// NewWishlistItemRepository creates a new instance of WishlistItemRepository
func NewWishlistItemRepository() WishlistItemRepository {
	return &wishlistItemRepositoryImpl{}
}

// getDB returns the database connection
func (r *wishlistItemRepositoryImpl) getDB() *gorm.DB {
	return db.GetDB()
}

// Create creates a new wishlist item
func (r *wishlistItemRepositoryImpl) Create(ctx context.Context, item *entity.WishlistItem) error {
	return r.getDB().WithContext(ctx).Create(item).Error
}

// FindByID finds a wishlist item by ID
func (r *wishlistItemRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.WishlistItem, error) {
	var item entity.WishlistItem
	err := r.getDB().WithContext(ctx).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// FindByWishlistIDAndVariantID finds item by wishlist and variant
func (r *wishlistItemRepositoryImpl) FindByWishlistIDAndVariantID(
	ctx context.Context,
	wishlistID, variantID uint,
) (*entity.WishlistItem, error) {
	var item entity.WishlistItem
	err := r.getDB().WithContext(ctx).
		Where("wishlist_id = ? AND variant_id = ?", wishlistID, variantID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Update updates a wishlist item
func (r *wishlistItemRepositoryImpl) Update(ctx context.Context, item *entity.WishlistItem) error {
	return r.getDB().WithContext(ctx).Save(item).Error
}

// Delete deletes a wishlist item (soft delete)
func (r *wishlistItemRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return r.getDB().WithContext(ctx).Delete(&entity.WishlistItem{}, id).Error
}

// ExistsByWishlistIDAndVariantID checks if item exists in wishlist
func (r *wishlistItemRepositoryImpl) ExistsByWishlistIDAndVariantID(
	ctx context.Context,
	wishlistID, variantID uint,
) (bool, error) {
	var count int64
	err := r.getDB().WithContext(ctx).
		Model(&entity.WishlistItem{}).
		Where("wishlist_id = ? AND variant_id = ?", wishlistID, variantID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsVariantInUserWishlist checks if a variant is in any of user's wishlists
func (r *wishlistItemRepositoryImpl) IsVariantInUserWishlist(
	ctx context.Context,
	variantID, userID uint,
) (bool, error) {
	var isWishlisted bool
	err := r.getDB().WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1 FROM wishlist_item wi
			INNER JOIN wishlist w ON w.id = wi.wishlist_id
			WHERE wi.variant_id = ?
			  AND w.user_id = ?
		)
	`, variantID, userID).Scan(&isWishlisted).Error
	if err != nil {
		return false, err
	}
	return isWishlisted, nil
}

// AreVariantsInUserWishlist checks if multiple variants are in any of user's wishlists
// Returns a map of variantID -> isWishlisted for efficient batch lookup
func (r *wishlistItemRepositoryImpl) AreVariantsInUserWishlist(
	ctx context.Context,
	variantIDs []uint,
	userID uint,
) (map[uint]bool, error) {
	result := make(map[uint]bool, len(variantIDs))

	// Initialize all to false
	for _, id := range variantIDs {
		result[id] = false
	}

	if len(variantIDs) == 0 {
		return result, nil
	}

	// Get all wishlisted variant IDs in a single query
	var wishlistedIDs []uint
	err := r.getDB().WithContext(ctx).Raw(`
		SELECT DISTINCT wi.variant_id
		FROM wishlist_item wi
		INNER JOIN wishlist w ON w.id = wi.wishlist_id
		WHERE wi.variant_id IN (?)
		  AND w.user_id = ?
	`, variantIDs, userID).Scan(&wishlistedIDs).Error
	if err != nil {
		return nil, err
	}

	// Mark wishlisted variants as true
	for _, id := range wishlistedIDs {
		result[id] = true
	}

	return result, nil
}

// CountByWishlistID counts items in a wishlist
func (r *wishlistItemRepositoryImpl) CountByWishlistID(
	ctx context.Context,
	wishlistID uint,
) (int64, error) {
	var count int64
	err := r.getDB().WithContext(ctx).
		Model(&entity.WishlistItem{}).
		Where("wishlist_id = ?", wishlistID).
		Count(&count).Error
	return count, err
}

// DeleteOldestByWishlistID deletes the oldest item in a wishlist
func (r *wishlistItemRepositoryImpl) DeleteOldestByWishlistID(
	ctx context.Context,
	wishlistID uint,
) error {
	// Find the oldest item (by created_at ASC)
	var oldestItem entity.WishlistItem
	err := r.getDB().WithContext(ctx).
		Where("wishlist_id = ?", wishlistID).
		Order("created_at ASC").
		First(&oldestItem).Error
	if err != nil {
		return err
	}

	// Delete the oldest item
	return r.getDB().WithContext(ctx).Delete(&entity.WishlistItem{}, oldestItem.ID).Error
}

// FindVariantIDsByWishlistID finds all variant IDs for a wishlist with pagination
// Returns variant IDs ordered by created_at DESC (newest first)
func (r *wishlistItemRepositoryImpl) FindVariantIDsByWishlistID(
	ctx context.Context,
	wishlistID uint,
	page, limit int,
) ([]uint, int64, error) {
	var total int64
	var variantIDs []uint

	// Count total items
	err := r.getDB().WithContext(ctx).
		Model(&entity.WishlistItem{}).
		Where("wishlist_id = ?", wishlistID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Get variant IDs with pagination
	offset := (page - 1) * limit
	err = r.getDB().WithContext(ctx).
		Model(&entity.WishlistItem{}).
		Select("variant_id").
		Where("wishlist_id = ?", wishlistID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Pluck("variant_id", &variantIDs).Error
	if err != nil {
		return nil, 0, err
	}

	return variantIDs, total, nil
}
