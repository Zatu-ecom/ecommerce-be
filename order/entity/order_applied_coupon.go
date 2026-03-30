package entity

import "ecommerce-be/common/db"

// OrderAppliedCoupon stores an immutable coupon snapshot for an order.
// Coupon APIs are pending, but this entity is ready for when redemption is implemented.
type OrderAppliedCoupon struct {
	db.BaseEntity
	OrderID               uint       `json:"orderId"              gorm:"column:order_id;not null;index"`
	DiscountCodeID        *uint      `json:"discountCodeId"       gorm:"column:discount_code_id;index"`
	CouponCode            string     `json:"couponCode"           gorm:"column:coupon_code;size:100;not null;index"`
	CouponTitle           *string    `json:"couponTitle"          gorm:"column:coupon_title;size:255"`
	DiscountType          string     `json:"discountType"         gorm:"column:discount_type;size:50"`
	DiscountValue         *int64     `json:"discountValue"        gorm:"column:discount_value"`
	DiscountCents         int64      `json:"discountCents"        gorm:"column:discount_cents;not null;default:0"`
	ShippingDiscountCents int64      `json:"shippingDiscountCents" gorm:"column:shipping_discount_cents;not null;default:0"`
	IsCombinable          *bool      `json:"isCombinable"         gorm:"column:is_combinable"`
	Metadata              db.JSONMap `json:"metadata"             gorm:"column:metadata;type:jsonb;default:'{}'"`
}
