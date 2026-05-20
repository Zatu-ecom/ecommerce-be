package entity

import (
	"ecommerce-be/common/db"
)

// OwnerType represents the type of owner for a file or storage config
type OwnerType string

const (
	OwnerTypePlatform OwnerType = "PLATFORM"
	OwnerTypeSeller   OwnerType = "SELLER"
	OwnerTypeUser     OwnerType = "USER"
)

type AdapterType string

const (
	AdapterTypeS3Compatible AdapterType = "s3_compatible"
	AdapterTypeGCS          AdapterType = "gcs"
	AdapterTypeAzure        AdapterType = "azure"
)

// StorageProvider represents the master list of storage providers
type StorageProvider struct {
	db.BaseEntity
	Code        string      `gorm:"column:code;unique;not null;size:50"`
	Name        string      `gorm:"column:name;not null;size:100"`
	AdapterType AdapterType `gorm:"column:adapter_type;not null;size:50"`
	IsActive    bool        `gorm:"column:is_active;not null;default:true"`
}

func (StorageProvider) TableName() string {
	return "storage_provider"
}

// StorageConfig represents admin and seller-specific storage credentials
type StorageConfig struct {
	db.BaseEntity
	OwnerType         OwnerType `gorm:"column:owner_type;not null;size:20;index:idx_storage_config_owner"`
	OwnerID           *uint     `gorm:"column:owner_id;index:idx_storage_config_owner"`
	ProviderID        uint      `gorm:"column:provider_id;not null;index"`
	DisplayName       string    `gorm:"column:display_name;not null;size:150"`
	BucketOrContainer string    `gorm:"column:bucket_or_container;not null;size:255"`
	// ConfigData holds the AES-GCM encrypted JSON blob that contains all
	// provider-specific settings (credentials + routing). The structure of
	// the decrypted JSON is typed per adapter: GCSConfig, S3Config, AzureConfig.
	ConfigData db.JSONMap `gorm:"column:config_data;not null"`
	IsDefault  bool       `gorm:"column:is_default;not null;default:false"`
	IsActive   bool       `gorm:"column:is_active;not null;default:true"`

	Provider StorageProvider `gorm:"foreignKey:ProviderID"`
}

func (StorageConfig) TableName() string {
	return "storage_config"
}
