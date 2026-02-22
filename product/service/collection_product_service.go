package service

import (
	"context"

	"ecommerce-be/common/log"
	"ecommerce-be/product/repository"
)

// CollectionProductService defines the interface for collection-product operations
type CollectionProductService interface {
	GetProductIDsByCollectionIDs(ctx context.Context, collectionIDs []uint) ([]uint, error)
}

// CollectionProductServiceImpl implements the CollectionProductService interface
type CollectionProductServiceImpl struct {
	repo repository.CollectionProductRepository
}

// NewCollectionProductService creates a new instance of CollectionProductService
func NewCollectionProductService(
	repo repository.CollectionProductRepository,
) CollectionProductService {
	return &CollectionProductServiceImpl{repo: repo}
}

// GetProductIDsByCollectionIDs returns distinct product IDs belonging to the given collection IDs
func (s *CollectionProductServiceImpl) GetProductIDsByCollectionIDs(
	ctx context.Context,
	collectionIDs []uint,
) ([]uint, error) {
	log.InfoWithContext(ctx, "Getting product IDs by collection IDs")

	productIDs, err := s.repo.GetProductIDsByCollectionIDs(ctx, collectionIDs)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to get product IDs by collection IDs", err)
		return nil, err
	}

	return productIDs, nil
}
