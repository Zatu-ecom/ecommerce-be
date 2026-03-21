package entity

import (
	"ecommerce-be/common/db"
)

// GatewayEnvironment represents the environment type
type GatewayEnvironment string

const (
	EnvironmentSandbox    GatewayEnvironment = "sandbox"
	EnvironmentProduction GatewayEnvironment = "production"
)

// GatewayCredentials represents encrypted gateway credentials
type GatewayCredentials = db.JSONMap

// PaymentGatewayConfig represents seller's gateway configuration
type PaymentGatewayConfig struct {
	db.BaseEntity
	SellerID    uint               `json:"sellerId"    gorm:"column:seller_id;not null;index"`
	GatewayID   uint               `json:"gatewayId"   gorm:"column:gateway_id;not null;index"`
	Environment GatewayEnvironment `json:"environment" gorm:"column:environment;size:20;not null"`
	Credentials GatewayCredentials `json:"credentials" gorm:"column:credentials;type:jsonb;not null"`
	IsActive    bool               `json:"isActive"    gorm:"column:is_active;default:true"`
	Priority    int                `json:"priority"    gorm:"column:priority;default:0"`
	Country     string             `json:"country"     gorm:"column:country;size:2;not null"`

	// Relationships
	Gateway *PaymentGateway `json:"gateway,omitempty" gorm:"foreignKey:GatewayID"`
}

func (PaymentGatewayConfig) TableName() string {
	return "payment_gateway_config"
}
