package repository

import (
	"context"
	"errors"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"

	"gorm.io/gorm"
)

// ListPromotionFilter represents the filters for listing promotions
type ListPromotionFilter struct {
	SellerID      uint
	Status        *entity.CampaignStatus
	PromotionType *entity.PromotionType
	AppliesTo     *entity.ScopeType
	Page          int
	Limit         int
}

// PromotionRepository defines the interface for promotion-related database operations
type PromotionRepository interface {
	Create(ctx context.Context, promotion *entity.Promotion) error
	FindByID(ctx context.Context, id uint) (*entity.Promotion, error)
	FindBySlug(ctx context.Context, slug string, sellerID uint) (*entity.Promotion, error)
	Exists(ctx context.Context, id uint) error
	FindActiveBySellerID(ctx context.Context, sellerID uint) ([]*entity.Promotion, error)
	Update(ctx context.Context, promotion *entity.Promotion) error
	UpdateStatus(ctx context.Context, id uint, status entity.CampaignStatus) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters ListPromotionFilter) ([]*entity.Promotion, int64, error)
	CountUsageByUser(ctx context.Context, promotionID uint, userID uint) (int, error)
	IncrementUsageAtomically(ctx context.Context, promotionID uint, usageLimit int) (bool, error)
	AutoStartPromotions(ctx context.Context, now time.Time) (int64, error)
	AutoEndPromotions(ctx context.Context, now time.Time) (int64, error)
}

// PromotionRepositoryImpl implements the PromotionRepository interface
type PromotionRepositoryImpl struct{}

// NewPromotionRepository creates a new instance of PromotionRepository
func NewPromotionRepository() PromotionRepository {
	return &PromotionRepositoryImpl{}
}

// Create creates a new promotion
func (r *PromotionRepositoryImpl) Create(ctx context.Context, promotion *entity.Promotion) error {
	return db.DB(ctx).Create(promotion).Error
}

// FindByID finds a promotion by ID
func (r *PromotionRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.Promotion, error) {
	var promotion entity.Promotion
	result := db.DB(ctx).Where("id = ?", id).First(&promotion)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, result.Error
	}
	return &promotion, nil
}

// FindBySlug finds a promotion by slug and seller ID
func (r *PromotionRepositoryImpl) FindBySlug(
	ctx context.Context,
	slug string,
	sellerID uint,
) (*entity.Promotion, error) {
	var promotion entity.Promotion
	result := db.DB(ctx).Where("slug = ? AND seller_id = ?", slug, sellerID).First(&promotion)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &promotion, nil
}

// Exists checks if a promotion exists by ID
func (r *PromotionRepositoryImpl) Exists(ctx context.Context, id uint) error {
	var count int64
	err := db.DB(ctx).Model(&entity.Promotion{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// FindActiveBySellerID returns all active promotions for a seller where current time is within date range
func (r *PromotionRepositoryImpl) FindActiveBySellerID(
	ctx context.Context,
	sellerID uint,
) ([]*entity.Promotion, error) {
	var promotions []*entity.Promotion
	now := time.Now()

	err := db.DB(ctx).
		Where("seller_id = ? AND status = ?", sellerID, entity.StatusActive).
		Where("starts_at <= ?", now).
		Where("ends_at IS NULL OR ends_at >= ?", now).
		Order("priority DESC").
		Find(&promotions).Error
	if err != nil {
		return nil, err
	}

	return promotions, nil
}

// Update updates a promotion
func (r *PromotionRepositoryImpl) Update(ctx context.Context, promotion *entity.Promotion) error {
	return db.DB(ctx).Save(promotion).Error
}

// UpdateStatus updates the status of a promotion
func (r *PromotionRepositoryImpl) UpdateStatus(
	ctx context.Context,
	id uint,
	status entity.CampaignStatus,
) error {
	return db.DB(ctx).Model(&entity.Promotion{}).Where("id = ?", id).Update("status", status).Error
}

// Delete soft deletes a promotion
func (r *PromotionRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Delete(&entity.Promotion{}, id).Error
}

// List returns a paginated list of promotions based on filters
func (r *PromotionRepositoryImpl) List(
	ctx context.Context,
	filters ListPromotionFilter,
) ([]*entity.Promotion, int64, error) {
	var promotions []*entity.Promotion
	var total int64

	query := db.DB(ctx).Model(&entity.Promotion{}).Where("seller_id = ?", filters.SellerID)

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.PromotionType != nil {
		query = query.Where("promotion_type = ?", *filters.PromotionType)
	}
	if filters.AppliesTo != nil {
		query = query.Where("applies_to = ?", *filters.AppliesTo)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (filters.Page - 1) * filters.Limit
	if err := query.Order("created_at DESC").Limit(filters.Limit).Offset(offset).Find(&promotions).Error; err != nil {
		return nil, 0, err
	}

	return promotions, total, nil
}

// CountUsageByUser counts how many times a specific user has used a promotion
func (r *PromotionRepositoryImpl) CountUsageByUser(
	ctx context.Context,
	promotionID uint,
	userID uint,
) (int, error) {
	var count int64
	err := db.DB(ctx).
		Model(&entity.PromotionUsage{}).
		Where("promotion_id = ? AND user_id = ?", promotionID, userID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// IncrementUsageAtomically atomically increments the usage count if under the limit.
// Returns true if the increment succeeded (row was updated), false if limit was reached.
func (r *PromotionRepositoryImpl) IncrementUsageAtomically(
	ctx context.Context,
	promotionID uint,
	usageLimit int,
) (bool, error) {
	result := db.DB(ctx).
		Model(&entity.Promotion{}).
		Where("id = ? AND current_usage_count < ?", promotionID, usageLimit).
		Update("current_usage_count", gorm.Expr("current_usage_count + 1"))
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// AutoStartPromotions sweeps for scheduled promotions that should be active based on time
func (r *PromotionRepositoryImpl) AutoStartPromotions(
	ctx context.Context,
	now time.Time,
) (int64, error) {
	result := db.DB(ctx).Model(&entity.Promotion{}).
		Where("status = ?", entity.StatusScheduled).
		Where("auto_start = ?", true).
		Where("starts_at <= ?", now).
		Update("status", entity.StatusActive)

	return result.RowsAffected, result.Error
}

// AutoEndPromotions sweeps for active promotions that should be ended based on time
func (r *PromotionRepositoryImpl) AutoEndPromotions(
	ctx context.Context,
	now time.Time,
) (int64, error) {
	result := db.DB(ctx).Model(&entity.Promotion{}).
		Where("status = ?", entity.StatusActive).
		Where("auto_end = ?", true).
		Where("ends_at IS NOT NULL").
		Where("ends_at <= ?", now).
		Update("status", entity.StatusEnded)

	return result.RowsAffected, result.Error
}
