package service

import (
	"context"

	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

// PromotionScopeEligibilityService encapsulates cart-scope eligibility logic
// for a single promotion scope type and returns eligible cart line item IDs.
type PromotionScopeEligibilityService interface {
	IsCartEligible(
		ctx context.Context,
		promotionID uint,
		cart *model.CartValidationRequest,
	) (bool, []string)
}

type PromotionScopeEligibilityServiceFactory struct {
	promotionProductScopeServiceImpl    *PromotionProductScopeServiceImpl
	promotionCategoryScopeServiceImpl   *PromotionCategoryScopeServiceImpl
	promotionCollectionScopeServiceImpl *PromotionCollectionScopeServiceImpl
	promotionVariantScopeServiceImpl    *PromotionVariantScopeServiceImpl
}

func NewPromotionScopeEligibilityServiceFactory(
	promotionProductScopeServiceImpl *PromotionProductScopeServiceImpl,
	promotionCategoryScopeServiceImpl *PromotionCategoryScopeServiceImpl,
	promotionCollectionScopeServiceImpl *PromotionCollectionScopeServiceImpl,
	promotionVariantScopeServiceImpl *PromotionVariantScopeServiceImpl,
) *PromotionScopeEligibilityServiceFactory {
	return &PromotionScopeEligibilityServiceFactory{
		promotionProductScopeServiceImpl:    promotionProductScopeServiceImpl,
		promotionCategoryScopeServiceImpl:   promotionCategoryScopeServiceImpl,
		promotionCollectionScopeServiceImpl: promotionCollectionScopeServiceImpl,
		promotionVariantScopeServiceImpl:    promotionVariantScopeServiceImpl,
	}
}

// GetPromotionScopeEligibilityService returns the appropriate scope eligibility service based on the scope type
func (f *PromotionScopeEligibilityServiceFactory) GetPromotionScopeEligibilityService(
	scope entity.ScopeType,
) PromotionScopeEligibilityService {
	switch scope {
	case entity.ScopeAllProducts:
		return nil
	case entity.ScopeSpecificProducts:
		return f.promotionProductScopeServiceImpl
	case entity.ScopeSpecificCategories:
		return f.promotionCategoryScopeServiceImpl
	case entity.ScopeSpecificCollections:
		return f.promotionCollectionScopeServiceImpl
	case entity.ScopeSpecficVariant:
		return f.promotionVariantScopeServiceImpl
	default:
		return nil
	}
}
