package cachekit

import (
	"context"
	"time"
)

// AdmissionMarkerTTL bases P2 list admission markers (~60s, jittered at
// write like every SET so marker expiries cannot synchronize).
const AdmissionMarkerTTL = 60 * time.Second

// AdmissionMarkerKey mints the tiny first-sighting marker for a canonical
// list-query hash: seller:{id}:seen:{hash} (pre-spec §4.4). Seller-prefixed
// so the key builder accepts it with no allowlist change.
func AdmissionMarkerKey(sellerID uint, hash string) (string, error) {
	return BuildSellerKey(sellerID, "seen", hash)
}

// AdmitList gates combinatorial query caching on a repeat sighting
// (pre-spec §4.4). First sight claims the marker and returns admit=false:
// fetch from DB, return rows, NO full SET. A repeat within the marker window
// returns admit=true: the caller may async-SET the full response (1–2m
// jittered). Singleflight coalesces identical keys; admission is the only
// defense against random one-off floods (markers cost one cheap SETNX, no
// big SET, no LRU pollution).
func AdmitList(ctx context.Context, c Cache, sellerID uint, hash string) (bool, error) {
	key, err := AdmissionMarkerKey(sellerID, hash)
	if err != nil {
		return false, err
	}
	claimed, err := c.SetNX(ctx, key, []byte("1"), JitteredTTL(AdmissionMarkerTTL))
	if err != nil {
		// Backend down: fail open to DB without caching (same as first
		// sight, but the caller must not treat the error as a repeat).
		return false, err
	}
	return !claimed, nil
}
