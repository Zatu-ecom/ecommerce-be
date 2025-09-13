package query

// Category queries
const (
	FIND_ATTRIBUTES_BY_CATEGORY_ID_WITH_INHERITANCE_QUERY = `
		WITH RECURSIVE category_hierarchy AS (
				SELECT id, parent_id FROM category WHERE id = ?
				UNION ALL
				SELECT c.id, c.parent_id FROM category c JOIN category_hierarchy ch ON c.id = ch.parent_id
			)
			SELECT DISTINCT ad.* FROM attribute_definition ad
			JOIN category_attribute ca ON ad.id = ca.attribute_definition_id
			WHERE ca.category_id IN (SELECT id FROM category_hierarchy)`
)
