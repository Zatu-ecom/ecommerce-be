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

// PercentageStrategy implements PromotionStrategy for percentage_discount promotion type.
//
// Business Logic:
//   Reduces every eligible item's current line total (FinalPriceCents) by the configured
//   percentage. Each item's discount is computed independently and the results are summed.
//   If max_discount_cents is set and the raw total exceeds it, the capped amount is
//   redistributed proportionally across items so no individual item is over-discounted.
//   When promotions are stacked, FinalPriceCents already reflects earlier discounts, so
//   the percentage is applied to the reduced price — not the original.
//
// Config Fields:
//   - percentage        (required) : 0.01 – 100
//   - max_discount_cents (optional) : upper cap on total discount
//   - min_order_cents    (optional) : minimum eligible subtotal to qualify
//
// Example:
//   Config: { "percentage": 20, "max_discount_cents": 15000 }
//   Cart:
//     Item A  $500 x1  (line total = 50000)
//     Item B  $300 x1  (line total = 30000)
//   Eligible subtotal = 80000
//   Raw discount: Item A = 10000, Item B = 6000   => total = 16000
//   Capped to 15000 => redistribute proportionally:
//     Item A capped = 15000 * 10000 / 16000 = 9375
//     Item B capped = 15000 - 9375          = 5625
//   Final subtotal = 80000 - 15000 = 65000
type PercentageStrategy struct{}

// NewPercentageStrategy creates a new PercentageStrategy
func NewPercentageStrategy() PromotionStrategy {
	return &PercentageStrategy{}
}

// DescribeConfig returns the supported percentage discount fields and setup guidance.
func (s *PercentageStrategy) DescribeConfig() model.PromotionStrategyDescriptor {
	return model.PromotionStrategyDescriptor{
		PromotionType: entity.PromoTypePercentage,
		Name:          "Percentage Discount",
		Description:   "Reduces eligible item prices by a percentage.",
		Fields: []model.PromotionConfigFieldDescriptor{
			{
				Name:        "percentage",
				Type:        "float64",
				Required:    true,
				Description: "Discount percentage between 0.01 and 100.",
			},
			{
				Name:        "max_discount_cents",
				Type:        "int64",
				Required:    false,
				Description: "Optional cap on the total percentage discount.",
			},
			{
				Name:        "min_order_cents",
				Type:        "int64",
				Required:    false,
				Description: "Optional minimum eligible cart total required to apply the discount.",
			},
		},
		BestPractices: []string{
			"Use a max discount cap for high-value catalogs.",
			"Disable stacking unless combined discounts are an explicit business decision.",
		},
	}
}

// ValidateConfig validates the percentage discount configuration
func (s *PercentageStrategy) ValidateConfig(config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Invalid config format")
	}

	var percentageConfig model.PercentageDiscountConfig
	if err := json.Unmarshal(configJSON, &percentageConfig); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid percentage_discount config structure",
		)
	}

	if percentageConfig.Percentage <= 0 || percentageConfig.Percentage > 100 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"percentage must be between 0.01 and 100",
		)
	}

	if percentageConfig.MinOrderCents != nil && *percentageConfig.MinOrderCents < 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"min_order_cents must be greater than or equal to 0",
		)
	}

	return nil
}

// CalculateDiscount updates the passed AppliedPromotionSummary in-place and
// returns an error if the promotion cannot be applied.
func (s *PercentageStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) (*model.SkippedPromotionReason, error) {
	eligibleItemsSet := helper.ToSet(eligibleItems)

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.PercentageDiscountConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid percentage_discount promotion configuration",
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
		totalEffective += item.FinalPriceCents
	}

	if totalEffective <= 0 {
		return &model.SkippedPromotionReason{
			Reason: "No eligible items for percentage discount promotion",
		}, nil
	}

	if config.MinOrderCents != nil && totalEffective < *config.MinOrderCents {
		shortfall := *config.MinOrderCents - totalEffective
		requirementFormat := "Add $%.2f more to qualify"
		potentialSavings := int64(float64(*config.MinOrderCents) * config.Percentage / 100)
		if config.MaxDiscountCents != nil && potentialSavings > *config.MaxDiscountCents {
			potentialSavings = *config.MaxDiscountCents
		}
		return &model.SkippedPromotionReason{
			Reason:           "Minimum order amount not met",
			Requirement:      fmt.Sprintf(requirementFormat, float64(shortfall)/100.0),
			PotentialSavings: potentialSavings,
		}, nil
	}

	// Compute per-item discount as a fraction of each item's current line total.
	itemDiscounts := make([]ItemDiscountDetail, 0, len(eligibleSummaryItems))
	var totalDiscount int64

	for _, item := range eligibleSummaryItems {
		if item.FinalPriceCents <= 0 {
			continue
		}
		itemDiscount := int64(float64(item.FinalPriceCents) * config.Percentage / 100)
		if itemDiscount <= 0 {
			continue
		}
		totalDiscount += itemDiscount
		itemDiscounts = append(itemDiscounts, ItemDiscountDetail{
			ItemID:        item.ItemID,
			DiscountCents: itemDiscount,
		})
	}

	if totalDiscount == 0 {
		return &model.SkippedPromotionReason{
			Reason: "No discount applicable for percentage discount promotion",
		}, nil
	}

	// Apply caps. If the total is capped, redistribute the reduction proportionally
	// across items so individual ItemDiscountDetail values stay consistent.
	effectiveCap := totalDiscount
	if config.MaxDiscountCents != nil && effectiveCap > *config.MaxDiscountCents {
		effectiveCap = *config.MaxDiscountCents
	}

	if effectiveCap < totalDiscount {
		// Redistribute the capped total proportionally; last item absorbs rounding remainder.
		var distributed int64
		for i := range itemDiscounts {
			var capped int64
			if i == len(itemDiscounts)-1 {
				capped = effectiveCap - distributed
			} else {
				capped = effectiveCap * itemDiscounts[i].DiscountCents / totalDiscount
			}
			itemDiscounts[i].DiscountCents = capped
			distributed += capped
		}
		totalDiscount = effectiveCap
	}

	ApplyDiscountToSummary(summary, promotion, itemDiscounts, totalDiscount, 0)
	return nil, nil
}
