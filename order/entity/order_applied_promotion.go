package entity

import "ecommerce-be/common/db"

// OrderAppliedPromotion stores an immutable snapshot of promotion impact for an order.
// This supports conversion reporting even if promotion configuration changes later.
type OrderAppliedPromotion struct {
	db.BaseEntity
	OrderID               uint       `json:"orderId"               gorm:"column:order_id;not null;index"`
	PromotionID           *uint      `json:"promotionId"           gorm:"column:promotion_id;index"`
	PromotionName         string     `json:"promotionName"         gorm:"column:promotion_name;size:255;not null"`
	PromotionType         string     `json:"promotionType"         gorm:"column:promotion_type;size:50;not null;index"`
	DiscountCents         int64      `json:"discountCents"         gorm:"column:discount_cents;not null;default:0"`
	ShippingDiscountCents int64      `json:"shippingDiscountCents" gorm:"column:shipping_discount_cents;not null;default:0"`
	IsStackable           *bool      `json:"isStackable"           gorm:"column:is_stackable"`
	Priority              int        `json:"priority"              gorm:"column:priority;not null;default:0"`
	Metadata              db.JSONMap `json:"metadata"              gorm:"column:metadata;type:jsonb;default:'{}'"`
}
