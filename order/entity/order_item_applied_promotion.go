package entity

import "ecommerce-be/common/db"

// OrderItemAppliedPromotion captures item-level promotion allocation on an order item.
// This enables exact reporting for scoped promotions (e.g., electronics-only offers).
type OrderItemAppliedPromotion struct {
	db.BaseEntity
	OrderID       uint       `json:"orderId"         gorm:"column:order_id;not null;index"`
	OrderItemID   uint       `json:"orderItemId"     gorm:"column:order_item_id;not null;index"`
	PromotionID   *uint      `json:"promotionId"     gorm:"column:promotion_id;index"`
	PromotionName string     `json:"promotionName"   gorm:"column:promotion_name;size:255;not null"`
	PromotionType string     `json:"promotionType"   gorm:"column:promotion_type;size:50;not null;index"`
	DiscountCents int64      `json:"discountCents"   gorm:"column:discount_cents;not null;default:0"`
	OriginalCents int64      `json:"originalCents"   gorm:"column:original_cents;not null;default:0"`
	FinalCents    int64      `json:"finalCents"      gorm:"column:final_cents;not null;default:0"`
	FreeQuantity  int        `json:"freeQuantity"    gorm:"column:free_quantity;not null;default:0"`
	Metadata      db.JSONMap `json:"metadata"        gorm:"column:metadata;type:jsonb;default:'{}'"`
}
