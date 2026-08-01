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

type DiscountCodeCategoryScopeService interface {
	AddCategories(ctx context.Context, req model.AddDiscountCodeCategoryRequest, sellerID uint) error
	RemoveCategories(ctx context.Context, req model.RemoveDiscountCodeCategoryRequest, sellerID uint) error
	RemoveAllCategories(ctx context.Context, discountCodeID, sellerID uint) error
	GetCategories(
		ctx context.Context,
		req model.GetDiscountCodeCategoriesRequest,
		sellerID uint,
	) (*model.GetDiscountCodeCategoriesResponse, error)
}

type DiscountCodeCategoryScopeServiceImpl struct {
	repo             repository.DiscountCodeCategoryScopeRepository
	discountCodeRepo repository.DiscountCodeRepository
}

func NewDiscountCodeCategoryScopeService(
	repo repository.DiscountCodeCategoryScopeRepository,
	discountCodeRepo repository.DiscountCodeRepository,
) DiscountCodeCategoryScopeService {
	return &DiscountCodeCategoryScopeServiceImpl{
		repo:             repo,
		discountCodeRepo: discountCodeRepo,
	}
}

func (s *DiscountCodeCategoryScopeServiceImpl) AddCategories(
	ctx context.Context,
	req model.AddDiscountCodeCategoryRequest,
	sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificCategories,
	); err != nil {
		return err
	}

	entities := make([]entity.DiscountCodeCategory, 0, len(req.Categories))
	for _, item := range req.Categories {
		entities = append(entities, entity.DiscountCodeCategory{
			DiscountCodeID: req.DiscountCodeID,
			CategoryID:     item.CategoryID,
		})
	}

	if err := s.repo.AddCategories(ctx, entities); err != nil {
		log.ErrorWithContext(ctx, "Failed to add discount code categories", err)
		return err
	}
	return nil
}

func (s *DiscountCodeCategoryScopeServiceImpl) RemoveCategories(
	ctx context.Context,
	req model.RemoveDiscountCodeCategoryRequest,
	sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificCategories,
	); err != nil {
		return err
	}
	return s.repo.DeleteCategories(ctx, req.DiscountCodeID, req.CategoryIDs)
}

func (s *DiscountCodeCategoryScopeServiceImpl) RemoveAllCategories(
	ctx context.Context,
	discountCodeID, sellerID uint,
) error {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, discountCodeID, sellerID, entity.ScopeSpecificCategories,
	); err != nil {
		return err
	}
	return s.repo.DeleteAllCategories(ctx, discountCodeID)
}

func (s *DiscountCodeCategoryScopeServiceImpl) GetCategories(
	ctx context.Context,
	req model.GetDiscountCodeCategoriesRequest,
	sellerID uint,
) (*model.GetDiscountCodeCategoriesResponse, error) {
	if _, err := ensureDiscountCodeScope(
		ctx, s.discountCodeRepo, req.DiscountCodeID, sellerID, entity.ScopeSpecificCategories,
	); err != nil {
		return nil, err
	}

	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	categories, total, err := s.repo.GetCategories(
		ctx, req.DiscountCodeID, req.CategoryIDs, offset, req.PageSize,
	)
	if err != nil {
		return nil, err
	}

	response := &model.GetDiscountCodeCategoriesResponse{
		BaseDiscountCodeScopeResponse: model.BaseDiscountCodeScopeResponse{
			DiscountCodeID: req.DiscountCodeID,
		},
		Categories: make([]model.DiscountCodeCategoryResponse, len(categories)),
		Pagination: common.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, c := range categories {
		response.Categories[i] = model.DiscountCodeCategoryResponse{
			BaseDiscountCodeScopeResponse: model.BaseDiscountCodeScopeResponse{
				DiscountCodeID: req.DiscountCodeID,
			},
			CategoryID:           c.CategoryID,
			IncludeSubcategories: true, // DB column not present yet; default mirrors promotion
		}
	}

	return response, nil
}
