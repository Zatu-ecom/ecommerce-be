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

type PromotionVariantScopeService interface {
	AddVariants(ctx context.Context, req model.AddPromotionVariantRequest) error
	RemoveVariants(ctx context.Context, req model.RemovePromotionVariantRequest) error
	RemoveAllVariants(ctx context.Context, promotionID uint) error
	GetVariants(
		ctx context.Context,
		req model.GetPromotionVariantsRequest,
	) (*model.GetPromotionVariantsResponse, error)
}

type PromotionVariantScopeServiceImpl struct {
	repo repository.PromotionProductVariantScopeRepository
}

func NewPromotionVariantScopeService(
	repo repository.PromotionProductVariantScopeRepository,
) PromotionVariantScopeService {
	return &PromotionVariantScopeServiceImpl{repo: repo}
}

func (s *PromotionVariantScopeServiceImpl) AddVariants(
	ctx context.Context,
	req model.AddPromotionVariantRequest,
) error {
	log.InfoWithContext(ctx, "Adding variants to promotion scope")

	// Convert request to entities
	var entities []entity.PromotionProductVariant
	for _, vid := range req.VariantIDs {
		entities = append(entities, entity.PromotionProductVariant{
			PromotionID: req.PromotionID,
			VariantID:   vid,
		})
	}

	if err := s.repo.AddPromotionProductVariants(ctx, entities); err != nil {
		log.ErrorWithContext(ctx, "Failed to add promotion variants", err)
		return err
	}

	return nil
}

func (s *PromotionVariantScopeServiceImpl) RemoveVariants(
	ctx context.Context,
	req model.RemovePromotionVariantRequest,
) error {
	log.InfoWithContext(ctx, "Removing variants from promotion scope")
	return s.repo.DeletePromotionProductVariants(ctx, req.PromotionID, req.VariantIDs)
}

func (s *PromotionVariantScopeServiceImpl) RemoveAllVariants(
	ctx context.Context,
	promotionID uint,
) error {
	log.InfoWithContext(ctx, "Removing all variants from promotion scope")
	return s.repo.DeletePromotionProductVariantByPromotionID(ctx, promotionID)
}

func (s *PromotionVariantScopeServiceImpl) GetVariants(
	ctx context.Context,
	req model.GetPromotionVariantsRequest,
) (*model.GetPromotionVariantsResponse, error) {
	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	variants, total, err := s.repo.GetPromotionProductVariants(
		ctx,
		req.PromotionID,
		req.VariantIDs,
		offset,
		req.PageSize,
	)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to get promotion variants", err)
		return nil, err
	}

	response := &model.GetPromotionVariantsResponse{
		BasePromotionScopeResponse: model.BasePromotionScopeResponse{PromotionID: req.PromotionID},
		Variants:                   make([]model.PromotionVariantResponse, len(variants)),
		Pagination:                 common.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, v := range variants {
		response.Variants[i] = model.PromotionVariantResponse{
			BasePromotionScopeResponse: model.BasePromotionScopeResponse{
				PromotionID: req.PromotionID,
			},
			VariantID: v.VariantID,
			// Note: Variant details would populate here if relations were loaded
		}
	}

	return response, nil
}
