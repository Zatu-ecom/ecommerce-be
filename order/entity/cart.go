package entity

import "ecommerce-be/common/db"

// ============================================================================
// Cart Entity
// ============================================================================

// Cart represents a user's shopping cart
// Workflow: Cart exists while active → Deleted after successful order creation
type Cart struct {
	db.BaseEntity
	UserID   *uint                  `json:"userId"   gorm:"column:user_id;index"`
	Metadata map[string]interface{} `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`

	// Relationships
	Items []CartItem `json:"items,omitempty" gorm:"foreignKey:CartID"`
}

// ============================================================================
// Cart Item Entity
// ============================================================================

type CartItem struct {
	db.BaseEntity
	CartID    uint `json:"cartId"    gorm:"column:cart_id;not null;index"`
	VariantID uint `json:"variantId" gorm:"column:variant_id;not null;index"`
	Quantity  int  `json:"quantity"  gorm:"column:quantity;not null"`
}
