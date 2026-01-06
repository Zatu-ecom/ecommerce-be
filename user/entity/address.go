package entity

import "ecommerce-be/common/db"

type Address struct {
	db.BaseEntity
	UserID    uint   `json:"userId"`
	Street    string `json:"street"    binding:"required"`
	City      string `json:"city"      binding:"required"`
	State     string `json:"state"     binding:"required"`
	ZipCode   string `json:"zipCode"   binding:"required"`
	CountryID uint   `json:"countryId" gorm:"not null"`
	IsDefault bool   `json:"isDefault" gorm:"default:false"`

	// Relationships
	Country Country `json:"country,omitempty" gorm:"foreignKey:CountryID"`
}
