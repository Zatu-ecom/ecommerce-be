package entity

import (
	"time"

	"ecommerce-be/common/db"
)

type Order struct {
	db.BaseEntity
	UserID        uint                   `json:"userId"        gorm:"column:user_id;not null;index"`
	SellerID      *uint                  `json:"sellerId"      gorm:"column:seller_id;index"`
	OrderNumber   string                 `json:"orderNumber"   gorm:"column:order_number;uniqueIndex"`
	Status        string                 `json:"status"        gorm:"column:status;size:32;default:pending;index"`
	SubtotalCents int64                  `json:"subtotalCents" gorm:"column:subtotal_cents;default:0"`
	TaxCents      int64                  `json:"taxCents"      gorm:"column:tax_cents;default:0"`
	ShippingCents int64                  `json:"shippingCents" gorm:"column:shipping_cents;default:0"`
	DiscountCents int64                  `json:"discountCents" gorm:"column:discount_cents;default:0"`
	TotalCents    int64                  `json:"totalCents"    gorm:"column:total_cents;default:0"`
	PlacedAt      *time.Time             `json:"placedAt"      gorm:"column:placed_at"`
	PaidAt        *time.Time             `json:"paidAt"        gorm:"column:paid_at"`
	Metadata      map[string]interface{} `json:"metadata"      gorm:"column:metadata;type:jsonb;default:'{}'"`
	TransactionID string                 `json:"transactionId" gorm:"column:transaction_id"`
}

type OrderItem struct {
	db.BaseEntity
	OrderID        uint                   `json:"orderId"        gorm:"column:order_id;not null;index"`
	ProductID      *uint                  `json:"productId"      gorm:"column:product_id;index"`
	SKU            *string                `json:"sku"            gorm:"column:sku;size:255"`
	Quantity       int                    `json:"quantity"       gorm:"column:quantity;not null"`
	UnitPriceCents int64                  `json:"unitPriceCents" gorm:"column:unit_price_cents;not null"`
	LineTotalCents int64                  `json:"lineTotalCents" gorm:"column:line_total_cents;not null"`
	Attributes     map[string]interface{} `json:"attributes"     gorm:"column:attributes;type:jsonb;default:'{}'"`
}

type OrderShipment struct {
	db.BaseEntity
	OrderID     uint                   `json:"orderId"     gorm:"column:order_id;not null;index"`
	Carrier     string                 `json:"carrier"     gorm:"column:carrier;size:50"`
	TrackingNo  string                 `json:"trackingNo"  gorm:"column:tracking_no;size:100"`
	Status      string                 `json:"status"      gorm:"column:status;size:32;default:pending;index"`
	ShippedAt   *time.Time             `json:"shippedAt"   gorm:"column:shipped_at"`
	DeliveredAt *time.Time             `json:"deliveredAt" gorm:"column:delivered_at"`
	Metadata    map[string]interface{} `json:"metadata"    gorm:"column:metadata;type:jsonb;default:'{}'"`
}
