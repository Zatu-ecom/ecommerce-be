package model

import (
	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
)

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

// ListStorageConfigQueryParams represents the incoming filtering and pagination query params
type ListStorageConfigQueryParams struct {
	common.BaseListParams
	Ids                string  `form:"ids"` // comma-separated
	ProviderIds        string  `form:"providerIds"`
	ValidationStatuses string  `form:"validationStatuses"`
	IsActive           *bool   `form:"isActive"`
	IsDefault          *bool   `form:"isDefault"`
	AdapterType        *string `form:"adapterType"`
	Search             *string `form:"search"`
}

// ToFilter converts raw query params into a normalized list filter.
func (p ListStorageConfigQueryParams) ToFilter() ListStorageConfigFilter {
	return ListStorageConfigFilter{
		BaseListParams:     p.BaseListParams,
		IDs:                helper.ParseCommaSeparated[uint](p.Ids),
		ProviderIDs:        helper.ParseCommaSeparated[uint](p.ProviderIds),
		ValidationStatuses: helper.ParseCommaSeparated[string](p.ValidationStatuses),
		IsActive:           p.IsActive,
		IsDefault:          p.IsDefault,
		AdapterType:        p.AdapterType,
		Search:             p.Search,
	}
}

// ListStorageConfigFilter represents the normalized list options for the repository
type ListStorageConfigFilter struct {
	common.BaseListParams
	OwnerType          entity.OwnerType
	OwnerID            *uint
	IDs                []uint
	ProviderIDs        []uint
	ValidationStatuses []string
	IsActive           *bool
	IsDefault          *bool
	AdapterType        *string
	Search             *string
}

// StorageConfigListItem represents a single row in the list response
type StorageConfigListItem struct {
	ID                uint   `json:"id"`
	ProviderID        uint   `json:"providerId"`
	OwnerType         string `json:"ownerType"`
	DisplayName       string `json:"displayName"`
	BucketOrContainer string `json:"bucketOrContainer"`
	Region            string `json:"region,omitempty"`
	Endpoint          string `json:"endpoint,omitempty"`
	BasePath          string `json:"basePath,omitempty"`
	ForcePathStyle    bool   `json:"forcePathStyle"`
	IsActive          bool   `json:"isActive"`
	IsDefault         bool   `json:"isDefault"`
	ValidationStatus  string `json:"validationStatus"`
}

// ListStorageConfigsResponse represents the paginated response
type ListStorageConfigsResponse struct {
	Configs    []StorageConfigListItem   `json:"configs"`
	Pagination common.PaginationResponse `json:"pagination"`
}

// ActivateStorageConfigResponse represents the activation success data
type ActivateStorageConfigResponse struct {
	ID        uint   `json:"id"`
	IsActive  bool   `json:"isActive"`
	OwnerType string `json:"ownerType"`
	OwnerID   *uint  `json:"ownerId,omitempty"`
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

// MapConfigToListItem maps the db entity to a list item response
func MapConfigToListItem(config entity.StorageConfig) StorageConfigListItem {
	return StorageConfigListItem{
		ID:                config.ID,
		ProviderID:        config.ProviderID,
		OwnerType:         string(config.OwnerType),
		DisplayName:       config.DisplayName,
		BucketOrContainer: config.BucketOrContainer,
		Region:            config.Region,
		Endpoint:          config.Endpoint,
		BasePath:          config.BasePath,
		ForcePathStyle:    config.ForcePathStyle,
		IsActive:          config.IsActive,
		IsDefault:         config.IsDefault,
		ValidationStatus:  config.ValidationStatus,
	}
}

// MapConfigToActivateResponse maps the db entity to the activation response
func MapConfigToActivateResponse(config entity.StorageConfig) ActivateStorageConfigResponse {
	return ActivateStorageConfigResponse{
		ID:        config.ID,
		IsActive:  config.IsActive,
		OwnerType: string(config.OwnerType),
		OwnerID:   config.OwnerID,
	}
}
