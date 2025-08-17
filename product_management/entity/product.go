package entity

import "time"

type Product struct {
	ID               uint             `json:"id" gorm:"primaryKey"`
	Name             string           `json:"name" binding:"required"`
	Category         string           `json:"category" binding:"required"`
	Price            float64          `json:"price" binding:"required"`
	ShortDescription string           `json:"shortDescription"`
	LongDescription  string           `json:"longDescription"`
	Images           []string         `json:"images" gorm:"type:text[]"`
	InStock          bool             `json:"inStock" gorm:"default:true"`
	IsPopular        bool             `json:"isPopular" gorm:"default:false"`
	Discount         int              `json:"discount" gorm:"default:0"`
	PackageOptions   []PackageOption  `json:"packageOptions" gorm:"foreignKey:ProductID"`
	NutritionalInfo  *NutritionalInfo `json:"nutritionalInfo" gorm:"embedded"`
	Ingredients      []string         `json:"ingredients" gorm:"type:text[]"`
	Benefits         []string         `json:"benefits" gorm:"type:text[]"`
	Instructions     string           `json:"instructions"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}