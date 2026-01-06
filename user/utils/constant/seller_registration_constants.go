package constant

// ========================================
// SELLER REGISTRATION ERROR CODES
// ========================================
const (
	EMAIL_ALREADY_EXISTS_CODE       = "EMAIL_ALREADY_EXISTS"
	TAX_ID_ALREADY_EXISTS_CODE      = "TAX_ID_ALREADY_EXISTS"
	TAX_ID_CHECK_FAILED_CODE        = "TAX_ID_CHECK_FAILED"
	USER_CREATE_FAILED_CODE         = "USER_CREATE_FAILED"
	SELLER_ID_UPDATE_FAILED_CODE    = "SELLER_ID_UPDATE_FAILED"
	PROFILE_CREATE_FAILED_CODE      = "PROFILE_CREATE_FAILED"
	SETTINGS_CREATE_FAILED_CODE     = "SETTINGS_CREATE_FAILED"
	TOKEN_GENERATION_FAILED_CODE    = "TOKEN_GENERATION_FAILED"
	SELLER_REGISTRATION_FAILED_CODE = "SELLER_REGISTRATION_FAILED"
)

// ========================================
// SELLER REGISTRATION ERROR MESSAGES
// ========================================
const (
	EMAIL_ALREADY_EXISTS_MSG       = "Email already registered"
	TAX_ID_ALREADY_EXISTS_MSG      = "Tax ID already registered"
	TAX_ID_CHECK_FAILED_MSG        = "Failed to validate tax ID"
	USER_CREATE_FAILED_MSG         = "Failed to create user"
	SELLER_ID_UPDATE_FAILED_MSG    = "Failed to update seller ID"
	PROFILE_CREATE_FAILED_MSG      = "Failed to create seller profile"
	SETTINGS_CREATE_FAILED_MSG     = "Failed to create seller settings"
	TOKEN_GENERATION_FAILED_MSG    = "Failed to generate authentication token"
	SELLER_REGISTRATION_FAILED_MSG = "Seller registration failed"
)

// ========================================
// SELLER REGISTRATION SUCCESS MESSAGES
// ========================================
const (
	SELLER_REGISTERED_SUCCESS_MSG = "Seller registered successfully"
	SELLER_PROFILE_FETCHED_MSG    = "Seller profile fetched successfully"
	SELLER_PROFILE_UPDATED_MSG    = "Seller profile updated successfully"
)

// ========================================
// SELLER PROFILE ERROR CODES
// ========================================
const (
	SELLER_PROFILE_NOT_FOUND_CODE   = "SELLER_PROFILE_NOT_FOUND"
	SELLER_PROFILE_EXISTS_CODE      = "SELLER_PROFILE_EXISTS"
	SELLER_PROFILE_UPDATE_FAILED_CODE = "SELLER_PROFILE_UPDATE_FAILED"
)

// ========================================
// SELLER PROFILE ERROR MESSAGES
// ========================================
const (
	SELLER_PROFILE_NOT_FOUND_MSG   = "Seller profile not found"
	SELLER_PROFILE_EXISTS_MSG      = "Seller profile already exists"
	SELLER_PROFILE_UPDATE_FAILED_MSG = "Failed to update seller profile"
)
