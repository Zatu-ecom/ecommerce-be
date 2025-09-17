package entity

import (
	"ecommerce-be/common/db"
)

type OrderItem struct {
	db.BaseEntity
	OrderID     uint    `json:"orderId"`
	SellerID    uint    `json:"sellerId"    binding:"required"`
	ProductID   uint    `json:"productId"`
	PackageSize string  `json:"packageSize"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}
