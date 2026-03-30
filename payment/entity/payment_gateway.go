package entity

import (
	"ecommerce-be/common/db"
)

// PaymentGateway represents a payment gateway provider
type PaymentGateway struct {
	db.BaseEntity
	Code                    string         `json:"code"                    gorm:"column:code;size:50;not null;uniqueIndex"`
	Name                    string         `json:"name"                    gorm:"column:name;size:100;not null"`
	Description             string         `json:"description"             gorm:"column:description;type:text"`
	LogoURL                 string         `json:"logoUrl"                 gorm:"column:logo_url;size:500"`
	IsActive                bool           `json:"isActive"                gorm:"column:is_active;default:true"`
	SupportedCountries      db.StringArray `json:"supportedCountries"      gorm:"column:supported_countries;type:varchar(2)[]"`
	SupportedCurrencies     db.StringArray `json:"supportedCurrencies"     gorm:"column:supported_currencies;type:varchar(3)[];not null"`
	SupportedPaymentMethods db.StringArray `json:"supportedPaymentMethods" gorm:"column:supported_payment_methods;type:text[];not null"`
	WebhookURL              string         `json:"webhookUrl"              gorm:"column:webhook_url;size:500"`
}

func (PaymentGateway) TableName() string {
	return "payment_gateway"
}
