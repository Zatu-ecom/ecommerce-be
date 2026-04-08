package entity

import (
	"time"

	"ecommerce-be/common/db"
)

// OwnerType represents the type of owner for a file or storage config
type OwnerType string

const (
	OwnerTypePlatform OwnerType = "PLATFORM"
	OwnerTypeSeller   OwnerType = "SELLER"
	OwnerTypeUser     OwnerType = "USER"
)

// StorageProvider represents the master list of storage providers
type StorageProvider struct {
	db.BaseEntity
	Code        string `gorm:"column:code;unique;not null;size:50"`
	Name        string `gorm:"column:name;not null;size:100"`
	AdapterType string `gorm:"column:adapter_type;not null;size:50"`
	IsActive    bool   `gorm:"column:is_active;not null;default:true"`
}

func (StorageProvider) TableName() string {
	return "storage_provider"
}

// StorageConfig represents admin and seller-specific storage credentials
type StorageConfig struct {
	db.BaseEntity
	OwnerType            OwnerType  `gorm:"column:owner_type;not null;size:20;index:idx_storage_config_owner"`
	OwnerID              *uint      `gorm:"column:owner_id;index:idx_storage_config_owner"`
	ProviderID           uint       `gorm:"column:provider_id;not null;index"`
	DisplayName          string     `gorm:"column:display_name;not null;size:150"`
	BucketOrContainer    string     `gorm:"column:bucket_or_container;not null;size:255"`
	Region               string     `gorm:"column:region;size:100"`
	Endpoint             string     `gorm:"column:endpoint;size:500"`
	BasePath             string     `gorm:"column:base_path;size:500"`
	ForcePathStyle       bool       `gorm:"column:force_path_style;default:false"`
	CredentialsEncrypted []byte     `gorm:"column:credentials_encrypted;not null"`
	ConfigJSON           db.JSONMap `gorm:"column:config_json;type:jsonb"`
	IsDefault            bool       `gorm:"column:is_default;not null;default:false"`
	IsActive             bool       `gorm:"column:is_active;not null;default:true"`
	LastValidatedAt      *time.Time `gorm:"column:last_validated_at"`
	ValidationStatus     string     `gorm:"column:validation_status;not null;default:'PENDING';size:30"`

	Provider StorageProvider `gorm:"foreignKey:ProviderID"`
}

func (StorageConfig) TableName() string {
	return "storage_config"
}

// SellerStorageBinding represents the binding table to choose which seller config is active
type SellerStorageBinding struct {
	db.BaseEntity
	SellerID        uint `gorm:"column:seller_id;not null;uniqueIndex:idx_seller_storage_binding_seller_active"`
	StorageConfigID uint `gorm:"column:storage_config_id;not null"`
	IsActive        bool `gorm:"column:is_active;not null;default:true;uniqueIndex:idx_seller_storage_binding_seller_active"`

	StorageConfig StorageConfig `gorm:"foreignKey:StorageConfigID"`
}

func (SellerStorageBinding) TableName() string {
	return "seller_storage_binding"
}
