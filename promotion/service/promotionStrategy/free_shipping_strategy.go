package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// FreeShippingStrategy implements PromotionStrategy for free_shipping promotion type
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

// CalculateDiscount calculates shipping discount (no per-item discounts)
func (s *FreeShippingStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) (*model.PromotionValidationResult, error) {
	result := &model.PromotionValidationResult{
		IsValid: false,
	}

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FreeShippingConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	if config.MinOrderCents != nil && cart.SubtotalCents < *config.MinOrderCents {
		result.Reason = "Minimum order amount not met for free shipping"
		return result, nil
	}

	shippingDiscount := cart.ShippingCents
	if config.MaxShippingDiscountCents != nil &&
		shippingDiscount > *config.MaxShippingDiscountCents {
		shippingDiscount = *config.MaxShippingDiscountCents
	}

	result.IsValid = true
	result.ShippingDiscount = shippingDiscount
	return result, nil
}
