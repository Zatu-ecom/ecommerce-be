package query

// Variant aggregation queries
const (
	// VARIANT_PRICE_AGGREGATION_QUERY gets min/max price and allow_purchase for a product.
	// Images are no longer stored on product_variant; use variant_media for asset resolution.
	// Parameters: productID (for WHERE clause)
	VARIANT_PRICE_AGGREGATION_QUERY = `
		MIN(price) as min_price,
		MAX(price) as max_price,
		BOOL_OR(allow_purchase) as allow_purchase
	`

	// VARIANT_ALL_FLAGS_AGGREGATION_QUERY aggregates flags and default price across all variants.
	VARIANT_ALL_FLAGS_AGGREGATION_QUERY = `
		COALESCE(MAX(CASE WHEN is_default THEN price END), MIN(price)) as default_price,
		BOOL_OR(allow_purchase) as allow_purchase,
		BOOL_OR(is_popular) as is_popular
	`

	// VARIANT_BATCH_ALL_FLAGS_AGGREGATION_QUERY aggregates flags per product (batch list path).
	VARIANT_BATCH_ALL_FLAGS_AGGREGATION_QUERY = `
		product_id,
		COALESCE(MAX(CASE WHEN is_default THEN price END), MIN(price)) as default_price,
		BOOL_OR(allow_purchase) as allow_purchase,
		BOOL_OR(is_popular) as is_popular
	`

	// VARIANT_OPTION_DERIVED_PRICE_AGGREGATION_QUERY min/max price for option-derived variants only.
	VARIANT_OPTION_DERIVED_PRICE_AGGREGATION_QUERY = `
		MIN(pv.price) as min_price,
		MAX(pv.price) as max_price
	`

	// VARIANT_BATCH_OPTION_DERIVED_PRICE_AGGREGATION_QUERY batch option-derived min/max per product.
	VARIANT_BATCH_OPTION_DERIVED_PRICE_AGGREGATION_QUERY = `
		pv.product_id,
		MIN(pv.price) as min_price,
		MAX(pv.price) as max_price
	`

	// WISHLIST_CHECK_SINGLE_PRODUCT checks if any variant of a product is in user's wishlist
	// Parameters: productID, userID
	WISHLIST_CHECK_SINGLE_PRODUCT = `
		SELECT EXISTS (
			SELECT 1 FROM wishlist_item wi
			INNER JOIN wishlist w ON w.id = wi.wishlist_id
			INNER JOIN product_variant pv ON pv.id = wi.variant_id
			WHERE pv.product_id = ?
			  AND w.user_id = ?
		)
	`

	// WISHLIST_CHECK_MULTIPLE_PRODUCTS gets product IDs that have wishlisted variants for a user
	// Parameters: productIDs (array), userID
	WISHLIST_CHECK_MULTIPLE_PRODUCTS = `
		SELECT DISTINCT pv.product_id
		FROM wishlist_item wi
		INNER JOIN wishlist w ON w.id = wi.wishlist_id
		INNER JOIN product_variant pv ON pv.id = wi.variant_id
		WHERE pv.product_id IN ?
		  AND w.user_id = ?
	`

	// WISHLIST_CHECK_SINGLE_VARIANT checks if a specific variant is in user's wishlist
	// Parameters: variantID, userID
	WISHLIST_CHECK_SINGLE_VARIANT = `
		SELECT EXISTS (
			SELECT 1 FROM wishlist_item wi
			INNER JOIN wishlist w ON w.id = wi.wishlist_id
			WHERE wi.variant_id = ?
			  AND w.user_id = ?
		)
	`
)
