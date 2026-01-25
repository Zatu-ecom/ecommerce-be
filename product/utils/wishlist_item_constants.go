package utils

// Wishlist Item error codes
const (
	WISHLIST_ITEM_NOT_FOUND_CODE    = "WISHLIST_ITEM_NOT_FOUND"
	WISHLIST_ITEM_EXISTS_CODE       = "WISHLIST_ITEM_EXISTS"
	UNAUTHORIZED_WISHLIST_ITEM_CODE = "UNAUTHORIZED_WISHLIST_ITEM"
	SAME_WISHLIST_MOVE_CODE         = "SAME_WISHLIST_MOVE"
)

// Wishlist Item error messages
const (
	WISHLIST_ITEM_NOT_FOUND_MSG    = "Wishlist item not found"
	WISHLIST_ITEM_EXISTS_MSG       = "Item already exists in this wishlist"
	UNAUTHORIZED_WISHLIST_ITEM_MSG = "You do not have permission to access this wishlist item"
	INVALID_WISHLIST_ITEM_ID_MSG   = "Invalid wishlist item ID"
	SAME_WISHLIST_MOVE_MSG         = "Cannot move item to the same wishlist"
	INVALID_TARGET_WISHLIST_ID_MSG = "Invalid target wishlist ID"
)

// Wishlist Item success messages
const (
	WISHLIST_ITEM_ADDED_MSG        = "Item added to wishlist successfully"
	WISHLIST_ITEM_REMOVED_MSG      = "Item removed from wishlist successfully"
	WISHLIST_ITEM_MOVED_MSG        = "Item moved to another wishlist successfully"
	WISHLIST_ITEM_ADDED_TO_CART_MSG = "Item added to cart successfully"
)

// Wishlist Item field names
const (
	WISHLIST_ITEM_FIELD_NAME  = "wishlistItem"
	WISHLIST_ITEMS_FIELD_NAME = "wishlistItems"
)

// Wishlist Item failure messages
const (
	FAILED_TO_ADD_WISHLIST_ITEM_MSG    = "Failed to add item to wishlist"
	FAILED_TO_REMOVE_WISHLIST_ITEM_MSG = "Failed to remove item from wishlist"
	FAILED_TO_MOVE_WISHLIST_ITEM_MSG   = "Failed to move item to another wishlist"
	FAILED_TO_ADD_TO_CART_MSG          = "Failed to add item to cart"
)
