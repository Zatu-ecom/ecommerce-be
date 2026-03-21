package promotionStrategy

import (
	"context"
	"encoding/json"
	"fmt"

	"ecommerce-be/common/helper"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// FixedAmountStrategy implements PromotionStrategy for fixed_amount promotion type.
//
// Business Logic:
//
//	A single flat discount (amount_cents) is subtracted from the total of all eligible
//	items in the cart. The discount is distributed proportionally across eligible items
//	based on each item's share of the eligible subtotal. If the discount exceeds the
//	eligible subtotal it is clamped so the total never goes negative.
//
// Config Fields:
//   - amount_cents  (required) : flat discount in smallest currency unit
//   - min_order_cents (optional) : minimum eligible subtotal before the discount applies
//
// Example:
//
//	Config: { "amount_cents": 30000 }     (i.e. $300 off)
//	Cart:
//	  Item A  $900 x1  (line total = 90000)    75% of eligible subtotal
//	  Item B  $300 x1  (line total = 30000)    25% of eligible subtotal
//	Eligible subtotal = 120000
//	Total discount    = 30000 (capped to eligible subtotal if larger)
//	Item A discount   = 30000 * 90000 / 120000 = 22500
//	Item B discount   = 30000 - 22500          =  7500  (last item gets remainder)
//	Final subtotal    = 120000 - 30000         = 90000
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
) (*model.SkippedPromotionReason, error) {
	eligibleItemsSet := helper.ToSet(eligibleItems)

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FixedAmountConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, promoErrors.ErrInvalidDiscountConfig.WithMessage(
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
		return &model.SkippedPromotionReason{
			Reason: "No eligible items for fixed amount promotion",
		}, nil
	}

	if config.MinOrderCents != nil && totalEffective < *config.MinOrderCents {
		shortfall := *config.MinOrderCents - totalEffective
		requirementFormat := "Add $%.2f more to qualify"
		return &model.SkippedPromotionReason{
			Reason:           "Minimum order amount not met",
			Requirement:      fmt.Sprintf(requirementFormat, float64(shortfall)/100.0),
			PotentialSavings: config.AmountCents,
		}, nil
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
		return &model.SkippedPromotionReason{
			Reason: "No discount applicable for fixed amount promotion",
		}, nil
	}

	ApplyDiscountToSummary(summary, promotion, itemDiscounts, distributed, 0)
	return nil, nil
}
