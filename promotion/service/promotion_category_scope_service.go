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

type PromotionCategoryScopeService interface {
	AddCategories(ctx context.Context, req model.AddPromotionCategoryRequest) error
	RemoveCategories(ctx context.Context, req model.RemovePromotionCategoryRequest) error
	RemoveAllCategories(ctx context.Context, promotionID uint) error
	GetCategories(
		ctx context.Context,
		req model.GetPromotionCategoriesRequest,
	) (*model.GetPromotionCategoriesResponse, error)
}

type PromotionCategoryScopeServiceImpl struct {
	repo repository.PromotionCategoryScopeRepository
}

func NewPromotionCategoryScopeService(
	repo repository.PromotionCategoryScopeRepository,
) PromotionCategoryScopeService {
	return &PromotionCategoryScopeServiceImpl{repo: repo}
}

func (s *PromotionCategoryScopeServiceImpl) AddCategories(
	ctx context.Context,
	req model.AddPromotionCategoryRequest,
) error {
	log.InfoWithContext(ctx, "Adding categories to promotion scope")

	// Convert request to entities
	var entities []entity.PromotionCategory
	for _, item := range req.Categories {
		entities = append(entities, entity.PromotionCategory{
			PromotionID:          req.PromotionID,
			CategoryID:           item.CategoryID,
			IncludeSubcategories: helper.BoolPtr(item.IncludeSubcategories),
		})
	}

	if err := s.repo.AddPromotionCategories(ctx, entities); err != nil {
		log.ErrorWithContext(ctx, "Failed to add promotion categories", err)
		return err
	}

	return nil
}

func (s *PromotionCategoryScopeServiceImpl) RemoveCategories(
	ctx context.Context,
	req model.RemovePromotionCategoryRequest,
) error {
	log.InfoWithContext(ctx, "Removing categories from promotion scope")
	return s.repo.DeletePromotionCategories(ctx, req.PromotionID, req.CategoryIDs)
}

func (s *PromotionCategoryScopeServiceImpl) RemoveAllCategories(
	ctx context.Context,
	promotionID uint,
) error {
	log.InfoWithContext(ctx, "Removing all categories from promotion scope")
	return s.repo.DeletePromotionCategoryByPromotionID(ctx, promotionID)
}

func (s *PromotionCategoryScopeServiceImpl) GetCategories(
	ctx context.Context,
	req model.GetPromotionCategoriesRequest,
) (*model.GetPromotionCategoriesResponse, error) {
	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	categories, total, err := s.repo.GetPromotionCategories(
		ctx,
		req.PromotionID,
		req.CategoryIDs,
		offset,
		req.PageSize,
	)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to get promotion categories", err)
		return nil, err
	}

	response := &model.GetPromotionCategoriesResponse{
		BasePromotionScopeResponse: model.BasePromotionScopeResponse{PromotionID: req.PromotionID},
		Categories:                 make([]model.PromotionCategoryResponse, len(categories)),
		Pagination:                 common.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, c := range categories {
		includeSub := true
		if c.IncludeSubcategories != nil {
			includeSub = *c.IncludeSubcategories
		}

		response.Categories[i] = model.PromotionCategoryResponse{
			BasePromotionScopeResponse: model.BasePromotionScopeResponse{
				PromotionID: req.PromotionID,
			},
			CategoryID:           c.CategoryID,
			IncludeSubcategories: includeSub,
			// Note: Category name would populate here if relations were loaded
		}
	}

	return response, nil
}
