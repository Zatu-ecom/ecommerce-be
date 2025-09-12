package entity

import (
	"ecommerce-be/common/entity"
)

type ProductAttribute struct {
	entity.BaseEntity
	ProductID             uint   `json:"productId"             gorm:"column:product_id;not null"`
	AttributeDefinitionID uint   `json:"attributeDefinitionId" gorm:"column:attribute_definition_id;not null"`
	Value                 string `json:"value"                 gorm:"column:value;not null"`
	SortOrder             uint   `json:"sortOrder"             gorm:"column:sort_order;default:0"`

	// Relationships - use pointers to avoid N+1 queries
	AttributeDefinition *AttributeDefinition `json:"attributeDefinition,omitempty" gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
