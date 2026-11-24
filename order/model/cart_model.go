package model

import (
	commonModel "ecommerce-be/common/model"
)

// ============================================================================
// Cart Request Models
// ============================================================================

// AddCartItemRequest represents the request to add an item to cart
type AddCartItemRequest struct {
	Items []AddCartItemDetail `json:"items" binding:"required,min=1,dive"`
}

// AddCartItemDetail represents one cart line in batch add-to-cart requests.
type AddCartItemDetail struct {
	VariantID uint `json:"variantId" binding:"required,gt=0"`
	Quantity  *int `json:"quantity"  binding:"required,gte=0,lte=99"`
}

// UpdateCartItemRequest represents the request to update cart item quantity
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0,lte=99"`
}

// MergeCartRequest is the request body for merging a device (guest) cart into a user (authenticated) cart.
// DeviceID is the UUID that identifies the guest device.
type MergeCartRequest struct {
	DeviceID string `json:"deviceId" binding:"required,min=1,max=64"`
}

// ============================================================================
// Shared/Base Components (DRY - Don't Repeat Yourself)
// ============================================================================

// CurrencyInfo contains currency details for display.
// Alias of the shared money contract so cart responses stay on one money language.
type CurrencyInfo = commonModel.CurrencyInfo

// ProductBasicInfo contains minimal product info for cart display
type ProductBasicInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// VariantOptionInfo contains option name-value pair
type VariantOptionInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// VariantInfo contains variant details for cart item display
// Reused by CartItemBase (embedded in multiple response types)
type VariantInfo struct {
	ID            uint                `json:"id"`
	SKU           string              `json:"sku"`
	Images        []string            `json:"images"`
	ImageFileID   *string             `json:"imageFileId,omitempty"`
	AllowPurchase bool                `json:"allowPurchase"`
	Product       ProductBasicInfo    `json:"product"`
	Options       []VariantOptionInfo `json:"options"`
}

// ============================================================================
// Cart Item Models (Composition Pattern)
// ============================================================================

// CartItemBase contains common cart item fields
// Embedded by CartItemResponse and CartItemWithPricingResponse
type CartItemBase struct {
	ID        uint        `json:"id"`
	CartID    uint        `json:"cartId"`
	VariantID uint        `json:"variantId"`
	Quantity  int         `json:"quantity"`
	Variant   VariantInfo `json:"variant"`
}

// CartItemResponse represents a cart item without pricing
// Used in Add/Update/Remove item responses
type CartItemResponse struct {
	CartItemBase // Embed base fields
}

