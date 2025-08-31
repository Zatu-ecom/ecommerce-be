package entity

import "ecommerce-be/common/entity"

type ProductAttribute struct {
	entity.BaseEntity
	ProductID uint   `json:"productId" gorm:"index"`
	Key       string `json:"key" binding:"required"` // matches AttributeDefinition.Key
	Value     string `json:"value" binding:"required"`

	// Composite unique index to prevent duplicate keys for same product
	// gorm:"uniqueIndex:idx_product_key,product_id,key"
}
