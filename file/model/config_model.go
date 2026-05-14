package model

import (
	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/file/entity"
)

// SaveConfigRequest represents the incoming payload for creating a storage config.
// All provider-specific settings (credentials + routing) are unified in Config.
// The adapter schema API (GET /storage-config/schema, optional ?adapterType=) describes
// what fields each provider requires.
// Omitted isActive / isDefault default to true on create.
type SaveConfigRequest struct {
	ProviderID        uint   `json:"providerId"        binding:"required"`
	DisplayName       string `json:"displayName"       binding:"required,max=150"`
	BucketOrContainer string `json:"bucketOrContainer" binding:"required,max=255"`
	// Config holds ALL provider-specific settings: credentials, endpoint, region, etc.
	// The exact required keys depend on the adapter type — see the schema API.
	// For GCS, config.service_account_json may be a JSON string (the key file) or a nested
	// JSON object with the same fields; both are accepted and stored as an encrypted string.
	Config    map[string]any `json:"config"            binding:"required"`
	IsActive  *bool          `json:"isActive,omitempty"`
	IsDefault *bool          `json:"isDefault,omitempty"`
}

// UpdateStorageConfigRequest is the body for PUT /storage-config/:id.
type UpdateStorageConfigRequest struct {
	ProviderID        uint           `json:"providerId"        binding:"required"`
	DisplayName       string         `json:"displayName"       binding:"required,max=150"`
	BucketOrContainer string         `json:"bucketOrContainer" binding:"required,max=255"`
	Config            map[string]any `json:"config"            binding:"required"`
	IsActive          bool           `json:"isActive"`
	IsDefault         bool           `json:"isDefault"`
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
}

// ProviderResponse represents a supported cloud storage provider
type ProviderResponse struct {
	ID          uint               `json:"id"`
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	AdapterType entity.AdapterType `json:"adapterType"`
}

// ListStorageConfigQueryParams represents the incoming filtering and pagination query params
type ListStorageConfigQueryParams struct {
	common.BaseListParams
	Ids         string              `form:"ids"` // comma-separated
	ProviderIds string              `form:"providerIds"`
	IsActive    *bool               `form:"isActive"`
	IsDefault   *bool               `form:"isDefault"`
	AdapterType *entity.AdapterType `form:"adapterType"`
	Search      *string             `form:"search"`
}

// ToFilter converts raw query params into a normalized list filter.
func (p ListStorageConfigQueryParams) ToFilter() ListStorageConfigFilter {
	return ListStorageConfigFilter{
		BaseListParams: p.BaseListParams,
		IDs:            helper.ParseCommaSeparated[uint](p.Ids),
		ProviderIDs:    helper.ParseCommaSeparated[uint](p.ProviderIds),
		IsActive:       p.IsActive,
		IsDefault:      p.IsDefault,
		AdapterType:    p.AdapterType,
		Search:         p.Search,
	}
}

// ListStorageConfigFilter represents the normalized list options for the repository
type ListStorageConfigFilter struct {
	common.BaseListParams
	OwnerType   entity.OwnerType
	OwnerID     *uint
	IDs         []uint
	ProviderIDs []uint
	IsActive    *bool
	IsDefault   *bool
	AdapterType *entity.AdapterType
	Search      *string
}

// StorageConfigListItem represents a single row in the list response.
// Routing details (region, endpoint, etc.) are not returned because they now
// live inside the encrypted config_data blob.
type StorageConfigListItem struct {
	ID                uint           `json:"id"`
	ProviderID        uint           `json:"providerId"`
	OwnerType         string         `json:"ownerType"`
	DisplayName       string         `json:"displayName"`
	BucketOrContainer string         `json:"bucketOrContainer"`
	IsActive          bool           `json:"isActive"`
	ConfigData        map[string]any `json:"configData"`
	IsDefault         bool           `json:"isDefault"`
}

// ListStorageConfigsResponse represents the paginated response
type ListStorageConfigsResponse struct {
	Configs    []StorageConfigListItem   `json:"configs"`
	Pagination common.PaginationResponse `json:"pagination"`
}

// TestStorageConfigResponse is returned when a dry-run connectivity check succeeds.
type TestStorageConfigResponse struct {
	OK bool `json:"ok"`
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
func MapConfigToListItem(
	config entity.StorageConfig,
	cnf map[string]any,
) StorageConfigListItem {
	return StorageConfigListItem{
		ID:                config.ID,
		ProviderID:        config.ProviderID,
		OwnerType:         string(config.OwnerType),
		DisplayName:       config.DisplayName,
		BucketOrContainer: config.BucketOrContainer,
		IsActive:          config.IsActive,
		IsDefault:         config.IsDefault,
		ConfigData:        cnf,
	}
}
