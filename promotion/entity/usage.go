package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// PromotionUsage tracks when and how promotions are used in orders
type PromotionUsage struct {
	db.BaseEntity

	// References
	PromotionID uint `json:"promotionId" gorm:"column:promotion_id;not null;index"`
	UserID      uint `json:"userId"      gorm:"column:user_id;not null;index"`
	OrderID     uint `json:"orderId"     gorm:"column:order_id;not null;index"`

	// Discount Applied
	DiscountAmountCents int64 `json:"discountAmountCents" gorm:"column:discount_amount_cents;not null"`
	OriginalAmountCents int64 `json:"originalAmountCents" gorm:"column:original_amount_cents;not null"`

	// Timestamp
	UsedAt time.Time `json:"usedAt" gorm:"column:used_at;not null;index"`

	// Metadata
	Metadata JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`

	// Relationships
	Promotion *Promotion `json:"promotion,omitempty" gorm:"foreignKey:PromotionID"`
}

// TableName specifies the table name
func (PromotionUsage) TableName() string {
	return "promotion_usage"
}

// DiscountCodeUsage tracks when and how discount codes are used in orders
type DiscountCodeUsage struct {
	db.BaseEntity

	// References
	DiscountCodeID uint `json:"discountCodeId" gorm:"column:discount_code_id;not null;index"`
	UserID         uint `json:"userId"         gorm:"column:user_id;not null;index"`
	OrderID        uint `json:"orderId"        gorm:"column:order_id;not null;index"`

	// Discount Applied
	DiscountAmountCents int64 `json:"discountAmountCents" gorm:"column:discount_amount_cents;not null"`
	OriginalAmountCents int64 `json:"originalAmountCents" gorm:"column:original_amount_cents;not null"`

	// Timestamp
	UsedAt time.Time `json:"usedAt" gorm:"column:used_at;not null;index"`

	// Metadata
	Metadata JSONMap `json:"metadata" gorm:"column:metadata;type:jsonb;default:'{}'"`

	// Relationships
	DiscountCode *DiscountCode `json:"discountCode,omitempty" gorm:"foreignKey:DiscountCodeID"`
}

// TableName specifies the table name
func (DiscountCodeUsage) TableName() string {
	return "discount_code_usage"
}
