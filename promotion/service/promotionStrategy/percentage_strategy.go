package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/common/helper"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// PercentageStrategy implements PromotionStrategy for percentage_discount promotion type
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

// CalculateDiscount calculates per-item percentage discount
func (s *PercentageStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) (*model.PromotionValidationResult, error) {
	result := &model.PromotionValidationResult{
		IsValid: false,
	}

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.PercentageDiscountConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	var totalDiscount int64
	var itemDiscounts []model.ItemDiscount

	for _, item := range cart.Items {
		effectivePrice := effectivePrices[item.ItemID]
		if effectivePrice <= 0 {
			continue
		}

		itemTotal := effectivePrice * int64(item.Quantity)
		itemDiscount := int64(float64(itemTotal) * config.Percentage / 100)

		if itemDiscount > 0 {
			totalDiscount += itemDiscount
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

	if totalDiscount == 0 {
		result.Reason = "No discount applicable"
		return result, nil
	}

	// Apply max discount caps
	if config.MaxDiscountCents != nil && totalDiscount > *config.MaxDiscountCents {
		totalDiscount = *config.MaxDiscountCents
	}
	if promotion.MaxDiscountAmountCents != nil &&
		totalDiscount > *promotion.MaxDiscountAmountCents {
		totalDiscount = *promotion.MaxDiscountAmountCents
	}

	result.IsValid = true
	result.DiscountCents = totalDiscount
	result.ItemDiscounts = itemDiscounts
	return result, nil
}

// CalculateDiscountV2 is the enhanced version of CalculateDiscount that will update the summary in-place and
// return error if promotion cannot be applied
func (s *PercentageStrategy) CalculateDiscountV2(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) error {
	eligibleItemsSet := helper.ToSet(eligibleItems)

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.PercentageDiscountConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
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
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No eligible items for percentage discount promotion",
		)
	}

	if config.MinOrderCents != nil && totalEffective < *config.MinOrderCents {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Minimum order amount not met")
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
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No discount applicable for percentage discount promotion",
		)
	}

	// Apply caps. If the total is capped, redistribute the reduction proportionally
	// across items so individual ItemDiscountDetail values stay consistent.
	effectiveCap := totalDiscount
	if config.MaxDiscountCents != nil && effectiveCap > *config.MaxDiscountCents {
		effectiveCap = *config.MaxDiscountCents
	}
	if promotion.MaxDiscountAmountCents != nil && effectiveCap > *promotion.MaxDiscountAmountCents {
		effectiveCap = *promotion.MaxDiscountAmountCents
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
	return nil
}
