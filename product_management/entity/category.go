package entity

import (
	"ecommerce-be/common/entity"
)

type Category struct {
	entity.BaseEntity
	Name        string `json:"name"        binding:"required" gorm:"column:name"`
	ParentID    *uint  `json:"parentId"                       gorm:"column:parent_id"`
	Description string `json:"description"                    gorm:"column:description"`

	// Relationships - use pointers to avoid N+1 queries
	Parent   *Category  `json:"parent,omitempty"   gorm:"foreignKey:parent_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Children []Category `json:"children,omitempty" gorm:"foreignKey:parent_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Products []Product  `json:"products,omitempty" gorm:"foreignKey:category_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
