package utils

// Cache keys for product management
const (
	// Category cache keys
	CATEGORY_CACHE_KEY_PREFIX = "product:category:"
	CATEGORIES_CACHE_KEY      = "product:categories:all"
	CATEGORY_PARENT_CACHE_KEY = "product:categories:parent:"

	// Attribute cache keys
	ATTRIBUTE_CACHE_KEY_PREFIX = "product:attribute:"
	ATTRIBUTES_CACHE_KEY       = "product:attributes:all"

	// Product cache keys
	PRODUCT_CACHE_KEY_PREFIX = "product:product:"
	PRODUCTS_CACHE_KEY       = "product:products:list:"
	PRODUCT_SEARCH_CACHE_KEY = "product:search:"

	// Filter cache keys
	FILTERS_CACHE_KEY = "product:filters:"

	// Related products cache keys
	RELATED_PRODUCTS_CACHE_KEY = "product:related:"
)

// Cache TTL constants (in seconds)
const (
	// Product Lists: Cache for 5 minutes
	PRODUCT_LIST_CACHE_TTL = 300

	// Product Details: Cache for 15 minutes
	PRODUCT_DETAIL_CACHE_TTL = 900

	// Category Lists: Cache for 1 hour
	CATEGORY_LIST_CACHE_TTL = 3600

	// Filters: Cache for 30 minutes
	FILTERS_CACHE_TTL = 1800

	// Search Results: Cache for 10 minutes
	SEARCH_CACHE_TTL = 600

	// Related Products: Cache for 20 minutes
	RELATED_PRODUCTS_CACHE_TTL = 1200
)
