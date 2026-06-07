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

	// FILTER_IN_STOCK_SUBQUERY filters products that have at least one purchasable variant
	// with available inventory (quantity - reserved_quantity - threshold > 0) at any location.
	FILTER_IN_STOCK_SUBQUERY = `EXISTS (
		SELECT 1 FROM product_variant pv
		INNER JOIN inventory inv ON inv.variant_id = pv.id
		WHERE pv.product_id = product.id
		AND pv.allow_purchase = true
		AND (inv.quantity - inv.reserved_quantity - inv.threshold) > 0
	)`

	// FILTER_OUT_OF_STOCK_SUBQUERY filters products with no purchasable in-stock variants
	FILTER_OUT_OF_STOCK_SUBQUERY = `NOT EXISTS (
		SELECT 1 FROM product_variant pv
		INNER JOIN inventory inv ON inv.variant_id = pv.id
		WHERE pv.product_id = product.id
		AND pv.allow_purchase = true
		AND (inv.quantity - inv.reserved_quantity - inv.threshold) > 0
	)`

	// FILTER_IS_POPULAR_SUBQUERY filters products with at least one popular variant
	FILTER_IS_POPULAR_SUBQUERY = `EXISTS (
		SELECT 1 FROM product_variant pv 
		WHERE pv.product_id = product.id 
		AND pv.is_popular = ?
	)`

	// FILTER_VARIANT_IDS_SUBQUERY filters products that have any of the specified variant IDs
	FILTER_VARIANT_IDS_SUBQUERY = `EXISTS (
		SELECT 1 FROM product_variant pv 
		WHERE pv.product_id = product.id 
		AND pv.id IN ?
	)`
)
