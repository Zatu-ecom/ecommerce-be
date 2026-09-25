package helpers

import promotionModel "ecommerce-be/promotion/model"

// CouponCartRequestBuilder builds promotion CouponCartRequest values for service-level tests.
type CouponCartRequestBuilder struct {
	req promotionModel.CouponCartRequest
}

// NewCouponCartRequest starts a request for the given seller/customer with coupons allowed.
func NewCouponCartRequest(sellerID, customerID uint) *CouponCartRequestBuilder {
	uid := customerID
	return &CouponCartRequestBuilder{
		req: promotionModel.CouponCartRequest{
			SellerID:               sellerID,
			CustomerID:             &uid,
			ShippingCents:          5000,
			PromotionsAllowCoupons: true,
			Items:                  make([]promotionModel.CartItem, 0),
			AppliedDiscountCodeIDs: make([]uint, 0),
		},
	}
}

func (b *CouponCartRequestBuilder) WithItem(
	itemID string,
	productID uint,
	quantity int,
	priceCents, totalCents int64,
) *CouponCartRequestBuilder {
	b.req.Items = append(b.req.Items, promotionModel.CartItem{
		ItemID:     itemID,
		ProductID:  productID,
		Quantity:   quantity,
		PriceCents: priceCents,
		TotalCents: totalCents,
	})
	b.req.SubtotalCents += totalCents
	return b
}

func (b *CouponCartRequestBuilder) WithApplied(discountCodeIDs ...uint) *CouponCartRequestBuilder {
	b.req.AppliedDiscountCodeIDs = append(b.req.AppliedDiscountCodeIDs, discountCodeIDs...)
	return b
}

func (b *CouponCartRequestBuilder) ShippingCents(cents int64) *CouponCartRequestBuilder {
	b.req.ShippingCents = cents
	return b
}

func (b *CouponCartRequestBuilder) PromotionsAllowCoupons(allow bool) *CouponCartRequestBuilder {
	b.req.PromotionsAllowCoupons = allow
	return b
}

func (b *CouponCartRequestBuilder) IsFirstOrder(first bool) *CouponCartRequestBuilder {
	b.req.IsFirstOrder = first
	return b
}

func (b *CouponCartRequestBuilder) Build() *promotionModel.CouponCartRequest {
	out := b.req
	return &out
}

// DefaultCheckoutCouponCartRequest is the common single-item cart used in checkout revalidate tests.
func DefaultCheckoutCouponCartRequest(sellerID, customerID, discountCodeID uint) *promotionModel.CouponCartRequest {
	return NewCouponCartRequest(sellerID, customerID).
		WithItem("1", 1, 1, 99900, 99900).
		WithApplied(discountCodeID).
		Build()
}
