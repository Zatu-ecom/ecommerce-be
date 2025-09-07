package entity

import (
	"ecommerce-be/common/entity"
)

// TODO : Remove isActive field remove all 
type Product struct {
	entity.BaseEntity
	Name             string   `json:"name" binding:"required" gorm:"column:name"`
	CategoryID       uint     `json:"categoryId" binding:"required" gorm:"column:category_id"`
	Brand            string   `json:"brand" gorm:"column:brand"`
	SKU              string   `json:"sku" binding:"required" gorm:"column:sku;uniqueIndex"`
	Price            float64  `json:"price" binding:"required" gorm:"column:price"`
	Currency         string   `json:"currency" gorm:"column:currency;default:USD"`
	ShortDescription string   `json:"shortDescription" gorm:"column:short_description"`
	LongDescription  string   `json:"longDescription" gorm:"column:long_description"`
	Images           []string `json:"images" gorm:"column:images;type:text[]"`
	InStock          bool     `json:"inStock" gorm:"column:in_stock;default:true"`
	IsPopular        bool     `json:"isPopular" gorm:"column:is_popular;default:false"`
	Discount         int      `json:"discount" gorm:"column:discount;default:0"`
	Tags             []string `json:"tags" gorm:"column:tags;type:text[]"`

	// Relationships - use pointers to avoid N+1 queries
	Category *Category `json:"category,omitempty" gorm:"foreignKey:category_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}
