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

type DiscountCodeCollectionScopeService interface {
	AddCollections(ctx context.Context, req model.AddDiscountCodeCollectionRequest, sellerID uint) error
	RemoveCollections(ctx context.Context, req model.RemoveDiscountCodeCollectionRequest, sellerID uint) error
	RemoveAllCollections(ctx context.Context, discountCodeID, sellerID uint) error
	GetCollections(
		ctx context.Context,
		req model.GetDiscountCodeCollectionsRequest,
		sellerID uint,
	) (*model.GetDiscountCodeCollectionsResponse, error)
}

type DiscountCodeCollectionScopeServiceImpl struct {
	repo             repository.DiscountCodeCollectionScopeRepository
	discountCodeRepo repository.DiscountCodeRepository
}

func NewDiscountCodeCollectionScopeService(
	repo repository.DiscountCodeCollectionScopeRepository,
	discountCodeRepo repository.DiscountCodeRepository,
) DiscountCodeCollectionScopeService {
	return &DiscountCodeCollectionScopeServiceImpl{
		repo:             repo,
		discountCodeRepo: discountCodeRepo,
	}
}

func (s *DiscountCodeCollectionScopeServiceImpl) AddCollections(
	ctx context.Context,
	req model.AddDiscountCodeCollectionRequest,
	sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificCollections,
	); err != nil {
		return err
	}

	entities := make([]entity.DiscountCodeCollection, 0, len(req.CollectionIDs))
	for _, cid := range req.CollectionIDs {
		entities = append(entities, entity.DiscountCodeCollection{
			DiscountCodeID: req.DiscountCodeID,
			CollectionID:   cid,
		})
	}

	if err := s.repo.AddCollections(ctx, entities); err != nil {
		log.ErrorWithContext(ctx, "Failed to add discount code collections", err)
		return err
	}
	return nil
}

func (s *DiscountCodeCollectionScopeServiceImpl) RemoveCollections(
	ctx context.Context,
	req model.RemoveDiscountCodeCollectionRequest,
	sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificCollections,
	); err != nil {
		return err
	}
	return s.repo.DeleteCollections(ctx, req.DiscountCodeID, req.CollectionIDs)
}

func (s *DiscountCodeCollectionScopeServiceImpl) RemoveAllCollections(
	ctx context.Context,
	discountCodeID, sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, discountCodeID, sellerID, entity.ScopeSpecificCollections,
	); err != nil {
		return err
	}
	return s.repo.DeleteAllCollections(ctx, discountCodeID)
}

func (s *DiscountCodeCollectionScopeServiceImpl) GetCollections(
	ctx context.Context,
	req model.GetDiscountCodeCollectionsRequest,
	sellerID uint,
) (*model.GetDiscountCodeCollectionsResponse, error) {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificCollections,
	); err != nil {
		return nil, err
	}

	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	collections, total, err := s.repo.GetCollections(
		ctx, req.DiscountCodeID, req.CollectionIDs, offset, req.PageSize,
	)
	if err != nil {
		return nil, err
	}

	response := &model.GetDiscountCodeCollectionsResponse{
		BaseDiscountCodeScopeResponse: model.BaseDiscountCodeScopeResponse{
			DiscountCodeID: req.DiscountCodeID,
		},
		Collections: make([]model.DiscountCodeCollectionResponse, len(collections)),
		Pagination:  common.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, c := range collections {
		response.Collections[i] = model.DiscountCodeCollectionResponse{
			BaseDiscountCodeScopeResponse: model.BaseDiscountCodeScopeResponse{
				DiscountCodeID: req.DiscountCodeID,
			},
			CollectionID: c.CollectionID,
		}
	}

	return response, nil
}
