package entity

import (
	"ecommerce-be/common/db"
)

// ============================================================================
// Wishlist Entity
// ============================================================================

type Wishlist struct {
	db.BaseEntity
	UserID    uint   `json:"userId"    gorm:"column:user_id;not null;index"`
	Name      string `json:"name"      gorm:"column:name;size:255;default:default"`
	IsDefault bool   `json:"isDefault" gorm:"column:is_default;default:false"`

	// Relationships
	Items []WishlistItem `json:"items,omitempty" gorm:"foreignKey:WishlistID"`
}

// ============================================================================
// Wishlist Item Entity
// ============================================================================

type WishlistItem struct {
	db.BaseEntity
	WishlistID uint `json:"wishlistId" gorm:"column:wishlist_id;not null;index"`
	VariantID  uint `json:"variantId"  gorm:"column:variant_id;not null;index"`
}
