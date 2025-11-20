package constants

// Role-based authentication constants
const (
	// Role-based authentication messages
	INSUFFICIENT_PERMISSIONS_MSG     = "Insufficient permissions"
	SELLER_SUBSCRIPTION_INACTIVE_MSG = "Seller subscription is inactive"
	SELLER_NOT_VERIFIED_MSG          = "Seller account is not verified"
	INVALID_SELLER_MSG               = "Invalid seller information"
	ROLE_NOT_FOUND_MSG               = "User Role not found"

	// Customer-specific messages (when associated seller has issues)
	CUSTOMER_NO_SELLER_MSG                        = "Customer account must be associated with a seller"
	SELLER_SUBSCRIPTION_INACTIVE_FOR_CUSTOMER_MSG = "Associated seller's subscription is inactive. Please contact your seller."
	SELLER_NOT_VERIFIED_FOR_CUSTOMER_MSG          = "Associated seller account is not verified. Please contact your seller."
	SELLER_INACTIVE_FOR_CUSTOMER_MSG              = "Associated seller account is inactive. Please contact your seller."

	// Role-based error codes
	INSUFFICIENT_PERMISSIONS_CODE     = "INSUFFICIENT_PERMISSIONS"
	SELLER_SUBSCRIPTION_INACTIVE_CODE = "SELLER_SUBSCRIPTION_INACTIVE"
	SELLER_NOT_VERIFIED_CODE          = "SELLER_NOT_VERIFIED"
	INVALID_SELLER_CODE               = "INVALID_SELLER"
	ROLE_NOT_FOUND_CODE               = "ROLE_NOT_FOUND"

	// Customer-specific error codes
	CUSTOMER_NO_SELLER_CODE = "CUSTOMER_NO_SELLER"

	// Role levels (lower number = higher authority)
	ADMIN_ROLE_LEVEL    uint = 1
	SELLER_ROLE_LEVEL   uint = 2
	CUSTOMER_ROLE_LEVEL uint = 3

	// Role names
	ADMIN_ROLE_NAME    = "ADMIN"
	SELLER_ROLE_NAME   = "SELLER"
	CUSTOMER_ROLE_NAME = "CUSTOMER"

	// Subscription statuses
	SUBSCRIPTION_STATUS_ACTIVE    = "active"
	SUBSCRIPTION_STATUS_PENDING   = "pending"
	SUBSCRIPTION_STATUS_EXPIRED   = "expired"
	SUBSCRIPTION_STATUS_CANCELLED = "cancelled"
)
