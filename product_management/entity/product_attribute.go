package entity

import (
	"ecommerce-be/common/entity"
)

type ProductAttribute struct {
	entity.BaseEntity
	ProductID             uint   `json:"productId" gorm:"column:productId;not null"`
	AttributeDefinitionID uint   `json:"attributeDefinitionId" gorm:"column:attributeDefinitionId;not null"`
	Key                   string `json:"key" gorm:"column:key;not null"`
	Value                 string `json:"value" gorm:"column:value;not null"`
	IsActive              bool   `json:"isActive" gorm:"column:isActive;default:true"`

	// Relationships - use pointers to avoid N+1 queries
	Product             *Product             `json:"product,omitempty" gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AttributeDefinition *AttributeDefinition `json:"attributeDefinition,omitempty" gorm:"foreignKey:AttributeDefinitionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
