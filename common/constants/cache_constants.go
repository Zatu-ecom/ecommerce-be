package constants

import "time"

// Cache-related constants.
//
// Seller-validation keys are built by cachekit (seller:{id}:seller:complete);
// invalidation funnels through cache.Invalidate*SellerCache. Do not
// reintroduce per-concern key prefixes: they were never written and their
// invalidators silently missed the live key (012 C4).
const (
	SELLER_CACHE_EXPIRATION       = time.Minute * 15   // 15 minutes (legacy default; strategies set their own TTLs)
	SELLER_CACHE_SHORT_EXPIRATION = time.Minute * 2    // 2 minutes for failed validations
)
