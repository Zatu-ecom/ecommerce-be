package entity

import (
	"ecommerce-be/common/db"
)

// Role defines the user roles in the system.
type Role struct {
	db.BaseEntity
	Name        string `json:"name"        gorm:"unique;not null;size:50"` // e.g., "ADMIN", "SELLER", "CUSTOMER"
	Description string `json:"description"`                                // A brief description of the role's purpose.
	Level       uint   `json:"level"       gorm:"not null"`                // 1=ADMIN, 2=SELLER, 3=CUSTOMER
}
