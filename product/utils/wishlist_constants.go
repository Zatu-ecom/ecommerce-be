package utils

// Wishlist error codes
const (
	WISHLIST_NOT_FOUND_CODE    = "WISHLIST_NOT_FOUND"
	WISHLIST_NAME_EXISTS_CODE  = "WISHLIST_NAME_EXISTS"
	MAX_WISHLISTS_REACHED_CODE = "MAX_WISHLISTS_REACHED"
	UNAUTHORIZED_WISHLIST_CODE = "UNAUTHORIZED_WISHLIST"
	CANNOT_DELETE_DEFAULT_CODE = "CANNOT_DELETE_DEFAULT"
)

// Wishlist error messages
const (
	WISHLIST_NOT_FOUND_MSG    = "Wishlist not found"
	WISHLIST_NAME_EXISTS_MSG  = "Wishlist with this name already exists"
	MAX_WISHLISTS_REACHED_MSG = "Maximum number of wishlists reached"
	UNAUTHORIZED_WISHLIST_MSG = "You do not have permission to access this wishlist"
	CANNOT_DELETE_DEFAULT_MSG = "Cannot delete the default wishlist"
	INVALID_WISHLIST_ID_MSG   = "Invalid wishlist ID"
)

// Wishlist success messages
const (
	WISHLISTS_RETRIEVED_MSG = "Wishlists retrieved successfully"
	WISHLIST_CREATED_MSG    = "Wishlist created successfully"
	WISHLIST_RETRIEVED_MSG  = "Wishlist retrieved successfully"
	WISHLIST_UPDATED_MSG    = "Wishlist updated successfully"
	WISHLIST_DELETED_MSG    = "Wishlist deleted successfully"
)

// Wishlist field names
const (
	WISHLIST_FIELD_NAME  = "wishlist"
	WISHLISTS_FIELD_NAME = "wishlists"
)

// Wishlist failure messages
const (
	FAILED_TO_GET_WISHLISTS_MSG   = "Failed to retrieve wishlists"
	FAILED_TO_CREATE_WISHLIST_MSG = "Failed to create wishlist"
	FAILED_TO_GET_WISHLIST_MSG    = "Failed to retrieve wishlist"
	FAILED_TO_UPDATE_WISHLIST_MSG = "Failed to update wishlist"
	FAILED_TO_DELETE_WISHLIST_MSG = "Failed to delete wishlist"
)

// Wishlist limits - these are default values
// Actual values are loaded from config (environment variables)
// Use config.Get().Wishlist.MaxWishlistsPerUser, etc. instead
const (
	DEFAULT_WISHLIST_NAME = "My Wishlist" // Kept for backward compatibility
)
