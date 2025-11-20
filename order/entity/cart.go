package entity

import "ecommerce-be/common/db"

type Cart struct {
	db.BaseEntity
	UserID   *uint                  `json:"userId"   gorm:"column:user_id;index"`
	Status   string                 `json:"status"   gorm:"column:status;size:32;default:active;index"`
	Metadata map[string]interface{} `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`
}

type CartItem struct {
	db.BaseEntity
	CartID         uint                   `json:"cartId"         gorm:"column:cart_id;not null;index"`
	ProductID      *uint                  `json:"productId"      gorm:"column:product_id;index"`
	SKU            *string                `json:"sku"            gorm:"column:sku;size:255"`
	Quantity       int                    `json:"quantity"       gorm:"column:quantity;not null"`
	UnitPriceCents int64                  `json:"unitPriceCents" gorm:"column:unit_price_cents;not null"`
	LineTotalCents int64                  `json:"lineTotalCents" gorm:"column:line_total_cents;not null"`
	Attributes     map[string]interface{} `json:"attributes"     gorm:"column:attributes;type:jsonb;default:'{}'"`
}
