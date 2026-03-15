package promotionStrategy

import (
	"context"
	"encoding/json"

	"ecommerce-be/common/helper"
	"ecommerce-be/promotion/entity"
	promoErrors "ecommerce-be/promotion/error"
	"ecommerce-be/promotion/model"
)

// FlashSaleStrategy implements PromotionStrategy for flash_sale promotion type
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
) error {
	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FlashSaleConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"Invalid flash_sale promotion configuration",
		)
	}

	if config.StockLimit != nil {
		soldCount := 0
		if config.SoldCount != nil {
			soldCount = *config.SoldCount
		}
		if soldCount >= *config.StockLimit {
			return promoErrors.ErrInvalidDiscountConfig.WithMessage(
				"Flash sale stock limit reached",
			)
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
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No eligible items for flash sale promotion",
		)
	}

	if config.MinOrderCents != nil && totalEffective < *config.MinOrderCents {
		return promoErrors.ErrInvalidDiscountConfig.WithMessage("Minimum order amount not met")
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
		return promoErrors.ErrInvalidDiscountConfig.WithMessage(
			"No discount applicable for flash sale promotion",
		)
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
	return nil
}
