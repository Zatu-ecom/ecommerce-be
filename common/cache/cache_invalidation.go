package cache

import (
	"ecommerce-be/common/cachekit"
)

/****************************************************
*			Cache invalidation functions			*
*****************************************************/

// invalidateCompleteCache deletes the live seller-validation entry written by
// ValidateSellerCompleteCached (key: seller:{id}:seller:complete).
func invalidateCompleteCache(sellerID uint) error {
	cacheKey, err := cachekit.BuildSellerKey(sellerID, "seller", "complete")
	if err != nil {
		return err
	}
	return Del(cacheKey)
}

// InvalidateSellerSubscriptionCache invalidates cached seller validation after
// a subscription/plan change. Kept for existing callers; targets the live key.
func InvalidateSellerSubscriptionCache(sellerID uint) error {
	return invalidateCompleteCache(sellerID)
}

// InvalidateSellerDetailsCache invalidates cached seller validation after a
// profile/active-flag change. Kept for existing callers; targets the live key.
func InvalidateSellerDetailsCache(sellerID uint) error {
	return invalidateCompleteCache(sellerID)
}

// InvalidateAllSellerCache invalidates all cached seller validation entries
// for a seller (single live key).
func InvalidateAllSellerCache(sellerID uint) error {
	return invalidateCompleteCache(sellerID)
}
