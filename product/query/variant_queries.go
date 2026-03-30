package query

// Variant aggregation queries
const (
	// VARIANT_PRICE_AGGREGATION_QUERY gets min/max price, allow_purchase, and main image for a product
	// Parameters: productID (for subquery), productID (for WHERE clause)
	VARIANT_PRICE_AGGREGATION_QUERY = `
		MIN(price) as min_price,
		MAX(price) as max_price,
		BOOL_OR(allow_purchase) as allow_purchase,
		(SELECT images FROM product_variant WHERE product_id = ? AND is_default = true AND images IS NOT NULL AND images != '{}' LIMIT 1) as main_image
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
