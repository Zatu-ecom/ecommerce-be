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
)
