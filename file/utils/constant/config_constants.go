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
	FILE_NOT_IMPLEMENTED_CODE          = "NOT_IMPLEMENTED"
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
	FILE_CONFIG_NOT_IMPLEMENTED_MSG   = "Storage config test endpoint is not implemented yet"
	FILE_ACTIVATE_NOT_IMPLEMENTED_MSG = "Activate storage config endpoint is not implemented yet"
	FILE_ACTIVE_NOT_IMPLEMENTED_MSG   = "Get active storage config endpoint is not implemented yet"
	FILE_AUTH_REQUIRED_MSG            = "Authentication required"
	FILE_ROLE_DATA_MISSING_MSG        = "Role data missing in token context"
	FILE_CONFIG_NOT_LOADED_MSG        = "application config is not loaded"
	FILE_CONFIG_PENDING_STATUS        = "PENDING"
)

// ========================================
// FILE CONFIG SUCCESS MESSAGES
// ========================================

const (
	FILE_PROVIDERS_FETCHED_MSG = "Storage providers fetched successfully"
	FILE_CONFIG_SAVED_MSG      = "Storage config saved successfully"
)

// ========================================
// FILE CONFIG FAILURE CONTEXT MESSAGES
// ========================================

const (
	FAILED_TO_FETCH_PROVIDERS_MSG        = "Failed to fetch storage providers"
	FAILED_TO_SAVE_CONFIG_MSG            = "Failed to save storage config"
	FILE_PROVIDER_LOOKUP_FAILED_FMT      = "Provider lookup failed: %v"
	FILE_INVALID_CREDENTIALS_PAYLOAD_FMT = "Invalid credentials payload: %v"
	FILE_SAVE_CONFIG_FAILED_FMT          = "Failed to save config: %v"
	FILE_CONFIG_LOOKUP_FAILED_FMT        = "Config lookup failed: %v"
)
