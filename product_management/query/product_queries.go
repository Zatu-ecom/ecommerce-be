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
)
