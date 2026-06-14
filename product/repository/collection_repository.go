package repository

import (
	"context"
	"errors"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"

	"gorm.io/gorm"
)

// CollectionRepository defines the interface for collection-related database operations
type CollectionRepository interface {
	Create(ctx context.Context, collection *entity.Collection) error
	Update(ctx context.Context, collection *entity.Collection) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*entity.Collection, error)
	FindAll(ctx context.Context, sellerID *uint) ([]entity.Collection, error)
	FindBySlugAndSeller(ctx context.Context, slug string, sellerID uint) (*entity.Collection, error)
	CountProducts(ctx context.Context, collectionID uint) (int64, error)
	Exists(ctx context.Context, id uint) error
}

// CollectionRepositoryImpl implements CollectionRepository
type CollectionRepositoryImpl struct{}

// NewCollectionRepository creates a new CollectionRepository
func NewCollectionRepository() CollectionRepository {
	return &CollectionRepositoryImpl{}
}

func (r *CollectionRepositoryImpl) Create(ctx context.Context, collection *entity.Collection) error {
	return db.DB(ctx).Create(collection).Error
}

func (r *CollectionRepositoryImpl) Update(ctx context.Context, collection *entity.Collection) error {
	return db.DB(ctx).Model(collection).
		Select("Name", "Description", "ImageFileID", "IsActive", "UpdatedAt").
		Updates(map[string]any{
			"name":          collection.Name,
			"description":   collection.Description,
			"image_file_id": collection.ImageFileID,
			"is_active":     collection.IsActive,
			"updated_at":    collection.UpdatedAt,
		}).Error
}

func (r *CollectionRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return db.DB(ctx).Delete(&entity.Collection{}, id).Error
}

func (r *CollectionRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Collection, error) {
	var collection entity.Collection
	result := db.DB(ctx).Where("id = ?", id).First(&collection)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrCollectionNotFound
		}
		return nil, result.Error
	}
	return &collection, nil
}

func (r *CollectionRepositoryImpl) FindAll(ctx context.Context, sellerID *uint) ([]entity.Collection, error) {
	var collections []entity.Collection
	q := db.DB(ctx).Model(&entity.Collection{})
	if sellerID != nil {
		q = q.Where("seller_id = ?", *sellerID)
	}
	if err := q.Order("name ASC").Find(&collections).Error; err != nil {
		return nil, err
	}
	return collections, nil
}

func (r *CollectionRepositoryImpl) FindBySlugAndSeller(
	ctx context.Context,
	slug string,
	sellerID uint,
) (*entity.Collection, error) {
	var collection entity.Collection
	result := db.DB(ctx).Where("slug = ? AND seller_id = ?", slug, sellerID).First(&collection)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &collection, nil
}

func (r *CollectionRepositoryImpl) CountProducts(ctx context.Context, collectionID uint) (int64, error) {
	var count int64
	err := db.DB(ctx).
		Model(&entity.CollectionProduct{}).
		Where("collection_id = ?", collectionID).
		Count(&count).Error
	return count, err
}

func (r *CollectionRepositoryImpl) Exists(ctx context.Context, id uint) error {
	var count int64
	result := db.DB(ctx).Model(&entity.Collection{}).Where("id = ?", id).Count(&count)
	if result.Error != nil {
		return result.Error
	}
	if count == 0 {
		return prodErrors.ErrCollectionNotFound
	}
	return nil
}
