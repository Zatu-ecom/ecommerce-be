package helpers

// PromotionPayloadBuilder builds POST /api/promotion bodies for tests.
type PromotionPayloadBuilder struct {
	payload map[string]any
}

// NewPercentagePromotion starts an active percentage promotion on all products.
func NewPercentagePromotion(name string, percentage float64) *PromotionPayloadBuilder {
	return &PromotionPayloadBuilder{
		payload: map[string]any{
			"name":          name,
			"promotionType": "percentage_discount",
			"discountConfig": map[string]any{
				"percentage": percentage,
			},
			"appliesTo":   "all_products",
			"eligibleFor": "everyone",
			"startsAt":    "2023-01-01T00:00:00Z",
			"endsAt":      "2029-12-31T23:59:59Z",
			"status":      "active",
		},
	}
}

// PercentagePromotionPayload is a one-liner for the common promotion create body.
func PercentagePromotionPayload(name string, percentage float64) map[string]any {
	return NewPercentagePromotion(name, percentage).Build()
}

func (b *PromotionPayloadBuilder) CanStackWithCoupons(can bool) *PromotionPayloadBuilder {
	b.payload["canStackWithCoupons"] = can
	return b
}

func (b *PromotionPayloadBuilder) CanStackWithOtherPromotions(can bool) *PromotionPayloadBuilder {
	b.payload["canStackWithOtherPromotions"] = can
	return b
}

func (b *PromotionPayloadBuilder) Status(status string) *PromotionPayloadBuilder {
	b.payload["status"] = status
	return b
}

func (b *PromotionPayloadBuilder) Build() map[string]any {
	out := make(map[string]any, len(b.payload))
	for k, v := range b.payload {
		out[k] = v
	}
	return out
}
