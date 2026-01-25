package entity

import "ecommerce-be/common/db"

// ============================================================================
// Order Item Entity
// ============================================================================

type OrderItem struct {
	db.BaseEntity
	OrderID        uint                   `json:"orderId"        gorm:"column:order_id;not null;index"`
	ProductID      *uint                  `json:"productId"      gorm:"column:product_id;index"`
	VariantID      *uint                  `json:"variantId"      gorm:"column:variant_id;index"`
	SKU            *string                `json:"sku"            gorm:"column:sku;size:255"`
	ProductName    string                 `json:"productName"    gorm:"column:product_name;size:255;not null"`
	VariantName    *string                `json:"variantName"    gorm:"column:variant_name;size:255"`
	ImageURL       *string                `json:"imageUrl"       gorm:"column:image_url;size:500"`
	Quantity       int                    `json:"quantity"       gorm:"column:quantity;not null"`
	UnitPriceCents int64                  `json:"unitPriceCents" gorm:"column:unit_price_cents;not null"`
	LineTotalCents int64                  `json:"lineTotalCents" gorm:"column:line_total_cents;not null"`
	Attributes     map[string]interface{} `json:"attributes"     gorm:"column:attributes;type:jsonb;default:'{}'"`
}
