package entity

import (
	"ecommerce-be/common/db"
)

// SellerSettings contains business configuration for a seller
type SellerSettings struct {
	db.BaseEntity
	SellerID uint `json:"sellerId" gorm:"uniqueIndex;not null"` // References user.id (seller)

	// Business location & currency
	BusinessCountryID    uint `json:"businessCountryId"    gorm:"not null"` // Country where business is registered
	BaseCurrencyID       uint `json:"baseCurrencyId"       gorm:"not null"` // Prices stored in this currency
	SettlementCurrencyID uint `json:"settlementCurrencyId"`                 // Payouts in this currency (NULL = same as base)

	// Display preferences
	DisplayPricesInBuyerCurrency bool `json:"displayPricesInBuyerCurrency" gorm:"default:false"` // Convert prices for buyers
}
