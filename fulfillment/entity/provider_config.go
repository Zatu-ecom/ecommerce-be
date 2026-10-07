package entity

import (
	"ecommerce-be/common/db"
)

// ============================================================================
// CourierProviderConfig — hybrid credentials (032, T3).
// seller_id NULL = platform default; a seller row overrides it at book time.
// The shipment freezes provider_config_id so later webhooks/tracking use the
// same row even after credential rotation. Carries tenant automation flags.
// ============================================================================

type CourierProviderConfig struct {
	db.BaseEntity
	// NULL = platform default. Pointer distinguishes override from default.
	SellerID     *uint  `json:"sellerId"     gorm:"column:seller_id;index"`
	ProviderCode string `json:"providerCode" gorm:"column:provider_code;size:50;not null;index"`

	Environment string `json:"environment" gorm:"column:environment;size:20;not null;default:production"`

	// AES-256-GCM encrypted credential map; shape validated against fields.
	Credentials db.JSONMap `json:"-" gorm:"column:credentials;type:jsonb;not null"`

	// Tenant automation flags.
	AutoBook           bool    `json:"autoBook"          gorm:"column:auto_book;not null;default:false"`
	RatePreference     *string `json:"ratePreference"    gorm:"column:rate_preference;size:20"`
	DefaultWeightGrams *int    `json:"defaultWeightGrams" gorm:"column:default_weight_grams"`

	IsActive bool `json:"isActive" gorm:"column:is_active;not null;default:true"`
}

// TableName returns the singular table name.
func (CourierProviderConfig) TableName() string {
	return "courier_provider_config"
}
