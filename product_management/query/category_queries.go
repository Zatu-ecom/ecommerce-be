package query

// Category queries
const (
	GET_ALL_CATEGORIES_QUERY = `
		SELECT 
			c.id, c.name, c.parentId, c.description, c.isActive, 
			c.createdAt, c.updatedAt
		FROM categories c
		WHERE c.isActive = true
		ORDER BY c.name ASC
	`

	GET_CATEGORY_BY_ID_QUERY = `
		SELECT 
			c.id, c.name, c.parentId, c.description, c.isActive,
			c.createdAt, c.updatedAt
		FROM categories c
		WHERE c.id = ? AND c.isActive = true
	`

	GET_CATEGORIES_BY_PARENT_ID_QUERY = `
		SELECT 
			c.id, c.name, c.parentId, c.description, c.isActive,
			c.createdAt, c.updatedAt
		FROM categories c
		WHERE c.parentId = ? AND c.isActive = true
		ORDER BY c.name ASC
	`

	CREATE_CATEGORY_QUERY = `
		INSERT INTO categories (name, parentId, description, isActive, createdAt, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	UPDATE_CATEGORY_QUERY = `
		UPDATE categories 
		SET name = ?, parentId = ?, description = ?, isActive = ?, updatedAt = ?
		WHERE id = ?
	`

	SOFT_DELETE_CATEGORY_QUERY = `
		UPDATE categories 
		SET isActive = false, updatedAt = ?
		WHERE id = ?
	`

	CHECK_CATEGORY_HAS_PRODUCTS_QUERY = `
		SELECT COUNT(*) FROM products WHERE categoryId = ? AND isActive = true
	`

	CHECK_CATEGORY_HAS_CHILDREN_QUERY = `
		SELECT COUNT(*) FROM categories WHERE parentId = ? AND isActive = true
	`
)
