package helpers

import "time"

// DiscountCodePayloadBuilder builds POST /api/promotion/discount-code bodies for tests.
type DiscountCodePayloadBuilder struct {
	payload map[string]any
}

// NewDiscountCodePayload starts a percentage / all_products payload with a wide active window.
func NewDiscountCodePayload(code string) *DiscountCodePayloadBuilder {
	return &DiscountCodePayloadBuilder{
		payload: map[string]any{
			"code":         code,
			"title":        code + " Title",
			"discountType": "percentage",
			"value":        int64(10),
			"appliesTo":    "all_products",
			"startsAt":     "2026-01-01T00:00:00Z",
			"endsAt":       "2026-12-31T23:59:59Z",
		},
	}
}

// PercentageDiscountCode is a one-liner for the common percentage coupon payload.
func PercentageDiscountCode(code string, value int64) map[string]any {
	return NewDiscountCodePayload(code).Percentage(value).Build()
}

func (b *DiscountCodePayloadBuilder) Title(title string) *DiscountCodePayloadBuilder {
	b.payload["title"] = title
	return b
}

func (b *DiscountCodePayloadBuilder) Percentage(value int64) *DiscountCodePayloadBuilder {
	b.payload["discountType"] = "percentage"
	b.payload["value"] = value
	return b
}

func (b *DiscountCodePayloadBuilder) FixedAmount(cents int64) *DiscountCodePayloadBuilder {
	b.payload["discountType"] = "fixed_amount"
	b.payload["value"] = cents
	return b
}

func (b *DiscountCodePayloadBuilder) FreeShipping() *DiscountCodePayloadBuilder {
	b.payload["discountType"] = "free_shipping"
	b.payload["value"] = int64(0)
	return b
}

func (b *DiscountCodePayloadBuilder) BuyXGetY(buyQty, getQty int, getDiscountPercent float64) *DiscountCodePayloadBuilder {
	b.payload["discountType"] = "buy_x_get_y"
	b.payload["value"] = int64(0)
	b.payload["metadata"] = map[string]any{
		"buyQuantity":        buyQty,
		"getQuantity":        getQty,
		"getDiscountPercent": getDiscountPercent,
	}
	return b
}

func (b *DiscountCodePayloadBuilder) Value(value int64) *DiscountCodePayloadBuilder {
	b.payload["value"] = value
	return b
}

func (b *DiscountCodePayloadBuilder) AppliesTo(appliesTo string) *DiscountCodePayloadBuilder {
	b.payload["appliesTo"] = appliesTo
	return b
}

func (b *DiscountCodePayloadBuilder) StartsAt(rfc3339 string) *DiscountCodePayloadBuilder {
	b.payload["startsAt"] = rfc3339
	return b
}

func (b *DiscountCodePayloadBuilder) EndsAt(rfc3339 string) *DiscountCodePayloadBuilder {
	b.payload["endsAt"] = rfc3339
	return b
}

func (b *DiscountCodePayloadBuilder) MinPurchaseCents(cents int64) *DiscountCodePayloadBuilder {
	b.payload["minPurchaseAmountCents"] = cents
	return b
}

func (b *DiscountCodePayloadBuilder) MinQuantity(qty int) *DiscountCodePayloadBuilder {
	b.payload["minQuantity"] = qty
	return b
}

func (b *DiscountCodePayloadBuilder) UsageLimitTotal(limit int) *DiscountCodePayloadBuilder {
	b.payload["usageLimitTotal"] = limit
	return b
}

func (b *DiscountCodePayloadBuilder) UsageLimitPerCustomer(limit int) *DiscountCodePayloadBuilder {
	b.payload["usageLimitPerCustomer"] = limit
	return b
}

func (b *DiscountCodePayloadBuilder) UsageReset(timeType string, amount int) *DiscountCodePayloadBuilder {
	b.payload["usageResetTimeType"] = timeType
	b.payload["usageResetAmount"] = amount
	return b
}

func (b *DiscountCodePayloadBuilder) CanCombineWithOtherDiscounts(can bool) *DiscountCodePayloadBuilder {
	b.payload["canCombineWithOtherDiscounts"] = can
	return b
}

func (b *DiscountCodePayloadBuilder) CustomerEligibility(eligibility string) *DiscountCodePayloadBuilder {
	b.payload["customerEligibility"] = eligibility
	return b
}

func (b *DiscountCodePayloadBuilder) CustomerSegmentID(id uint) *DiscountCodePayloadBuilder {
	b.payload["customerSegmentId"] = id
	return b
}

func (b *DiscountCodePayloadBuilder) Metadata(meta map[string]any) *DiscountCodePayloadBuilder {
	b.payload["metadata"] = meta
	return b
}

func (b *DiscountCodePayloadBuilder) Build() map[string]any {
	out := make(map[string]any, len(b.payload))
	for k, v := range b.payload {
		out[k] = v
	}
	return out
}

// PastRFC3339 returns an RFC3339 timestamp days in the past (UTC).
func PastRFC3339(days int) string {
	return time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
}

// FutureRFC3339 returns an RFC3339 timestamp days in the future (UTC).
func FutureRFC3339(days int) string {
	return time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
}
