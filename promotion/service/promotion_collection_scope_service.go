package service

import (
	"context"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/common/log"
	productService "ecommerce-be/product/service"
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
)

type PromotionCollectionScopeService interface {
	AddCollections(ctx context.Context, req model.AddPromotionCollectionRequest) error
	RemoveCollections(ctx context.Context, req model.RemovePromotionCollectionRequest) error
	RemoveAllCollections(ctx context.Context, promotionID uint) error
	GetCollections(
		ctx context.Context,
		req model.GetPromotionCollectionsRequest,
	) (*model.GetPromotionCollectionsResponse, error)
}

type PromotionCollectionScopeServiceImpl struct {
	repo                     repository.PromotionCollectionScopeRepository
	collectionProductService productService.CollectionProductService
}

func NewPromotionCollectionScopeServiceImpl(
	repo repository.PromotionCollectionScopeRepository,
	collectionProductService productService.CollectionProductService,
) *PromotionCollectionScopeServiceImpl {
	return &PromotionCollectionScopeServiceImpl{
		repo:                     repo,
		collectionProductService: collectionProductService,
	}
}

func (s *PromotionCollectionScopeServiceImpl) AddCollections(
	ctx context.Context,
	req model.AddPromotionCollectionRequest,
) error {
	log.InfoWithContext(ctx, "Adding collections to promotion scope")

	// Convert request to entities
	var entities []entity.PromotionCollection
	for _, cid := range req.CollectionIDs {
		entities = append(entities, entity.PromotionCollection{
			PromotionID:  req.PromotionID,
			CollectionID: cid,
		})
	}

	if err := s.repo.AddPromotionCollections(ctx, entities); err != nil {
		log.ErrorWithContext(ctx, "Failed to add promotion collections", err)
		return err
	}

	return nil
}

func (s *PromotionCollectionScopeServiceImpl) RemoveCollections(
	ctx context.Context,
	req model.RemovePromotionCollectionRequest,
) error {
	log.InfoWithContext(ctx, "Removing collections from promotion scope")
	return s.repo.DeletePromotionCollections(ctx, req.PromotionID, req.CollectionIDs)
}

func (s *PromotionCollectionScopeServiceImpl) RemoveAllCollections(
	ctx context.Context,
	promotionID uint,
) error {
	log.InfoWithContext(ctx, "Removing all collections from promotion scope")
	return s.repo.DeletePromotionCollectionByPromotionID(ctx, promotionID)
}

func (s *PromotionCollectionScopeServiceImpl) GetCollections(
	ctx context.Context,
	req model.GetPromotionCollectionsRequest,
) (*model.GetPromotionCollectionsResponse, error) {
	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	collections, total, err := s.repo.GetPromotionCollections(
		ctx,
		req.PromotionID,
		req.CollectionIDs,
		offset,
		req.PageSize,
	)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to get promotion collections", err)
		return nil, err
	}

	response := &model.GetPromotionCollectionsResponse{
		BasePromotionScopeResponse: model.BasePromotionScopeResponse{PromotionID: req.PromotionID},
		Collections:                make([]model.PromotionCollectionResponse, len(collections)),
		Pagination:                 common.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, c := range collections {
		response.Collections[i] = model.PromotionCollectionResponse{
			BasePromotionScopeResponse: model.BasePromotionScopeResponse{
				PromotionID: req.PromotionID,
			},
			CollectionID: c.CollectionID,
			// Note: Collection details would populate here if relations were loaded
		}
	}

	return response, nil
}

func (s *PromotionCollectionScopeServiceImpl) IsCartEligible(
	ctx context.Context,
	promotionID uint,
	cart *model.CartValidationRequest,
) (bool, []string) {
	// Step 1: Get collections for this promotion
	collResp, err := s.GetCollections(
		ctx,
		model.GetPromotionCollectionsRequest{
			GetPromotionScopeRequest: model.GetPromotionScopeRequest{
				BasePromotionScopeRequest: model.BasePromotionScopeRequest{
					PromotionID: promotionID,
				},
			},
		},
	)
	if err != nil || collResp == nil || len(collResp.Collections) == 0 {
		return false, nil
	}

	// Step 2: Extract collection IDs
	collectionIDs := make([]uint, len(collResp.Collections))
	for i, c := range collResp.Collections {
		collectionIDs[i] = c.CollectionID
	}

	// Step 3: Get product IDs belonging to those collections (via product service)
	productIDs, err := s.collectionProductService.GetProductIDsByCollectionIDs(ctx, collectionIDs)
	if err != nil || len(productIDs) == 0 {
		return false, nil
	}

	// Step 4: Check if any cart item matches
	eligibleProducts := make(map[uint]bool, len(productIDs))
	for _, pid := range productIDs {
		eligibleProducts[pid] = true
	}

	eligibleLineItems := []string{}

	for _, item := range cart.Items {
		if eligibleProducts[item.ProductID] {
			eligibleLineItems = append(eligibleLineItems, item.ItemID)
		}
	}
	return len(eligibleLineItems) > 0, eligibleLineItems
}
