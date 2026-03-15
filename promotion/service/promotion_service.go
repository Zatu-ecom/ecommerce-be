package service

import (
	"context"

	productService "ecommerce-be/product/service"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
)

// PromotionService defines the interface for promotion-related business logic
type PromotionService interface {
	CreatePromotion(
		ctx context.Context,
		req model.CreatePromotionRequest,
		sellerID uint,
	) (*model.PromotionResponse, error)

	GetPromotionByID(
		ctx context.Context,
		id uint,
		sellerID uint,
	) (*model.PromotionResponse, error)

	ListPromotions(
		ctx context.Context,
		req model.ListPromotionsRequest,
	) (*model.ListPromotionsResponse, error)

	UpdatePromotion(
		ctx context.Context,
		id uint,
		req model.UpdatePromotionRequest,
		sellerID uint,
	) (*model.PromotionResponse, error)

	UpdateStatus(
		ctx context.Context,
		id uint,
		req model.UpdateStatusRequest,
		sellerID uint,
	) (*model.PromotionResponse, error)

	DeletePromotion(
		ctx context.Context,
		id uint,
		sellerID uint,
	) error

	// ApplyPromotionsToCart applies multiple promotions to a cart with priority and stacking logic
	// Returns per-item discount breakdown and summary
	ApplyPromotionsToCart(
		ctx context.Context,
		cart *model.CartValidationRequest,
	) (*model.AppliedPromotionSummary, error)
}

// PromotionServiceImpl implements the PromotionService interface
type PromotionServiceImpl struct {
	promotionRepo                           repository.PromotionRepository
	productScopeService                     PromotionProductScopeService
	categoryScopeService                    PromotionCategoryScopeService
	collectionScopeService                  PromotionCollectionScopeService
	collectionProductService                productService.CollectionProductService
	promotionScopeEligibilityServiceFactory *PromotionScopeEligibilityServiceFactory
}

// NewPromotionService creates a new instance of PromotionService
func NewPromotionService(
	promotionRepo repository.PromotionRepository,
	productScopeService PromotionProductScopeService,
	categoryScopeService PromotionCategoryScopeService,
	collectionScopeService PromotionCollectionScopeService,
	collectionProductService productService.CollectionProductService,
	promotionScopeEligibilityServiceFactory *PromotionScopeEligibilityServiceFactory,
) PromotionService {
	return &PromotionServiceImpl{
		promotionRepo:                           promotionRepo,
		productScopeService:                     productScopeService,
		categoryScopeService:                    categoryScopeService,
		collectionScopeService:                  collectionScopeService,
		collectionProductService:                collectionProductService,
		promotionScopeEligibilityServiceFactory: promotionScopeEligibilityServiceFactory,
	}
}
