package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// FixedAmountStrategy implements PromotionStrategy for fixed_amount promotion type
type FixedAmountStrategy struct{}

// NewFixedAmountStrategy creates a new FixedAmountStrategy
func NewFixedAmountStrategy() PromotionStrategy {
	return &FixedAmountStrategy{}
}

// DescribeConfig returns the supported fixed-amount discount fields and setup guidance.
func (s *FixedAmountStrategy) DescribeConfig() model.PromotionStrategyDescriptor {
	return model.PromotionStrategyDescriptor{
		PromotionType: entity.PromoTypeFixedAmount,
		Name:          "Fixed Amount Discount",
		Description:   "Applies a fixed cart-level discount distributed across eligible items.",
		Fields: []model.PromotionConfigFieldDescriptor{
			{Name: "amount_cents", Type: "int64", Required: true, Description: "Fixed discount amount in smallest currency unit."},
		},
		BestPractices: []string{
			"Keep the amount below the typical cart subtotal to avoid zeroing most carts.",
			"Disable stacking unless combined discounts are intentionally allowed.",
		},
	}
}

// ValidateConfig validates the fixed amount configuration
func (s *FixedAmountStrategy) ValidateConfig(config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Invalid config format")
	}

	var fixedConfig model.FixedAmountConfig
	if err := json.Unmarshal(configJSON, &fixedConfig); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid fixed_amount config structure",
		)
	}

	if fixedConfig.AmountCents <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"amount_cents must be greater than 0",
		)
	}

	return nil
}

// CalculateDiscount calculates per-item fixed amount discount (distributed proportionally)
func (s *FixedAmountStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) (*model.PromotionValidationResult, error) {
	result := &model.PromotionValidationResult{
		IsValid: false,
	}

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FixedAmountConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	// Calculate total effective cart value
	var totalEffective int64
	for _, item := range cart.Items {
		totalEffective += effectivePrices[item.ItemID] * int64(item.Quantity)
	}

	if totalEffective <= 0 {
		result.Reason = "No eligible items"
		return result, nil
	}

	discountCents := config.AmountCents
	if discountCents > totalEffective {
		discountCents = totalEffective
	}
	if promotion.MaxDiscountAmountCents != nil &&
		discountCents > *promotion.MaxDiscountAmountCents {
		discountCents = *promotion.MaxDiscountAmountCents
	}

	// Distribute discount proportionally across items
	var itemDiscounts []model.ItemDiscount
	var distributed int64

	for i, item := range cart.Items {
		effectivePrice := effectivePrices[item.ItemID]
		if effectivePrice <= 0 {
			continue
		}

		itemTotal := effectivePrice * int64(item.Quantity)
		var itemDiscount int64

		if i == len(cart.Items)-1 {
			// Last item gets remainder to avoid rounding issues
			itemDiscount = discountCents - distributed
		} else {
			itemDiscount = discountCents * itemTotal / totalEffective
		}

		if itemDiscount > itemTotal {
			itemDiscount = itemTotal
		}
		if itemDiscount > 0 {
			distributed += itemDiscount
			itemDiscounts = append(itemDiscounts, model.ItemDiscount{
				ItemID:        item.ItemID,
				ProductID:     item.ProductID,
				PromotionID:   promotion.ID,
				PromotionName: promotion.Name,
				DiscountCents: itemDiscount,
				OriginalCents: effectivePrice,
				FinalCents:    effectivePrice - (itemDiscount / int64(item.Quantity)),
			})
		}
	}

	result.IsValid = true
	result.DiscountCents = discountCents
	result.ItemDiscounts = itemDiscounts
	return result, nil
}
