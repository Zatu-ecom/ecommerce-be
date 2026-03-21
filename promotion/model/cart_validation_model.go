package model

// CartItem represents an item in the cart for promotion validation
type CartItem struct {
	ItemID     string `json:"itemId"              binding:"required"` // Unique ID for tracking (e.g. "cart-item-1" or UUID)
	ProductID  uint   `json:"productId"`
	VariantID  *uint  `json:"variantId,omitempty"`
	CategoryID uint   `json:"categoryId"`
	Quantity   int    `json:"quantity"`
	PriceCents int64  `json:"priceCents"` // Price per unit in cents
	TotalCents int64  `json:"totalCents"` // PriceCents * Quantity
}

// CartValidationRequest represents the cart data for promotion validation
type CartValidationRequest struct {
	SellerID      uint       `json:"sellerId"             binding:"required"`
	Items         []CartItem `json:"items"                binding:"required,min=1,dive"`
	SubtotalCents int64      `json:"subtotalCents"        binding:"required,min=0"`
	ShippingCents int64      `json:"shippingCents"        binding:"min=0"`
	CustomerID    *uint      `json:"customerId,omitempty"`
	IsFirstOrder  bool       `json:"isFirstOrder"` // True if this is customer's first order (set by order service)
}

// ItemPromotionDetail is the response struct nested inside CartItemSummary.
// It omits ItemID/ProductID since the parent CartItemSummary already has them.
type ItemPromotionDetail struct {
	PromotionID   uint   `json:"promotionId"`
	PromotionName string `json:"promotionName"`
	DiscountCents int64  `json:"discountCents"`          // Discount amount for this item from this promotion
	OriginalCents int64  `json:"originalCents"`          // Item price before this promotion
	FinalCents    int64  `json:"finalCents"`             // Item price after this promotion
	FreeQuantity  int    `json:"freeQuantity,omitempty"` // For buy X get Y: number of free items
}

// PromotionValidationResult embeds the standard PromotionResponse (same as get-promotion API)
// and adds cart-specific validation/discount fields on top.
type PromotionValidationResult struct {
	// Full promotion details — reuses the same response model as get-promotion API
	Promotion *PromotionResponse `json:"promotion,omitempty"`

	// Cart-specific fields
	IsValid          bool   `json:"isValid"`
	DiscountCents    int64  `json:"discountCents"`    // Total discount from this promotion
	ShippingDiscount int64  `json:"shippingDiscount"` // Shipping discount from this promotion
	Reason           string `json:"reason,omitempty"`
}

// SkippedPromotionReason stores why a promotion wasn't applied along with user-actionable requirements
type SkippedPromotionReason struct {
	Reason           string `json:"reason"`
	Requirement      string `json:"requirement,omitempty"`
	PotentialSavings int64  `json:"potentialSavings,omitempty"`
}

// SkippedPromotionResult represents a skipped promotion sent in the API response
type SkippedPromotionResult struct {
	Promotion *PromotionResponse `json:"promotion,omitempty"`
	SkippedPromotionReason
}

// AppliedPromotionSummary represents the final summary after applying all promotions
type AppliedPromotionSummary struct {
	Items              []CartItemSummary           `json:"items"`
	AppliedPromotions  []PromotionValidationResult `json:"appliedPromotions"`
	SkippedPromotions  []SkippedPromotionResult    `json:"skippedPromotions,omitempty"`
	TotalDiscountCents int64                       `json:"totalDiscountCents"`
	ShippingDiscount   int64                       `json:"shippingDiscount"`
	OriginalSubtotal   int64                       `json:"originalSubtotal"`
	FinalSubtotal      int64                       `json:"finalSubtotal"`
}

// CartItemSummary represents the final state of a cart item after all promotions applied
type CartItemSummary struct {
	ItemID                 string                `json:"itemId"`
	ProductID              uint                  `json:"productId"`
	VariantID              *uint                 `json:"variantId,omitempty"`
	Quantity               int                   `json:"quantity"`
	OriginalUnitPriceCents int64                 `json:"originalUnitPriceCents"` // Original unit price
	FinalPriceCents        int64                 `json:"finalPriceCents"`        // Final unit price after all discounts
	TotalDiscountCents     int64                 `json:"totalDiscountCents"`     // Sum of all discounts on this item
	AppliedPromotions      []ItemPromotionDetail `json:"appliedPromotions"`      // All promotions applied to this item
}
