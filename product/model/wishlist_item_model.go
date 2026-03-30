package model

import "time"

// ============================================================================
// Wishlist Item Request Models
// ============================================================================

// WishlistItemCreateRequest represents the request to add an item to wishlist
type WishlistItemCreateRequest struct {
	VariantID uint `json:"variantId" binding:"required"`
}

// WishlistItemMoveRequest represents the request to move an item to another wishlist
type WishlistItemMoveRequest struct {
	TargetWishlistID uint `json:"targetWishlistId" binding:"required"`
}

// WishlistItemAddToCartRequest represents the request to add wishlist item to cart
type WishlistItemAddToCartRequest struct {
	Quantity int `json:"quantity" binding:"omitempty,min=1"`
}

// WishlistItemResponse represents a wishlist item in API responses
type WishlistItemResponse struct {
	ID        uint      `json:"id"`
	VariantID uint      `json:"variantId"`
	CreatedAt time.Time `json:"createdAt"`
}

// CartItemResponse represents a cart item in API responses
type CartItemResponse struct {
	ID        uint `json:"id"`
	VariantID uint `json:"variantId"`
	Quantity  int  `json:"quantity"`
}
