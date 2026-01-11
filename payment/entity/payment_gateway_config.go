package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"ecommerce-be/common/db"
)

// GatewayEnvironment represents the environment type
type GatewayEnvironment string

const (
	EnvironmentSandbox    GatewayEnvironment = "sandbox"
	EnvironmentProduction GatewayEnvironment = "production"
)

// GatewayCredentials represents encrypted gateway credentials
type GatewayCredentials map[string]any

// Scan implements sql.Scanner interface
func (c *GatewayCredentials) Scan(value any) error {
	if value == nil {
		*c = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal GatewayCredentials value: %v", value)
	}
	return json.Unmarshal(bytes, c)
}

// Value implements driver.Valuer interface
func (c GatewayCredentials) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

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
