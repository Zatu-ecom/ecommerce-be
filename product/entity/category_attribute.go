package entity

import (
	"ecommerce-be/common/db"
)

type CategoryAttribute struct {
	db.BaseEntity
	CategoryID            uint `json:"categoryId"            gorm:"column:category_id;not null"`
	AttributeDefinitionID uint `json:"attributeDefinitionId" gorm:"column:attribute_definition_id;not null"`

	// Relationships - use pointers to avoid N+1 queries
	Category            *Category            `json:"category,omitempty"            gorm:"foreignKey:category_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AttributeDefinition *AttributeDefinition `json:"attributeDefinition,omitempty" gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
