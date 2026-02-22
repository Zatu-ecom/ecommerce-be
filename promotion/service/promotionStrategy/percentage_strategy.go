package promotionStrategy

import (
	"context"
	"encoding/json"

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
