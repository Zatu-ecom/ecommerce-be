package cache

import "context"

// CacheAware is implemented by product-module write services that accept the
// detail cache strategy for post-commit invalidation. Factories assert this
// interface instead of concrete types, keeping wiring decoupled.
type CacheAware interface {
	SetProductCache(*ProductCache)
}

// CategoryCacheAware is implemented by category-scope write services that
// accept the reference-data strategy for post-commit invalidation.
type CategoryCacheAware interface {
	SetCategoryCache(*CategoryCache)
}

// VariantLister lists all variant IDs of a product. Provided by wiring
// (factory closure over the variant repository) so strategies never touch
// repositories directly.
type VariantLister func(ctx context.Context, productID uint) ([]uint, error)
