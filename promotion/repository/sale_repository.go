package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"

	"gorm.io/gorm"
)

// SaleRepository defines the interface for sale-related database operations
type SaleRepository interface {
	Create(ctx context.Context, sale *entity.Sale) error
	Update(ctx context.Context, sale *entity.Sale) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*entity.Sale, error)
	FindBySlugAndSeller(ctx context.Context, slug string, sellerID uint) (*entity.Sale, error)
	FindAllBySellerID(ctx context.Context, sellerID uint) ([]entity.Sale, error)
	UpdateStatus(ctx context.Context, id uint, status entity.CampaignStatus) error
}

// SaleRepositoryImpl implements SaleRepository
type SaleRepositoryImpl struct{}

// NewSaleRepository creates a new SaleRepository
func NewSaleRepository() SaleRepository {
	return &SaleRepositoryImpl{}
}

func (r *SaleRepositoryImpl) Create(ctx context.Context, sale *entity.Sale) error {
	return db.DB(ctx).Create(sale).Error
}

func (r *SaleRepositoryImpl) Update(ctx context.Context, sale *entity.Sale) error {
	return db.DB(ctx).Model(sale).
		Select("Name", "Description", "Slug", "BannerImages", "Status", "StartAt", "EndAt", "UpdatedAt").
		Updates(map[string]any{
			"name":          sale.Name,
			"description":   sale.Description,
			"slug":          sale.Slug,
			"banner_images": sale.BannerImages,
			"status":        sale.Status,
			"start_at":      sale.StartAt,
			"end_at":        sale.EndAt,
			"updated_at":    sale.UpdatedAt,
		}).Error
}

func (r *SaleRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Delete(&entity.Sale{}, id).Error
}

func (r *SaleRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Sale, error) {
	var sale entity.Sale
	result := db.DB(ctx).Where("id = ?", id).First(&sale)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, promoErrors.ErrSaleNotFound
		}
		return nil, result.Error
	}
	return &sale, nil
}

func (r *SaleRepositoryImpl) FindBySlugAndSeller(
	ctx context.Context,
	slug string,
	sellerID uint,
) (*entity.Sale, error) {
	var sale entity.Sale
	result := db.DB(ctx).Where("slug = ? AND seller_id = ?", slug, sellerID).First(&sale)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &sale, nil
}

func (r *SaleRepositoryImpl) FindAllBySellerID(
	ctx context.Context,
	sellerID uint,
) ([]entity.Sale, error) {
	var sales []entity.Sale
	err := db.DB(ctx).
		Where("seller_id = ?", sellerID).
		Order("name ASC").
		Find(&sales).Error
	if err != nil {
		return nil, err
	}
	return sales, nil
}

func (r *SaleRepositoryImpl) UpdateStatus(
	ctx context.Context,
	id uint,
	status entity.CampaignStatus,
) error {
	return db.DB(ctx).
		Model(&entity.Sale{}).
		Where("id = ?", id).
		Update("status", status).Error
}
