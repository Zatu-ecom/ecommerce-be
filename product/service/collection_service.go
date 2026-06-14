package service

import (
	"context"
	"strings"

	"ecommerce-be/common/log"
	"ecommerce-be/product/factory"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
	"ecommerce-be/product/validator"
)

// CollectionService defines the interface for collection-related business logic
type CollectionService interface {
	CreateCollection(
		ctx context.Context,
		req model.CollectionCreateRequest,
		roleLevel uint,
		sellerID uint,
	) (*model.CollectionResponse, error)
	UpdateCollection(
		ctx context.Context,
		id uint,
		req model.CollectionUpdateRequest,
		roleLevel uint,
		sellerID uint,
	) (*model.CollectionResponse, error)
	DeleteCollection(ctx context.Context, id uint, roleLevel uint, sellerID uint) error
	GetAllCollections(ctx context.Context, sellerID *uint) (*model.CollectionsResponse, error)
	GetCollectionByID(ctx context.Context, id uint, sellerID *uint) (*model.CollectionResponse, error)
}

// CollectionServiceImpl implements CollectionService
type CollectionServiceImpl struct {
	collectionRepo repository.CollectionRepository
}

// NewCollectionService creates a new CollectionService
func NewCollectionService(collectionRepo repository.CollectionRepository) CollectionService {
	return &CollectionServiceImpl{collectionRepo: collectionRepo}
}

func (s *CollectionServiceImpl) CreateCollection(
	ctx context.Context,
	req model.CollectionCreateRequest,
	roleLevel uint,
	sellerID uint,
) (*model.CollectionResponse, error) {
	log.InfoWithContext(ctx, "Creating collection")

	collection := factory.BuildCollectionEntityFromCreateRequest(req, sellerID)
	if err := s.collectionRepo.Create(ctx, collection); err != nil {
		if isUniqueViolation(err) {
			return nil, prodErrors.ErrCollectionExists
		}
		log.ErrorWithContext(ctx, "Failed to create collection", err)
		return nil, err
	}

	return factory.BuildCollectionResponse(collection, 0), nil
}

func (s *CollectionServiceImpl) UpdateCollection(
	ctx context.Context,
	id uint,
	req model.CollectionUpdateRequest,
	roleLevel uint,
	sellerID uint,
) (*model.CollectionResponse, error) {
	log.InfoWithContext(ctx, "Updating collection")

	collection, err := s.collectionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := validator.ValidateCollectionOwnershipOrAdminAccess(roleLevel, sellerID, collection); err != nil {
		return nil, err
	}

	collection = factory.ApplyCollectionUpdateRequest(collection, req)
	if err := s.collectionRepo.Update(ctx, collection); err != nil {
		if isUniqueViolation(err) {
			return nil, prodErrors.ErrCollectionExists
		}
		log.ErrorWithContext(ctx, "Failed to update collection", err)
		return nil, err
	}

	count, err := s.collectionRepo.CountProducts(ctx, id)
	if err != nil {
		return nil, err
	}

	return factory.BuildCollectionResponse(collection, count), nil
}

func (s *CollectionServiceImpl) DeleteCollection(
	ctx context.Context,
	id uint,
	roleLevel uint,
	sellerID uint,
) error {
	log.InfoWithContext(ctx, "Deleting collection")

	collection, err := s.collectionRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := validator.ValidateCollectionOwnershipOrAdminAccess(roleLevel, sellerID, collection); err != nil {
		return err
	}

	if err := s.collectionRepo.Delete(ctx, id); err != nil {
		log.ErrorWithContext(ctx, "Failed to delete collection", err)
		return err
	}

	return nil
}

func (s *CollectionServiceImpl) GetAllCollections(
	ctx context.Context,
	sellerID *uint,
) (*model.CollectionsResponse, error) {
	log.InfoWithContext(ctx, "Getting all collections")

	collections, err := s.collectionRepo.FindAll(ctx, sellerID)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to get collections", err)
		return nil, err
	}

	responses := make([]model.CollectionResponse, 0, len(collections))
	for i := range collections {
		count, countErr := s.collectionRepo.CountProducts(ctx, collections[i].ID)
		if countErr != nil {
			return nil, countErr
		}
		responses = append(responses, *factory.BuildCollectionResponse(&collections[i], count))
	}

	return &model.CollectionsResponse{Collections: responses}, nil
}

func (s *CollectionServiceImpl) GetCollectionByID(
	ctx context.Context,
	id uint,
	sellerID *uint,
) (*model.CollectionResponse, error) {
	log.InfoWithContext(ctx, "Getting collection by ID")

	collection, err := s.collectionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := validator.ValidateCollectionReadable(sellerID, collection); err != nil {
		return nil, err
	}

	count, err := s.collectionRepo.CountProducts(ctx, id)
	if err != nil {
		return nil, err
	}

	return factory.BuildCollectionResponse(collection, count), nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique_seller_collection_slug") || strings.Contains(msg, "duplicate key")
}
