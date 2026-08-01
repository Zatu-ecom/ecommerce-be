package model

// CouponCartRequest is built by the order module and passed to CouponApplyService
type CouponCartRequest struct {
	SellerID               uint
	CustomerID             *uint
	IsFirstOrder           bool
	Items                  []CartItem // after-promo adjusted line totals preferred
	SubtotalCents          int64      // after promotions
	ShippingCents          int64
	AppliedDiscountCodeIDs []uint
	PromotionsAllowCoupons bool // false if any applied promo CanStackWithCoupons=false
}

// CouponValidationResult embeds discount-code details plus cart-specific discount fields
type CouponValidationResult struct {
	DiscountCode     *DiscountCodeResponse `json:"discountCode,omitempty"`
	IsValid          bool                  `json:"isValid"`
	DiscountCents    int64                 `json:"discountCents"`
	ShippingDiscount int64                 `json:"shippingDiscount"`
	Reason           string                `json:"reason,omitempty"`
}

// SkippedCouponResult represents a coupon that was not applied
type SkippedCouponResult struct {
	DiscountCode     *DiscountCodeResponse `json:"discountCode,omitempty"`
	Reason           string                `json:"reason"`
	Requirement      string                `json:"requirement,omitempty"`
	PotentialSavings int64                 `json:"potentialSavings,omitempty"`
}

// AppliedCouponSummary is the result of applying coupons to a cart
type AppliedCouponSummary struct {
	AppliedCoupons     []CouponValidationResult `json:"appliedCoupons"`
	SkippedCoupons     []SkippedCouponResult    `json:"skippedCoupons,omitempty"`
	TotalDiscountCents int64                    `json:"totalDiscountCents"`
	ShippingDiscount   int64                    `json:"shippingDiscount"`
}

// CouponUsageRecord is used when recording usage at checkout
type CouponUsageRecord struct {
	DiscountCodeID      uint
	DiscountAmountCents int64
	OriginalAmountCents int64
}

// AvailableCouponInfo is a coupon that can currently be applied to the cart
type AvailableCouponInfo struct {
	ID                           uint    `json:"id"`
	Code                         string  `json:"code"`
	Title                        string  `json:"title"`
	DiscountType                 string  `json:"discountType"`
	Value                        int64   `json:"value"`
	MaxDiscountAmountCents       *int64  `json:"maxDiscountAmountCents,omitempty"`
	PotentialDiscount            int64   `json:"potentialDiscount"`
	PotentialDiscountFormatted   string  `json:"potentialDiscountFormatted"`
	MinPurchaseAmountCents       *int64  `json:"minPurchaseAmountCents,omitempty"`
	CanCombineWithOtherDiscounts bool    `json:"canCombineWithOtherDiscounts"`
	StartsAt                     string  `json:"startsAt"`
	EndsAt                       *string `json:"endsAt,omitempty"`
}

// UnavailableCouponInfo is a coupon that exists but is not currently applicable
type UnavailableCouponInfo struct {
	ID     uint   `json:"id"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

// AvailableCouponsResponse splits applicable vs not-applicable coupons for a cart
type AvailableCouponsResponse struct {
	Applicable    []AvailableCouponInfo   `json:"applicable"`
	NotApplicable []UnavailableCouponInfo `json:"notApplicable"`
}

// ApplyCouponRequest is the HTTP body for POST /api/order/cart/coupon
type ApplyCouponRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50"`
}
