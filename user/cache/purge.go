package cache

import (
	"context"
	"errors"

	"ecommerce-be/common/cachekit"
)

// PurgeSellerCache removes every volatile entry keyed to a seller (and the
// given users) with exact deletes. Call AFTER a hard account delete commits.
//
// Exact keys only — prefix globs are forbidden on the request path
// (pre-spec §5.4) — and no tombstones: a tombstone would retain the deleted
// user's bytes under the same key. Covered keys (data-model §volatile role):
//   - seller:{id}:seller:complete (validation verdict)
//   - seller:{id}:currency:default (seller default resolution)
//   - seller:{id}:currency:user:{uid} (per-user preference, per userID)
//   - seller:{id}:settings (private seller settings)
//
// A nil cache means caching is unwired: nothing to purge, nil error. Per-key
// failures are joined (best-effort across keys) so one sick key never hides
// another; the short TTLs bound any miss.
func PurgeSellerCache(ctx context.Context, c cachekit.Cache, sellerID uint, userIDs []uint) error {
	if c == nil {
		return nil
	}
	keys := make([]string, 0, 3+len(userIDs))
	for _, segments := range [][]string{
		{"seller", "complete"},
		{"currency", "default"},
		{"settings"},
	} {
		if k, err := cachekit.BuildSellerKey(sellerID, segments...); err == nil {
			keys = append(keys, k)
		}
	}
	for _, uid := range userIDs {
		if k, err := cachekit.BuildSellerKey(sellerID, "currency", "user", uintToString(uid)); err == nil {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	if err := c.Del(ctx, keys...); err != nil {
		return errors.Join(err)
	}
	return nil
}
