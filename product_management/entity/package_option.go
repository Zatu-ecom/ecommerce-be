package entity

import "ecommerce-be/common/entity"

type PackageOption struct {
	entity.BaseEntity
	ProductID uint    `json:"productId" gorm:"index"`
	Name      string  `json:"name" binding:"required"`
	Price     float64 `json:"price" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required"`
}
