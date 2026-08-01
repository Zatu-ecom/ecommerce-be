package model

import (
	"time"

	"ecommerce-be/common"
)

// ============================================================================
// Wishlist Management - Request Models
// ============================================================================

// WishlistCreateRequest represents the request body for creating a wishlist
type WishlistCreateRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// WishlistUpdateRequest represents the request body for updating a wishlist
type WishlistUpdateRequest struct {
	Name      *string `json:"name"      binding:"omitempty,min=1,max=100"`
	IsDefault *bool   `json:"isDefault" binding:"omitempty"`
}

// ============================================================================
// Wishlist Management - Response Models
// ============================================================================

// WishlistResponse represents a single wishlist in API responses
type WishlistResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"isDefault"`
	ItemCount int       `json:"itemCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WishlistsResponse represents the response for getting all wishlists
type WishlistsResponse struct {
	Wishlists []WishlistResponse `json:"wishlists"`
}

// WishlistProductItem represents a single wishlist item with the resolved product
type WishlistProductItem struct {
	WishlistItemID uint            `json:"wishlistItemId"` // ID of the wishlist item (for delete/move)
	VariantID      uint            `json:"variantId"`      // The specific variant that was wishlisted
	AddedAt        time.Time       `json:"addedAt"`        // When the item was added to the wishlist
	Product        ProductResponse `json:"product"`        // Full resolved product with variants
}

// WishlistDetailResponse represents the response for getting a wishlist with products
// Uses WishlistProductItem for per-item metadata with full product details
// Pagination is top-level (not nested inside a sub-object)
type WishlistDetailResponse struct {
	ID         uint                      `json:"id"`
	Name       string                    `json:"name"`
	IsDefault  bool                      `json:"isDefault"`
	Items      []WishlistProductItem     `json:"items"`      // Paginated items with product details
	Pagination common.PaginationResponse `json:"pagination"` // Pagination metadata
	CreatedAt  time.Time                 `json:"createdAt"`
	UpdatedAt  time.Time                 `json:"updatedAt"`
}
