package promotionStrategy

import (
	"context"

	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

// PromotionStrategy defines the interface for promotion validation and application
type PromotionStrategy interface {
	// ValidateConfig validates the discount config structure for the promotion type
	ValidateConfig(config map[string]interface{}) error

	// DescribeConfig returns the supported config fields and setup guidance for the promotion type.
	DescribeConfig() model.PromotionStrategyDescriptor

	// CalculateDiscount calculates per-item discounts for the promotion
	// effectivePrices maps ItemID -> current effective price per unit (after previous promotions)
	// This allows stacking: second promotion applies on the discounted price from the first
	CalculateDiscount(
		ctx context.Context,
		promotion *entity.Promotion,
		cart *model.CartValidationRequest,
		effectivePrices map[string]int64,
	) (*model.PromotionValidationResult, error)
}
