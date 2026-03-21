package promotionStrategy

import (
	"context"
	"encoding/json"
	"fmt"

	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// FreeShippingStrategy implements PromotionStrategy for free_shipping promotion type.
//
// Business Logic:
//
//	Waives the cart's shipping charge (cart.ShippingCents). This is the only strategy
//	that operates on shipping rather than item prices — no per-item discounts are
//	produced. The eligibleItems (scope) parameter is intentionally ignored because
//	shipping is a cart-wide cost, not per-item.
//
//	When stacking, the min-order check uses summary.FinalSubtotal (the effective
//	subtotal after all earlier promotions) so that a preceding deep discount can
//	legitimately push the subtotal below the free-shipping threshold.
//
// Config Fields:
//   - min_order_cents           (optional) : minimum subtotal to unlock free shipping
//   - max_shipping_discount_cents (optional) : cap on how much shipping to waive
//
// Example:
//
//	Config: { "min_order_cents": 50000, "max_shipping_discount_cents": 5000 }
//	Cart:
//	  Item A $600 x1  (line total = 60000)
//	  Shipping = 8000
//	FinalSubtotal (after earlier promos) = 60000  >=  50000 threshold  => qualifies
//	Shipping discount = min(8000, 5000) = 5000
//	No item discounts, but ShippingDiscount on the summary = 5000
type FreeShippingStrategy struct{}

// NewFreeShippingStrategy creates a new FreeShippingStrategy
func NewFreeShippingStrategy() PromotionStrategy {
	return &FreeShippingStrategy{}
}

// DescribeConfig returns the supported free-shipping fields and setup guidance.
func (s *FreeShippingStrategy) DescribeConfig() model.PromotionStrategyDescriptor {
	return model.PromotionStrategyDescriptor{
		PromotionType: entity.PromoTypeFreeShipping,
		Name:          "Free Shipping",
		Description:   "Waives shipping charges for eligible carts.",
		Fields: []model.PromotionConfigFieldDescriptor{
			{
				Name:        "min_order_cents",
				Type:        "int64",
				Required:    false,
				Description: "Optional minimum subtotal required to unlock free shipping.",
			},
			{
				Name:        "max_shipping_discount_cents",
				Type:        "int64",
				Required:    false,
				Description: "Optional cap on the shipping discount.",
			},
		},
		BestPractices: []string{
			"Use a minimum subtotal to protect margin on low-value orders.",
			"Keep stacking disabled unless free shipping is intended to combine with deep item discounts.",
			"Scope (appliesTo) is not applied to this promotion type: free shipping is always a cart-wide benefit regardless of which products are in scope.",
		},
	}
}

// ValidateConfig validates the free shipping configuration
func (s *FreeShippingStrategy) ValidateConfig(config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Invalid config format")
	}

	var shippingConfig model.FreeShippingConfig
	if err := json.Unmarshal(configJSON, &shippingConfig); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid free_shipping config structure",
		)
	}

	return nil
}

// CalculateDiscount updates the passed AppliedPromotionSummary in-place and
// returns an error if the promotion cannot be applied.
//
// Free shipping discounts the shipping charge only — no per-item discounts are applied.
// The min-order check uses summary.FinalSubtotal (effective subtotal after prior promotions)
// rather than the original cart subtotal, which is the correct behaviour when stacking.
// eligibleItems is intentionally ignored: the shipping discount is cart-wide, not item-scoped.
func (s *FreeShippingStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) (*model.SkippedPromotionReason, error) {
	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FreeShippingConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid free_shipping promotion configuration",
		)
	}
	if cart.ShippingCents == 0 {
		return &model.SkippedPromotionReason{
			Reason:      "NOT_APPLICABLE",
			Requirement: "Shipping is already free",
		}, nil
	}

	if config.MinOrderCents != nil && summary.FinalSubtotal < *config.MinOrderCents {
		shortfall := *config.MinOrderCents - summary.FinalSubtotal
		requirementFormat := "Add $%.2f more to qualify"
		return &model.SkippedPromotionReason{
			Reason:           "Minimum order amount not met for free shipping",
			Requirement:      fmt.Sprintf(requirementFormat, float64(shortfall)/100.0),
			PotentialSavings: cart.ShippingCents,
		}, nil
	}

	shippingDiscount := cart.ShippingCents
	if config.MaxShippingDiscountCents != nil &&
		shippingDiscount > *config.MaxShippingDiscountCents {
		shippingDiscount = *config.MaxShippingDiscountCents
	}

	if shippingDiscount <= 0 {
		return &model.SkippedPromotionReason{
			Reason: "No shipping discount applicable",
		}, nil
	}

	ApplyDiscountToSummary(summary, promotion, nil, 0, shippingDiscount)
	return nil, nil
}
