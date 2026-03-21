package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/common/helper"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// BundleStrategy implements PromotionStrategy for bundle promotion type.
//
// Business Logic:
//   Requires all configured bundle_items (product + optional variant, with min quantities)
//   to be present in the cart. The number of "complete sets" is the minimum across all
//   bundle items of (cart qty / required qty). The discount applies only to the units
//   that form complete sets, not the entire cart line.
//
//   Three discount modes are supported via bundle_discount_type:
//     - "percentage"    : takes bundle_discount_value % off the bundle subtotal
//     - "fixed_amount"  : subtracts bundle_discount_value per complete set
//     - "fixed_price"   : sets the bundle to bundle_price_cents per complete set;
//                         discount = bundleSubtotal - (bundlePriceCents * completeSets)
//
//   The computed discount is distributed proportionally across the matched items
//   based on each item's share of the bundle subtotal. When promotions are stacked,
//   effective prices from FinalPriceCents (post-earlier-discount) are used.
//
// Config Fields:
//   - bundle_items          (required) : array of { product_id, variant_id?, quantity }
//   - bundle_discount_type  (required) : "percentage" | "fixed_amount" | "fixed_price"
//   - bundle_discount_value (required for percentage/fixed_amount)
//   - bundle_price_cents    (required for fixed_price)
//
// Example (fixed_price):
//   Config: { bundle_items: [{product_id:1, qty:1}, {product_id:2, qty:1}],
//             bundle_discount_type: "fixed_price", bundle_price_cents: 80000 }
//   Cart:
//     Item A  product 1  $600 x1  (line total = 60000)
//     Item B  product 2  $400 x1  (line total = 40000)
//   Complete sets = 1
//   Bundle subtotal = 100000
//   Discount = 100000 - 80000 = 20000
//   Item A discount = 20000 * 60000 / 100000 = 12000
//   Item B discount = 20000 - 12000          =  8000
//   Final subtotal  = 100000 - 20000         = 80000
type BundleStrategy struct{}

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
				Name:        "bundle_discount_type",
				Type:        "string",
				Required:    true,
				Description: "Bundle discount calculation mode.",
				AllowedValues: []string{
					string(model.DiscountTypePercentage),
					string(model.DiscountTypeFixedAmount),
					string(model.DiscountTypeFixedPrice),
				},
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

	if bundleConfig.BundleDiscountType == model.DiscountTypeFixedPrice {
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

// CalculateDiscount updates the passed AppliedPromotionSummary in-place and
// returns an error if the promotion cannot be applied.
func (s *BundleStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) (*model.SkippedPromotionReason, error) {
	config, ok := s.parseBundleConfig(promotion.DiscountConfig)
	if !ok {
		return nil, promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid bundle promotion configuration",
		)
	}

	eligibleItemsSet := helper.ToSet(eligibleItems)

	matchedItems, completeSets, reason := s.matchBundleItemsV2(
		config,
		cart,
		summary,
		eligibleItemsSet,
	)
	if reason != "" {
		return &model.SkippedPromotionReason{
			Reason:      "NOT_MET",
			Requirement: "Add required bundle items to qualify",
		}, nil
	}

	bundleTotalCents := s.calculateBundleTotalCentsV2(matchedItems, completeSets)
	totalDiscount := s.calculateTotalDiscount(config, bundleTotalCents, completeSets)

	if totalDiscount > bundleTotalCents {
		totalDiscount = bundleTotalCents
	}
	if totalDiscount <= 0 {
		return &model.SkippedPromotionReason{
			Reason: "No discount applicable for bundle",
		}, nil
	}

	// Distribute proportionally across matched items; last item gets remainder
	// to prevent integer truncation from losing cents.
	itemDiscounts := make([]ItemDiscountDetail, 0, len(matchedItems))
	var distributed int64

	for i, item := range matchedItems {
		usedQty := item.required * completeSets
		// Per-unit effective price derived from the current line total so stacked
		// promotions reduce the base for subsequent promotions.
		perUnitEffective := item.summaryItem.FinalPriceCents / int64(item.summaryItem.Quantity)
		itemLineTotal := perUnitEffective * int64(usedQty)

		var itemDiscount int64
		if i == len(matchedItems)-1 {
			itemDiscount = totalDiscount - distributed
		} else {
			itemDiscount = totalDiscount * itemLineTotal / bundleTotalCents
		}

		if itemDiscount > itemLineTotal {
			itemDiscount = itemLineTotal
		}
		if itemDiscount <= 0 {
			continue
		}

		distributed += itemDiscount
		itemDiscounts = append(itemDiscounts, ItemDiscountDetail{
			ItemID:        item.summaryItem.ItemID,
			DiscountCents: itemDiscount,
		})
	}

	if distributed == 0 {
		return &model.SkippedPromotionReason{
			Reason: "No discount applicable for bundle",
		}, nil
	}

	ApplyDiscountToSummary(summary, promotion, itemDiscounts, distributed, 0)
	return nil, nil
}

