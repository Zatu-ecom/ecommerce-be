package promotionStrategy

import (
	"ecommerce-be/promotion/entity"
	"ecommerce-be/promotion/model"
)

// GetPromotionStrategy returns the appropriate strategy for the given promotion type
func GetPromotionStrategy(promotionType entity.PromotionType) PromotionStrategy {
	switch promotionType {
	case entity.PromoTypePercentage:
		return NewPercentageStrategy()
	case entity.PromoTypeFixedAmount:
		return NewFixedAmountStrategy()
	case entity.PromoTypeFreeShipping:
		return NewFreeShippingStrategy()
	case entity.PromoTypeBuyXGetY:
		return NewBuyXGetYStrategy()
	case entity.PromoTypeBundle:
		return NewBundleStrategy()
	case entity.PromoTypeTiered:
		return NewTieredStrategy()
	case entity.PromoTypeFlashSale:
		return NewFlashSaleStrategy()
	default:
		return nil
	}
}

// GetPromotionStrategyDescriptor returns the field schema and setup guidance for a promotion type.
func GetPromotionStrategyDescriptor(
	promotionType entity.PromotionType,
) *model.PromotionStrategyDescriptor {
	strategy := GetPromotionStrategy(promotionType)
	if strategy == nil {
		return nil
	}

	descriptor := strategy.DescribeConfig()
	return &descriptor
}
