package repository

import (
	"context"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
)

// DiscountCodeUsageRepository defines the interface for discount-code usage operations
type DiscountCodeUsageRepository interface {
	Create(ctx context.Context, usage *entity.DiscountCodeUsage) error
	CountByUserInWindow(
		ctx context.Context,
		discountCodeID uint,
		userID uint,
		windowStart *time.Time,
	) (int64, error)
}

// DiscountCodeUsageRepositoryImpl implements DiscountCodeUsageRepository
type DiscountCodeUsageRepositoryImpl struct{}

// NewDiscountCodeUsageRepository creates a new DiscountCodeUsageRepository
func NewDiscountCodeUsageRepository() DiscountCodeUsageRepository {
	return &DiscountCodeUsageRepositoryImpl{}
}

func (r *DiscountCodeUsageRepositoryImpl) Create(
	ctx context.Context,
	usage *entity.DiscountCodeUsage,
) error {
	return db.DB(ctx).Create(usage).Error
}

// CountByUserInWindow counts usages for a user+code. When windowStart is nil, counts all-time
// (lifetime / ResetTimeType none). When set, only rows with used_at >= windowStart are counted.
func (r *DiscountCodeUsageRepositoryImpl) CountByUserInWindow(
	ctx context.Context,
	discountCodeID uint,
	userID uint,
	windowStart *time.Time,
) (int64, error) {
	var count int64
	query := db.DB(ctx).
		Model(&entity.DiscountCodeUsage{}).
		Where("discount_code_id = ? AND user_id = ?", discountCodeID, userID)

	if windowStart != nil {
		query = query.Where("used_at >= ?", *windowStart)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
