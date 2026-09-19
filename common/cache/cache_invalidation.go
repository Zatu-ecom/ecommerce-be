package cache

import (
	"context"
	"time"

	"ecommerce-be/common/cachekit"
)

/****************************************************
*			Cache invalidation functions			*
*****************************************************/

// sellerCompleteKey builds the live seller-validation key written by
// ValidateSellerCompleteCached (seller:{id}:seller:complete).
func sellerCompleteKey(sellerID uint) (string, error) {
	return cachekit.BuildSellerKey(sellerID, "seller", "complete")
}

// invalidateCompleteCacheCtx deletes the live seller-validation entry from
// the VOLATILE cachekit role — the same store ValidateSellerCompleteCached
// fills. The legacy single-Redis Del targeted a different backend
// (REDIS_ADDR vs CACHE_ADDR) and silently missed; exact key, post-commit,
// failures counted by the client and never returned (the 5m TTL bounds any
// miss). A nil default cache means caching is unwired: nothing to clear.
func invalidateCompleteCacheCtx(ctx context.Context, sellerID uint) error {
	cacheKey, err := sellerCompleteKey(sellerID)
	if err != nil {
		return err
	}
	c := cachekit.DefaultCache()
	if c == nil {
		return nil
	}
	return c.Del(ctx, cacheKey)
}

// invalidateCompleteCache is the legacy no-context wrapper (10s bound).
// New writers MUST take a request context and call the Ctx variant so
// correlation IDs flow into cache logs.
func invalidateCompleteCache(sellerID uint) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return invalidateCompleteCacheCtx(ctx, sellerID)
}

// InvalidateSellerSubscriptionCacheCtx invalidates cached seller validation
// after a subscription/plan change. Every writer of `subscription`, `plan`,
// or `user.is_active` MUST call this AFTER commit (there is no subscription
// HTTP module today — hook the writer, including non-HTTP writers).
func InvalidateSellerSubscriptionCacheCtx(ctx context.Context, sellerID uint) error {
	return invalidateCompleteCacheCtx(ctx, sellerID)
}

// InvalidateSellerSubscriptionCache invalidates cached seller validation after
// a subscription/plan change. Kept for existing callers; targets the live key.
func InvalidateSellerSubscriptionCache(sellerID uint) error {
	return invalidateCompleteCache(sellerID)
}

// InvalidateSellerDetailsCacheCtx invalidates cached seller validation after
// a profile/active-flag change. Targets the live volatile key.
func InvalidateSellerDetailsCacheCtx(ctx context.Context, sellerID uint) error {
	return invalidateCompleteCacheCtx(ctx, sellerID)
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

// InvalidateAllSellerCacheCtx invalidates all cached seller validation
// entries for a seller (single live key).
func InvalidateAllSellerCacheCtx(ctx context.Context, sellerID uint) error {
	return invalidateCompleteCacheCtx(ctx, sellerID)
}
