package entity

import (
	"ecommerce-be/common/entity"
)

type AttributeDefinition struct {
	entity.BaseEntity
	Key           string             `json:"key"           binding:"required" gorm:"column:key;uniqueIndex"`
	Name          string             `json:"name"          binding:"required" gorm:"column:name"`
	Unit          string             `json:"unit"                             gorm:"column:unit"`
	AllowedValues entity.StringArray `json:"allowedValues"                    gorm:"column:allowed_values;type:text[]"`

	// Relationships - use pointers to avoid N+1 queries
	CategoryAttributes []CategoryAttribute `json:"categoryAttributes,omitempty" gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	ProductAttributes  []ProductAttribute  `json:"productAttributes,omitempty"  gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
