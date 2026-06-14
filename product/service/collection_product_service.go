package service

import (
	"context"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/common/log"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/factory"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
	productHelper "ecommerce-be/product/utils/helper"
	"ecommerce-be/product/validator"
)

// CollectionProductService defines the interface for collection-product operations
type CollectionProductService interface {
	GetProductIDsByCollectionIDs(ctx context.Context, collectionIDs []uint) ([]uint, error)
	AddProducts(
		ctx context.Context,
		collectionID uint,
		req model.AddCollectionProductsRequest,
		roleLevel uint,
		sellerID uint,
	) error
	RemoveProducts(
		ctx context.Context,
		collectionID uint,
		req model.RemoveCollectionProductsRequest,
		roleLevel uint,
		sellerID uint,
	) error
	GetProducts(
		ctx context.Context,
		collectionID uint,
		req model.GetCollectionProductsRequest,
		sellerID *uint,
	) (*model.GetCollectionProductsResponse, error)
	ReorderProducts(
		ctx context.Context,
		collectionID uint,
		req model.ReorderCollectionProductsRequest,
		roleLevel uint,
		sellerID uint,
	) error
}

// CollectionProductServiceImpl implements the CollectionProductService interface
type CollectionProductServiceImpl struct {
	repo           repository.CollectionProductRepository
	collectionRepo repository.CollectionRepository
	productRepo    repository.ProductRepository
}

// NewCollectionProductService creates a new instance of CollectionProductService
func NewCollectionProductService(
	repo repository.CollectionProductRepository,
	collectionRepo repository.CollectionRepository,
	productRepo repository.ProductRepository,
) CollectionProductService {
	return &CollectionProductServiceImpl{
		repo:           repo,
		collectionRepo: collectionRepo,
		productRepo:    productRepo,
	}
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

func (s *CollectionProductServiceImpl) AddProducts(
	ctx context.Context,
	collectionID uint,
	req model.AddCollectionProductsRequest,
	roleLevel uint,
	sellerID uint,
) error {
	log.InfoWithContext(ctx, "Adding products to collection")

	collection, err := s.getCollectionForWrite(ctx, collectionID, roleLevel, sellerID)
	if err != nil {
		return err
	}

	if err := s.validateProductsBelongToSeller(ctx, req.ProductIDs, collection.SellerID); err != nil {
		return err
	}

	existing, err := s.repo.GetExistingProductIDsInCollection(ctx, collectionID, req.ProductIDs)
	if err != nil {
		return err
	}
	existingSet := make(map[uint]struct{}, len(existing))
	for _, id := range existing {
		existingSet[id] = struct{}{}
	}

	maxPosition, err := s.repo.GetMaxPosition(ctx, collectionID)
	if err != nil {
		return err
	}

	var toAdd []entity.CollectionProduct
	position := maxPosition + 1
	for _, productID := range req.ProductIDs {
		if _, ok := existingSet[productID]; ok {
			continue
		}
		toAdd = append(toAdd, entity.CollectionProduct{
			CollectionID: collectionID,
			ProductID:    productID,
			Position:     position,
			BaseEntity:   productHelper.NewBaseEntity(),
		})
		position++
	}

	return s.repo.AddProducts(ctx, toAdd)
}

func (s *CollectionProductServiceImpl) RemoveProducts(
	ctx context.Context,
	collectionID uint,
	req model.RemoveCollectionProductsRequest,
	roleLevel uint,
	sellerID uint,
) error {
	log.InfoWithContext(ctx, "Removing products from collection")

	if _, err := s.getCollectionForWrite(ctx, collectionID, roleLevel, sellerID); err != nil {
		return err
	}

	return s.repo.RemoveProducts(ctx, collectionID, req.ProductIDs)
}

func (s *CollectionProductServiceImpl) GetProducts(
	ctx context.Context,
	collectionID uint,
	req model.GetCollectionProductsRequest,
	sellerID *uint,
) (*model.GetCollectionProductsResponse, error) {
	log.InfoWithContext(ctx, "Getting products in collection")

	collection, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return nil, err
	}

	if err := validator.ValidateCollectionReadable(sellerID, collection); err != nil {
		return nil, err
	}

	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	products, total, err := s.repo.GetProductsByCollectionID(
		ctx,
		collectionID,
		req.ProductIDs,
		offset,
		req.PageSize,
	)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to get collection products", err)
		return nil, err
	}

	responses := make([]model.CollectionProductResponse, 0, len(products))
	for i := range products {
		responses = append(responses, factory.BuildCollectionProductResponse(&products[i]))
	}

	return &model.GetCollectionProductsResponse{
		Products:   responses,
		Pagination: common.NewPaginationResponse(req.Page, req.PageSize, total),
	}, nil
}

func (s *CollectionProductServiceImpl) ReorderProducts(
	ctx context.Context,
	collectionID uint,
	req model.ReorderCollectionProductsRequest,
	roleLevel uint,
	sellerID uint,
) error {
	log.InfoWithContext(ctx, "Reordering products in collection")

	if _, err := s.getCollectionForWrite(ctx, collectionID, roleLevel, sellerID); err != nil {
		return err
	}

	updates := make([]repository.CollectionProductPositionUpdate, 0, len(req.Items))
	for _, item := range req.Items {
		updates = append(updates, repository.CollectionProductPositionUpdate{
			ProductID: item.ProductID,
			Position:  item.Position,
		})
	}

	return s.repo.UpdatePositions(ctx, collectionID, updates)
}

func (s *CollectionProductServiceImpl) getCollectionForWrite(
	ctx context.Context,
	collectionID uint,
	roleLevel uint,
	sellerID uint,
) (*entity.Collection, error) {
	collection, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	if err := validator.ValidateCollectionOwnershipOrAdminAccess(roleLevel, sellerID, collection); err != nil {
		return nil, err
	}
	return collection, nil
}

func (s *CollectionProductServiceImpl) validateProductsBelongToSeller(
	ctx context.Context,
	productIDs []uint,
	sellerID uint,
) error {
	products, err := s.productRepo.FindByIDs(ctx, productIDs)
	if err != nil {
		return err
	}

	found := make(map[uint]struct{}, len(products))
	for _, p := range products {
		if p.SellerID != sellerID {
			return prodErrors.ErrInvalidCollectionProduct
		}
		found[p.ID] = struct{}{}
	}

	for _, id := range productIDs {
		if _, ok := found[id]; !ok {
			return prodErrors.ErrInvalidCollectionProduct
		}
	}

	return nil
}
