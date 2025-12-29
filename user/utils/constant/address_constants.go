package constant

// ========================================
// ADDRESS ERROR CODES
// ========================================
const (
	ADDRESS_NOT_FOUND_CODE      = "ADDRESS_NOT_FOUND"
	DEFAULT_ADDRESS_EXISTS_CODE = "DEFAULT_ADDRESS_EXISTS"
	CANNOT_DELETE_DEFAULT_CODE  = "CANNOT_DELETE_DEFAULT"
)

// ========================================
// ADDRESS ERROR MESSAGES
// ========================================
const (
	ADDRESS_NOT_FOUND_MSG                  = "Address not found"
	DEFAULT_ADDRESS_EXISTS_MSG             = "Default address already exists"
	CANNOT_DELETE_ONLY_DEFAULT_ADDRESS_MSG = "cannot delete the only default address"
	CANNOT_DELETE_DEFAULT_MSG              = "Cannot delete default address. Please set another address as default first."
	INVALID_ADDRESS_ID_MSG                 = "Invalid address ID"
)

// ========================================
// ADDRESS OPERATION FAILURE MESSAGES
// ========================================
const (
	FAILED_TO_GET_ADDRESSES_MSG       = "Failed to get addresses"
	FAILED_TO_ADD_ADDRESS_MSG         = "Failed to add address"
	FAILED_TO_UPDATE_ADDRESS_MSG      = "Failed to update address"
	FAILED_TO_DELETE_ADDRESS_MSG      = "Failed to delete address"
	FAILED_TO_SET_DEFAULT_ADDRESS_MSG = "Failed to set default address"
)

// ========================================
// ADDRESS SUCCESS MESSAGES
// ========================================
const (
	ADDRESS_CREATED_MSG         = "Address created successfully"
	ADDRESS_RETRIEVED_MSG       = "Address retrieved successfully"
	ADDRESS_UPDATED_MSG         = "Address updated successfully"
	ADDRESS_DELETED_MSG         = "Address deleted successfully"
	ADDRESSES_RETRIEVED_MSG     = "Addresses retrieved successfully"
	DEFAULT_ADDRESS_UPDATED_MSG = "Default address updated successfully"
)

// ========================================
// ADDRESS FIELD NAMES
// ========================================
const (
	ADDRESS_FIELD_NAME   = "address"
	ADDRESSES_FIELD_NAME = "addresses"
)
