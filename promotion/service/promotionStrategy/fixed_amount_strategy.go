package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/common/helper"
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
			{
				Name:        "amount_cents",
				Type:        "int64",
				Required:    true,
				Description: "Fixed discount amount in smallest currency unit.",
			},
			{
				Name:        "min_order_cents",
				Type:        "int64",
				Required:    false,
				Description: "Optional minimum eligible cart total (in smallest currency unit) required to apply the discount.",
			},
		},
		BestPractices: []string{
			"Use a minimum order amount to protect margin on low-value orders.",
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

	if fixedConfig.MinOrderCents != nil && *fixedConfig.MinOrderCents < 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"min_order_cents must be greater than or equal to 0",
		)
	}

	return nil
}

// CalculateDiscount updates the passed AppliedPromotionSummary in-place and
// returns an error if the promotion cannot be applied.
func (s *FixedAmountStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) error {
	eligibleItemsSet := helper.ToSet(eligibleItems)

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FixedAmountConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid fixed_amount promotion configuration",
		)
	}

	// Collect eligible summary items and compute total effective value using current
	// FinalPriceCents (line total) so previously applied promotions are factored in.
	var eligibleSummaryItems []*model.CartItemSummary
	var totalEffective int64
	for i := range summary.Items {
		item := &summary.Items[i]
		if !eligibleItemsSet[item.ItemID] {
			continue
		}
		eligibleSummaryItems = append(eligibleSummaryItems, item)
		totalEffective += item.FinalPriceCents // FinalPriceCents is the line total (PriceCents * Quantity)
	}

	if totalEffective <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No eligible items for fixed amount promotion",
		)
	}

	if config.MinOrderCents != nil && totalEffective < *config.MinOrderCents {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Minimum order amount not met")
	}

	discountCents := config.AmountCents
	if discountCents > totalEffective {
		discountCents = totalEffective
	}

	// Distribute discount proportionally across eligible items; last item gets the
	// remainder to prevent cents from being lost to integer truncation.
	itemDiscounts := make([]ItemDiscountDetail, 0, len(eligibleSummaryItems))
	var distributed int64

	for i, item := range eligibleSummaryItems {
		if item.FinalPriceCents <= 0 {
			continue
		}

		// FinalPriceCents is the line total; use it directly for proportional distribution.
		var itemDiscount int64
		if i == len(eligibleSummaryItems)-1 {
			itemDiscount = discountCents - distributed
		} else {
			itemDiscount = discountCents * item.FinalPriceCents / totalEffective
		}

		if itemDiscount > item.FinalPriceCents {
			itemDiscount = item.FinalPriceCents
		}
		if itemDiscount > 0 {
			distributed += itemDiscount
			itemDiscounts = append(itemDiscounts, ItemDiscountDetail{
				ItemID:        item.ItemID,
				DiscountCents: itemDiscount,
			})
		}
	}

	if distributed == 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No discount applicable for fixed amount promotion",
		)
	}

	ApplyDiscountToSummary(summary, promotion, itemDiscounts, distributed, 0)
	return nil
}
