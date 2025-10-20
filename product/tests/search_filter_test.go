package tests

// import (
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// // TestSearchFunctionality tests product search functionality
// func TestSearchFunctionality(t *testing.T) {
// 	t.Log("🧪 Testing product search functionality...")

// 	t.Run("Basic Search Query", func(t *testing.T) {
// 		// Test basic search query builder
// 		searchReq := BuildSearchQuery("smartphone", nil, nil, nil, nil, 1, 10)
// 		assert.Equal(t, "smartphone", searchReq.Query)
// 		assert.Equal(t, 1, searchReq.Page)
// 		assert.Equal(t, 10, searchReq.Limit)
// 	})

// 	t.Run("Search with Filters", func(t *testing.T) {
// 		// Test search with various filters
// 		categoryID := uint(1)
// 		minPrice := 100.0
// 		maxPrice := 500.0
// 		attributes := map[string]string{"color": "black", "brand": "apple"}

// 		searchReq := BuildSearchQuery("laptop", &categoryID, &minPrice, &maxPrice, attributes, 1, 20)
// 		assert.Equal(t, "laptop", searchReq.Query)
// 		assert.Contains(t, searchReq.Filters, "categoryId")
// 		assert.Contains(t, searchReq.Filters, "minPrice")
// 		assert.Contains(t, searchReq.Filters, "maxPrice")
// 		assert.Contains(t, searchReq.Filters, "attributes")
// 		assert.Equal(t, 1, searchReq.Page)
// 		assert.Equal(t, 20, searchReq.Limit)
// 	})

// 	t.Run("Pagination", func(t *testing.T) {
// 		// Test pagination parameters
// 		searchReq := BuildSearchQuery("test", nil, nil, nil, nil, 2, 25)
// 		assert.Equal(t, 2, searchReq.Page)
// 		assert.Equal(t, 25, searchReq.Limit)
// 	})

// 	t.Log("✅ Search functionality tests completed")
// }

// // TestFilterFunctionality tests product filtering functionality
// func TestFilterFunctionality(t *testing.T) {
// 	t.Log("🧪 Testing product filtering functionality...")

// 	t.Run("Price Range Filtering", func(t *testing.T) {
// 		// Test price range filtering
// 		minPrice := 50.0
// 		maxPrice := 200.0

// 		searchReq := BuildSearchQuery("", nil, &minPrice, &maxPrice, nil, 1, 10)
// 		assert.Contains(t, searchReq.Filters, "minPrice")
// 		assert.Contains(t, searchReq.Filters, "maxPrice")
// 	})

// 	t.Run("Category Filtering", func(t *testing.T) {
// 		// Test category filtering
// 		categoryID := uint(5)

// 		searchReq := BuildSearchQuery("", &categoryID, nil, nil, nil, 1, 10)
// 		assert.Contains(t, searchReq.Filters, "categoryId")
// 	})

// 	t.Run("Attribute Filtering", func(t *testing.T) {
// 		// Test attribute filtering
// 		attributes := map[string]string{
// 			"color":    "red",
// 			"size":     "large",
// 			"brand":    "nike",
// 			"material": "cotton",
// 		}

// 		searchReq := BuildSearchQuery("", nil, nil, nil, attributes, 1, 10)
// 		assert.Contains(t, searchReq.Filters, "attributes")
// 		assert.Len(t, searchReq.Filters, 1)
// 	})

// 	t.Log("✅ Filter functionality tests completed")
// }

// // TestSearchQueryBuilder tests the search query builder function
// func TestSearchQueryBuilder(t *testing.T) {
// 	t.Log("🧪 Testing search query builder...")

// 	t.Run("Empty Query", func(t *testing.T) {
// 		// Test with empty search query
// 		searchReq := BuildSearchQuery("", nil, nil, nil, nil, 1, 10)
// 		assert.Equal(t, "", searchReq.Query)
// 		assert.Empty(t, searchReq.Filters)
// 	})

// 	t.Run("Full Query", func(t *testing.T) {
// 		// Test with all parameters
// 		categoryID := uint(10)
// 		minPrice := 25.0
// 		maxPrice := 1000.0
// 		attributes := map[string]string{"availability": "in_stock"}

// 		searchReq := BuildSearchQuery("gaming laptop", &categoryID, &minPrice, &maxPrice, attributes, 3, 50)
// 		assert.Equal(t, "gaming laptop", searchReq.Query)
// 		assert.Contains(t, searchReq.Filters, "categoryId")
// 		assert.Contains(t, searchReq.Filters, "minPrice")
// 		assert.Contains(t, searchReq.Filters, "maxPrice")
// 		assert.Contains(t, searchReq.Filters, "attributes")
// 		assert.Equal(t, 3, searchReq.Page)
// 		assert.Equal(t, 50, searchReq.Limit)
// 	})

// 	t.Run("Edge Cases", func(t *testing.T) {
// 		// Test edge cases
// 		searchReq := BuildSearchQuery("", nil, nil, nil, nil, 0, 0)
// 		assert.Equal(t, 0, searchReq.Page)
// 		assert.Equal(t, 0, searchReq.Limit)

// 		// Test with very large values
// 		searchReq = BuildSearchQuery("test", nil, nil, nil, nil, 999, 1000)
// 		assert.Equal(t, 999, searchReq.Page)
// 		assert.Equal(t, 1000, searchReq.Limit)
// 	})

// 	t.Log("✅ Search query builder tests completed")
// }

// // TestFilterCombinations tests various filter combinations
// func TestFilterCombinations(t *testing.T) {
// 	t.Log("🧪 Testing filter combinations...")

// 	t.Run("Category and Price", func(t *testing.T) {
// 		// Test category + price combination
// 		categoryID := uint(3)
// 		minPrice := 100.0
// 		maxPrice := 500.0

// 		searchReq := BuildSearchQuery("", &categoryID, &minPrice, &maxPrice, nil, 1, 10)
// 		assert.Contains(t, searchReq.Filters, "categoryId")
// 		assert.Contains(t, searchReq.Filters, "minPrice")
// 		assert.Contains(t, searchReq.Filters, "maxPrice")
// 		assert.Len(t, searchReq.Filters, 3)
// 	})

// 	t.Run("Price and Attributes", func(t *testing.T) {
// 		// Test price + attributes combination
// 		minPrice := 50.0
// 		maxPrice := 200.0
// 		attributes := map[string]string{"color": "blue"}

// 		searchReq := BuildSearchQuery("", nil, &minPrice, &maxPrice, attributes, 1, 10)
// 		assert.Contains(t, searchReq.Filters, "minPrice")
// 		assert.Contains(t, searchReq.Filters, "maxPrice")
// 		assert.Contains(t, searchReq.Filters, "attributes")
// 		assert.Len(t, searchReq.Filters, 3)
// 	})

// 	t.Run("Category and Attributes", func(t *testing.T) {
// 		// Test category + attributes combination
// 		categoryID := uint(7)
// 		attributes := map[string]string{"brand": "sony", "condition": "new"}

// 		searchReq := BuildSearchQuery("", &categoryID, nil, nil, attributes, 1, 10)
// 		assert.Contains(t, searchReq.Filters, "categoryId")
// 		assert.Contains(t, searchReq.Filters, "attributes")
// 		assert.Len(t, searchReq.Filters, 2)
// 	})

// 	t.Log("✅ Filter combinations tests completed")
// }
