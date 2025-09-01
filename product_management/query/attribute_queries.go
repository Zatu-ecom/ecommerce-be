package query

// Attribute Definition queries
const (
	GET_ALL_ATTRIBUTE_DEFINITIONS_QUERY = `
		SELECT 
			id, key, name, dataType, unit, description, allowedValues, isActive, createdAt
		FROM attributeDefinitions
		WHERE isActive = true
		ORDER BY name ASC
	`

	GET_ATTRIBUTE_DEFINITION_BY_ID_QUERY = `
		SELECT 
			id, key, name, dataType, unit, description, allowedValues, isActive, createdAt
		FROM attributeDefinitions
		WHERE id = ? AND isActive = true
	`

	GET_ATTRIBUTE_DEFINITION_BY_KEY_QUERY = `
		SELECT 
			id, key, name, dataType, unit, description, allowedValues, isActive, createdAt
		FROM attributeDefinitions
		WHERE key = ? AND isActive = true
	`

	CREATE_ATTRIBUTE_DEFINITION_QUERY = `
		INSERT INTO attributeDefinitions (key, name, dataType, unit, description, allowedValues, isActive, createdAt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	UPDATE_ATTRIBUTE_DEFINITION_QUERY = `
		UPDATE attributeDefinitions 
		SET name = ?, dataType = ?, unit = ?, description = ?, allowedValues = ?, isActive = ?, updatedAt = ?
		WHERE id = ?
	`
)

// Category Attribute queries
const (
	GET_CATEGORY_ATTRIBUTES_QUERY = `
		SELECT 
			ca.id, ca.isRequired, ca.isSearchable, ca.isFilterable, ca.sortOrder, 
			ca.defaultValue, ca.isActive,
			ad.id as attrDefId, ad.key, ad.name, ad.dataType, ad.unit, ad.allowedValues
		FROM categoryAttributes ca
		JOIN attributeDefinitions ad ON ca.attributeDefinitionId = ad.id
		WHERE ca.categoryId = ? AND ca.isActive = true
		ORDER BY ca.sortOrder ASC
	`

	CREATE_CATEGORY_ATTRIBUTE_QUERY = `
		INSERT INTO categoryAttributes (categoryId, attributeDefinitionId, isRequired, isSearchable, isFilterable, sortOrder, defaultValue, isActive, createdAt, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	UPDATE_CATEGORY_ATTRIBUTE_QUERY = `
		UPDATE categoryAttributes 
		SET isRequired = ?, isSearchable = ?, isFilterable = ?, sortOrder = ?, defaultValue = ?, isActive = ?, updatedAt = ?
		WHERE id = ?
	`

	DELETE_CATEGORY_ATTRIBUTE_QUERY = `
		DELETE FROM categoryAttributes WHERE categoryId = ? AND attributeDefinitionId = ?
	`
)
