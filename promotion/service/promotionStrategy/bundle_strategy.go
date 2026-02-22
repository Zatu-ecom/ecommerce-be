package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// BundleStrategy implements PromotionStrategy for bundle promotion type
type BundleStrategy struct{}

// NewBundleStrategy creates a new BundleStrategy
func NewBundleStrategy() PromotionStrategy {
	return &BundleStrategy{}
}

// ValidateConfig validates the bundle configuration
func (s *BundleStrategy) ValidateConfig(config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Invalid config format")
	}

	var bundleConfig model.BundleConfig
	if err := json.Unmarshal(configJSON, &bundleConfig); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid bundle config structure",
		)
	}

	if len(bundleConfig.BundleItems) == 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("bundle_items cannot be empty")
	}

	if bundleConfig.BundleDiscountType == "fixed_price" {
		if bundleConfig.BundlePriceCents == nil {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage(
				"bundle_price_cents required for fixed_price type",
			)
		}
	} else {
		if bundleConfig.BundleDiscountValue == nil {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage(
				"bundle_discount_value required for percentage/fixed_amount type",
			)
		}
	}

	return nil
}

// CalculateDiscount calculates per-item bundle discount
func (s *BundleStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) (*model.PromotionValidationResult, error) {
	result := &model.PromotionValidationResult{
		IsValid: false,
	}

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.BundleConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	// Track matched cart items for the bundle
	type matchedItem struct {
		cartItem model.CartItem
		quantity int
	}
	var matched []matchedItem
	var bundleTotalCents int64

	for _, bundleItem := range config.BundleItems {
		found := false
		for _, cartItem := range cart.Items {
			if cartItem.ProductID == bundleItem.ProductID {
				if bundleItem.VariantID == nil ||
					(cartItem.VariantID != nil && *cartItem.VariantID == *bundleItem.VariantID) {
					if cartItem.Quantity >= bundleItem.Quantity {
						found = true
						effectivePrice := effectivePrices[cartItem.ItemID]
						bundleTotalCents += effectivePrice * int64(bundleItem.Quantity)
						matched = append(
							matched,
							matchedItem{cartItem: cartItem, quantity: bundleItem.Quantity},
						)
						break
					}
				}
			}
		}
		if !found {
			result.Reason = "Cart does not contain all required bundle items"
			return result, nil
		}
	}

	var totalDiscount int64
	switch config.BundleDiscountType {
	case "fixed_price":
		totalDiscount = bundleTotalCents - *config.BundlePriceCents
	case "percentage":
		totalDiscount = int64(float64(bundleTotalCents) * (*config.BundleDiscountValue) / 100)
	case "fixed_amount":
		totalDiscount = int64(*config.BundleDiscountValue)
	}

	if totalDiscount > bundleTotalCents {
		totalDiscount = bundleTotalCents
	}
	if totalDiscount <= 0 {
		result.Reason = "No discount applicable for bundle"
		return result, nil
	}

	// Distribute discount proportionally across bundle items
	var itemDiscounts []model.ItemDiscount
	var distributed int64

	for i, m := range matched {
		effectivePrice := effectivePrices[m.cartItem.ItemID]
		itemTotal := effectivePrice * int64(m.quantity)
		var itemDiscount int64

		if i == len(matched)-1 {
			itemDiscount = totalDiscount - distributed
		} else {
			itemDiscount = totalDiscount * itemTotal / bundleTotalCents
		}

		if itemDiscount > itemTotal {
			itemDiscount = itemTotal
		}
		if itemDiscount > 0 {
			distributed += itemDiscount
			itemDiscounts = append(itemDiscounts, model.ItemDiscount{
				ItemID:        m.cartItem.ItemID,
				ProductID:     m.cartItem.ProductID,
				PromotionID:   promotion.ID,
				PromotionName: promotion.Name,
				DiscountCents: itemDiscount,
				OriginalCents: effectivePrice,
				FinalCents:    effectivePrice - (itemDiscount / int64(m.quantity)),
			})
		}
	}

	result.IsValid = true
	result.DiscountCents = totalDiscount
	result.ItemDiscounts = itemDiscounts
	return result, nil
}
