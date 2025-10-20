package entity

import (
	"ecommerce-be/common/db"
)

type Product struct {
	db.BaseEntity
	Name             string         `json:"name"             binding:"required" gorm:"column:name"`
	CategoryID       uint           `json:"categoryId"       binding:"required" gorm:"column:category_id"`
	Brand            string         `json:"brand"                               gorm:"column:brand"`
	BaseSKU          string         `json:"baseSku"          binding:"required" gorm:"column:base_sku;uniqueIndex"`
	ShortDescription string         `json:"shortDescription"                    gorm:"column:short_description"`
	LongDescription  string         `json:"longDescription"                     gorm:"column:long_description"`
	Tags             db.StringArray `json:"tags"                                gorm:"column:tags;type:text[]"`
	SellerID         uint           `json:"sellerId"                            gorm:"column:seller_id"`

	// Relationships - use pointers to avoid N+1 queries
	Category *Category `json:"category,omitempty" gorm:"foreignKey:category_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
