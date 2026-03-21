package entity

import (
	"ecommerce-be/common/db"
)

// PaymentMethodType represents the type of payment method
type PaymentMethodType string

const (
	PaymentMethodTypeCard        PaymentMethodType = "card"
	PaymentMethodTypeUPI         PaymentMethodType = "upi"
	PaymentMethodTypeWallet      PaymentMethodType = "wallet"
	PaymentMethodTypeBankAccount PaymentMethodType = "bank_account"
)

// PaymentMethodMetadata represents additional payment method metadata
type PaymentMethodMetadata = db.JSONMap

// PaymentMethod represents a saved payment method
type PaymentMethod struct {
	db.BaseEntity
	UserID                 uint                  `json:"userId"                 gorm:"column:user_id;not null;index"`
	GatewayID              uint                  `json:"gatewayId"              gorm:"column:gateway_id;not null"`
	Type                   PaymentMethodType     `json:"type"                   gorm:"column:type;size:50;not null"`
	GatewayCustomerID      string                `json:"gatewayCustomerId"      gorm:"column:gateway_customer_id;size:255"`
	GatewayPaymentMethodID string                `json:"gatewayPaymentMethodId" gorm:"column:gateway_payment_method_id;size:255;not null;index"`
	DisplayName            string                `json:"displayName"            gorm:"column:display_name;size:200"`
	Metadata               PaymentMethodMetadata `json:"metadata"               gorm:"column:metadata;type:jsonb"`
	IsDefault              bool                  `json:"isDefault"              gorm:"column:is_default;default:false"`
	// Relationships
	Gateway *PaymentGateway `json:"gateway,omitempty"      gorm:"foreignKey:GatewayID"`
}

func (PaymentMethod) TableName() string {
	return "payment_method"
}
