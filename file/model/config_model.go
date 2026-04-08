package model

import "ecommerce-be/file/entity"

// SaveConfigRequest represents the incoming payload for saving a config
type SaveConfigRequest struct {
	ID                *uint                  `json:"id"`
	ProviderID        uint                   `json:"providerId" binding:"required"`
	DisplayName       string                 `json:"displayName" binding:"required,max=150"`
	BucketOrContainer string                 `json:"bucketOrContainer" binding:"required,max=255"`
	Region            string                 `json:"region"`
	Endpoint          string                 `json:"endpoint"`
	BasePath          string                 `json:"basePath"`
	ForcePathStyle    bool                   `json:"forcePathStyle"`
	Credentials       map[string]interface{} `json:"credentials" binding:"required"`
	ConfigJSON        map[string]interface{} `json:"configJson"`
	IsDefault         bool                   `json:"isDefault"`
}

// ConfigResponse represents the outgoing storage configuration (without secrets)
type ConfigResponse struct {
	ID                uint   `json:"id"`
	ProviderID        uint   `json:"providerId"`
	OwnerType         string `json:"ownerType"`
	DisplayName       string `json:"displayName"`
	BucketOrContainer string `json:"bucketOrContainer"`
	IsActive          bool   `json:"isActive"`
	IsDefault         bool   `json:"isDefault"`
	ValidationStatus  string `json:"validationStatus"`
}

// ProviderResponse represents a supported cloud storage provider
type ProviderResponse struct {
	ID          uint   `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	AdapterType string `json:"adapterType"`
}

// MapConfigToResponse maps the db entity to the API response
func MapConfigToResponse(config entity.StorageConfig) ConfigResponse {
	return ConfigResponse{
		ID:                config.ID,
		ProviderID:        config.ProviderID,
		OwnerType:         string(config.OwnerType),
		DisplayName:       config.DisplayName,
		BucketOrContainer: config.BucketOrContainer,
		IsActive:          config.IsActive,
		IsDefault:         config.IsDefault,
		ValidationStatus:  config.ValidationStatus,
	}
}

// MapProviderToResponse maps the db entity to the API response
func MapProviderToResponse(provider entity.StorageProvider) ProviderResponse {
	return ProviderResponse{
		ID:          provider.ID,
		Code:        provider.Code,
		Name:        provider.Name,
		AdapterType: provider.AdapterType,
	}
}
