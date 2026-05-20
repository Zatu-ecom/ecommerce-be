package constant

// ========================================
// FILE CONFIG ERROR CODES
// ========================================

const (
	FILE_PROVIDER_NOT_FOUND_CODE       = "FILE_PROVIDER_NOT_FOUND"
	FILE_INVALID_ROLE_CODE             = "FILE_INVALID_ROLE"
	FILE_CONFIG_NOT_FOUND_CODE         = "FILE_CONFIG_NOT_FOUND"
	FILE_CONFIG_FORBIDDEN_CODE         = "FILE_CONFIG_FORBIDDEN"
	FILE_CONFIG_SERIALIZATION_ERR_CODE = "FILE_CONFIG_SERIALIZATION_FAILED"
	FILE_CONFIG_ENCRYPTION_ERR_CODE    = "FILE_CONFIG_ENCRYPTION_FAILED"
	FILE_CONFIG_PERSISTENCE_ERR_CODE   = "FILE_CONFIG_PERSISTENCE_FAILED"
	FILE_CONFIG_LIST_ERR_CODE              = "FILE_CONFIG_LIST_FAILED"
	FILE_LIST_VALIDATION_ERR_CODE          = "VALIDATION_ERROR"
	FILE_CONFIG_INVALID_CREDENTIALS_CODE   = "FILE_CONFIG_INVALID_CREDENTIALS"
	FILE_ADAPTER_SCHEMA_NOT_FOUND_CODE     = "FILE_ADAPTER_SCHEMA_NOT_FOUND"
)

// ========================================
// FILE CONFIG ERROR MESSAGES
// ========================================

const (
	FILE_PROVIDER_NOT_FOUND_MSG       = "Storage provider not found or inactive"
	FILE_INVALID_ROLE_MSG             = "Only seller or admin users can manage storage configurations"
	FILE_CONFIG_NOT_FOUND_MSG         = "Storage config not found"
	FILE_CONFIG_FORBIDDEN_MSG         = "You are not authorized to manage this storage config"
	FILE_CONFIG_SERIALIZATION_ERR_MSG = "Failed to process configuration data"
	FILE_CONFIG_ENCRYPTION_ERR_MSG    = "Failed to encrypt credentials"
	FILE_CONFIG_PERSISTENCE_ERR_MSG   = "Failed to persist storage configuration"
	FILE_CONFIG_LIST_ERR_MSG          = "Failed to list storage configurations"
	FILE_ACTIVATE_NOT_IMPLEMENTED_MSG = "Activate storage config endpoint is not implemented yet"
	FILE_ACTIVE_NOT_IMPLEMENTED_MSG   = "Get active storage config endpoint is not implemented yet"
	FILE_AUTH_REQUIRED_MSG            = "Authentication required"
	FILE_ROLE_DATA_MISSING_MSG        = "Role data missing in token context"
	FILE_CONFIG_NOT_LOADED_MSG        = "application config is not loaded"
	FILE_CONFIG_INVALID_ID_MSG               = "Invalid config ID"
	FILE_LIST_SELLER_ID_FIELD               = "sellerId"
	FILE_LIST_SELLER_ID_ERR_MSG             = "sellerId is not an allowed filter"
	FILE_LIST_VALIDATION_ERR_MSG            = "Validation failed"
	FILE_CONFIG_INVALID_CREDENTIALS_MSG     = "Invalid credentials for the selected storage provider"
	FILE_ADAPTER_SCHEMA_NOT_FOUND_MSG       = "No schema found for the requested adapter type"
	FILE_ADAPTER_SCHEMA_FETCHED_MSG         = "Adapter schema fetched successfully"
	FILE_ADAPTER_TYPE_QUERY_PARAM           = "adapterType"
)

// ========================================
// FILE CONFIG SUCCESS MESSAGES
// ========================================

const (
	FILE_PROVIDERS_FETCHED_MSG = "Storage providers fetched successfully"
	FILE_CONFIG_SAVED_MSG      = "Storage config saved successfully"
	FILE_CONFIG_UPDATED_MSG    = "Storage config updated successfully"
	FILE_CONFIG_LISTED_MSG     = "Storage configs retrieved successfully"
	FILE_CONFIG_TEST_SUCCEEDED_MSG = "Storage configuration is reachable"
)

// ========================================
// BLOB ADAPTER ERROR CODES
// ========================================

const (
	BLOB_ADAPTER_NOT_FOUND_CODE         = "BLOB_ADAPTER_NOT_FOUND"
	BLOB_ADAPTER_PERMISSION_DENIED_CODE = "BLOB_ADAPTER_PERMISSION_DENIED"
	BLOB_ADAPTER_NETWORK_ERR_CODE       = "BLOB_ADAPTER_NETWORK_ERROR"
	BLOB_ADAPTER_VALIDATION_ERR_CODE    = "BLOB_ADAPTER_VALIDATION_ERROR"
	BLOB_ADAPTER_INTERNAL_ERR_CODE      = "BLOB_ADAPTER_INTERNAL_ERROR"
	BLOB_FACTORY_INIT_ERR_CODE          = "BLOB_ADAPTER_FACTORY_INIT_FAILED"
)

// ========================================
// BLOB ADAPTER ERROR MESSAGES
// ========================================

const (
	BLOB_ADAPTER_NOT_FOUND_MSG         = "The requested object or bucket was not found"
	BLOB_ADAPTER_PERMISSION_DENIED_MSG = "Access denied by the storage provider"
	BLOB_ADAPTER_NETWORK_ERR_MSG       = "A network error occurred while communicating with the storage provider"
	BLOB_ADAPTER_VALIDATION_ERR_MSG    = "Invalid parameters supplied to the blob adapter operation"
	BLOB_ADAPTER_INTERNAL_ERR_MSG      = "An unexpected error occurred in the blob adapter"
	BLOB_FACTORY_INIT_ERR_MSG          = "Failed to initialise the blob adapter factory"
)

// ========================================
// FILE CONFIG FAILURE CONTEXT MESSAGES
// ========================================

const (
	FAILED_TO_FETCH_PROVIDERS_MSG        = "Failed to fetch storage providers"
	FAILED_TO_SAVE_CONFIG_MSG            = "Failed to save storage config"
	FAILED_TO_UPDATE_CONFIG_MSG          = "Failed to update storage config"
	FAILED_TO_TEST_CONFIG_MSG            = "Storage configuration test failed"
	FILE_PROVIDER_LOOKUP_FAILED_FMT      = "Provider lookup failed: %v"
	FILE_INVALID_CREDENTIALS_PAYLOAD_FMT = "Invalid credentials payload: %v"
	FILE_SAVE_CONFIG_FAILED_FMT          = "Failed to save config: %v"
	FILE_LIST_CONFIG_FAILED_FMT          = "Failed to list configs: %v"
	FILE_CONFIG_LOOKUP_FAILED_FMT        = "Config lookup failed: %v"
)
