package entity

import (
	"ecommerce-be/common/db"
)

// ProductMedia represents a product-owned association to an uploaded file-module asset.
// Product cascades deletion to ProductMedia rows; there is no DB FK to file tables.
type ProductMedia struct {
	db.BaseEntity
	ProductID    uint   `gorm:"column:product_id;not null"`
	FileID       string `gorm:"column:file_id;not null"`
	IsPrimary    bool   `gorm:"column:is_primary;not null;default:false"`
	DisplayOrder int    `gorm:"column:display_order;not null;default:0"`
}

func (ProductMedia) TableName() string {
	return "product_media"
}
