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

	// CalculateDiscount updates the passed AppliedPromotionSummary in-place and
	// returns an error if the promotion cannot be applied.
	CalculateDiscount(
		ctx context.Context,
		promotion *entity.Promotion,
		cart *model.CartValidationRequest,
		summary *model.AppliedPromotionSummary,
		eligibleItems []string,
	) (*model.SkippedPromotionReason, error)
}
