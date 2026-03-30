package constant

// ========================================
// CURRENCY ERROR CODES
// ========================================
const (
	CURRENCY_NOT_FOUND_CODE      = "CURRENCY_NOT_FOUND"
	DUPLICATE_CURRENCY_CODE_CODE = "DUPLICATE_CURRENCY_CODE"
	CURRENCY_INACTIVE_CODE       = "CURRENCY_INACTIVE"
	CURRENCY_HAS_REFERENCES_CODE = "CURRENCY_HAS_REFERENCES"
)

// ========================================
// CURRENCY ERROR MESSAGES
// ========================================
const (
	CURRENCY_NOT_FOUND_MSG      = "Currency not found"
	DUPLICATE_CURRENCY_CODE_MSG = "Currency with this code already exists"
	CURRENCY_INACTIVE_MSG       = "Currency is inactive"
	CURRENCY_HAS_REFERENCES_MSG = "Currency cannot be deleted as it has references"
)

// ========================================
// CURRENCY OPERATION FAILURE MESSAGES
// ========================================
const (
	FAILED_TO_CREATE_CURRENCY_MSG = "Failed to create currency"
	FAILED_TO_UPDATE_CURRENCY_MSG = "Failed to update currency"
	FAILED_TO_DELETE_CURRENCY_MSG = "Failed to delete currency"
	FAILED_TO_GET_CURRENCY_MSG    = "Failed to get currency"
	FAILED_TO_LIST_CURRENCIES_MSG = "Failed to list currencies"
	INVALID_CURRENCY_ID_MSG       = "Invalid currency ID"
)

// ========================================
// CURRENCY SUCCESS MESSAGES
// ========================================
const (
	CURRENCY_CREATED_MSG   = "Currency created successfully"
	CURRENCY_UPDATED_MSG   = "Currency updated successfully"
	CURRENCY_DELETED_MSG   = "Currency deleted successfully"
	CURRENCY_RETRIEVED_MSG = "Currency retrieved successfully"
	CURRENCIES_LISTED_MSG  = "Currencies retrieved successfully"
)

// ========================================
// CURRENCY FIELD NAMES
// ========================================
const (
	CURRENCY_FIELD_NAME   = "currency"
	CURRENCIES_FIELD_NAME = "currencies"
)
