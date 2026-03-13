package promotionStrategy

import (
	"context"
	"encoding/json"

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
				Name:          "discount_type",
				Type:          "string",
				Required:      true,
				Description:   "Flash sale discount mode.",
				AllowedValues: []string{"percentage", "fixed_amount"},
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

	return nil
}

// CalculateDiscount calculates per-item flash sale discount
func (s *FlashSaleStrategy) CalculateDiscount(
	ctx context.Context,
	promotion *entity.Promotion,
	cart *model.CartValidationRequest,
	effectivePrices map[string]int64,
) (*model.PromotionValidationResult, error) {
	result := &model.PromotionValidationResult{
		IsValid: false,
	}

	configJSON, _ := json.Marshal(promotion.DiscountConfig)
	var config model.FlashSaleConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		result.Reason = "Invalid promotion configuration"
		return result, nil
	}

	if config.StockLimit != nil {
		soldCount := 0
		if config.SoldCount != nil {
			soldCount = *config.SoldCount
		}
		if soldCount >= *config.StockLimit {
			result.Reason = "Flash sale stock limit reached"
			return result, nil
		}
	}

	var totalDiscount int64
	var itemDiscounts []model.ItemDiscount

	for _, item := range cart.Items {
		effectivePrice := effectivePrices[item.ItemID]
		if effectivePrice <= 0 {
			continue
		}

		itemTotal := effectivePrice * int64(item.Quantity)
		var itemDiscount int64

		switch config.DiscountType {
		case "percentage":
			itemDiscount = int64(float64(itemTotal) * config.DiscountValue / 100)
		case "fixed_amount":
			itemDiscount = int64(config.DiscountValue)
			if itemDiscount > itemTotal {
				itemDiscount = itemTotal
			}
		}

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
