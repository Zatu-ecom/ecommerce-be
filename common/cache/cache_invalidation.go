package cache

import (
	"fmt"

	"ecommerce-be/common/constants"
)

/****************************************************
*			Cache invalidation functions			*
*****************************************************/

// InvalidateSellerSubscriptionCache invalidates the subscription cache for a seller
func InvalidateSellerSubscriptionCache(sellerID uint) error {
	cacheKey := fmt.Sprintf("%s%d", constants.SELLER_SUBSCRIPTION_CACHE_KEY, sellerID)
	return DelKey(cacheKey)
}

// InvalidateSellerDetailsCache invalidates the seller details cache for a seller
func InvalidateSellerDetailsCache(sellerID uint) error {
	cacheKey := fmt.Sprintf("%s%d", constants.SELLER_DETAILS_CACHE_KEY, sellerID)
	return DelKey(cacheKey)
}

// InvalidateAllSellerCache invalidates both subscription and details cache for a seller
func InvalidateAllSellerCache(sellerID uint) error {
	if err := InvalidateSellerSubscriptionCache(sellerID); err != nil {
		return err
	}
	return InvalidateSellerDetailsCache(sellerID)
}
