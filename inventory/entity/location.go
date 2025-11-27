package entity

import (
	"ecommerce-be/common/db"
	"ecommerce-be/user/entity"
)

// Added for Address entity

type LocationType string

const (
	LocWarehouse    LocationType = "WAREHOUSE"
	LocStore        LocationType = "STORE"
	LocReturnCenter LocationType = "RETURN_CENTER" // Kept as per original, but Type field in struct is now string
)

// Location represents a physical place where inventory is stored
type Location struct {
	db.BaseEntity

	// Basic Info
	Name     string `json:"name"     gorm:"column:name;size:255;not null"`
	Type     string `json:"type"     gorm:"column:type;size:20;default:WAREHOUSE"`
	IsActive bool   `json:"isActive" gorm:"column:is_active;default:true"`
	Priority int    `json:"priority" gorm:"column:priority;default:0"` // Higher number = Higher priority (or lower, depending on logic. Usually 1 is highest)

	// Address Reference
	AddressID uint            `json:"addressId"         gorm:"column:address_id;not null"`
	Address   *entity.Address `json:"address,omitempty" gorm:"foreignKey:AddressID"`
}
