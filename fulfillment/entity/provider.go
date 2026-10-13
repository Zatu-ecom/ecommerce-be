package entity

import (
	"ecommerce-be/common/db"
)

// ============================================================================
// CourierProvider — catalog of supported couriers (032, T1).
// New courier (aggregator OR direct) = 1 seed row. No DDL, no code change.
// ============================================================================

type CourierProvider struct {
	db.BaseEntity
	Code string `json:"code" gorm:"column:code;size:50;not null;uniqueIndex"`
	Name string `json:"name" gorm:"column:name;size:100;not null"`
	Kind string `json:"kind" gorm:"column:kind;size:20;not null"` // 'aggregator', 'direct'

	// Capability flags (UI gates; direct couriers often lack NDR).
	SupportsPickup   bool `json:"supportsPickup"    gorm:"column:supports_pickup;not null;default:true"`
	SupportsNDR      bool `json:"supportsNdr"       gorm:"column:supports_ndr;not null;default:true"`
	SupportsReturn   bool `json:"supportsReturn"    gorm:"column:supports_return;not null;default:true"`
	SupportsCOD      bool `json:"supportsCod"       gorm:"column:supports_cod;not null;default:true"`
	WebhookSupported bool `json:"webhookSupported"  gorm:"column:webhook_supported;not null;default:true"`

	IsActive bool `json:"isActive" gorm:"column:is_active;not null;default:true"`
}

// TableName returns the singular table name.
func (CourierProvider) TableName() string {
	return "courier_provider"
}

// ============================================================================
// CourierProviderField — dynamic credential form per provider (032, T2).
// New provider's form = seed rows. No migration per provider.
// ============================================================================

type CourierProviderField struct {
	db.BaseEntity
	ProviderCode string `json:"providerCode" gorm:"column:provider_code;size:50;not null;index"`

	FieldName   string `json:"fieldName"   gorm:"column:field_name;size:100;not null"`
	DisplayName string `json:"displayName" gorm:"column:display_name;size:200;not null"`

	FieldType   string  `json:"fieldType"   gorm:"column:field_type;size:50;not null"`
	Description *string `json:"description" gorm:"column:description"`
	Placeholder *string `json:"placeholder" gorm:"column:placeholder;size:200"`

	IsRequired  bool `json:"isRequired"  gorm:"column:is_required;not null;default:true"`
	IsSensitive bool `json:"isSensitive" gorm:"column:is_sensitive;not null;default:false"`

	DisplayOrder    int        `json:"displayOrder"    gorm:"column:display_order;not null;default:0"`
	ValidationRules db.JSONMap `json:"validationRules" gorm:"column:validation_rules;type:jsonb"`
}

// TableName returns the singular table name.
func (CourierProviderField) TableName() string {
	return "courier_provider_field"
}
