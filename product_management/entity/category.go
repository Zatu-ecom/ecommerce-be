package entity

import (
	"ecommerce-be/common/entity"
)

type Category struct {
	entity.BaseEntity
	Name        string `json:"name" binding:"required" gorm:"column:name"`
	ParentID    uint   `json:"parentId" gorm:"column:parent_id;default:0"`
	Description string `json:"description" gorm:"column:description"`
	IsActive    bool   `json:"isActive" gorm:"column:is_active;default:true"`

	// Relationships - use pointers to avoid N+1 queries
	Parent   *Category  `json:"parent,omitempty" gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Children []Category `json:"children,omitempty" gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Products []Product  `json:"products,omitempty" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
