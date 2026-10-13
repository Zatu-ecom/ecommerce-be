package repository

import (
	"context"
	"errors"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"

	"gorm.io/gorm"
)

// ListDiscountCodeFilter represents the filters for listing discount codes
type ListDiscountCodeFilter struct {
	SellerID     uint
	IsActive     *bool
	DiscountType *entity.DiscountType
	AppliesTo    *entity.ScopeType
	Page         int
	Limit        int
}

// DiscountCodeRepository defines the interface for discount-code database operations
type DiscountCodeRepository interface {
	Create(ctx context.Context, code *entity.DiscountCode) error
	FindByID(ctx context.Context, id uint) (*entity.DiscountCode, error)
	FindByCode(ctx context.Context, sellerID uint, code string) (*entity.DiscountCode, error)
	List(ctx context.Context, filters ListDiscountCodeFilter) ([]*entity.DiscountCode, int64, error)
	Update(ctx context.Context, code *entity.DiscountCode) error
	UpdateActive(ctx context.Context, id uint, sellerID uint, isActive bool) error
	Delete(ctx context.Context, id uint) error
	CountUsage(ctx context.Context, discountCodeID uint) (int64, error)
	IncrementUsage(ctx context.Context, discountCodeID uint) error
	IncrementUsageAtomically(ctx context.Context, discountCodeID uint, usageLimit int) (bool, error)
	AutoStartDiscountCodes(ctx context.Context, now time.Time) (int64, error)
	AutoEndDiscountCodes(ctx context.Context, now time.Time) (int64, error)
}

// DiscountCodeRepositoryImpl implements DiscountCodeRepository
type DiscountCodeRepositoryImpl struct{}

// NewDiscountCodeRepository creates a new DiscountCodeRepository
func NewDiscountCodeRepository() DiscountCodeRepository {
	return &DiscountCodeRepositoryImpl{}
}

func (r *DiscountCodeRepositoryImpl) Create(ctx context.Context, code *entity.DiscountCode) error {
	return db.DB(ctx).Create(code).Error
}

func (r *DiscountCodeRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.DiscountCode, error) {
	var code entity.DiscountCode
	result := db.DB(ctx).Where("id = ?", id).First(&code)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, promoErrors.ErrDiscountCodeNotFound
		}
		return nil, result.Error
	}
	return &code, nil
}

func (r *DiscountCodeRepositoryImpl) FindByCode(
	ctx context.Context,
	sellerID uint,
	code string,
) (*entity.DiscountCode, error) {
	var discountCode entity.DiscountCode
	result := db.DB(ctx).
		Where("seller_id = ? AND code = ?", sellerID, code).
		First(&discountCode)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &discountCode, nil
}

func (r *DiscountCodeRepositoryImpl) List(
	ctx context.Context,
	filters ListDiscountCodeFilter,
) ([]*entity.DiscountCode, int64, error) {
	var codes []*entity.DiscountCode
	var total int64

	query := db.DB(ctx).Model(&entity.DiscountCode{}).Where("seller_id = ?", filters.SellerID)

	if filters.IsActive != nil {
		query = query.Where("is_active = ?", *filters.IsActive)
	}
	if filters.DiscountType != nil {
		query = query.Where("discount_type = ?", *filters.DiscountType)
	}
	if filters.AppliesTo != nil {
		query = query.Where("applies_to = ?", *filters.AppliesTo)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&codes).Error; err != nil {
		return nil, 0, err
	}

	return codes, total, nil
}

func (r *DiscountCodeRepositoryImpl) Update(ctx context.Context, code *entity.DiscountCode) error {
	return db.DB(ctx).Model(code).
		Select(
			"Title",
			"Description",
			"DiscountType",
			"Value",
			"MaxDiscountAmountCents",
			"AppliesTo",
			"MinPurchaseAmountCents",
			"MinQuantity",
			"CustomerEligibility",
			"CustomerSegmentID",
			"UsageLimitTotal",
			"UsageLimitPerCustomer",
			"UsageResetTimeType",
			"UsageResetAmount",
			"CanCombineWithOtherDiscounts",
			"StartsAt",
			"EndsAt",
			"IsActive",
			"AutoStart",
			"AutoEnd",
			"Metadata",
			"UpdatedAt",
		).
		Updates(code).Error
}

func (r *DiscountCodeRepositoryImpl) UpdateActive(
	ctx context.Context,
	id uint,
	sellerID uint,
	isActive bool,
) error {
	updates := map[string]any{"is_active": isActive}
	// Prevent cron from re-activating a manually deactivated code.
	if !isActive {
		updates["auto_start"] = false
	}
	return db.DB(ctx).
		Model(&entity.DiscountCode{}).
		Where("id = ? AND seller_id = ?", id, sellerID).
		Updates(updates).Error
}

// AutoStartDiscountCodes activates inactive codes whose start window has begun.
func (r *DiscountCodeRepositoryImpl) AutoStartDiscountCodes(
	ctx context.Context,
	now time.Time,
) (int64, error) {
	result := db.DB(ctx).Model(&entity.DiscountCode{}).
		Where("is_active = ?", false).
		Where("auto_start = ?", true).
		Where("starts_at <= ?", now).
		Where("ends_at IS NULL OR ends_at > ?", now).
		Update("is_active", true)

	return result.RowsAffected, result.Error
}

// AutoEndDiscountCodes deactivates active codes whose end window has passed.
func (r *DiscountCodeRepositoryImpl) AutoEndDiscountCodes(
	ctx context.Context,
	now time.Time,
) (int64, error) {
	result := db.DB(ctx).Model(&entity.DiscountCode{}).
		Where("is_active = ?", true).
		Where("auto_end = ?", true).
		Where("ends_at IS NOT NULL").
		Where("ends_at <= ?", now).
		Update("is_active", false)

	return result.RowsAffected, result.Error
}

func (r *DiscountCodeRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Delete(&entity.DiscountCode{}, id).Error
}

func (r *DiscountCodeRepositoryImpl) CountUsage(
	ctx context.Context,
	discountCodeID uint,
) (int64, error) {
	var count int64
	err := db.DB(ctx).
		Model(&entity.DiscountCodeUsage{}).
		Where("discount_code_id = ?", discountCodeID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// IncrementUsage unconditionally increments current_usage_count (unlimited codes).
func (r *DiscountCodeRepositoryImpl) IncrementUsage(
	ctx context.Context,
	discountCodeID uint,
) error {
	result := db.DB(ctx).
		Model(&entity.DiscountCode{}).
		Where("id = ?", discountCodeID).
		Update("current_usage_count", gorm.Expr("current_usage_count + 1"))
	return result.Error
}

// IncrementUsageAtomically atomically increments current_usage_count if under the limit.
// Returns true if the increment succeeded (row was updated), false if limit was reached.
func (r *DiscountCodeRepositoryImpl) IncrementUsageAtomically(
	ctx context.Context,
	discountCodeID uint,
	usageLimit int,
) (bool, error) {
	result := db.DB(ctx).
		Model(&entity.DiscountCode{}).
		Where("id = ? AND current_usage_count < ?", discountCodeID, usageLimit).
		Update("current_usage_count", gorm.Expr("current_usage_count + 1"))
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
