package entity

import (
	"ecommerce-be/common/entity"
)

type PackageOption struct {
	entity.BaseEntity
	ProductID   uint    `json:"productId"   gorm:"column:product_id;not null"`
	Name        string  `json:"name"        gorm:"column:name"                binding:"required"`
	Description string  `json:"description" gorm:"column:description"`
	Price       float64 `json:"price"       gorm:"column:price"               binding:"required,gt=0"`
	Quantity    int     `json:"quantity"    gorm:"column:quantity"            binding:"required,gt=0"`

	// Relationships - use pointers to avoid N+1 queries
	Product *Product `json:"product,omitempty" gorm:"foreignKey:product_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
