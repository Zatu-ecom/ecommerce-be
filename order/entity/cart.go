package entity

import (
	"ecommerce-be/common/db"
)

// ============================================================================
// Cart Entity
// ============================================================================

// Cart represents a user's shopping cart
// Workflow: Cart exists while active → Deleted after successful order creation
// Note: SellerID is derived from User.SellerID (each user belongs to one seller)
// Note: All prices are calculated at runtime - no prices stored in cart tables
type Cart struct {
	db.BaseEntity
	UserID   uint       `json:"userId"   gorm:"column:user_id;uniqueIndex;not null"`
	Metadata db.JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`
}

// ============================================================================
// Cart Item Entity
// ============================================================================

// CartItem represents an item in the cart
// Note: Price is NOT stored - calculated at runtime from variant's current price
type CartItem struct {
	db.BaseEntity
	CartID    uint `json:"cartId"    gorm:"column:cart_id;not null;index"`
	VariantID uint `json:"variantId" gorm:"column:variant_id;not null;index"`
	Quantity  int  `json:"quantity"  gorm:"column:quantity;not null"`
}

// ============================================================================
// Cart Applied Coupon Entity (Many-to-Many: Multiple coupons per cart)
// ============================================================================

// CartAppliedCoupon tracks discount codes (coupons) applied to the cart
// Multiple coupons can be applied based on can_combine_with_other_discounts setting
// Note: Discount amount is NOT stored - calculated at runtime based on current rules
type CartAppliedCoupon struct {
	db.BaseEntity
	CartID         uint `json:"cartId"         gorm:"column:cart_id;not null;index"`
	DiscountCodeID uint `json:"discountCodeId" gorm:"column:discount_code_id;not null;index"`
}

// ============================================================================
// Cart Item Promotion Entity (Many-to-Many: Multiple promotions per cart item)
// ============================================================================

// CartItemPromotion tracks promotions applied to individual cart items
// Multiple promotions can be applied based on can_combine_with_other_discounts setting
// Note: Promotion effects are NOT stored - calculated at runtime based on current rules
type CartItemPromotion struct {
	db.BaseEntity
	CartItemID  uint `json:"cartItemId"  gorm:"column:cart_item_id;not null;index"`
	PromotionID uint `json:"promotionId" gorm:"column:promotion_id;not null;index"`
}
