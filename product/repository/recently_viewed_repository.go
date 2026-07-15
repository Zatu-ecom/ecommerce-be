package repository

import (
	"context"
	"errors"
	"fmt"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ============================================================================
// RecentlyViewedRepository Interface
// ============================================================================

// RecentlyViewedRepository defines the interface for recently viewed
// product database operations. Each method is focused on a single concern
// following the Interface Segregation Principle.
type RecentlyViewedRepository interface {
	// UpsertByUserAndProduct creates or updates a recently viewed record.
	// Uses ON CONFLICT (user_id, product_id) to atomically insert or update
	// the viewed_at timestamp and seller_id when a customer re-views a product.
	UpsertByUserAndProduct(
		ctx context.Context,
		userID, sellerID, productID uint,
	) error

	// CountByUserID returns the number of recently viewed records for a user.
	// Used by the service to determine if trimming is needed after an upsert.
	CountByUserID(ctx context.Context, userID uint) (int64, error)

	// DeleteOldestByUserID removes the oldest recently viewed records for a user
	// beyond the specified limit. Uses a subquery approach to atomically delete
	// only the rows that exceed the retention limit.
	DeleteOldestByUserID(
		ctx context.Context,
		userID uint,
		keepCount int,
	) error

	// FindByUserID retrieves recently viewed product IDs for a user,
	// ordered by viewed_at descending (most recent first).
	// The caller specifies the maximum number of records to return.
	FindByUserID(
		ctx context.Context,
		userID uint,
		limit int,
	) ([]entity.RecentlyViewed, error)
}

// ============================================================================
// RecentlyViewedRepositoryImpl Implementation
// ============================================================================

// RecentlyViewedRepositoryImpl implements RecentlyViewedRepository
// using GORM for PostgreSQL database access.
type RecentlyViewedRepositoryImpl struct{}

// NewRecentlyViewedRepository creates a new RecentlyViewedRepositoryImpl.
func NewRecentlyViewedRepository() RecentlyViewedRepository {
	return &RecentlyViewedRepositoryImpl{}
}

// UpsertByUserAndProduct creates or updates a recently viewed record atomically
// using GORM's OnConflict clause. This avoids the race condition inherent in
// SELECT-then-INSERT patterns.
func (r *RecentlyViewedRepositoryImpl) UpsertByUserAndProduct(
	ctx context.Context,
	userID, sellerID, productID uint,
) error {
	record := entity.RecentlyViewed{
		UserID:    userID,
		SellerID:  sellerID,
		ProductID: productID,
	}

	err := db.DB(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "product_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"viewed_at",
				"seller_id",
			}),
		}).
		Create(&record).Error
	if err != nil {
		return fmt.Errorf("failed to upsert recently viewed record: %w", err)
	}

	return nil
}

// CountByUserID returns the number of recently viewed records for a user.
func (r *RecentlyViewedRepositoryImpl) CountByUserID(
	ctx context.Context,
	userID uint,
) (int64, error) {
	var count int64
	err := db.DB(ctx).
		Model(&entity.RecentlyViewed{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count recently viewed records: %w", err)
	}

	return count, nil
}

// DeleteOldestByUserID removes the oldest recently viewed records for a user,
// keeping only the top `keepCount` most recent entries.
// Uses a subquery with a self-join to atomically identify and delete only
// the rows exceeding the retention limit.
func (r *RecentlyViewedRepositoryImpl) DeleteOldestByUserID(
	ctx context.Context,
	userID uint,
	keepCount int,
) error {
	// Subquery: select IDs of records to keep (most recent N)
	// Then delete all records for the user NOT in that set
	subQuery := db.DB(ctx).
		Model(&entity.RecentlyViewed{}).
		Where("user_id = ?", userID).
		Order("viewed_at DESC").
		Limit(keepCount).
		Select("id")

	err := db.DB(ctx).
		Where("user_id = ? AND id NOT IN (?)", userID, subQuery).
		Delete(&entity.RecentlyViewed{}).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to trim recently viewed records: %w", err)
	}

	return nil
}

// FindByUserID retrieves recently viewed records for a user,
// ordered by viewed_at descending (most recent first), limited to `limit` rows.
func (r *RecentlyViewedRepositoryImpl) FindByUserID(
	ctx context.Context,
	userID uint,
	limit int,
) ([]entity.RecentlyViewed, error) {
	var records []entity.RecentlyViewed
	err := db.DB(ctx).
		Where("user_id = ?", userID).
		Order("viewed_at DESC").
		Limit(limit).
		Find(&records).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to find recently viewed records: %w", err)
	}

	return records, nil
}
