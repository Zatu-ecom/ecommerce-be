package constant

// Cart handler messages (short error/success strings)
const (
	SELLER_CONTEXT_REQUIRED_MSG    = "Seller context required"
	FAILED_TO_ADD_ITEM_TO_CART_MSG = "Failed to add item to cart"
	FAILED_TO_GET_CART_MSG         = "Failed to get cart"
	FAILED_TO_DELETE_CART_MSG      = "Failed to delete cart"
	ITEM_ADDED_TO_CART_MSG         = "Item added to cart"
	CART_FETCHED_MSG               = "Cart fetched successfully"
	CART_DELETED_MSG               = "Cart deleted successfully"
)

// Cart repository — error messages returned to API clients (not log lines)
const (
	CART_NOT_FOUND_MSG               = "Cart not found"
	FAILED_TO_FETCH_CART_MSG         = "Failed to fetch cart"
	FAILED_TO_INSERT_CART_RECORD_MSG = "Failed to insert record"
	FAILED_TO_FETCH_CART_ITEM_MSG    = "Failed to fetch cart item"
	FAILED_TO_FETCH_CART_ITEMS_MSG   = "Failed to fetch cart items"
	FAILED_TO_UPDATE_CART_RECORD_MSG = "Failed to update record"
	FAILED_TO_DELETE_CART_RECORD_MSG = "Failed to delete record"
)

// Cart merge messages
const (
	CART_MERGED_MSG            = "Guest cart merged successfully"
	FAILED_TO_MERGE_CART_MSG   = "Failed to merge guest cart"
	GUEST_NOT_FOUND_MSG        = "No guest cart found"
	DEVICE_CONTEXT_MISSING_MSG = "Device context is required"
)

// Cart coupon (customer apply/remove) messages
const (
	COUPON_APPLIED_MSG                   = "Coupon applied successfully"
	COUPON_REMOVED_MSG                   = "Coupon removed successfully"
	ALL_COUPONS_REMOVED_MSG              = "All coupons removed successfully"
	AVAILABLE_COUPONS_RETRIEVED_MSG      = "Available coupons retrieved successfully"
	FAILED_TO_APPLY_COUPON_MSG           = "Failed to apply coupon"
	FAILED_TO_REMOVE_COUPON_MSG          = "Failed to remove coupon"
	FAILED_TO_LIST_AVAILABLE_COUPONS_MSG = "Failed to list available coupons"
	COUPON_APPLY_RATE_LIMITED_MSG        = "Too many coupon apply attempts"
)
