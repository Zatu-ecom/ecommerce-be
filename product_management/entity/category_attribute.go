package entity

import (
	"ecommerce-be/common/entity"
)

type CategoryAttribute struct {
	entity.BaseEntity
	CategoryID            uint   `json:"categoryId" gorm:"index"`
	AttributeDefinitionID uint   `json:"attributeDefinitionId" gorm:"index"`
	IsRequired            bool   `json:"isRequired" gorm:"default:false"`
	IsSearchable          bool   `json:"isSearchable" gorm:"default:true"`
	IsFilterable          bool   `json:"isFilterable" gorm:"default:true"`
	SortOrder             int    `json:"sortOrder" gorm:"default:0"`
	DefaultValue          string `json:"defaultValue"`
	IsActive              bool   `json:"isActive" gorm:"default:true"`
}
