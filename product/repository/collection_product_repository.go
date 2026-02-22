package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
)

// CollectionProductRepository defines the interface for collection-product operations
type CollectionProductRepository interface {
	GetProductIDsByCollectionIDs(ctx context.Context, collectionIDs []uint) ([]uint, error)
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
