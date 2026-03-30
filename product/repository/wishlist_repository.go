package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/mapper"

	"gorm.io/gorm"
)

// WishlistRepository defines the interface for wishlist-related database operations
type WishlistRepository interface {
	Create(ctx context.Context, wishlist *entity.Wishlist) error
	FindByID(ctx context.Context, id uint) (*entity.Wishlist, error)
	FindByIDWithItems(ctx context.Context, id uint) (*entity.Wishlist, error)
	FindByUserID(ctx context.Context, userID uint) ([]entity.Wishlist, error)
	FindByUserIDWithItemCount(
		ctx context.Context,
		userID uint,
	) ([]mapper.WishlistWithItemCount, error)
	FindByUserIDAndName(ctx context.Context, userID uint, name string) (*entity.Wishlist, error)
	Update(ctx context.Context, wishlist *entity.Wishlist) error
	Delete(ctx context.Context, id uint) error
	ClearDefaultForUser(ctx context.Context, userID uint) error
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	CountItemsByWishlistID(ctx context.Context, wishlistID uint) (int64, error)
}

// WishlistRepositoryImpl implements the WishlistRepository interface
type WishlistRepositoryImpl struct{}

// NewWishlistRepository creates a new instance of WishlistRepository
func NewWishlistRepository() WishlistRepository {
	return &WishlistRepositoryImpl{}
}

// Create creates a new wishlist
func (r *WishlistRepositoryImpl) Create(ctx context.Context, wishlist *entity.Wishlist) error {
	return db.DB(ctx).Create(wishlist).Error
}

// FindByID finds a wishlist by ID
func (r *WishlistRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Wishlist, error) {
	var wishlist entity.Wishlist
	result := db.DB(ctx).Where("id = ?", id).First(&wishlist)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrWishlistNotFound
		}
		return nil, result.Error
	}
	return &wishlist, nil
}

// FindByIDWithItems finds a wishlist by ID with its items
func (r *WishlistRepositoryImpl) FindByIDWithItems(
	ctx context.Context,
	id uint,
) (*entity.Wishlist, error) {
	var wishlist entity.Wishlist
	result := db.DB(ctx).
		Preload("Items").
		Where("id = ?", id).
		First(&wishlist)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrWishlistNotFound
		}
		return nil, result.Error
	}
	return &wishlist, nil
}

// FindByUserID finds all wishlists for a user
func (r *WishlistRepositoryImpl) FindByUserID(
	ctx context.Context,
	userID uint,
) ([]entity.Wishlist, error) {
	var wishlists []entity.Wishlist
	result := db.DB(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, created_at ASC").
		Find(&wishlists)
	if result.Error != nil {
		return nil, result.Error
	}
	return wishlists, nil
}

// FindByUserIDWithItemCount finds all wishlists for a user with item counts in a single query
func (r *WishlistRepositoryImpl) FindByUserIDWithItemCount(
	ctx context.Context,
	userID uint,
) ([]mapper.WishlistWithItemCount, error) {
	var wishlists []mapper.WishlistWithItemCount
	result := db.DB(ctx).
		Table("wishlist w").
		Select(`
			w.id,
			w.user_id,
			w.name,
			w.is_default,
			w.created_at,
			w.updated_at,
			COALESCE(COUNT(wi.id), 0) AS item_count
		`).
		Joins("LEFT JOIN wishlist_item wi ON wi.wishlist_id = w.id").
		Where("w.user_id = ?", userID).
		Group("w.id, w.user_id, w.name, w.is_default, w.created_at, w.updated_at").
		Order("w.is_default DESC, w.created_at ASC").
		Scan(&wishlists)
	if result.Error != nil {
		return nil, result.Error
	}
	return wishlists, nil
}

// FindByUserIDAndName finds a wishlist by user ID and name
func (r *WishlistRepositoryImpl) FindByUserIDAndName(
	ctx context.Context,
	userID uint,
	name string,
) (*entity.Wishlist, error) {
	var wishlist entity.Wishlist
	result := db.DB(ctx).
		Where("user_id = ? AND name = ?", userID, name).
		First(&wishlist)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Not found is not an error for this check
		}
		return nil, result.Error
	}
	return &wishlist, nil
}

// CountByUserID counts wishlists for a user
func (r *WishlistRepositoryImpl) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	result := db.DB(ctx).
		Model(&entity.Wishlist{}).
		Where("user_id = ?", userID).
		Count(&count)
	return count, result.Error
}

// CountItemsByWishlistID counts items in a wishlist
func (r *WishlistRepositoryImpl) CountItemsByWishlistID(
	ctx context.Context,
	wishlistID uint,
) (int64, error) {
	var count int64
	result := db.DB(ctx).
		Model(&entity.WishlistItem{}).
		Where("wishlist_id = ?", wishlistID).
		Count(&count)
	return count, result.Error
}

// Update updates a wishlist
func (r *WishlistRepositoryImpl) Update(ctx context.Context, wishlist *entity.Wishlist) error {
	return db.DB(ctx).Save(wishlist).Error
}

// Delete soft deletes a wishlist and its items
func (r *WishlistRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete wishlist items first
		if err := tx.Where("wishlist_id = ?", id).Delete(&entity.WishlistItem{}).Error; err != nil {
			return err
		}
		// Delete wishlist
		return tx.Delete(&entity.Wishlist{}, id).Error
	})
}

// ClearDefaultForUser clears the default flag for all wishlists of a user
func (r *WishlistRepositoryImpl) ClearDefaultForUser(ctx context.Context, userID uint) error {
	return db.DB(ctx).
		Model(&entity.Wishlist{}).
		Where("user_id = ? AND is_default = ?", userID, true).
		Update("is_default", false).Error
}
