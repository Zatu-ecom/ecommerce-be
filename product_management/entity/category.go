package entity

import (
	"ecommerce-be/common/entity"
)

type Category struct {
	entity.BaseEntity
	Name        string `json:"name" binding:"required" gorm:"uniqueIndex"`
	ParentID    uint   `json:"parentId" gorm:"index"`
	Description string `json:"description"`
	IsActive    bool   `json:"isActive" gorm:"default:true"`
}
