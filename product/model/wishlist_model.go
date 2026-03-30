package model

import "time"

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

// WishlistDetailResponse represents the response for getting a wishlist with products
// Uses ProductsResponse for full product details with pagination support
type WishlistDetailResponse struct {
	ID        uint             `json:"id"`
	Name      string           `json:"name"`
	IsDefault bool             `json:"isDefault"`
	ItemCount int              `json:"itemCount"` // Total items in wishlist (not paginated count)
	Products  ProductsResponse `json:"products"`  // Paginated products with full details
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}
