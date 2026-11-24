package cache

import (
	"context"

	"ecommerce-be/common/cachekit"
)

// InvalidateSellerValidation deletes the live seller-validation entry
// (seller:{id}:seller:complete) from the VOLATILE cachekit role — the same
// store ValidateSellerCompleteCached fills. Call AFTER commit from every
// writer that affects the verdict: profile/active-flag changes,
// subscription/plan writes (no HTTP subscription module exists today — hook
// the writer, including non-HTTP writers), and user-delete purge paths.
//
// Exact key, post-commit; backend failures are returned (callers log and
// continue: the 5m subscription-capped TTL bounds any miss). A nil cache
// means caching is unwired: nothing to clear.
func InvalidateSellerValidation(ctx context.Context, sellerID uint) error {
	cacheKey, err := cachekit.BuildSellerKey(sellerID, "seller", "complete")
	if err != nil {
		return err
	}
	c := cachekit.DefaultCache()
	if c == nil {
		return nil
	}
	return c.Del(ctx, cacheKey)
}
