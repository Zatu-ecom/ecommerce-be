package constant

// ========================================
// COUNTRY-CURRENCY ERROR CODES
// ========================================
const (
	COUNTRY_CURRENCY_NOT_FOUND_CODE  = "COUNTRY_CURRENCY_NOT_FOUND"
	COUNTRY_CURRENCY_EXISTS_CODE     = "COUNTRY_CURRENCY_EXISTS"
	PRIMARY_CURRENCY_REQUIRED_CODE   = "PRIMARY_CURRENCY_REQUIRED"
	MULTIPLE_PRIMARY_CURRENCIES_CODE = "MULTIPLE_PRIMARY_CURRENCIES"
)

// ========================================
// COUNTRY-CURRENCY ERROR MESSAGES
// ========================================
const (
	COUNTRY_CURRENCY_NOT_FOUND_MSG  = "Country-currency mapping not found"
	COUNTRY_CURRENCY_EXISTS_MSG     = "This currency is already mapped to this country"
	PRIMARY_CURRENCY_REQUIRED_MSG   = "At least one primary currency is required for a country"
	MULTIPLE_PRIMARY_CURRENCIES_MSG = "Only one currency can be marked as primary per country"
)

// ========================================
// COUNTRY-CURRENCY OPERATION FAILURE MESSAGES
// ========================================
const (
	FAILED_TO_ADD_COUNTRY_CURRENCY_MSG      = "Failed to add currency to country"
	FAILED_TO_UPDATE_COUNTRY_CURRENCY_MSG   = "Failed to update country-currency mapping"
	FAILED_TO_REMOVE_COUNTRY_CURRENCY_MSG   = "Failed to remove currency from country"
	FAILED_TO_LIST_COUNTRY_CURRENCY_MSG     = "Failed to list country currencies"
	FAILED_TO_BULK_ADD_COUNTRY_CURRENCY_MSG = "Failed to bulk add currencies to country"
)

// ========================================
// COUNTRY-CURRENCY SUCCESS MESSAGES
// ========================================
const (
	COUNTRY_CURRENCY_ADDED_MSG        = "Currency added to country successfully"
	COUNTRY_CURRENCY_UPDATED_MSG      = "Country-currency mapping updated successfully"
	COUNTRY_CURRENCY_REMOVED_MSG      = "Currency removed from country successfully"
	COUNTRY_CURRENCIES_LISTED_MSG     = "Country currencies retrieved successfully"
	COUNTRY_CURRENCIES_BULK_ADDED_MSG = "Currencies added to country successfully"
)
