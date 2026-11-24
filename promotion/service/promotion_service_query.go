package service

import (
	"context"
	"fmt"
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	commonModel "ecommerce-be/common/model"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
)

// GetPromotionByID retrieves a promotion by ID
func (s *PromotionServiceImpl) GetPromotionByID(
	ctx context.Context,
	id uint,
	sellerID uint,
) (*model.PromotionResponse, error) {
	log.InfoWithContext(ctx, fmt.Sprintf("Retrieving promotion %d for seller %d", id, sellerID))

	promotion, err := s.promotionRepo.FindByID(ctx, id)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to retrieve promotion", err)
		return nil, promoErrors.ErrPromotionNotFound
	}

	if promotion.SellerID != sellerID {
		return nil, promoErrors.ErrUnauthorizedPromotionAccess
	}

	ccy, err := s.sellerCurrency(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	return factory.PromotionEntityToResponse(promotion, ccy), nil
}

// ListPromotions returns a list of promotions based on the provided filters
func (s *PromotionServiceImpl) ListPromotions(
	ctx context.Context,
	req model.ListPromotionsRequest,
) (*model.ListPromotionsResponse, error) {
	log.InfoWithContext(ctx, fmt.Sprintf("Listing promotions for seller %d", req.SellerID))

	req.SetDefaults()

	filters := repository.ListPromotionFilter{
		SellerID:      req.SellerID,
		Status:        req.Status,
		PromotionType: req.PromotionType,
		AppliesTo:     req.AppliesTo,
		Page:          req.Page,
		Limit:         req.PageSize,
	}

	promotions, total, err := s.promotionRepo.List(ctx, filters)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to list promotions", err)
		return nil, commonError.NewAppError(
			"PROMOTION_LIST_FAILED",
			"Failed to list promotions",
			http.StatusInternalServerError,
		)
	}

	ccy, err := s.sellerCurrency(ctx, req.SellerID)
	if err != nil {
		return nil, err
	}

	var responseList []*model.PromotionResponse
	for _, p := range promotions {
		responseList = append(responseList, factory.PromotionEntityToResponse(p, ccy))
	}

	if responseList == nil {
		responseList = make([]*model.PromotionResponse, 0)
	}

	return &model.ListPromotionsResponse{
		Promotions: responseList,
		Pagination: commonModel.NewPaginationResponse(req.Page, req.PageSize, total),
	}, nil
}
