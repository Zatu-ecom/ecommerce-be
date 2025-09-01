package query

// Product queries
const (
	GET_ALL_PRODUCTS_QUERY = `
		SELECT 
			p.id, p.name, p.categoryId, p.brand, p.sku, p.price, p.currency,
			p.shortDescription, p.longDescription, p.images, p.inStock, p.isPopular,
			p.isActive, p.discount, p.tags, p.createdAt, p.updatedAt,
			c.name as categoryName
		FROM products p
		JOIN categories c ON p.categoryId = c.id
		WHERE p.isActive = true
	`

	GET_PRODUCT_BY_ID_QUERY = `
		SELECT 
			p.id, p.name, p.categoryId, p.brand, p.sku, p.price, p.currency,
			p.shortDescription, p.longDescription, p.images, p.inStock, p.isPopular,
			p.isActive, p.discount, p.tags, p.createdAt, p.updatedAt,
			c.name as categoryName
		FROM products p
		JOIN categories c ON p.categoryId = c.id
		WHERE p.id = ? AND p.isActive = true
	`

	GET_PRODUCT_BY_SKU_QUERY = `
		SELECT 
			p.id, p.name, p.categoryId, p.brand, p.sku, p.price, p.currency,
			p.shortDescription, p.longDescription, p.images, p.inStock, p.isPopular,
			p.isActive, p.discount, p.tags, p.createdAt, p.updatedAt
		FROM products p
		WHERE p.sku = ? AND p.isActive = true
	`

	CREATE_PRODUCT_QUERY = `
		INSERT INTO products (name, categoryId, brand, sku, price, currency, shortDescription, 
			longDescription, images, inStock, isPopular, isActive, discount, tags, createdAt, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	UPDATE_PRODUCT_QUERY = `
		UPDATE products 
		SET name = ?, categoryId = ?, brand = ?, price = ?, currency = ?, shortDescription = ?,
			longDescription = ?, images = ?, inStock = ?, isPopular = ?, discount = ?, tags = ?, updatedAt = ?
		WHERE id = ?
	`

	SOFT_DELETE_PRODUCT_QUERY = `
		UPDATE products 
		SET isActive = false, updatedAt = ?
		WHERE id = ?
	`

	UPDATE_PRODUCT_STOCK_QUERY = `
		UPDATE products 
		SET inStock = ?, updatedAt = ?
		WHERE id = ?
	`
)

// Product Attribute queries
const (
	GET_PRODUCT_ATTRIBUTES_QUERY = `
		SELECT 
			pa.key, pa.value,
			ad.name as attrName, ad.dataType, ad.unit, ad.allowedValues
		FROM productAttributes pa
		JOIN attributeDefinitions ad ON pa.attributeDefinitionId = ad.id
		WHERE pa.productId = ?
	`

	CREATE_PRODUCT_ATTRIBUTE_QUERY = `
		INSERT INTO productAttributes (productId, attributeDefinitionId, key, value, createdAt, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	UPDATE_PRODUCT_ATTRIBUTE_QUERY = `
		UPDATE productAttributes 
		SET value = ?, updatedAt = ?
		WHERE productId = ? AND attributeDefinitionId = ?
	`

	DELETE_PRODUCT_ATTRIBUTE_QUERY = `
		DELETE FROM productAttributes WHERE productId = ? AND attributeDefinitionId = ?
	`
)

// Package Option queries
const (
	GET_PRODUCT_PACKAGE_OPTIONS_QUERY = `
		SELECT 
			id, name, description, price, quantity, isActive, createdAt, updatedAt
		FROM packageOptions
		WHERE productId = ? AND isActive = true
		ORDER BY price ASC
	`

	CREATE_PACKAGE_OPTION_QUERY = `
		INSERT INTO packageOptions (productId, name, description, price, quantity, isActive, createdAt, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	UPDATE_PACKAGE_OPTION_QUERY = `
		UPDATE packageOptions 
		SET name = ?, description = ?, price = ?, quantity = ?, isActive = ?, updatedAt = ?
		WHERE id = ?
	`

	DELETE_PACKAGE_OPTION_QUERY = `
		DELETE FROM packageOptions WHERE id = ?
	`
)

// Search and Filter queries
const (
	SEARCH_PRODUCTS_QUERY = `
		SELECT 
			p.id, p.name, p.categoryId, p.brand, p.sku, p.price, p.currency,
			p.shortDescription, p.images, p.inStock, p.isPopular, p.isActive,
			p.discount, p.tags, p.createdAt, p.updatedAt,
			c.name as categoryName
		FROM products p
		JOIN categories c ON p.categoryId = c.id
		WHERE p.isActive = true
	`

	GET_PRODUCT_FILTERS_QUERY = `
		SELECT 
			c.id, c.name, COUNT(p.id) as productCount
		FROM categories c
		LEFT JOIN products p ON c.id = p.categoryId AND p.isActive = true
		WHERE c.isActive = true
		GROUP BY c.id, c.name
		ORDER BY c.name ASC
	`

	GET_RELATED_PRODUCTS_QUERY = `
		SELECT 
			p.id, p.name, p.price, p.shortDescription, p.images, p.createdAt
		FROM products p
		WHERE p.categoryId = ? AND p.id != ? AND p.isActive = true
		ORDER BY p.createdAt DESC
		LIMIT ?
	`
)
