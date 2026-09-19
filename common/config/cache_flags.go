package config

import (
	"os"
	"strconv"
)

// CacheFlags gates every cacheable area independently. All flags default off
// except SetWrites: with everything off the system behaves exactly as before
// caching existed (fail-open to the database). See 012 pre-spec §8.5.
// Flags never disable durable KV behavior (scheduler, SETNX, denylist, limiter).
type CacheFlags struct {
	// Enabled is the global kill switch; off turns all domain Get/Set into no-ops.
	// Env: CACHE_ENABLED (default false).
	Enabled bool
	// SellerValidation caches seller_complete (5m). Env: CACHE_SELLER_VALIDATION.
	SellerValidation bool
	// Currency caches seller/user currency resolution (1h). Env: CACHE_CURRENCY.
	Currency bool
	// ProductDetail caches product + variant DTOs. Env: CACHE_PRODUCT_DETAIL.
	ProductDetail bool
	// Category caches category/attribute reads. Env: CACHE_CATEGORY.
	Category bool
	// Geo caches public country/currency reference. Env: CACHE_GEO.
	Geo bool
	// SellerSettings caches per-seller settings (private). Env: CACHE_SELLER_SETTINGS.
	SellerSettings bool
	// GatewayCatalog caches the gateway catalog slice (seller overlay stays live).
	// Env: CACHE_GATEWAY_CATALOG.
	GatewayCatalog bool
	// FileRef caches storage providers + adapter schemas. Env: CACHE_FILE_REF.
	FileRef bool
	// InventoryAvail caches internal availability reads (requires atomic guard).
	// Env: CACHE_INVENTORY_AVAIL.
	InventoryAvail bool
	// ProductList caches versioned list/search results with admission.
	// Env: CACHE_PRODUCT_LIST.
	ProductList bool
	// SetWrites is the manual write-shed lever (default ON); off serves GETs
	// while skipping all domain SETs. Env: CACHE_SET_WRITES (default true).
	SetWrites bool
}

// loadCacheFlags loads per-area cache flags from environment variables.
func loadCacheFlags() CacheFlags {
	return CacheFlags{
		Enabled:          getEnvAsBoolOrDefault("CACHE_ENABLED", false),
		SellerValidation: getEnvAsBoolOrDefault("CACHE_SELLER_VALIDATION", false),
		Currency:         getEnvAsBoolOrDefault("CACHE_CURRENCY", false),
		ProductDetail:    getEnvAsBoolOrDefault("CACHE_PRODUCT_DETAIL", false),
		Category:         getEnvAsBoolOrDefault("CACHE_CATEGORY", false),
		Geo:              getEnvAsBoolOrDefault("CACHE_GEO", false),
		SellerSettings:   getEnvAsBoolOrDefault("CACHE_SELLER_SETTINGS", false),
		GatewayCatalog:   getEnvAsBoolOrDefault("CACHE_GATEWAY_CATALOG", false),
		FileRef:          getEnvAsBoolOrDefault("CACHE_FILE_REF", false),
		InventoryAvail:   getEnvAsBoolOrDefault("CACHE_INVENTORY_AVAIL", false),
		ProductList:      getEnvAsBoolOrDefault("CACHE_PRODUCT_LIST", false),
		SetWrites:        getEnvAsBoolOrDefault("CACHE_SET_WRITES", true),
	}
}

// getEnvAsBoolOrDefault reads an environment variable as bool with a default.
// Accepts 1/t/true and 0/f/false (case-insensitive); anything else yields default.
func getEnvAsBoolOrDefault(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}
