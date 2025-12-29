package constant

// ========================================
// COUNTRY ERROR CODES
// ========================================
const (
	COUNTRY_NOT_FOUND_CODE      = "COUNTRY_NOT_FOUND"
	DUPLICATE_COUNTRY_CODE_CODE = "DUPLICATE_COUNTRY_CODE"
	COUNTRY_INACTIVE_CODE       = "COUNTRY_INACTIVE"
	COUNTRY_HAS_REFERENCES_CODE = "COUNTRY_HAS_REFERENCES"
)

// ========================================
// COUNTRY ERROR MESSAGES
// ========================================
const (
	COUNTRY_NOT_FOUND_MSG      = "Country not found"
	DUPLICATE_COUNTRY_CODE_MSG = "Country with this code already exists"
	COUNTRY_INACTIVE_MSG       = "Country is inactive"
	COUNTRY_HAS_REFERENCES_MSG = "Country cannot be deleted as it has references"
)

// ========================================
// COUNTRY OPERATION FAILURE MESSAGES
// ========================================
const (
	FAILED_TO_CREATE_COUNTRY_MSG = "Failed to create country"
	FAILED_TO_UPDATE_COUNTRY_MSG = "Failed to update country"
	FAILED_TO_DELETE_COUNTRY_MSG = "Failed to delete country"
	FAILED_TO_GET_COUNTRY_MSG    = "Failed to get country"
	FAILED_TO_LIST_COUNTRIES_MSG = "Failed to list countries"
	INVALID_COUNTRY_ID_MSG       = "Invalid country ID"
)

// ========================================
// COUNTRY SUCCESS MESSAGES
// ========================================
const (
	COUNTRY_CREATED_MSG   = "Country created successfully"
	COUNTRY_UPDATED_MSG   = "Country updated successfully"
	COUNTRY_DELETED_MSG   = "Country deleted successfully"
	COUNTRY_RETRIEVED_MSG = "Country retrieved successfully"
	COUNTRIES_LISTED_MSG  = "Countries retrieved successfully"
)

// ========================================
// COUNTRY FIELD NAMES
// ========================================
const (
	COUNTRY_FIELD_NAME   = "country"
	COUNTRIES_FIELD_NAME = "countries"
)