// ItemAppliedPromotionInfo contains promotion details applied to a cart item
type ItemAppliedPromotionInfo struct {
	PromotionID uint              `json:"promotionId"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Discount    commonModel.Money `json:"discount"`
	BadgeText   string            `json:"badgeText,omitempty"`
	BadgeColor  string            `json:"badgeColor,omitempty"`
}

// CartItemWithPricingResponse represents a cart item with full pricing details
// Used in Get Cart response
type CartItemWithPricingResponse struct {
	CartItemBase                                      // Embed base fields
	UnitPrice              commonModel.Money          `json:"unitPrice"`
	LineTotal              commonModel.Money          `json:"lineTotal"`
	AppliedPromotions      []ItemAppliedPromotionInfo `json:"appliedPromotions"`
	TotalPromotionDiscount commonModel.Money          `json:"totalPromotionDiscount"`
	DiscountedLineTotal    commonModel.Money          `json:"discountedLineTotal"`
}

// ============================================================================
// Cart Summary Models
// ============================================================================

// SavingsInfo contains savings summary for display
type SavingsInfo struct {
	Amount     commonModel.Money `json:"amount"`
	Percentage float64           `json:"percentage"`
	Message    string            `json:"message"`
}

// CartSummary contains cart totals for display (used in full cart response)
type CartSummary struct {
	ItemCount   int `json:"itemCount"`
	UniqueItems int `json:"uniqueItems"`

	Subtotal commonModel.Money `json:"subtotal"`

	PromotionCount    int               `json:"promotionCount"`
	PromotionDiscount commonModel.Money `json:"promotionDiscount"`

	CouponCount    int               `json:"couponCount"`
	CouponDiscount commonModel.Money `json:"couponDiscount"`

	TotalDiscount commonModel.Money `json:"totalDiscount"`

	AfterDiscount commonModel.Money `json:"afterDiscount"`

	Tax          commonModel.Money  `json:"tax"`
	Shipping     *commonModel.Money `json:"shipping"`
	FreeShipping bool               `json:"freeShipping"`

	Total commonModel.Money `json:"total"`

	Savings *SavingsInfo `json:"savings,omitempty"`
}

// CartSummaryBrief contains minimal cart summary for header/badge display
type CartSummaryBrief struct {
	ItemCount     int               `json:"itemCount"`
	UniqueItems   int               `json:"uniqueItems"`
	Subtotal      commonModel.Money `json:"subtotal"`
	Total         commonModel.Money `json:"total"`
	TotalDiscount commonModel.Money `json:"totalDiscount"`
	Currency      CurrencyInfo      `json:"currency"`
}

// ============================================================================
// Main Cart Response Models (Composition Pattern)
// ============================================================================

// CartBase contains common cart fields
// Embedded by CartBasicResponse and CartResponse
// UserID is nil for guest/device carts (before merge).
type CartBase struct {
	ID       uint           `json:"id"`
	UserID   *uint          `json:"userId,omitempty"`
	Currency CurrencyInfo   `json:"currency"`
	Metadata map[string]any `json:"metadata"`
}

// CartBasicResponse represents cart response without pricing calculations
// Used in Add Item, Update Item, Remove Item, Clear Cart responses
type CartBasicResponse struct {
	CartBase                    // Embed base fields
	Items    []CartItemResponse `json:"items"`
}

// AppliedCouponInfo contains coupon details for cart response
type AppliedCouponInfo struct {
	ID               uint              `json:"id"`
	DiscountCodeID   uint              `json:"discountCodeId"`
	Code             string            `json:"code"`
	Title            string            `json:"title"`
	DiscountType     string            `json:"discountType"`
	Discount         commonModel.Money `json:"discount"`
	ShippingDiscount commonModel.Money `json:"shippingDiscount"`
}

// AvailablePromotionInfo represents a promotion that can be unlocked
type AvailablePromotionInfo struct {
	ID               uint              `json:"id"`
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Reason           string            `json:"reason"`
	Requirement      string            `json:"requirement,omitempty"`
	PotentialSavings commonModel.Money `json:"potentialSavings,omitempty"`
}

// AppliedPromotionInfo represents a promotion applied at cart level
type AppliedPromotionInfo struct {
	PromotionID      uint              `json:"promotionId"`
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Discount         commonModel.Money `json:"discount"`
	ShippingDiscount commonModel.Money `json:"shippingDiscount"`
}

// CartAvailableCouponsResponse splits applicable vs not-applicable coupons on cart GET
type CartAvailableCouponsResponse struct {
	Applicable    []CartAvailableCouponInfo   `json:"applicable"`
	NotApplicable []CartUnavailableCouponInfo `json:"notApplicable"`
}

// CartAvailableCouponInfo is a coupon that can currently be applied
type CartAvailableCouponInfo struct {
	ID                           uint               `json:"id"`
	Code                         string             `json:"code"`
	Title                        string             `json:"title"`
	DiscountType                 string             `json:"discountType"`
	Value                        commonModel.Money  `json:"value"`
	MaxDiscountAmount            *commonModel.Money `json:"maxDiscountAmount,omitempty"`
	PotentialDiscount            commonModel.Money  `json:"potentialDiscount"`
	MinPurchaseAmount            *commonModel.Money `json:"minPurchaseAmount,omitempty"`
	CanCombineWithOtherDiscounts bool               `json:"canCombineWithOtherDiscounts"`
	StartsAt                     string             `json:"startsAt"`
	EndsAt                       *string            `json:"endsAt,omitempty"`
}

// CartUnavailableCouponInfo is a coupon that exists but is not currently applicable
type CartUnavailableCouponInfo struct {
	ID     uint   `json:"id"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

// CartResponse represents the full cart response with pricing, promotions, and coupons
// Used in Get Cart API
type CartResponse struct {
	CartBase                                          // Embed base fields
	Items               []CartItemWithPricingResponse `json:"items"`
	AppliedPromotions   []AppliedPromotionInfo        `json:"appliedPromotions"`
	AppliedCoupons      []AppliedCouponInfo           `json:"appliedCoupons"`
	Summary             CartSummary                   `json:"summary"`
	AvailablePromotions []AvailablePromotionInfo      `json:"availablePromotions,omitempty"`
	AvailableCoupons    *CartAvailableCouponsResponse `json:"availableCoupons,omitempty"`
}