// matchedBundleItemV2 pairs a cart item (for product/variant matching) with its
// corresponding CartItemSummary (for current effective price) and the required qty.
type matchedBundleItemV2 struct {
	cartItem    model.CartItem
	summaryItem *model.CartItemSummary
	required    int
}

// matchBundleItemsV2 mirrors matchBundleItems but filters by eligibleItemsSet and
// resolves the live CartItemSummary for each match so V2 uses current FinalPriceCents.
func (s *BundleStrategy) matchBundleItemsV2(
	config model.BundleConfig,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItemsSet map[string]bool,
) ([]matchedBundleItemV2, int, string) {
	// Build a lookup from ItemID -> index in summary.Items for O(1) access
	summaryIndexByID := make(map[string]int, len(summary.Items))
	for i, item := range summary.Items {
		summaryIndexByID[item.ItemID] = i
	}

	var matchedItems []matchedBundleItemV2
	completeSets := -1

	for _, bundleItem := range config.BundleItems {
		found := false
		for _, cartItem := range cart.Items {
			if !eligibleItemsSet[cartItem.ItemID] {
				continue
			}
			if cartItem.ProductID != bundleItem.ProductID {
				continue
			}

			variantMatch := bundleItem.VariantID == nil ||
				(cartItem.VariantID != nil && *cartItem.VariantID == *bundleItem.VariantID)
			if !variantMatch || cartItem.Quantity < bundleItem.Quantity {
				continue
			}

			summaryIdx, ok := summaryIndexByID[cartItem.ItemID]
			if !ok {
				continue
			}

			found = true
			setsForItem := cartItem.Quantity / bundleItem.Quantity
			if completeSets == -1 || setsForItem < completeSets {
				completeSets = setsForItem
			}
			matchedItems = append(matchedItems, matchedBundleItemV2{
				cartItem:    cartItem,
				summaryItem: &summary.Items[summaryIdx],
				required:    bundleItem.Quantity,
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

// calculateBundleTotalCentsV2 sums the effective line value for the units participating
// in complete bundle sets. Effective per-unit price is derived from the current
// FinalPriceCents (line total) so any previously applied stacked promotions are
// reflected in the bundle base value.
func (s *BundleStrategy) calculateBundleTotalCentsV2(
	matchedItems []matchedBundleItemV2,
	completeSets int,
) int64 {
	var bundleTotalCents int64
	for _, item := range matchedItems {
		usedQty := item.required * completeSets
		perUnitEffective := item.summaryItem.FinalPriceCents / int64(item.summaryItem.Quantity)
		bundleTotalCents += perUnitEffective * int64(usedQty)
	}
	return bundleTotalCents
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

func (s *BundleStrategy) calculateTotalDiscount(
	config model.BundleConfig,
	bundleTotalCents int64,
	completeSets int,
) int64 {
	switch model.DiscountType(config.BundleDiscountType) {
	case model.DiscountTypeFixedPrice:
		return bundleTotalCents - (*config.BundlePriceCents * int64(completeSets))
	case model.DiscountTypePercentage:
		return int64(float64(bundleTotalCents) * (*config.BundleDiscountValue) / 100)
	case model.DiscountTypeFixedAmount:
		return int64(*config.BundleDiscountValue) * int64(completeSets)
	default:
		return 0
	}
}
