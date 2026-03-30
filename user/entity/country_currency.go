package entity

import (
	"time"
)

// CountryCurrency represents the many-to-many relationship between Country and Currency
type CountryCurrency struct {
	ID         uint      `json:"id"         gorm:"primaryKey"`
	CountryID  uint      `json:"countryId"  gorm:"not null;uniqueIndex:idx_country_currency"`
	CurrencyID uint      `json:"currencyId" gorm:"not null;uniqueIndex:idx_country_currency"`
	IsPrimary  bool      `json:"isPrimary"  gorm:"default:false"` // Primary currency for this country
	CreatedAt  time.Time `json:"createdAt"  gorm:"autoCreateTime"`

	// Relationships
	Country  Country  `json:"country,omitempty"  gorm:"foreignKey:CountryID"`
	Currency Currency `json:"currency,omitempty" gorm:"foreignKey:CurrencyID"`
}
