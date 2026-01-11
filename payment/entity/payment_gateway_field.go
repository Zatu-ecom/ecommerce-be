package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"ecommerce-be/common/db"
)

// FieldType represents the type of configuration field
type FieldType string

const (
	FieldTypeString  FieldType = "string"
	FieldTypeNumber  FieldType = "number"
	FieldTypeBoolean FieldType = "boolean"
	FieldTypeURL     FieldType = "url"
	FieldTypeEmail   FieldType = "email"
)

// ValidationRules represents validation rules for a field
type ValidationRules map[string]any

// Scan implements sql.Scanner interface
func (v *ValidationRules) Scan(value any) error {
	if value == nil {
		*v = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal ValidationRules value: %v", value)
	}
	return json.Unmarshal(bytes, v)
}

// Value implements driver.Valuer interface
func (v ValidationRules) Value() (driver.Value, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// PaymentGatewayField represents a configuration field required by a gateway
type PaymentGatewayField struct {
	db.BaseEntity
	GatewayID       uint            `json:"gatewayId"       gorm:"column:gateway_id;not null;index"`
	FieldName       string          `json:"fieldName"       gorm:"column:field_name;size:100;not null"`
	DisplayName     string          `json:"displayName"     gorm:"column:display_name;size:200;not null"`
	FieldType       FieldType       `json:"fieldType"       gorm:"column:field_type;size:50;not null"`
	Description     string          `json:"description"     gorm:"column:description;type:text"`
	Placeholder     string          `json:"placeholder"     gorm:"column:placeholder;size:200"`
	IsRequired      bool            `json:"isRequired"      gorm:"column:is_required;default:true"`
	IsSensitive     bool            `json:"isSensitive"     gorm:"column:is_sensitive;default:false"`
	DisplayOrder    int             `json:"displayOrder"    gorm:"column:display_order;default:0"`
	ValidationRules ValidationRules `json:"validationRules" gorm:"column:validation_rules;type:jsonb"`

	// Relationships
	Gateway *PaymentGateway `json:"gateway,omitempty" gorm:"foreignKey:GatewayID"`
}

func (PaymentGatewayField) TableName() string {
	return "payment_gateway_field"
}
