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

// TieredStrategy implements PromotionStrategy for tiered promotion type.
//
// Business Logic:
//   Applies a discount tier based on the total quantity or spend of eligible items.
//   Only items in the eligibleItems set (determined by scope/appliesTo) contribute to
//   the check value AND receive the resulting discount. Items outside the scope are
//   completely excluded from both threshold evaluation and discount distribution.
//
//   tier_type controls what is measured:
//     - "quantity" : sums the Quantity of all eligible items
//     - "spend"    : sums the FinalPriceCents (line total after earlier promos) of eligible items
//
//   The first tier whose [min, max] range contains the check value is selected.
//   Within the selected tier, discount_type determines the calculation:
//     - "percentage"   : totalDiscount = eligibleSubtotal * discount_value / 100
//     - "fixed_amount" : totalDiscount = discount_value (flat, regardless of item count)
//
//   The total discount is clamped to the eligible subtotal, then distributed
//   proportionally across eligible items (last item gets the rounding remainder).
//
// Config Fields:
//   - tier_type (required) : "quantity" | "spend"
//   - tiers     (required) : ordered array of TierConfig:
//       - min            (required) : inclusive lower bound
//       - max            (optional) : inclusive upper bound (omit for open-ended top tier)
//       - discount_type  (required) : "percentage" | "fixed_amount"
//       - discount_value (required) : amount or percentage for the tier
//
// Example (quantity tiers):
//   Config: { tier_type: "quantity", tiers: [
//     { min: 10, discount_type: "percentage", discount_value: 20 },
//     { min: 5,  max: 9, discount_type: "percentage", discount_value: 10 },
//   ]}
//   Cart (all eligible):
//     Item A  $200 x4  (line total = 80000)
//     Item B  $100 x3  (line total = 30000)
//   Total quantity = 7  =>  matches tier [5,9] => 10% discount
//   Eligible subtotal = 110000
//   Total discount    = 11000
//   Item A discount   = 11000 * 80000 / 110000 = 8000
//   Item B discount   = 11000 - 8000           = 3000
//   Final subtotal    = 110000 - 11000         = 99000
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
				Name:        "tier_type",
				Type:        "string",
				Required:    true,
				Description: "How tiers are evaluated: 'quantity' counts total eligible item units, 'spend' sums eligible item totals in cents.",
				AllowedValues: []string{
					string(model.TierTypeQuantity),
					string(model.TierTypeSpend),
				},
			},
			{
				Name:        "tiers",
				Type:        "[]TierConfig",
				Required:    true,
				Description: "Ordered list of tier definitions. Each entry is a TierConfig object (see sub-fields below).",
			},
			{
				Name:        "tiers[].min",
				Type:        "int",
				Required:    true,
				Description: "Minimum threshold (inclusive) for this tier. Use 0 for the lowest tier.",
			},
			{
				Name:        "tiers[].max",
				Type:        "int",
				Required:    false,
				Description: "Maximum threshold (inclusive) for this tier. Omit to create an open-ended top tier.",
			},
			{
				Name:        "tiers[].discount_type",
				Type:        "string",
				Required:    true,
				Description: "Discount calculation mode for this tier.",
				AllowedValues: []string{
					string(model.DiscountTypePercentage),
					string(model.DiscountTypeFixedAmount),
				},
			},
			{
				Name:        "tiers[].discount_value",
				Type:        "float64",
				Required:    true,
				Description: "Discount amount for this tier. For 'percentage' must be between 0.01 and 100. For 'fixed_amount' must be > 0 (in smallest currency unit).",
			},
		},
		BestPractices: []string{
			"Ensure tiers are ordered from strongest threshold to weakest to avoid ambiguity.",
			"Use spend tiers when cart mix varies heavily and quantity tiers when unit count is the real incentive.",
			"Scope (appliesTo) is honoured: only eligible items contribute to the quantity/spend threshold and receive the discount. Items outside the scope are excluded from both the check value and the distribution.",
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

	if tieredConfig.TierType != model.TierTypeQuantity &&
		tieredConfig.TierType != model.TierTypeSpend {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"tier_type must be 'quantity' or 'spend'",
		)
	}

	for i, tier := range tieredConfig.Tiers {
		label := fmt.Sprintf("tiers[%d]", i)

		if tier.DiscountType != model.DiscountTypePercentage &&
			tier.DiscountType != model.DiscountTypeFixedAmount {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage(
				label + ".discount_type must be 'percentage' or 'fixed_amount'",
			)
		}

		if tier.DiscountValue <= 0 {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage(
				label + ".discount_value must be greater than 0",
			)
		}

		if tier.DiscountType == model.DiscountTypePercentage && tier.DiscountValue > 100 {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage(
				label + ".discount_value must be between 0.01 and 100 for percentage tiers",
			)
		}

		if tier.Min < 0 {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage(
				label + ".min must be greater than or equal to 0",
			)
		}

		if tier.Max != nil && *tier.Max < tier.Min {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage(
				label + ".max must be greater than or equal to min",
			)
		}
	}

	return nil
}

// CalculateDiscount updates the passed AppliedPromotionSummary in-place and
// returns an error if the promotion cannot be applied.
//
// Only eligible items (those in the eligibleItems set) contribute to the
// quantity/spend check value and receive the proportional discount.
func (s *TieredStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) error {
	eligibleItemsSet := helper.ToSet(eligibleItems)

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.TieredConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid tiered promotion configuration",
		)
	}

	// Collect eligible summary items and compute totals using current FinalPriceCents
	// (line total after prior promotions) so stacking is accounted for.
	var eligibleSummaryItems []*model.CartItemSummary
	var totalEffective int64
	var totalQuantity int

	for i := range summary.Items {
		item := &summary.Items[i]
		if !eligibleItemsSet[item.ItemID] {
			continue
		}
		eligibleSummaryItems = append(eligibleSummaryItems, item)
		totalEffective += item.FinalPriceCents

		// Recover quantity from the original cart request since CartItemSummary carries it.
		totalQuantity += item.Quantity
	}

	if totalEffective <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No eligible items for tiered promotion",
		)
	}

	// Determine the check value used for tier matching.
	var checkValue int
	if config.TierType == model.TierTypeQuantity {
		checkValue = totalQuantity
	} else {
		checkValue = int(totalEffective)
	}

	// Find the first tier whose range contains checkValue.
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
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Does not meet minimum tier requirements",
		)
	}

	// Compute total discount from the matched tier.
	var totalDiscount int64
	switch applicableTier.DiscountType {
	case model.DiscountTypePercentage:
		totalDiscount = int64(float64(totalEffective) * applicableTier.DiscountValue / 100)
	case model.DiscountTypeFixedAmount:
		totalDiscount = int64(applicableTier.DiscountValue)
	}

	if totalDiscount > totalEffective {
		totalDiscount = totalEffective
	}

	if totalDiscount <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No discount applicable for tiered promotion",
		)
	}

	// Distribute proportionally across eligible items; last item absorbs rounding remainder.
	itemDiscounts := make([]ItemDiscountDetail, 0, len(eligibleSummaryItems))
	var distributed int64

	for i, item := range eligibleSummaryItems {
		if item.FinalPriceCents <= 0 {
			continue
		}

		var itemDiscount int64
		if i == len(eligibleSummaryItems)-1 {
			itemDiscount = totalDiscount - distributed
		} else {
			itemDiscount = totalDiscount * item.FinalPriceCents / totalEffective
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
			"No discount applicable for tiered promotion",
		)
	}

	ApplyDiscountToSummary(summary, promotion, itemDiscounts, distributed, 0)
	return nil
}
