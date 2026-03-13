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

type matchedBundleItem struct {
	cartItem model.CartItem
	required int
}

// NewBundleStrategy creates a new BundleStrategy
func NewBundleStrategy() PromotionStrategy {
	return &BundleStrategy{}
}

// DescribeConfig returns the supported bundle fields and setup guidance.
func (s *BundleStrategy) DescribeConfig() model.PromotionStrategyDescriptor {
	return model.PromotionStrategyDescriptor{
		PromotionType: entity.PromoTypeBundle,
		Name:          "Bundle",
		Description:   "Applies a bundle-specific discount when the required items are present together.",
		Fields: []model.PromotionConfigFieldDescriptor{
			{
				Name:        "bundle_items",
				Type:        "[]BundleItemConfig",
				Required:    true,
				Description: "Required products or variants and their quantities.",
			},
			{
				Name:          "bundle_discount_type",
				Type:          "string",
				Required:      true,
				Description:   "Bundle discount calculation mode.",
				AllowedValues: []string{"percentage", "fixed_amount", "fixed_price"},
			},
			{
				Name:        "bundle_discount_value",
				Type:        "float64",
				Required:    false,
				Description: "Required for percentage and fixed_amount bundles.",
			},
			{
				Name:        "bundle_price_cents",
				Type:        "int64",
				Required:    false,
				Description: "Required for fixed_price bundles.",
			},
		},
		BestPractices: []string{
			"Use exact variants for high-value bundles so unintended substitutions do not qualify.",
			"Prefer non-stackable bundles unless layered discounts are intentional.",
		},
	}
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

	config, ok := s.parseBundleConfig(promotion.DiscountConfig)
	if !ok {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	matchedItems, completeSets, reason := s.matchBundleItems(config, cart)
	if reason != "" {
		result.Reason = reason
		return result, nil
	}

	bundleTotalCents := s.calculateBundleTotalCents(matchedItems, completeSets, effectivePrices)
	totalDiscount := s.calculateTotalDiscount(config, bundleTotalCents, completeSets)

	if totalDiscount > bundleTotalCents {
		totalDiscount = bundleTotalCents
	}
	if totalDiscount <= 0 {
		result.Reason = "No discount applicable for bundle"
		return result, nil
	}

	result.IsValid = true
	result.DiscountCents = totalDiscount
	result.ItemDiscounts = s.distributeDiscountAcrossItems(
		promotion,
		matchedItems,
		completeSets,
		bundleTotalCents,
		totalDiscount,
		effectivePrices,
	)
	return result, nil
}

func (s *BundleStrategy) parseBundleConfig(
	discountConfig map[string]interface{},
) (model.BundleConfig, bool) {
	configJSON, _ := json.Marshal(discountConfig)

	var config model.BundleConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return model.BundleConfig{}, false
	}

	return config, true
}

func (s *BundleStrategy) matchBundleItems(
	config model.BundleConfig,
	cart *model.CartValidationRequest,
) ([]matchedBundleItem, int, string) {
	var matchedItems []matchedBundleItem
	completeSets := -1

	for _, bundleItem := range config.BundleItems {
		found := false
		for _, cartItem := range cart.Items {
			if cartItem.ProductID != bundleItem.ProductID {
				continue
			}

			variantMatch := bundleItem.VariantID == nil ||
				(cartItem.VariantID != nil && *cartItem.VariantID == *bundleItem.VariantID)
			if !variantMatch || cartItem.Quantity < bundleItem.Quantity {
				continue
			}

			found = true
			setsForItem := cartItem.Quantity / bundleItem.Quantity
			if completeSets == -1 || setsForItem < completeSets {
				completeSets = setsForItem
			}
			matchedItems = append(matchedItems, matchedBundleItem{
				cartItem: cartItem,
				required: bundleItem.Quantity,
			})
			break
		}

		if !found {
			return nil, 0, "Cart does not contain all required bundle items"
		}
	}

	if completeSets <= 0 {
		return nil, 0, "No complete bundle sets found"
	}

	return matchedItems, completeSets, ""
}

func (s *BundleStrategy) calculateBundleTotalCents(
	matchedItems []matchedBundleItem,
	completeSets int,
	effectivePrices map[string]int64,
) int64 {
	var bundleTotalCents int64
	for _, item := range matchedItems {
		effectivePrice := effectivePrices[item.cartItem.ItemID]
		usedQty := item.required * completeSets
		bundleTotalCents += effectivePrice * int64(usedQty)
	}

	return bundleTotalCents
}

func (s *BundleStrategy) calculateTotalDiscount(
	config model.BundleConfig,
	bundleTotalCents int64,
	completeSets int,
) int64 {
	switch config.BundleDiscountType {
	case "fixed_price":
		return bundleTotalCents - (*config.BundlePriceCents * int64(completeSets))
	case "percentage":
		return int64(float64(bundleTotalCents) * (*config.BundleDiscountValue) / 100)
	case "fixed_amount":
		return int64(*config.BundleDiscountValue) * int64(completeSets)
	default:
		return 0
	}
}

func (s *BundleStrategy) distributeDiscountAcrossItems(
	promotion *entity.Promotion,
	matchedItems []matchedBundleItem,
	completeSets int,
	bundleTotalCents int64,
	totalDiscount int64,
	effectivePrices map[string]int64,
) []model.ItemDiscount {
	var itemDiscounts []model.ItemDiscount
	var distributed int64

	for i, item := range matchedItems {
		effectivePrice := effectivePrices[item.cartItem.ItemID]
		usedQty := item.required * completeSets
		itemTotal := effectivePrice * int64(usedQty)
		itemDiscount := totalDiscount * itemTotal / bundleTotalCents
		if i == len(matchedItems)-1 {
			itemDiscount = totalDiscount - distributed
		}

		if itemDiscount > itemTotal {
			itemDiscount = itemTotal
		}
		if itemDiscount <= 0 {
			continue
		}

		distributed += itemDiscount
		itemDiscounts = append(itemDiscounts, model.ItemDiscount{
			ItemID:        item.cartItem.ItemID,
			ProductID:     item.cartItem.ProductID,
			PromotionID:   promotion.ID,
			PromotionName: promotion.Name,
			DiscountCents: itemDiscount,
			OriginalCents: effectivePrice,
			// Discount is distributed over the full cart line quantity so downstream
			// summary/stacking logic uses a consistent effective unit price.
			FinalCents: effectivePrice - (itemDiscount / int64(item.cartItem.Quantity)),
		})
	}

	return itemDiscounts
}
