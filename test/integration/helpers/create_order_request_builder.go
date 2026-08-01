package helpers

// CreateOrderRequestBuilder builds POST /api/order bodies for tests.
type CreateOrderRequestBuilder struct {
	payload map[string]any
}

// NewCreateOrderRequest starts a directship order against seeded address IDs.
func NewCreateOrderRequest() *CreateOrderRequestBuilder {
	return &CreateOrderRequestBuilder{
		payload: map[string]any{
			"shippingAddressId": uint(1),
			"billingAddressId":  uint(1),
			"fulfillmentType":   "directship",
			"metadata":          map[string]any{"source": "integration-test"},
		},
	}
}

// DefaultCreateOrderRequest is a one-liner for the common checkout body.
func DefaultCreateOrderRequest() map[string]any {
	return NewCreateOrderRequest().Build()
}

func (b *CreateOrderRequestBuilder) ShippingAddressID(id uint) *CreateOrderRequestBuilder {
	b.payload["shippingAddressId"] = id
	return b
}

func (b *CreateOrderRequestBuilder) BillingAddressID(id uint) *CreateOrderRequestBuilder {
	b.payload["billingAddressId"] = id
	return b
}

func (b *CreateOrderRequestBuilder) FulfillmentType(ft string) *CreateOrderRequestBuilder {
	b.payload["fulfillmentType"] = ft
	return b
}

func (b *CreateOrderRequestBuilder) Metadata(meta map[string]any) *CreateOrderRequestBuilder {
	b.payload["metadata"] = meta
	return b
}

func (b *CreateOrderRequestBuilder) Build() map[string]any {
	out := make(map[string]any, len(b.payload))
	for k, v := range b.payload {
		out[k] = v
	}
	return out
}

// AddCartItemsPayload builds POST /api/order/cart/item bodies.
func AddCartItemsPayload(variantID uint, quantity int) map[string]any {
	return map[string]any{
		"items": []map[string]any{
			{"variantId": variantID, "quantity": quantity},
		},
	}
}

// ApplyCouponPayload builds POST /api/order/cart/coupon bodies.
func ApplyCouponPayload(code string) map[string]any {
	return map[string]any{"code": code}
}
