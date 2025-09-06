package entity

import (
	"ecommerce-be/common/entity"
)

type CategoryAttribute struct {
	entity.BaseEntity
	CategoryID            uint   `json:"categoryId" gorm:"column:category_id;not null"`
	AttributeDefinitionID uint   `json:"attributeDefinitionId" gorm:"column:attribute_definition_id;not null"`
	IsRequired            bool   `json:"isRequired" gorm:"column:is_required;default:false"`
	IsSearchable          bool   `json:"isSearchable" gorm:"column:is_searchable;default:false"`
	IsFilterable          bool   `json:"isFilterable" gorm:"column:is_filterable;default:false"`
	SortOrder             int    `json:"sortOrder" gorm:"column:sort_order;default:0"`
	DefaultValue          string `json:"defaultValue" gorm:"column:defaultValue"`
	IsActive              bool   `json:"isActive" gorm:"column:is_active;default:true"`

	// Relationships - use pointers to avoid N+1 queries
	Category            *Category            `json:"category,omitempty" gorm:"foreignKey:category_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AttributeDefinition *AttributeDefinition `json:"attributeDefinition,omitempty" gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
