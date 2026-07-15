package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// ============================================================================
// RecentlyViewed Entity
// Represents a single product view event by a customer.
// Each (user_id, product_id) pair is unique — re-viewing the same product
// updates the viewed_at timestamp via ON CONFLICT upsert rather than creating
// a new row.
// ============================================================================

// RecentlyViewed represents a record of a customer viewing a product.
type RecentlyViewed struct {
	db.BaseEntity
	UserID    uint      `json:"userId"    gorm:"column:user_id;not null;index"`
	SellerID  uint      `json:"sellerId"  gorm:"column:seller_id;not null"`
	ProductID uint      `json:"productId" gorm:"column:product_id;not null;index"`
	ViewedAt  time.Time `json:"viewedAt"  gorm:"column:viewed_at;not null;default:CURRENT_TIMESTAMP"`
}

// TableName overrides the default GORM table name.
// Returns the singular table name as per project convention (SingularTable: true).
func (RecentlyViewed) TableName() string {
	return "user_recently_viewed"
}
