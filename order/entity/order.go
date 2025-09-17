package entity

import (
	"ecommerce-be/common/db"
)

type Order struct {
	db.BaseEntity
	UserID      uint        `json:"userId"      binding:"required"`
	SellerID    uint        `json:"sellerId"    binding:"required"`
	OrderNumber string      `json:"orderNumber"                    gorm:"unique"`
	Status      string      `json:"status"                         gorm:"default:'pending'"`
	Subtotal    float64     `json:"subtotal"`
	Tax         float64     `json:"tax"`
	Shipping    float64     `json:"shipping"`
	Total       float64     `json:"total"`
	Items       []OrderItem `json:"items"                          gorm:"foreignKey:OrderID"`
}
