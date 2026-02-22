package promotionStrategy

import (
	"ecommerce-be/promotion/entity"
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
