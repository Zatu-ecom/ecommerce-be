package entity

import (
	"ecommerce-be/common/entity"
)

type AttributeDefinition struct {
	entity.BaseEntity
	Key           string   `json:"key" binding:"required" gorm:"column:key;uniqueIndex"`
	Name          string   `json:"name" binding:"required" gorm:"column:name"`
	DataType      string   `json:"dataType" binding:"required" gorm:"column:data_type"` // string, number, boolean, array
	Unit          string   `json:"unit" gorm:"column:unit"`
	Description   string   `json:"description" gorm:"column:description"`
	AllowedValues []string `json:"allowedValues" gorm:"column:allowed_values;type:text[]"`
	IsActive      bool     `json:"isActive" gorm:"column:is_active;default:true"`

	// Relationships - use pointers to avoid N+1 queries
	CategoryAttributes []CategoryAttribute `json:"categoryAttributes,omitempty" gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	ProductAttributes  []ProductAttribute  `json:"productAttributes,omitempty" gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
