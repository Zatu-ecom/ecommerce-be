package entity

import (
	"ecommerce-be/common/db"
)

// DiscountCodeProduct represents products eligible for a discount code
type DiscountCodeProduct struct {
	db.BaseEntity

	// References
	DiscountCodeID uint  `json:"discountCodeId" gorm:"column:discount_code_id;not null;index"`
	ProductID      uint  `json:"productId"      gorm:"column:product_id;not null;index"`
	VariantID      *uint `json:"variantId"      gorm:"column:variant_id;index"`

	// Relationships
	DiscountCode *DiscountCode `json:"discountCode,omitempty" gorm:"foreignKey:DiscountCodeID"`
}

// TableName specifies the table name
func (DiscountCodeProduct) TableName() string {
	return "discount_code_product"
}

// DiscountCodeCategory represents categories eligible for a discount code
type DiscountCodeCategory struct {
	db.BaseEntity

	// References
	DiscountCodeID uint `json:"discountCodeId" gorm:"column:discount_code_id;not null;index"`
	CategoryID     uint `json:"categoryId"     gorm:"column:category_id;not null;index"`

	// Relationships
	DiscountCode *DiscountCode `json:"discountCode,omitempty" gorm:"foreignKey:DiscountCodeID"`
}

// TableName specifies the table name
func (DiscountCodeCategory) TableName() string {
	return "discount_code_category"
}

// DiscountCodeCollection represents collections eligible for a discount code
type DiscountCodeCollection struct {
	db.BaseEntity

	// References
	DiscountCodeID uint `json:"discountCodeId" gorm:"column:discount_code_id;not null;index"`
	CollectionID   uint `json:"collectionId"   gorm:"column:collection_id;not null;index"`

	// Relationships
	DiscountCode *DiscountCode `json:"discountCode,omitempty" gorm:"foreignKey:DiscountCodeID"`
}

// TableName specifies the table name
func (DiscountCodeCollection) TableName() string {
	return "discount_code_collection"
}
