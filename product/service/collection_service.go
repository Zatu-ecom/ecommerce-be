package service

import (
	"context"
	"strings"

	"ecommerce-be/common/filegateway"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	fileGateway "ecommerce-be/file/gateway"
	"ecommerce-be/product/factory"
	prodErrors "ecommerce-be/product/error"
	"ecommerce-be/product/entity"
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
	fileGateway    filegateway.FileDisplayGateway
}

// NewCollectionService creates a new CollectionService
func NewCollectionService(
	collectionRepo repository.CollectionRepository,
	fileGateway filegateway.FileDisplayGateway,
) CollectionService {
	return &CollectionServiceImpl{
		collectionRepo: collectionRepo,
		fileGateway:    fileGateway,
	}
}

func (s *CollectionServiceImpl) CreateCollection(
	ctx context.Context,
	req model.CollectionCreateRequest,
	roleLevel uint,
	sellerID uint,
) (*model.CollectionResponse, error) {
	log.InfoWithContext(ctx, "Creating collection")

	if err := s.validateImageFileID(ctx, req.ImageFileID, sellerID); err != nil {
		return nil, err
	}

	collection := factory.BuildCollectionEntityFromCreateRequest(req, sellerID)
	if err := s.collectionRepo.Create(ctx, collection); err != nil {
		if isUniqueViolation(err) {
			return nil, prodErrors.ErrCollectionExists
		}
		log.ErrorWithContext(ctx, "Failed to create collection", err)
		return nil, err
	}

	return s.buildCollectionResponse(ctx, collection, 0), nil
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

	if err := s.validateImageFileID(ctx, req.ImageFileID, sellerID); err != nil {
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

	return s.buildCollectionResponse(ctx, collection, count), nil
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

	imageMap := s.batchResolveCollectionImages(ctx, collections)
	responses := make([]model.CollectionResponse, 0, len(collections))
	for i := range collections {
		count, countErr := s.collectionRepo.CountProducts(ctx, collections[i].ID)
		if countErr != nil {
			return nil, countErr
		}
		image := imageMap[collections[i].ID]
		responses = append(responses, *factory.BuildCollectionResponse(&collections[i], count, image))
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

	return s.buildCollectionResponse(ctx, collection, count), nil
}

func (s *CollectionServiceImpl) validateImageFileID(
	ctx context.Context,
	imageFileID *string,
	sellerID uint,
) error {
	if imageFileID == nil || *imageFileID == "" {
		return nil
	}
	_, err := filegateway.ResolveSingle(ctx, s.fileGateway, *imageFileID, &sellerID)
	if err != nil {
		if fileGateway.IsFileNotFound(err) || err == commonError.ErrFileNotAccessible {
			return prodErrors.ErrCollectionInvalidFile
		}
		return err
	}
	return nil
}

func (s *CollectionServiceImpl) buildCollectionResponse(
	ctx context.Context,
	collection *entity.Collection,
	productCount int64,
) *model.CollectionResponse {
	sid := collection.SellerID
	image := filegateway.ResolveOptional(ctx, s.fileGateway, collection.ImageFileID, &sid)
	return factory.BuildCollectionResponse(collection, productCount, image)
}

func (s *CollectionServiceImpl) batchResolveCollectionImages(
	ctx context.Context,
	collections []entity.Collection,
) map[uint]*filegateway.FileAssetResponse {
	result := make(map[uint]*filegateway.FileAssetResponse, len(collections))
	if len(collections) == 0 {
		return result
	}

	fileIDs := make([]string, 0, len(collections))
	fileIDToCollectionIDs := make(map[string][]uint)
	for _, c := range collections {
		if c.ImageFileID == nil || *c.ImageFileID == "" {
			continue
		}
		fid := *c.ImageFileID
		if _, seen := fileIDToCollectionIDs[fid]; !seen {
			fileIDs = append(fileIDs, fid)
		}
		fileIDToCollectionIDs[fid] = append(fileIDToCollectionIDs[fid], c.ID)
	}
	if len(fileIDs) == 0 {
		return result
	}

	var sellerID *uint
	if len(collections) > 0 {
		sid := collections[0].SellerID
		sellerID = &sid
	}
	fileMap, _ := s.fileGateway.GetFilesWithURLs(ctx, fileIDs, sellerID)
	for fid, info := range fileMap {
		asset := filegateway.ToFileAssetResponse(info)
		for _, collectionID := range fileIDToCollectionIDs[fid] {
			result[collectionID] = asset
		}
	}
	return result
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique_seller_collection_slug") || strings.Contains(msg, "duplicate key")
}
