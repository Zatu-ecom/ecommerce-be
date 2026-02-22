package service

import (
	"context"
	"net/http"
	"time"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/log"
	productService "ecommerce-be/product/service"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/factory"
	"ecommerce-be/promotion/model"
	"ecommerce-be/promotion/repository"
	"ecommerce-be/promotion/service/promotionStrategy"
)

// PromotionService defines the interface for promotion-related business logic
type PromotionService interface {
	CreatePromotion(
		ctx context.Context,
		req model.CreatePromotionRequest,
		sellerID uint,
	) (*model.PromotionResponse, error)

	// ApplyPromotionsToCart applies multiple promotions to a cart with priority and stacking logic
	// Returns per-item discount breakdown and summary
	ApplyPromotionsToCart(
		ctx context.Context,
		cart *model.CartValidationRequest,
	) (*model.AppliedPromotionSummary, error)
}

// PromotionServiceImpl implements the PromotionService interface
type PromotionServiceImpl struct {
	promotionRepo            repository.PromotionRepository
	productScopeService      PromotionProductScopeService
	categoryScopeService     PromotionCategoryScopeService
	collectionScopeService   PromotionCollectionScopeService
	collectionProductService productService.CollectionProductService
}

// NewPromotionService creates a new instance of PromotionService
func NewPromotionService(
	promotionRepo repository.PromotionRepository,
	productScopeService PromotionProductScopeService,
	categoryScopeService PromotionCategoryScopeService,
	collectionScopeService PromotionCollectionScopeService,
	collectionProductService productService.CollectionProductService,
) PromotionService {
	return &PromotionServiceImpl{
		promotionRepo:            promotionRepo,
		productScopeService:      productScopeService,
		categoryScopeService:     categoryScopeService,
		collectionScopeService:   collectionScopeService,
		collectionProductService: collectionProductService,
	}
}

// CreatePromotion creates a new promotion
func (s *PromotionServiceImpl) CreatePromotion(
	ctx context.Context,
	req model.CreatePromotionRequest,
	sellerID uint,
) (*model.PromotionResponse, error) {
	log.InfoWithContext(ctx, "Creating new promotion")

	// Validate discount config using strategy pattern
	strategy := promotionStrategy.GetPromotionStrategy(req.PromotionType)
	if strategy == nil {
		return nil, promoErrors.ErrInvalidDiscountConfig.WithMessage("Unsupported promotion type")
	}

	if err := strategy.ValidateConfig(req.DiscountConfig); err != nil {
		log.ErrorWithContext(ctx, "Invalid discount config", err)
		return nil, err
	}

	// Validate date ranges
	if err := s.validateDateRanges(req.StartsAt, req.EndsAt); err != nil {
		log.ErrorWithContext(ctx, "Invalid date ranges", err)
		return nil, err
	}

	// Validate eligibility
	if err := s.validateEligibility(req.EligibleFor, req.CustomerSegmentID); err != nil {
		log.ErrorWithContext(ctx, "Invalid eligibility", err)
		return nil, err
	}

	// Check for duplicate slug if provided
	if req.Slug != nil && *req.Slug != "" {
		existingPromotion, err := s.promotionRepo.FindBySlug(ctx, *req.Slug, sellerID)
		if err != nil {
			log.ErrorWithContext(ctx, "Error checking slug uniqueness", err)
			return nil, err
		}
		if existingPromotion != nil {
			return nil, promoErrors.ErrPromotionSlugExists
		}
	}

	// Convert request to entity
	promotion := factory.PromotionRequestToEntity(req, sellerID)

	// Create promotion in database
	if err := s.promotionRepo.Create(ctx, promotion); err != nil {
		log.ErrorWithContext(ctx, "Failed to create promotion", err)
		return nil, commonError.NewAppError(
			"PROMOTION_CREATE_FAILED",
			"Failed to create promotion",
			http.StatusInternalServerError,
		)
	}

	log.InfoWithContext(ctx, "Promotion created successfully")

	// Convert entity to response
	response := factory.PromotionEntityToResponse(promotion)
	return response, nil
}

// validateDateRanges validates that StartsAt is before EndsAt
func (s *PromotionServiceImpl) validateDateRanges(startsAt *string, endsAt *string) error {
	if startsAt == nil {
		return promoErrors.ErrInvalidDateRange.WithMessage("startsAt is required")
	}

	startsAtTime, err := time.Parse(time.RFC3339, *startsAt)
	if err != nil {
		return promoErrors.ErrInvalidDateRange.WithMessage("startsAt must be in RFC3339 format")
	}

	if endsAt != nil && *endsAt != "" {
		endsAtTime, err := time.Parse(time.RFC3339, *endsAt)
		if err != nil {
			return promoErrors.ErrInvalidDateRange.WithMessage("endsAt must be in RFC3339 format")
		}

		if !endsAtTime.After(startsAtTime) {
			return promoErrors.ErrInvalidDateRange.WithMessage("endsAt must be after startsAt")
		}
	}

	return nil
}

// validateEligibility validates customer eligibility settings
func (s *PromotionServiceImpl) validateEligibility(
	eligibleFor entity.EligibilityType,
	customerSegmentID *uint,
) error {
	// If no eligibleFor is specified, default to everyone
	if eligibleFor == "" {
		return nil
	}

	// If specific_segment is selected, customerSegmentID is required
	if eligibleFor == entity.EligibleSpecificSegment {
		if customerSegmentID == nil || *customerSegmentID == 0 {
			return promoErrors.ErrInvalidEligibility.WithMessage(
				"customerSegmentId is required when eligibleFor is 'specific_segment'",
			)
		}
	}

	return nil
}
