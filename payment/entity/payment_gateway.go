package entity

import (
	"ecommerce-be/common/db"
)

// PaymentGateway represents a payment gateway provider.
// Geo support lives in the payment_gateway_country / payment_gateway_currency
// join tables (queried via EXISTS), not on string-array columns.
type PaymentGateway struct {
	db.BaseEntity
	Code                    string         `json:"code"                    gorm:"column:code;size:50;not null;uniqueIndex"`
	Name                    string         `json:"name"                    gorm:"column:name;size:100;not null"`
	Description             string         `json:"description"             gorm:"column:description;type:text"`
	LogoFileID              *string        `json:"logoFileId"              gorm:"column:logo_file_id;size:80"`
	IsActive                bool           `json:"isActive"                gorm:"column:is_active;default:true"`
	SupportedPaymentMethods db.StringArray `json:"supportedPaymentMethods" gorm:"column:supported_payment_methods;type:text[];not null"`
}

func (PaymentGateway) TableName() string {
	return "payment_gateway"
}
