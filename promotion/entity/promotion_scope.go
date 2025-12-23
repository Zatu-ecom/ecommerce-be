package entity

import (
	"ecommerce-be/common/db"
)

// PromotionProduct represents products included in a promotion
type PromotionProduct struct {
	db.BaseEntity

	// References
	PromotionID uint  `json:"promotionId" gorm:"column:promotion_id;not null;index"`
	ProductID   uint  `json:"productId"   gorm:"column:product_id;not null;index"`
	VariantID   *uint `json:"variantId"   gorm:"column:variant_id;index"`

	// Variant-specific override (optional)
	OverrideDiscountConfig *DiscountConfig `json:"overrideDiscountConfig" gorm:"column:override_discount_config;type:jsonb"`

	// Relationships
	Promotion *Promotion `json:"promotion,omitempty" gorm:"foreignKey:PromotionID"`
}

// TableName specifies the table name
func (PromotionProduct) TableName() string {
	return "promotion_product"
}

// PromotionCategory represents categories included in a promotion
type PromotionCategory struct {
	db.BaseEntity

	// References
	PromotionID uint `json:"promotionId" gorm:"column:promotion_id;not null;index"`
	CategoryID  uint `json:"categoryId"  gorm:"column:category_id;not null;index"`

	// Include subcategories?
	IncludeSubcategories *bool `json:"includeSubcategories" gorm:"column:include_subcategories;default:true"`

	// Relationships
	Promotion *Promotion `json:"promotion,omitempty" gorm:"foreignKey:PromotionID"`
}

// TableName specifies the table name
func (PromotionCategory) TableName() string {
	return "promotion_category"
}

// PromotionCollection represents collections included in a promotion
type PromotionCollection struct {
	db.BaseEntity

	// References
	PromotionID  uint `json:"promotionId"  gorm:"column:promotion_id;not null;index"`
	CollectionID uint `json:"collectionId" gorm:"column:collection_id;not null;index"`

	// Relationships
	Promotion *Promotion `json:"promotion,omitempty" gorm:"foreignKey:PromotionID"`
}

// TableName specifies the table name
func (PromotionCollection) TableName() string {
	return "promotion_collection"
}
