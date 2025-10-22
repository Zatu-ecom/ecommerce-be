package query

const (
	FIND_BRANDS_WITH_PRODUCT_COUNT_QUERY = `
		SELECT 
			count(brand) as product_count, 
			brand 
		from product 
		group by brand
		having count(brand) > 0
		order by product_count desc`

	FIND_CATEGORIES_WITH_PRODUCT_COUNT_QUERY = `
		SELECT 
			c.id AS category_id,
			c.name AS category_name,
			c.parent_id AS parent_id,
			COUNT(p.id) AS product_count
		FROM category c
		LEFT JOIN product p ON p.category_id = c.id
		GROUP BY c.id, c.name, c.parent_id 
		HAVING COUNT(p.id) > 0
		ORDER BY product_count desc`

	FIND_ATTRIBUTES_WITH_PRODUCT_COUNT_QUERY = `
		SELECT 
			count(ad.id) as product_count,
			ad.name as name,
			ad.key as key,
			ad.allowed_values as allowed_values
		from product_attribute pa
		left join attribute_definition ad on ad.id = pa.attribute_definition_id
		group by ad.id, ad.name, ad.key, ad.allowed_values
		having count(ad.id) > 0
		order by product_count desc`

	// Multi-tenant versions with seller_id filter
	FIND_BRANDS_WITH_PRODUCT_COUNT_BY_SELLER_QUERY = `
		SELECT 
			count(brand) as product_count, 
			brand 
		from product 
		where seller_id = ?
		group by brand
		having count(brand) > 0
		order by product_count desc`

	FIND_CATEGORIES_WITH_PRODUCT_COUNT_BY_SELLER_QUERY = `
		SELECT 
			c.id AS category_id,
			c.name AS category_name,
			c.parent_id AS parent_id,
			COUNT(p.id) AS product_count
		FROM category c
		LEFT JOIN product p ON p.category_id = c.id AND p.seller_id = ?
		WHERE c.is_global = true OR c.seller_id = ?
		GROUP BY c.id, c.name, c.parent_id 
		HAVING COUNT(p.id) > 0
		ORDER BY product_count desc`

	FIND_ATTRIBUTES_WITH_PRODUCT_COUNT_BY_SELLER_QUERY = `
		SELECT 
			count(ad.id) as product_count,
			ad.name as name,
			ad.key as key,
			ad.allowed_values as allowed_values
		from product_attribute pa
		left join product p on p.id = pa.product_id
		left join attribute_definition ad on ad.id = pa.attribute_definition_id
		where p.seller_id = ?
		group by ad.id, ad.name, ad.key, ad.allowed_values
		having count(ad.id) > 0
		order by product_count desc`

	// Variant-based filter queries
	FIND_PRICE_RANGE_QUERY = `
		SELECT 
			MIN(pv.price) as min_price,
			MAX(pv.price) as max_price,
			COUNT(DISTINCT p.id) as product_count
		FROM product_variant pv
		INNER JOIN product p ON p.id = pv.product_id`

	FIND_PRICE_RANGE_BY_SELLER_QUERY = `
		SELECT 
			MIN(pv.price) as min_price,
			MAX(pv.price) as max_price,
			COUNT(DISTINCT p.id) as product_count
		FROM product_variant pv
		INNER JOIN product p ON p.id = pv.product_id
		WHERE p.seller_id = ?`

	FIND_VARIANT_OPTIONS_QUERY = `
		SELECT 
			po.id as option_id,
			po.name as option_name,
			po.display_name as option_display_name,
			pov.id as value_id,
			pov.value as option_value,
			pov.display_name as value_display_name,
			pov.color_code as color_code,
			COUNT(DISTINCT p.id) as product_count
		FROM product_option po
		INNER JOIN product_option_value pov ON pov.option_id = po.id
		INNER JOIN product p ON p.id = po.product_id
		INNER JOIN product_variant pv ON pv.product_id = p.id
		INNER JOIN variant_option_value vov ON vov.variant_id = pv.id AND vov.option_value_id = pov.id
		GROUP BY po.id, po.name, po.display_name, pov.id, pov.value, pov.display_name, pov.color_code
		ORDER BY po.position, pov.position`

	FIND_VARIANT_OPTIONS_BY_SELLER_QUERY = `
		SELECT 
			po.id as option_id,
			po.name as option_name,
			po.display_name as option_display_name,
			pov.id as value_id,
			pov.value as option_value,
			pov.display_name as value_display_name,
			pov.color_code as color_code,
			COUNT(DISTINCT p.id) as product_count
		FROM product_option po
		INNER JOIN product_option_value pov ON pov.option_id = po.id
		INNER JOIN product p ON p.id = po.product_id
		INNER JOIN product_variant pv ON pv.product_id = p.id
		INNER JOIN variant_option_value vov ON vov.variant_id = pv.id AND vov.option_value_id = pov.id
		WHERE p.seller_id = ?
		GROUP BY po.id, po.name, po.display_name, pov.id, pov.value, pov.display_name, pov.color_code
		ORDER BY po.position, pov.position`

	FIND_STOCK_STATUS_QUERY = `
		SELECT 
			COUNT(DISTINCT CASE WHEN pv.in_stock = true AND pv.stock > 0 THEN p.id END) as in_stock,
			COUNT(DISTINCT CASE WHEN pv.in_stock = false OR pv.stock = 0 THEN p.id END) as out_of_stock,
			COUNT(DISTINCT p.id) as total_products
		FROM product p
		INNER JOIN product_variant pv ON pv.product_id = p.id`

	FIND_STOCK_STATUS_BY_SELLER_QUERY = `
		SELECT 
			COUNT(DISTINCT CASE WHEN pv.in_stock = true AND pv.stock > 0 THEN p.id END) as in_stock,
			COUNT(DISTINCT CASE WHEN pv.in_stock = false OR pv.stock = 0 THEN p.id END) as out_of_stock,
			COUNT(DISTINCT p.id) as total_products
		FROM product p
		INNER JOIN product_variant pv ON pv.product_id = p.id
		WHERE p.seller_id = ?`
)
