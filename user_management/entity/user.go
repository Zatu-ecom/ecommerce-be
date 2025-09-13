package entity

import (
	"ecommerce-be/common/entity"
)

type User struct {
	entity.BaseEntity
	FirstName   string `json:"firstName"   binding:"required"`
	LastName    string `json:"lastName"    binding:"required"`
	Email       string `json:"email"       binding:"required,email" gorm:"unique"`
	Password    string `json:"-"           binding:"required,min=6"`
	Phone       string `json:"phone"`
	DateOfBirth string `json:"dateOfBirth"`
	Gender      string `json:"gender"`
	IsActive    bool   `json:"isActive"                             gorm:"default:true"`

	// Relationships
	// Addresses []Address `json:"addresses" gorm:"foreignKey:UserID"`
	// Orders    []Order   `json:"orders" gorm:"foreignKey:UserID"`
}
