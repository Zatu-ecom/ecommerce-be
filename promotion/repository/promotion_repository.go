package repository

import (
	"context"
	"errors"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"

	"gorm.io/gorm"
)

// PromotionRepository defines the interface for promotion-related database operations
type PromotionRepository interface {
	Create(ctx context.Context, promotion *entity.Promotion) error
	FindByID(ctx context.Context, id uint) (*entity.Promotion, error)
	FindBySlug(ctx context.Context, slug string, sellerID uint) (*entity.Promotion, error)
	Exists(ctx context.Context, id uint) error
	FindActiveBySellerID(ctx context.Context, sellerID uint) ([]*entity.Promotion, error)
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
