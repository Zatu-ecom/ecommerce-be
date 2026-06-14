package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/error"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CollectionProductPositionUpdate represents a position update for a collection product
type CollectionProductPositionUpdate struct {
	ProductID uint
	Position  int
}

// CollectionProductRepository defines the interface for collection-product operations
type CollectionProductRepository interface {
	GetProductIDsByCollectionIDs(ctx context.Context, collectionIDs []uint) ([]uint, error)
	AddProducts(ctx context.Context, products []entity.CollectionProduct) error
	RemoveProducts(ctx context.Context, collectionID uint, productIDs []uint) error
	GetProductsByCollectionID(
		ctx context.Context,
		collectionID uint,
		productIDs []uint,
		offset, limit int,
	) ([]entity.CollectionProduct, int64, error)
	GetMaxPosition(ctx context.Context, collectionID uint) (int, error)
	GetExistingProductIDsInCollection(
		ctx context.Context,
		collectionID uint,
		productIDs []uint,
	) ([]uint, error)
	UpdatePositions(ctx context.Context, collectionID uint, items []CollectionProductPositionUpdate) error
}

// CollectionProductRepositoryImpl implements the CollectionProductRepository interface
type CollectionProductRepositoryImpl struct{}

// NewCollectionProductRepository creates a new instance of CollectionProductRepository
func NewCollectionProductRepository() CollectionProductRepository {
	return &CollectionProductRepositoryImpl{}
}

// GetProductIDsByCollectionIDs returns distinct product IDs belonging to the given collection IDs
func (r *CollectionProductRepositoryImpl) GetProductIDsByCollectionIDs(
	ctx context.Context,
	collectionIDs []uint,
) ([]uint, error) {
	var productIDs []uint
	err := db.DB(ctx).
		Model(&entity.CollectionProduct{}).
		Where("collection_id IN ?", collectionIDs).
		Distinct("product_id").
		Pluck("product_id", &productIDs).Error
	if err != nil {
		return nil, err
	}
	return productIDs, nil
}

// AddProducts bulk inserts collection-product mappings, ignoring duplicates
func (r *CollectionProductRepositoryImpl) AddProducts(
	ctx context.Context,
	products []entity.CollectionProduct,
) error {
	if len(products) == 0 {
		return nil
	}
	return db.DB(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "collection_id"}, {Name: "product_id"}},
			DoNothing: true,
		}).
		Create(&products).Error
}

// RemoveProducts deletes specific product mappings from a collection
func (r *CollectionProductRepositoryImpl) RemoveProducts(
	ctx context.Context,
	collectionID uint,
	productIDs []uint,
) error {
	if len(productIDs) == 0 {
		return nil
	}
	return db.DB(ctx).
		Where("collection_id = ? AND product_id IN ?", collectionID, productIDs).
		Delete(&entity.CollectionProduct{}).Error
}

// GetProductsByCollectionID retrieves products in a collection with pagination
func (r *CollectionProductRepositoryImpl) GetProductsByCollectionID(
	ctx context.Context,
	collectionID uint,
	productIDs []uint,
	offset, limit int,
) ([]entity.CollectionProduct, int64, error) {
	var products []entity.CollectionProduct
	var total int64

	query := db.DB(ctx).Model(&entity.CollectionProduct{}).Where("collection_id = ?", collectionID)
	if len(productIDs) > 0 {
		query = query.Where("product_id IN ?", productIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Product").
		Order("position ASC, created_at ASC").
		Offset(offset).
		Limit(limit).
		Find(&products).Error
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// GetMaxPosition returns the highest position in a collection, or -1 if empty
func (r *CollectionProductRepositoryImpl) GetMaxPosition(
	ctx context.Context,
	collectionID uint,
) (int, error) {
	var maxPosition *int
	err := db.DB(ctx).
		Model(&entity.CollectionProduct{}).
		Where("collection_id = ?", collectionID).
		Select("MAX(position)").
		Scan(&maxPosition).Error
	if err != nil {
		return 0, err
	}
	if maxPosition == nil {
		return -1, nil
	}
	return *maxPosition, nil
}

// GetExistingProductIDsInCollection returns which of the given product IDs are already in the collection
func (r *CollectionProductRepositoryImpl) GetExistingProductIDsInCollection(
	ctx context.Context,
	collectionID uint,
	productIDs []uint,
) ([]uint, error) {
	if len(productIDs) == 0 {
		return nil, nil
	}
	var existing []uint
	err := db.DB(ctx).
		Model(&entity.CollectionProduct{}).
		Where("collection_id = ? AND product_id IN ?", collectionID, productIDs).
		Pluck("product_id", &existing).Error
	return existing, err
}

// UpdatePositions updates display positions for products in a collection
func (r *CollectionProductRepositoryImpl) UpdatePositions(
	ctx context.Context,
	collectionID uint,
	items []CollectionProductPositionUpdate,
) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			result := tx.Model(&entity.CollectionProduct{}).
				Where("collection_id = ? AND product_id = ?", collectionID, item.ProductID).
				Update("position", item.Position)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return prodErrors.ErrProductNotInCollection
			}
		}
		return nil
	})
}
