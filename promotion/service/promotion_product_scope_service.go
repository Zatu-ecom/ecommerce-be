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

type PromotionProductScopeService interface {
	AddProducts(ctx context.Context, req model.AddPromotionProductRequest) error
	RemoveProducts(ctx context.Context, req model.RemovePromotionProductRequest) error
	RemoveAllProducts(ctx context.Context, promotionID uint) error
	GetProducts(
		ctx context.Context,
		req model.GetPromotionProductsRequest,
	) (*model.GetPromotionProductsResponse, error)
}

type PromotionProductScopeServiceImpl struct {
	repo repository.PromotionProductScopeRepository
}

func NewPromotionProductScopeServiceImpl(
	repo repository.PromotionProductScopeRepository,
) *PromotionProductScopeServiceImpl {
	return &PromotionProductScopeServiceImpl{repo: repo}
}

func (s *PromotionProductScopeServiceImpl) AddProducts(
	ctx context.Context,
	req model.AddPromotionProductRequest,
) error {
	log.InfoWithContext(ctx, "Adding products to promotion scope")

	// Convert request to entities
	var entities []entity.PromotionProduct
	for _, pid := range req.ProductIDs {
		entities = append(entities, entity.PromotionProduct{
			PromotionID: req.PromotionID,
			ProductID:   pid,
		})
	}

	if err := s.repo.AddPromotionProducts(ctx, entities); err != nil {
		log.ErrorWithContext(ctx, "Failed to add promotion products", err)
		return err
	}

	return nil
}

func (s *PromotionProductScopeServiceImpl) RemoveProducts(
	ctx context.Context,
	req model.RemovePromotionProductRequest,
) error {
	log.InfoWithContext(ctx, "Removing products from promotion scope")
	return s.repo.DeletePromotionProducts(ctx, req.PromotionID, req.ProductIDs)
}

func (s *PromotionProductScopeServiceImpl) RemoveAllProducts(
	ctx context.Context,
	promotionID uint,
) error {
	log.InfoWithContext(ctx, "Removing all products from promotion scope")
	return s.repo.DeletePromotionProductByPromotionID(ctx, promotionID)
}

func (s *PromotionProductScopeServiceImpl) GetProducts(
	ctx context.Context,
	req model.GetPromotionProductsRequest,
) (*model.GetPromotionProductsResponse, error) {
	req.SetDefaults()
	offset := helper.CalculateOffset(req.Page, req.PageSize)

	products, total, err := s.repo.GetPromotionProducts(
		ctx,
		req.PromotionID,
		req.ProductIDs,
		offset,
		req.PageSize,
	)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to get promotion products", err)
		return nil, err
	}

	response := &model.GetPromotionProductsResponse{
		BasePromotionScopeResponse: model.BasePromotionScopeResponse{PromotionID: req.PromotionID},
		Products:                   make([]model.PromotionProductResponse, len(products)),
		Pagination:                 common.NewPaginationResponse(req.Page, req.PageSize, total),
	}

	for i, p := range products {
		response.Products[i] = model.PromotionProductResponse{
			BasePromotionScopeResponse: model.BasePromotionScopeResponse{
				PromotionID: req.PromotionID,
			},
			ProductID: p.ProductID,
			// Note: Product details (Name, Slug, Image) would populate here
			// if we preloaded the 'Product' relation in the repo or fetched it separately.
			// For now, returning basic scope info.
		}
	}

	return response, nil
}

func (s *PromotionProductScopeServiceImpl) IsCartEligible(
	ctx context.Context,
	promotionID uint,
	cart *model.CartValidationRequest,
) (bool, []string) {
	// Collect product IDs from cart items
	cartProductIDs := make([]uint, len(cart.Items))
	for i, item := range cart.Items {
		cartProductIDs[i] = item.ProductID
	}

	// Call GetProducts with cart product IDs as filter
	// If any results come back, those cart products exist in the promotion scope
	resp, err := s.GetProducts(ctx, model.GetPromotionProductsRequest{
		GetPromotionScopeRequest: model.GetPromotionScopeRequest{
			BasePromotionScopeRequest: model.BasePromotionScopeRequest{
				PromotionID: promotionID,
			},
		},
		ProductIDs: cartProductIDs,
	})
	if err != nil || resp == nil {
		return false, nil
	}

	eligibleProductIDs := make(map[uint]bool)
	for _, product := range resp.Products {
		eligibleProductIDs[product.ProductID] = true
	}

	eligibleLineItems := []string{}

	for _, item := range cart.Items {
		if eligibleProductIDs[item.ProductID] {
			eligibleLineItems = append(eligibleLineItems, item.ItemID)
		}
	}

	return len(resp.Products) > 0, eligibleLineItems
}
