package cachekit

import (
	"context"
	"strconv"
)

// VersionKey mints the durable counter key for a seller-scoped resource
// family, e.g. VersionKey(7, "productlist") → seller:7:productlist:ver.
// Version keys live on DURABLE kv (noeviction): an evicted counter restarts
// at 1 and resurrects orphaned old-version list keys as fresh — a silent
// staleness-bound break (pre-spec §5.4).
func VersionKey(sellerID uint, family string) (string, error) {
	return BuildSellerKey(sellerID, family+":ver")
}

// ParseVersion decodes a counter value read back from the backend.
func ParseVersion(raw []byte) (uint64, error) {
	return strconv.ParseUint(string(raw), 10, 64)
}

// BumpVersion atomically advances a durable version counter.
func BumpVersion(ctx context.Context, d Durable, versionKey string) (uint64, error) {
	return d.Incr(ctx, versionKey)
}
