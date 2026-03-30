package model

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

// ============================================================================
// Shared/Base Components (DRY - Don't Repeat Yourself)
// ============================================================================

// CurrencyInfo contains currency details for display
type CurrencyInfo struct {
	Code          string `json:"code"`
	Symbol        string `json:"symbol"`
	DecimalDigits int    `json:"decimalDigits"`
}

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
	PromotionID       uint   `json:"promotionId"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	Discount          int64  `json:"discount"`
	DiscountFormatted string `json:"discountFormatted"`
	BadgeText         string `json:"badgeText,omitempty"`
	BadgeColor        string `json:"badgeColor,omitempty"`
}

// CartItemWithPricingResponse represents a cart item with full pricing details
// Used in Get Cart response
type CartItemWithPricingResponse struct {
	CartItemBase                                      // Embed base fields
	UnitPrice              int64                      `json:"unitPrice"`
	LineTotal              int64                      `json:"lineTotal"`
	AppliedPromotions      []ItemAppliedPromotionInfo `json:"appliedPromotions"`
	TotalPromotionDiscount int64                      `json:"totalPromotionDiscount"`
	DiscountedLineTotal    int64                      `json:"discountedLineTotal"`
}

// ============================================================================
// Cart Summary Models
// ============================================================================

// SavingsInfo contains savings summary for display
type SavingsInfo struct {
	Amount     int64   `json:"amount"`
	Percentage float64 `json:"percentage"`
	Message    string  `json:"message"`
}

// CartSummary contains cart totals for display (used in full cart response)
type CartSummary struct {
	ItemCount   int `json:"itemCount"`
	UniqueItems int `json:"uniqueItems"`

	Subtotal          int64  `json:"subtotal"`
	SubtotalFormatted string `json:"subtotalFormatted"`

	PromotionCount             int    `json:"promotionCount"`
	PromotionDiscount          int64  `json:"promotionDiscount"`
	PromotionDiscountFormatted string `json:"promotionDiscountFormatted"`

	CouponCount             int    `json:"couponCount"`
	CouponDiscount          int64  `json:"couponDiscount"`
	CouponDiscountFormatted string `json:"couponDiscountFormatted"`

	TotalDiscount          int64  `json:"totalDiscount"`
	TotalDiscountFormatted string `json:"totalDiscountFormatted"`

	AfterDiscount          int64  `json:"afterDiscount"`
	AfterDiscountFormatted string `json:"afterDiscountFormatted"`

	Tax          int64  `json:"tax"`
	TaxFormatted string `json:"taxFormatted"`

	Shipping          *int64  `json:"shipping"`
	ShippingFormatted *string `json:"shippingFormatted"`
	FreeShipping      bool    `json:"freeShipping"`

	Total          int64  `json:"total"`
	TotalFormatted string `json:"totalFormatted"`

	Savings *SavingsInfo `json:"savings,omitempty"`
}

// CartSummaryBrief contains minimal cart summary for header/badge display
type CartSummaryBrief struct {
	ItemCount     int          `json:"itemCount"`
	UniqueItems   int          `json:"uniqueItems"`
	Subtotal      int64        `json:"subtotal"`
	Total         int64        `json:"total"`
	TotalDiscount int64        `json:"totalDiscount"`
	Currency      CurrencyInfo `json:"currency"`
}

// ============================================================================
// Main Cart Response Models (Composition Pattern)
// ============================================================================

// CartBase contains common cart fields
// Embedded by CartBasicResponse and CartResponse
type CartBase struct {
	ID       uint           `json:"id"`
	UserID   uint           `json:"userId"`
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
	ID                uint   `json:"id"`
	DiscountCodeID    uint   `json:"discountCodeId"`
	Code              string `json:"code"`
	Title             string `json:"title"`
	DiscountType      string `json:"discountType"`
	Discount          int64  `json:"discount"`
	DiscountFormatted string `json:"discountFormatted"`
}

// AvailablePromotionInfo represents a promotion that can be unlocked
type AvailablePromotionInfo struct {
	ID                        uint   `json:"id"`
	Name                      string `json:"name"`
	Type                      string `json:"type"`
	Reason                    string `json:"reason"`
	Requirement               string `json:"requirement,omitempty"`
	PotentialSavings          int64  `json:"potentialSavings,omitempty"`
	PotentialSavingsFormatted string `json:"potentialSavingsFormatted,omitempty"`
}

// AppliedPromotionInfo represents a promotion applied at cart level
type AppliedPromotionInfo struct {
	PromotionID               uint   `json:"promotionId"`
	Name                      string `json:"name"`
	Type                      string `json:"type"`
	Discount                  int64  `json:"discount"`
	DiscountFormatted         string `json:"discountFormatted"`
	ShippingDiscount          int64  `json:"shippingDiscount"`
	ShippingDiscountFormatted string `json:"shippingDiscountFormatted"`
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
}
