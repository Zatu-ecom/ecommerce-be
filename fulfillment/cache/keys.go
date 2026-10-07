// Package cache owns fulfillment-module cache key shapes and TTL bases.
// Only rates, tracking snapshots, shipment details, lists (versioned),
// catalog metadata, and pickup addresses are volatile-cacheable. Shipment
// authoritative state, COD amounts, credentials, and webhook payloads are
// never cached. All operations go through cachekit interfaces.
package cache

import (
	"strconv"
	"strings"
	"time"

	"ecommerce-be/common/cachekit"
)

// TTL bases (always wrapped in cachekit.JitteredTTL at the call site).
const (
	// RateTTL bases rate-shopping responses (volatile, fail-open).
	RateTTL = 90 * time.Second
	// TrackTTL bases live tracking snapshots (volatile, fail-open).
	TrackTTL = 45 * time.Second
	// ShipmentTTL bases shipment detail reads (volatile, fail-open).
	ShipmentTTL = 60 * time.Second
	// ListTTL bases versioned seller shipment lists (volatile, fail-open).
	ListTTL = 60 * time.Second
	// CatalogTTL bases courier catalog metadata (volatile, 12h like payment).
	CatalogTTL = 12 * time.Hour
	// CatalogNegativeTTL bases unknown-provider tombstones.
	CatalogNegativeTTL = 60 * time.Second
	// PickupsTTL bases per-seller pickup snapshots (volatile).
	PickupsTTL = time.Hour
)

// RateKey builds seller:{id}:fulfill:rate:{hash} for a rate-shopping request.
// The hash covers pickup + drop + weight + dims + cod (built by the caller).
func RateKey(sellerID uint, hash string) (string, error) {
	return cachekit.BuildSellerKey(sellerID, "fulfill:rate:"+sanitize(hash))
}

// TrackKey builds seller:{id}:fulfill:track:{awb} for a tracking snapshot.
func TrackKey(sellerID uint, awb string) (string, error) {
	return cachekit.BuildSellerKey(sellerID, "fulfill:track:"+sanitize(awb))
}

// ShipmentKey builds seller:{id}:fulfill:shipment:{id} for a shipment detail read.
func ShipmentKey(sellerID uint, shipmentID uint) (string, error) {
	return cachekit.BuildSellerKey(sellerID, "fulfill:shipment:"+strconv.FormatUint(uint64(shipmentID), 10))
}

// ListVersionKey is the durable list-generation counter for a seller.
// Lists retire by bumping this key (cachekit.BumpVersion); prefix deletes
// on the request path are forbidden.
func ListVersionKey(sellerID uint) (string, error) {
	return cachekit.BuildSellerKey(sellerID, "fulfill:list:ver")
}

// ListKey builds the versioned list key seller:{id}:fulfill:list:v{ver}:{hash}.
func ListKey(sellerID uint, version, hash string) (string, error) {
	return cachekit.BuildSellerKey(sellerID, "fulfill:list:v"+version+":"+sanitize(hash))
}

// CatalogKey builds courier:catalog:{code} (platform allowlist, validate at use).
func CatalogKey(code string) string {
	return "courier:catalog:" + sanitize(code)
}

// CatalogAllKey builds courier:catalog:all:v{ver} for the versioned code list.
func CatalogAllKey(version string) string {
	return "courier:catalog:all:v" + sanitize(version)
}

// IdempotencyKey builds seller:{id}:fulfill:init:idem:{sha256} (durable, 24h).
// Seller-scoped: the DB constraint is UNIQUE(seller_id, idempotency_key) and
// this key is only the fast path. sha must be 64-hex (hash of the raw header).
func IdempotencyKey(sellerID uint, sha256Hex string) (string, error) {
	return cachekit.BuildSellerKey(sellerID, "fulfill:init:idem:"+sanitize(sha256Hex))
}

// RateLimitKey builds fulfill:rate:limit:{seller} (durable Lua INCR+EXPIRE 60s).
func RateLimitKey(sellerID uint) string {
	return "fulfill:rate:limit:" + strconv.FormatUint(uint64(sellerID), 10)
}

// PickupsKey builds seller:{id}:fulfill:pickups for the pickup snapshot.
func PickupsKey(sellerID uint) (string, error) {
	return cachekit.BuildSellerKey(sellerID, "fulfill:pickups")
}

// sanitize strips characters that would break key validation or SCAN safety.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == ':':
			return r
		default:
			return '-'
		}
	}, s)
}
