package entity

import (
	"ecommerce-be/common/entity"
)

type CategoryAttribute struct {
	entity.BaseEntity
	CategoryID            uint   `json:"categoryId" gorm:"column:categoryId;not null"`
	AttributeDefinitionID uint   `json:"attributeDefinitionId" gorm:"column:attributeDefinitionId;not null"`
	IsRequired            bool   `json:"isRequired" gorm:"column:isRequired;default:false"`
	IsSearchable          bool   `json:"isSearchable" gorm:"column:isSearchable;default:false"`
	IsFilterable          bool   `json:"isFilterable" gorm:"column:isFilterable;default:false"`
	SortOrder             int    `json:"sortOrder" gorm:"column:sortOrder;default:0"`
	DefaultValue          string `json:"defaultValue" gorm:"column:defaultValue"`
	IsActive              bool   `json:"isActive" gorm:"column:isActive;default:true"`

	// Relationships - use pointers to avoid N+1 queries
	Category            *Category            `json:"category,omitempty" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AttributeDefinition *AttributeDefinition `json:"attributeDefinition,omitempty" gorm:"foreignKey:AttributeDefinitionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
