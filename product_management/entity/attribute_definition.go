package entity

import "ecommerce-be/common/entity"


type AttributeDefinition struct {
	entity.BaseEntity
	Key           string   `json:"key" binding:"required" gorm:"uniqueIndex"`
	Name          string   `json:"name" binding:"required"`
	DataType      string   `json:"dataType" binding:"required"` // string, number, boolean, array
	Unit          string   `json:"unit"`
	Description   string   `json:"description"`
	AllowedValues []string `json:"allowedValues" gorm:"type:text[]"`
	IsActive      bool     `json:"isActive" gorm:"default:true"`
}
