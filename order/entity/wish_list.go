package entity

import (
	"time"

	"ecommerce-be/common/db"
)

type Wishlist struct {
	db.BaseEntity
	UserID    uint   `json:"userId"    gorm:"column:user_id;not null;index"`
	Name      string `json:"name"      gorm:"column:name;size:255;default:default"`
	IsDefault bool   `json:"isDefault" gorm:"column:is_default;default:false"`
}

type WishlistItem struct {
	db.BaseEntity
	WishlistID uint      `json:"wishlistId" gorm:"column:wishlist_id;not null;index"`
	ProductID  *uint     `json:"productId"  gorm:"column:product_id;index"`
	SKU        *string   `json:"sku"        gorm:"column:sku;size:255"`
	AddedAt    time.Time `json:"addedAt"    gorm:"column:added_at;autoCreateTime"`
}
