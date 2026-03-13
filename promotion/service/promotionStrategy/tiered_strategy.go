package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// TieredStrategy implements PromotionStrategy for tiered promotion type
type TieredStrategy struct{}

// NewTieredStrategy creates a new TieredStrategy
func NewTieredStrategy() PromotionStrategy {
	return &TieredStrategy{}
}

// DescribeConfig returns the supported tiered fields and setup guidance.
func (s *TieredStrategy) DescribeConfig() model.PromotionStrategyDescriptor {
	return model.PromotionStrategyDescriptor{
		PromotionType: entity.PromoTypeTiered,
		Name:          "Tiered Discount",
		Description:   "Applies a discount tier based on quantity or spend thresholds.",
		Fields: []model.PromotionConfigFieldDescriptor{
			{
				Name:          "tier_type",
				Type:          "string",
				Required:      true,
				Description:   "How tiers are evaluated.",
				AllowedValues: []string{"quantity", "spend"},
			},
			{
				Name:        "tiers",
				Type:        "[]TierConfig",
				Required:    true,
				Description: "Ordered tier definitions describing thresholds and discount values.",
			},
		},
		BestPractices: []string{
			"Ensure tiers are ordered from strongest threshold to weakest to avoid ambiguity.",
			"Use spend tiers when cart mix varies heavily and quantity tiers when unit count is the real incentive.",
		},
	}
}

// ValidateConfig validates the tiered configuration
func (s *TieredStrategy) ValidateConfig(config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Invalid config format")
	}

	var tieredConfig model.TieredConfig
	if err := json.Unmarshal(configJSON, &tieredConfig); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid tiered config structure",
		)
	}

	if len(tieredConfig.Tiers) == 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("tiers cannot be empty")
	}

	if tieredConfig.TierType != "quantity" && tieredConfig.TierType != "spend" {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"tier_type must be 'quantity' or 'spend'",
		)
	}

	return nil
}

// CalculateDiscount calculates per-item tiered discount
func (s *TieredStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) (*model.PromotionValidationResult, error) {
	result := &model.PromotionValidationResult{
		IsValid: false,
	}

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.TieredConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	// Calculate check value based on effective prices
	var totalEffective int64
	var totalQuantity int
	for _, item := range cart.Items {
		totalEffective += effectivePrices[item.ItemID] * int64(item.Quantity)
		totalQuantity += item.Quantity
	}

	var checkValue int
	if config.TierType == "quantity" {
		checkValue = totalQuantity
	} else {
		checkValue = int(totalEffective)
	}

	var applicableTier *model.TierConfig
	for i := range config.Tiers {
		tier := &config.Tiers[i]
		if checkValue >= tier.Min {
			if tier.Max == nil || checkValue <= *tier.Max {
				applicableTier = tier
				break
			}
		}
	}

	if applicableTier == nil {
		result.Reason = "Does not meet minimum tier requirements"
		return result, nil
	}

	// Calculate total discount first
	var totalDiscount int64
	switch applicableTier.DiscountType {
	case "percentage":
		totalDiscount = int64(float64(totalEffective) * applicableTier.DiscountValue / 100)
	case "fixed_amount":
		totalDiscount = int64(applicableTier.DiscountValue)
	}

	if promotion.MaxDiscountAmountCents != nil &&
		totalDiscount > *promotion.MaxDiscountAmountCents {
		totalDiscount = *promotion.MaxDiscountAmountCents
	}
	if totalDiscount > totalEffective {
		totalDiscount = totalEffective
	}

	// Distribute proportionally across items
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
			itemDiscount = totalDiscount - distributed
		} else {
			itemDiscount = totalDiscount * itemTotal / totalEffective
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
	result.DiscountCents = totalDiscount
	result.ItemDiscounts = itemDiscounts
	return result, nil
}
