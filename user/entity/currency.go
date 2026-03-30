package entity

import (
	"ecommerce-be/common/db"
)

// Currency represents a currency in the system (ISO 4217 standard)
type Currency struct {
	db.BaseEntity
	Code          string `json:"code"          gorm:"size:3;uniqueIndex;not null"` // ISO 4217: 'USD', 'INR', 'EUR'
	Name          string `json:"name"          gorm:"size:100;not null"`           // 'US Dollar', 'Indian Rupee'
	Symbol        string `json:"symbol"        gorm:"size:10;not null"`            // '$', '₹', '€'
	SymbolNative  string `json:"symbolNative"  gorm:"size:10"`                     // Native symbol
	DecimalDigits int    `json:"decimalDigits" gorm:"default:2"`                   // 2 for USD, 0 for JPY
	IsActive      bool   `json:"isActive"      gorm:"default:true"`                // Currency is active

	// Relationships
	Countries []Country `json:"countries,omitempty" gorm:"many2many:country_currency;"`
}
