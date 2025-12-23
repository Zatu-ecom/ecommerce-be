package constants

import "time"

// Cache-related constants
const (
	// Cache keys and expiration times
	SELLER_SUBSCRIPTION_CACHE_KEY = "seller_subscription:"
	SELLER_DETAILS_CACHE_KEY      = "seller_details:"
	SELLER_COMPLETE_CACHE_KEY     = "seller_complete:" // OPTIMIZED: Complete seller validation data
	SELLER_CACHE_EXPIRATION       = time.Minute * 15   // 15 minutes
	SELLER_CACHE_SHORT_EXPIRATION = time.Minute * 2    // 2 minutes for failed validations

	// Inventory Reservation cache keys
	// Key format: reservation:expiry:{referenceId}
	RESERVATION_EXPIRY_KEY_PREFIX = "reservation:expiry:"
)
