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

// FlashSaleStrategy implements PromotionStrategy for flash_sale promotion type.
//
// Business Logic:
//
//	Applies a temporary high-urgency discount with optional stock limits. Unlike
//	FixedAmountStrategy (which distributes one flat amount across the cart), the flash
//	sale's fixed_amount mode applies discount_value to EACH eligible item independently,
//	clamped to that item's FinalPriceCents. The percentage mode works per-item as well,
//	computing discount_value% of each item's current line total.
//
//	Before any discount calculation:
//	  1. Stock check: if stock_limit is set and sold_count >= stock_limit, the promotion
//	     is rejected immediately (the actual atomic decrement happens at order time).
//	  2. Min-order check: if min_order_cents is set, the sum of FinalPriceCents of all
//	     eligible items must meet or exceed it.
//
//	After per-item discounts are computed, if max_discount_cents is set and the raw total
//	exceeds it, the capped amount is redistributed proportionally across items.
//
// Config Fields:
//   - discount_type      (required) : "percentage" | "fixed_amount"
//   - discount_value     (required) : percentage (0–100) or flat amount per item (cents)
//   - max_discount_cents (optional) : cap on total discount across all items
//   - min_order_cents    (optional) : minimum eligible subtotal to qualify
//   - stock_limit        (optional) : total allotted stock for the flash sale
//   - sold_count         (optional) : current sold count for stock-limit validation
//
// Example (percentage, 30% off, max cap 20000):
//
//	Config: { discount_type: "percentage", discount_value: 30, max_discount_cents: 20000 }
//	Cart:
//	  Item A  $500 x1  (line total = 50000)
//	  Item B  $300 x1  (line total = 30000)
//	Per-item: Item A = 15000, Item B = 9000  => raw total = 24000
//	Capped to 20000 => redistribute:
//	  Item A capped = 20000 * 15000 / 24000 = 12500
//	  Item B capped = 20000 - 12500         =  7500
//	Final subtotal = 80000 - 20000 = 60000
//
// Example (fixed_amount, $200 off per item):
//
//	Config: { discount_type: "fixed_amount", discount_value: 20000 }
//	Cart:
//	  Item A  $500 x1  (line total = 50000)  => discount = 20000
//	  Item B  $150 x1  (line total = 15000)  => discount = 15000 (clamped to item price)
//	Total discount = 35000,  Final subtotal = 65000 - 35000 = 30000
type FlashSaleStrategy struct{}

// NewFlashSaleStrategy creates a new FlashSaleStrategy
func NewFlashSaleStrategy() PromotionStrategy {
	return &FlashSaleStrategy{}
}

// DescribeConfig returns the supported flash-sale fields and setup guidance.
func (s *FlashSaleStrategy) DescribeConfig() model.PromotionStrategyDescriptor {
	return model.PromotionStrategyDescriptor{
		PromotionType: entity.PromoTypeFlashSale,
		Name:          "Flash Sale",
		Description:   "Applies a temporary high-urgency discount, optionally with stock limits.",
		Fields: []model.PromotionConfigFieldDescriptor{
			{
				Name:        "discount_type",
				Type:        "string",
				Required:    true,
				Description: "Flash sale discount mode.",
				AllowedValues: []string{
					string(model.DiscountTypePercentage),
					string(model.DiscountTypeFixedAmount),
				},
			},
			{
				Name:        "discount_value",
				Type:        "float64",
				Required:    true,
				Description: "Discount amount or percentage depending on discount_type.",
			},
			{
				Name:        "max_discount_cents",
				Type:        "int64",
				Required:    false,
				Description: "Optional discount cap.",
			},
			{
				Name:        "min_order_cents",
				Type:        "int64",
				Required:    false,
				Description: "Optional minimum eligible-item subtotal (in cents) required before the flash sale discount applies.",
			},
			{
				Name:        "stock_limit",
				Type:        "int",
				Required:    false,
				Description: "Optional total stock allotment for the flash sale.",
			},
			{
				Name:        "sold_count",
				Type:        "int",
				Required:    false,
				Description: "Current sold count used when validating stock limits.",
			},
		},
		BestPractices: []string{
			"Use a stock limit for scarce inventory or time-boxed campaigns.",
			"Keep flash sales non-stackable unless the combined discount is explicitly planned.",
		},
	}
}

// ValidateConfig validates the flash sale configuration
func (s *FlashSaleStrategy) ValidateConfig(config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Invalid config format")
	}

	var flashConfig model.FlashSaleConfig
	if err := json.Unmarshal(configJSON, &flashConfig); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid flash_sale config structure",
		)
	}

	if flashConfig.DiscountValue <= 0 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"discount_value must be greater than 0",
		)
	}

	if model.DiscountType(flashConfig.DiscountType) == model.DiscountTypePercentage &&
		flashConfig.DiscountValue > 100 {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"discount_value for percentage type must be <= 100",
		)
	}

	return nil
}

// CalculateDiscount updates the passed AppliedPromotionSummary in-place and
// returns an error if the promotion cannot be applied.
func (s *FlashSaleStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	summary *model.AppliedPromotionSummary,
	eligibleItems []string,
) (*model.SkippedPromotionReason, error) {
	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FlashSaleConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid flash_sale promotion configuration",
		)
	}

	if config.StockLimit != nil {
		soldCount := 0
		if config.SoldCount != nil {
			soldCount = *config.SoldCount
		}
		if soldCount >= *config.StockLimit {
			return &model.SkippedPromotionReason{
				Reason: "Flash sale stock limit reached",
			}, nil
		}
	}

	eligibleItemsSet := helper.ToSet(eligibleItems)

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
			Reason: "No eligible items for flash sale promotion",
		}, nil
	}

	if config.MinOrderCents != nil && totalEffective < *config.MinOrderCents {
		shortfall := *config.MinOrderCents - totalEffective
		requirementFormat := "Add $%.2f more to qualify"
		var potentialSavings int64
		if config.DiscountType == "percentage" {
			potentialSavings = int64(float64(*config.MinOrderCents) * (config.DiscountValue / 100))
		} else {
			potentialSavings = int64(config.DiscountValue)
		}
		return &model.SkippedPromotionReason{
			Reason:           "NOT_MET",
			Requirement:      fmt.Sprintf(requirementFormat, float64(shortfall)/100.0),
			PotentialSavings: potentialSavings,
		}, nil
	}

	itemDiscounts := make([]ItemDiscountDetail, 0, len(eligibleSummaryItems))
	var totalDiscount int64

	for _, item := range eligibleSummaryItems {
		if item.FinalPriceCents <= 0 {
			continue
		}

		var itemDiscount int64
		switch model.DiscountType(config.DiscountType) {
		case model.DiscountTypePercentage:
			itemDiscount = int64(float64(item.FinalPriceCents) * config.DiscountValue / 100)
		case model.DiscountTypeFixedAmount:
			itemDiscount = int64(config.DiscountValue)
			if itemDiscount > item.FinalPriceCents {
				itemDiscount = item.FinalPriceCents
			}
		}

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
			Reason: "No discount applicable for flash sale promotion",
		}, nil
	}

	effectiveCap := totalDiscount
	if config.MaxDiscountCents != nil && effectiveCap > *config.MaxDiscountCents {
		effectiveCap = *config.MaxDiscountCents
	}

	if effectiveCap < totalDiscount {
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
