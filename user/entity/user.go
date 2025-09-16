package entity

import (
	"ecommerce-be/common/db"
)

type User struct {
	db.BaseEntity
	FirstName   string `json:"firstName"   binding:"required"`
	LastName    string `json:"lastName"    binding:"required"`
	Email       string `json:"email"       binding:"required,email" gorm:"unique"`
	Password    string `json:"-"           binding:"required,min=6"`
	Phone       string `json:"phone"`
	DateOfBirth string `json:"dateOfBirth"`
	Gender      string `json:"gender"`
	IsActive    bool   `json:"isActive"                             gorm:"default:true"`

	// --- Role and Profile Links ---
	RoleID uint `json:"roleId" gorm:"not null"`
}
