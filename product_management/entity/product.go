package entity

import (
	"ecommerce-be/common/entity"
)

type Product struct {
	entity.BaseEntity
	Name             string   `json:"name" binding:"required"`
	Category         uint     `json:"categoryId" binding:"required"`
	Price            float64  `json:"price" binding:"required"`
	ShortDescription string   `json:"shortDescription"`
	LongDescription  string   `json:"longDescription"`
	Images           []string `json:"images" gorm:"type:text[]"`
	InStock          bool     `json:"inStock" gorm:"default:true"`
	IsPopular        bool     `json:"isPopular" gorm:"default:false"`
	Discount         int      `json:"discount" gorm:"default:0"`
	PackageOptionID  uint     `json:"packageOptionId" binding:"required"`
}