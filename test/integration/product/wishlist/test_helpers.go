package wishlist

// getItemCount extracts the total item count from a GET by ID wishlist response.
// In the detail response, itemCount was replaced by pagination.totalItems.
func getItemCount(wishlist map[string]any) float64 {
	pagination, ok := wishlist["pagination"].(map[string]any)
	if !ok {
		return 0
	}
	totalItems, ok := pagination["totalItems"].(float64)
	if !ok {
		return 0
	}
	return totalItems
}
