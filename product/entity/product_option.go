package entity

import "ecommerce-be/common/db"

type ProductOption struct {
	db.BaseEntity
	ProductID   uint   `json:"productId"   gorm:"column:product_id;not null"`
	Name        string `json:"name"        gorm:"column:name"                binding:"required"`
	DisplayName string `json:"displayName" gorm:"column:display_name"`
	Position    int    `json:"position"    gorm:"column:position;default:0"`

	// Relationships
	Product *Product             `json:"product,omitempty" gorm:"foreignKey:product_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Values  []ProductOptionValue `json:"values,omitempty"  gorm:"foreignKey:option_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type ProductOptionValue struct {
	db.BaseEntity
	OptionID    uint   `json:"optionId"    gorm:"column:option_id;not null"`
	Value       string `json:"value"       gorm:"column:value"              binding:"required"`
	DisplayName string `json:"displayName" gorm:"column:display_name"`
	ColorCode   string `json:"colorCode"   gorm:"column:color_code"`
	Position    int    `json:"position"    gorm:"column:position;default:0"`

	// Relationships
	Option *ProductOption `json:"option,omitempty" gorm:"foreignKey:option_id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
