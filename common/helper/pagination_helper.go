package helper

// CalculateTotalPages calculates total pages for pagination
// totalItems: total number of items in the dataset
// pageSize: number of items per page
// Returns: total number of pages needed to display all items
func CalculateTotalPages(totalItems, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	pages := totalItems / pageSize
	if totalItems%pageSize > 0 {
		pages++
	}
	return pages
}

// CalculateOffset calculates the offset for database queries based on page and limit
// page: current page number (1-indexed)
// limit: number of items per page
// Returns: offset to skip in database query
func CalculateOffset(page, limit int) int {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	return (page - 1) * limit
}
