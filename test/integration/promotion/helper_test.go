package promotion_test

import (
	"ecommerce-be/promotion/model"
)

// DefaultPromotionPayload returns a valid default payload map for creating a promotion in tests.
func DefaultPromotionPayload() map[string]interface{} {
	return map[string]interface{}{
		"name":          "Default Promotion",
		"promotionType": "percentage",
		"discountConfig": model.PercentageDiscountConfig{
			Percentage: 10,
		},
		"appliesTo":           "specific_products",
		"customerEligibility": "everyone",
		"usageLimitTotal":     100,
		"startsAt":            "2023-01-01T00:00:00Z",
		"endsAt":              "2029-12-31T23:59:59Z",
		"isActive":            true,
	}
}

// BuildPromotionPayload is a helper to construct a promotion payload and overrides
// the most common fields (name, promotionType, discountConfig).
func BuildPromotionPayload(name, promoType string, discountConfig interface{}) map[string]interface{} {
	payload := DefaultPromotionPayload()
	if name != "" {
		payload["name"] = name
	}
	if promoType != "" {
		payload["promotionType"] = promoType
	}
	if discountConfig != nil {
		payload["discountConfig"] = discountConfig
	}
	return payload
}
