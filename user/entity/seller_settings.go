package entity

import (
	"ecommerce-be/common/db"
)

// Payments environment modes for the payment gateway platform.
const (
	PaymentsEnvironmentSandbox    = "sandbox"
	PaymentsEnvironmentProduction = "production"
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

	// PaymentsEnvironment selects which gateway credential set checkout uses:
	// 'sandbox' (default) or 'production'. Frozen onto each payment_transaction
	// at initiate time so a later toggle never reinterprets old payments.
	PaymentsEnvironment string `json:"paymentsEnvironment" gorm:"size:20;not null;default:sandbox"`
}
