package query

// Filter subquery constants for product filtering
// These are reusable WHERE clause fragments used in FindAll and Search operations
const (
	// FILTER_PRICE_MIN_SUBQUERY filters products by minimum variant price
	FILTER_PRICE_MIN_SUBQUERY = `EXISTS (
		SELECT 1 FROM product_variant pv 
		WHERE pv.product_id = product.id 
		AND pv.price >= ?
	)`

	// FILTER_PRICE_MAX_SUBQUERY filters products by maximum variant price
	FILTER_PRICE_MAX_SUBQUERY = `EXISTS (
		SELECT 1 FROM product_variant pv 
		WHERE pv.product_id = product.id 
		AND pv.price <= ?
	)`

	// FILTER_IN_STOCK_SUBQUERY filters products that have at least one in-stock variant
	FILTER_IN_STOCK_SUBQUERY = `EXISTS (
		SELECT 1 FROM product_variant pv 
		WHERE pv.product_id = product.id 
		AND pv.in_stock = true 
		AND pv.stock > 0
	)`

	// FILTER_OUT_OF_STOCK_SUBQUERY filters products with no in-stock variants
	FILTER_OUT_OF_STOCK_SUBQUERY = `NOT EXISTS (
		SELECT 1 FROM product_variant pv 
		WHERE pv.product_id = product.id 
		AND pv.in_stock = true 
		AND pv.stock > 0
	)`

	// FILTER_IS_POPULAR_SUBQUERY filters products with at least one popular variant
	FILTER_IS_POPULAR_SUBQUERY = `EXISTS (
		SELECT 1 FROM product_variant pv 
		WHERE pv.product_id = product.id 
		AND pv.is_popular = ?
	)`
)
