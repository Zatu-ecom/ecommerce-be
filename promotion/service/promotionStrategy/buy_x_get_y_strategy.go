package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// BuyXGetYStrategy implements PromotionStrategy for buy_x_get_y promotion type
type BuyXGetYStrategy struct{}

// NewBuyXGetYStrategy creates a new BuyXGetYStrategy
func NewBuyXGetYStrategy() PromotionStrategy {
	return &BuyXGetYStrategy{}
}

// ValidateConfig validates the buy X get Y configuration
func (s *BuyXGetYStrategy) ValidateConfig(config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Invalid config format")
	}

	var buyXGetYConfig model.BuyXGetYConfig
	if err := json.Unmarshal(configJSON, &buyXGetYConfig); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid buy_x_get_y config structure",
		)
	}

	if buyXGetYConfig.BuyQuantity <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"buy_quantity must be greater than 0",
		)
	}

	if buyXGetYConfig.GetQuantity <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"get_quantity must be greater than 0",
		)
	}

	if buyXGetYConfig.MaxSets != nil && *buyXGetYConfig.MaxSets <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"max_sets must be greater than 0 if specified",
		)
	}

	return nil
}

// CalculateDiscount calculates per-item buy X get Y discount
// Same product only - customer gets Y items free when buying X items
func (s *BuyXGetYStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) (*model.PromotionValidationResult, error) {
	result := &model.PromotionValidationResult{
		IsValid: false,
	}

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.BuyXGetYConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	var totalDiscountCents int64
	var itemDiscounts []model.ItemDiscount

	for _, item := range cart.Items {
		effectivePrice := effectivePrices[item.ItemID]
		if effectivePrice <= 0 || item.Quantity < config.BuyQuantity {
			continue
		}

		// Calculate how many complete sets (buy X + get Y)
		totalItemsPerSet := config.BuyQuantity + config.GetQuantity
		completeSets := item.Quantity / totalItemsPerSet
		if completeSets == 0 {
			completeSets = item.Quantity / config.BuyQuantity
		}

		if config.MaxSets != nil && completeSets > *config.MaxSets {
			completeSets = *config.MaxSets
		}

		if completeSets > 0 {
			freeItems := completeSets * config.GetQuantity
			itemDiscountCents := int64(freeItems) * effectivePrice
			totalDiscountCents += itemDiscountCents

			itemDiscounts = append(itemDiscounts, model.ItemDiscount{
				ItemID:        item.ItemID,
				ProductID:     item.ProductID,
				PromotionID:   promotion.ID,
				PromotionName: promotion.Name,
				DiscountCents: itemDiscountCents,
				OriginalCents: effectivePrice,
				FinalCents:    effectivePrice,
				FreeQuantity:  freeItems,
			})
		}
	}

	if totalDiscountCents == 0 {
		result.Reason = "Not enough items to qualify for buy X get Y promotion"
		return result, nil
	}

	result.IsValid = true
	result.DiscountCents = totalDiscountCents
	result.ItemDiscounts = itemDiscounts
	return result, nil
}
