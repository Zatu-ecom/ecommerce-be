package entity

import (
	"time"
)

type User struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	FirstName   string    `json:"firstName" binding:"required"`
	LastName    string    `json:"lastName" binding:"required"`
	Email       string    `json:"email" binding:"required,email" gorm:"unique"`
	Password    string    `json:"-" binding:"required,min=6"`
	Phone       string    `json:"phone"`
	DateOfBirth string    `json:"dateOfBirth"`
	Gender      string    `json:"gender"`
	IsActive    bool      `json:"isActive" gorm:"default:true"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// Relationships
	// Addresses []Address `json:"addresses" gorm:"foreignKey:UserID"`
	// Orders    []Order   `json:"orders" gorm:"foreignKey:UserID"`
}
