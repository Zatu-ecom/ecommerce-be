package entity

import (
	"ecommerce-be/common/db"
)

type ProductVariant struct {
	db.BaseEntity
	ProductID uint           `json:"productId" gorm:"column:product_id;not null"`
	SKU           string         `json:"sku"           gorm:"column:sku;uniqueIndex"                binding:"required"`
	Price         float64        `json:"price"         gorm:"column:price"                      binding:"required,gt=0"`
	Images        db.StringArray `json:"images"        gorm:"column:images;type:text[]"`
	AllowPurchase bool           `json:"allowPurchase" gorm:"column:allow_purchase;default:true"`
	IsPopular     bool           `json:"isPopular"     gorm:"column:is_popular;default:false"`
	IsDefault bool           `json:"isDefault" gorm:"column:is_default;default:false"`
}

type VariantOptionValue struct {
	db.BaseEntity
	VariantID     uint `json:"variantId"     gorm:"column:variant_id;not null"`
	OptionID      uint `json:"optionId"      gorm:"column:option_id;not null"`
	OptionValueID uint `json:"optionValueId" gorm:"column:option_value_id;not null"`
}
