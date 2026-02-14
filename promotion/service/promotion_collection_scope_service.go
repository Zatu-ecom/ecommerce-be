package service

import (
	"context"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/common/log"
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
	repo repository.PromotionCollectionScopeRepository
}

func NewPromotionCollectionScopeService(
	repo repository.PromotionCollectionScopeRepository,
) PromotionCollectionScopeService {
	return &PromotionCollectionScopeServiceImpl{repo: repo}
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
