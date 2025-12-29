package entity

import (
	"ecommerce-be/common/db"
)

// Country represents a country in the system (ISO 3166-1 standard)
type Country struct {
	db.BaseEntity
	Code       string `json:"code"       gorm:"size:2;uniqueIndex;not null"`   // ISO 3166-1 alpha-2: 'US', 'IN', 'GB'
	CodeAlpha3 string `json:"codeAlpha3" gorm:"size:3;uniqueIndex"`            // ISO 3166-1 alpha-3: 'USA', 'IND', 'GBR'
	Name       string `json:"name"       gorm:"size:100;not null"`             // 'United States', 'India'
	NativeName string `json:"nativeName" gorm:"size:100"`                      // 'भारत' for India
	PhoneCode  string `json:"phoneCode"  gorm:"size:10"`                       // '+1', '+91'
	Region     string `json:"region"     gorm:"size:50"`                       // 'Asia', 'Europe', 'Americas'
	FlagEmoji  string `json:"flagEmoji"  gorm:"size:10"`                       // '🇺🇸', '🇮🇳'
	IsActive   bool   `json:"isActive"   gorm:"default:true"`                  // Platform supports this country

	// Relationships
	Currencies []Currency `json:"currencies,omitempty" gorm:"many2many:country_currency;"`
}
