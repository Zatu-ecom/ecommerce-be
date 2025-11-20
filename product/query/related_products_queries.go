package query

// Related products stored procedure queries
const (
	// FIND_RELATED_PRODUCTS_SCORED_QUERY calls the stored procedure to get scored related products
	// Parameters: productID, sellerID (nullable), limit, offset, strategies
	FIND_RELATED_PRODUCTS_SCORED_QUERY = `
		SELECT 
			product_id, 
			product_name, 
			category_id, 
			category_name, 
			parent_category_id, 
			parent_category_name, 
			brand, 
			sku, 
			short_description, 
			long_description, 
			tags, 
			seller_id, 
			has_variants, 
			min_price, 
			max_price, 
			allow_purchase, 
			total_variants, 
			in_stock_variants, 
			created_at, 
			updated_at, 
			final_score, 
			relation_reason, 
			strategy_used 
		FROM 
			get_related_products_scored(
				$1 :: BIGINT, 
				$2 :: BIGINT, 
				$3 :: INT, 
				$4 :: INT, 
				$5 :: TEXT
			)`

	// FIND_RELATED_PRODUCTS_COUNT_QUERY gets the total count of related products
	// Parameters: productID, sellerID (nullable), strategies
	FIND_RELATED_PRODUCTS_COUNT_QUERY = `SELECT get_related_products_count($1, $2, $3)`
)
