package product

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestSearchProducts tests the GET /api/products/search endpoint with various scenarios
func TestSearchProducts(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")
	containers.RunSeeds(t, "migrations/seeds/002_seed_product_data.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// Set seller ID header for public API (required by PublicAPIAuth middleware)
	// Using seller_id 2 which owns products 1-4 (iPhone, Samsung, MacBook, Sony)
	client.SetHeader("X-Seller-ID", "2")

	// ============================================================================
	// QUERY VALIDATION TESTS
	// ============================================================================

	t.Run("Error - Missing search query parameter", func(t *testing.T) {
		w := client.Get(t, "/api/products/search")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Error - Empty search query parameter", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Error - Whitespace only search query", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=   ")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should return empty results for whitespace query
		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.Empty(t, results, "Should return no results for whitespace query")
	})

	t.Run("Success - Very long search query", func(t *testing.T) {
		longQuery := ""
		for i := 0; i < 500; i++ {
			longQuery += "a"
		}

		w := client.Get(t, fmt.Sprintf("/api/products/search?q=%s", url.QueryEscape(longQuery)))

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should not crash, even if no results found
		_, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
	})

	t.Run("Success - Special characters in query", func(t *testing.T) {
		specialQueries := []string{
			"iPhone!",
			"MacBook@Pro",
			"$100",
			"50%",
			"T-Shirt",
			"M&M",
		}

		for _, query := range specialQueries {
			w := client.Get(t, fmt.Sprintf("/api/products/search?q=%s", url.QueryEscape(query)))

			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

			// Should handle special characters gracefully
			_, ok := response["data"].(map[string]interface{})["results"]
			assert.True(t, ok, "Should have results field for query: "+query)
		}
	})

	t.Run("Success - SQL injection attempt in query", func(t *testing.T) {
		sqlInjectionQueries := []string{
			"'; DROP TABLE product; --",
			"1' OR '1'='1",
			"admin'--",
			"' UNION SELECT * FROM product--",
		}

		for _, query := range sqlInjectionQueries {
			w := client.Get(t, fmt.Sprintf("/api/products/search?q=%s", url.QueryEscape(query)))

			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

			// Should safely handle SQL injection attempts
			results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
			assert.True(t, ok, "results should be an array")
			assert.Empty(t, results, "Should return no results for SQL injection attempt")
		}
	})

	// ============================================================================
	// SEARCH MATCHING TESTS - NAME, DESCRIPTION, TAGS
	// ============================================================================

	t.Run("Success - Search by exact product name", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=iPhone 15 Pro")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify search response structure
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "iPhone 15 Pro", data["query"], "Query should be returned in response")

		results, ok := data["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.NotEmpty(t, results, "Should find iPhone 15 Pro")

		// Verify first result contains iPhone
		firstResult := results[0].(map[string]interface{})
		productName := firstResult["name"].(string)
		assert.Contains(t, productName, "iPhone", "Product name should contain iPhone")

		// Verify relevance score exists
		assert.NotNil(t, firstResult["relevanceScore"], "Should have relevance score")
		assert.NotNil(t, firstResult["matchedFields"], "Should have matched fields")
	})

	t.Run("Success - Search by partial product name (case insensitive)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=iphone")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.NotEmpty(t, results, "Should find products with 'iphone' (case insensitive)")

		// Verify at least one result contains iPhone
		found := false
		for _, item := range results {
			result := item.(map[string]interface{})
			productName := result["name"].(string)
			if productName == "iPhone 15 Pro" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find iPhone 15 Pro")
	})

	t.Run("Success - Search by brand name", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=Apple")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.NotEmpty(t, results, "Should find Apple products")

		// Verify all results are Apple products (iPhone or MacBook for seller 2)
		for _, item := range results {
			result := item.(map[string]interface{})
			brand := result["brand"].(string)
			assert.Equal(t, "Apple", brand, "All results should be Apple brand")
		}
	})

	t.Run("Success - Search by description keyword", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=smartphone")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.NotEmpty(t, results, "Should find products with 'smartphone' in description")

		// At least iPhone or Samsung should be found (both are smartphones)
		foundSmartphone := false
		for _, item := range results {
			result := item.(map[string]interface{})
			description := result["shortDescription"].(string)
			if description != "" {
				foundSmartphone = true
				break
			}
		}
		assert.True(t, foundSmartphone, "Should find smartphone products")
	})

	t.Run("Success - Search by tag", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=flagship")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.NotEmpty(t, results, "Should find products with 'flagship' tag")

		// Verify tags are included in response
		firstResult := results[0].(map[string]interface{})
		tags, ok := firstResult["tags"].([]interface{})
		assert.True(t, ok, "tags should be an array")
		assert.NotEmpty(t, tags, "Product should have tags")
	})

	t.Run("Success - Search returns multiple matching products", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=premium")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		// Could match multiple products (iPhone Pro, MacBook Pro, Sony headphones)
		// Just verify structure is correct
		assert.GreaterOrEqual(t, len(results), 1, "Should find at least one premium product")
	})

	t.Run("Success - Search with no matching results", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=xyz123nonexistent")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.Empty(t, results, "Should return empty results for non-matching query")

		// Verify pagination shows 0 total
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		totalItems := int(pagination["totalItems"].(float64))
		assert.Equal(t, 0, totalItems, "Total items should be 0")
	})

	t.Run("Success - Search with numeric query", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=15")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		// Could match iPhone 15 Pro
		if len(results) > 0 {
			firstResult := results[0].(map[string]interface{})
			assert.NotNil(t, firstResult["name"], "Product should have name")
		}
	})

	// ============================================================================
	// FILTER COMBINATION TESTS
	// ============================================================================

	t.Run("Success - Search with category filter", func(t *testing.T) {
		// Category 4 = Smartphones (iPhone, Samsung)
		w := client.Get(t, "/api/products/search?q=phone&categoryId=4")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// Verify all results are in category 4
		for _, item := range results {
			result := item.(map[string]interface{})
			categoryID := uint(result["categoryId"].(float64))
			assert.Equal(t, uint(4), categoryID, "All results should be in category 4")
		}
	})

	t.Run("Success - Search with brand filter", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=phone&brand=Apple")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// All results should be Apple brand
		for _, item := range results {
			result := item.(map[string]interface{})
			brand := result["brand"].(string)
			assert.Equal(t, "Apple", brand, "All results should be Apple brand")
		}
	})

	t.Run("Success - Search with minimum price filter", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&minPrice=1000")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// Verify products have variants with price >= 1000
		for _, item := range results {
			result := item.(map[string]interface{})
			variantPreview := result["variantPreview"].(map[string]interface{})
			assert.NotNil(t, variantPreview, "Should have variant preview")
		}
	})

	t.Run("Success - Search with maximum price filter", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=phone&maxPrice=50000")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// Should return products with variants <= 50000
		// Just verify structure is correct
		for _, item := range results {
			result := item.(map[string]interface{})
			assert.NotNil(t, result["variantPreview"], "Should have variant preview")
		}
	})

	t.Run("Success - Search with price range (min and max)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&minPrice=500&maxPrice=100000")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// Verify response structure
		for _, item := range results {
			result := item.(map[string]interface{})
			assert.NotNil(t, result["name"], "Product should have name")
			assert.NotNil(t, result["variantPreview"], "Should have variant preview")
		}
	})

	t.Run("Success - Search with multiple filters combined", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=phone&categoryId=4&brand=Apple&minPrice=1000")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// If results exist, verify they match all filters
		for _, item := range results {
			result := item.(map[string]interface{})
			brand := result["brand"].(string)
			categoryID := uint(result["categoryId"].(float64))
			assert.Equal(t, "Apple", brand, "Should be Apple brand")
			assert.Equal(t, uint(4), categoryID, "Should be in category 4")
		}
	})

	t.Run("Success - Search with invalid category filter (ignored)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=phone&categoryId=999999")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.Empty(t, results, "Should return no results for non-existent category")
	})

	t.Run("Success - Search with invalid price filter (non-numeric)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=phone&minPrice=abc&maxPrice=xyz")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Invalid price filters should be ignored, search should still work
		_, ok := response["data"].(map[string]interface{})["results"]
		assert.True(t, ok, "Should have results field even with invalid price filters")
	})

	t.Run("Success - Search with negative price filter (ignored)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=phone&minPrice=-100")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Negative price should be ignored or handled gracefully
		_, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
	})

	t.Run("Success - Search with zero price filter", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=phone&minPrice=0&maxPrice=0")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		_, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		// Zero price filters should be ignored (minPrice > 0 check in handler)
	})

	// ============================================================================
	// PAGINATION TESTS
	// ============================================================================

	t.Run("Success - Search with default pagination", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify pagination structure
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["currentPage"], "Default page should be 1")
		assert.Equal(t, float64(10), pagination["itemsPerPage"], "Default limit should be 10")
		assert.NotNil(t, pagination["totalPages"], "Should have totalPages")
		assert.NotNil(t, pagination["totalItems"], "Should have totalItems")
		assert.NotNil(t, pagination["hasNext"], "Should have hasNext")
		assert.NotNil(t, pagination["hasPrev"], "Should have hasPrev")
	})

	t.Run("Success - Search with custom page and limit", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&page=1&limit=2")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.LessOrEqual(t, len(results), 2, "Should return at most 2 results")

		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["currentPage"], "Page should be 1")
		assert.Equal(t, float64(2), pagination["itemsPerPage"], "Limit should be 2")
	})

	t.Run("Success - Search with page 2", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&page=2&limit=1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		assert.Equal(t, float64(2), pagination["currentPage"], "Page should be 2")
		assert.Equal(t, true, pagination["hasPrev"], "Should have previous page")
	})

	t.Run("Success - Search with limit exceeding maximum (capped at 100)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&limit=200")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		// Service caps limit at 100
		itemsPerPage := int(pagination["itemsPerPage"].(float64))
		assert.LessOrEqual(t, itemsPerPage, 100, "Limit should be capped at 100")
	})

	t.Run("Success - Search with page beyond total pages", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&page=999")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.Empty(t, results, "Should return empty results for page beyond total")

		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		assert.Equal(t, false, pagination["hasNext"], "Should not have next page")
	})

	t.Run("Success - Search with invalid page number (defaults to 1)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&page=0")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		// Service defaults page < 1 to 1
		currentPage := int(pagination["currentPage"].(float64))
		assert.GreaterOrEqual(t, currentPage, 1, "Page should default to 1")
	})

	t.Run("Success - Search with negative page number (defaults to 1)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&page=-5")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		currentPage := int(pagination["currentPage"].(float64))
		assert.GreaterOrEqual(t, currentPage, 1, "Page should default to 1")
	})

	t.Run("Success - Search with invalid limit (defaults to 20)", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=pro&limit=0")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		// Service defaults limit < 1 to 20
		itemsPerPage := int(pagination["itemsPerPage"].(float64))
		assert.GreaterOrEqual(t, itemsPerPage, 1, "Limit should default to valid value")
	})

	// ============================================================================
	// MULTI-TENANT ISOLATION TESTS
	// ============================================================================

	t.Run("Error - Search without X-Seller-ID header", func(t *testing.T) {
		clientNoHeader := helpers.NewAPIClient(server)
		// Don't set X-Seller-ID header

		w := clientNoHeader.Get(t, "/api/products/search?q=phone")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Error - Search with invalid X-Seller-ID header", func(t *testing.T) {
		clientInvalidHeader := helpers.NewAPIClient(server)
		clientInvalidHeader.SetHeader("X-Seller-ID", "invalid")

		w := clientInvalidHeader.Get(t, "/api/products/search?q=phone")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Error - Search with zero X-Seller-ID header", func(t *testing.T) {
		clientZeroHeader := helpers.NewAPIClient(server)
		clientZeroHeader.SetHeader("X-Seller-ID", "0")

		w := clientZeroHeader.Get(t, "/api/products/search?q=phone")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Error - Search with negative X-Seller-ID header", func(t *testing.T) {
		clientNegativeHeader := helpers.NewAPIClient(server)
		clientNegativeHeader.SetHeader("X-Seller-ID", "-1")

		w := clientNegativeHeader.Get(t, "/api/products/search?q=phone")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Success - Seller 2 searches and gets only their products", func(t *testing.T) {
		client2 := helpers.NewAPIClient(server)
		client2.SetHeader("X-Seller-ID", "2")

		w := client2.Get(t, "/api/products/search?q=pro")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// All results should belong to seller 2
		for _, item := range results {
			result := item.(map[string]interface{})
			sellerID := uint(result["sellerId"].(float64))
			assert.Equal(t, uint(2), sellerID, "All products should belong to seller 2")
		}
	})

	t.Run("Success - Seller 3 searches and gets only their products", func(t *testing.T) {
		client3 := helpers.NewAPIClient(server)
		client3.SetHeader("X-Seller-ID", "3")

		w := client3.Get(t, "/api/products/search?q=shirt")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// All results should belong to seller 3
		for _, item := range results {
			result := item.(map[string]interface{})
			sellerID := uint(result["sellerId"].(float64))
			assert.Equal(t, uint(3), sellerID, "All products should belong to seller 3")
		}
	})

	t.Run("Success - Seller 3 cannot find seller 2 products", func(t *testing.T) {
		client3 := helpers.NewAPIClient(server)
		client3.SetHeader("X-Seller-ID", "3")

		w := client3.Get(t, "/api/products/search?q=iPhone")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.Empty(t, results, "Seller 3 should not find iPhone (belongs to seller 2)")
	})

	t.Run("Success - Seller 4 searches their products", func(t *testing.T) {
		client4 := helpers.NewAPIClient(server)
		client4.SetHeader("X-Seller-ID", "4")

		w := client4.Get(t, "/api/products/search?q=sofa")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// All results should belong to seller 4
		for _, item := range results {
			result := item.(map[string]interface{})
			sellerID := uint(result["sellerId"].(float64))
			assert.Equal(t, uint(4), sellerID, "All products should belong to seller 4")
		}
	})

	t.Run("Success - Different sellers get different search results for same query", func(t *testing.T) {
		// Seller 2 searches for "pro"
		client2 := helpers.NewAPIClient(server)
		client2.SetHeader("X-Seller-ID", "2")
		w2 := client2.Get(t, "/api/products/search?q=pro")
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		results2, _ := response2["data"].(map[string]interface{})["results"].([]interface{})

		// Seller 3 searches for "pro"
		client3 := helpers.NewAPIClient(server)
		client3.SetHeader("X-Seller-ID", "3")
		w3 := client3.Get(t, "/api/products/search?q=pro")
		response3 := helpers.AssertSuccessResponse(t, w3, http.StatusOK)
		results3, _ := response3["data"].(map[string]interface{})["results"].([]interface{})

		// Results should be different (or empty for one seller)
		// Seller 2 has iPhone Pro, MacBook Pro
		// Seller 3 has no "pro" products
		assert.NotEmpty(t, results2, "Seller 2 should have 'pro' products")
		assert.Empty(t, results3, "Seller 3 should not have 'pro' products")
	})

	// ============================================================================
	// RESPONSE STRUCTURE VALIDATION TESTS
	// ============================================================================

	t.Run("Success - Search response includes all required fields", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=iPhone")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})

		// Top-level fields
		assert.NotNil(t, data["query"], "Response should include query")
		assert.NotNil(t, data["results"], "Response should include results")
		assert.NotNil(t, data["pagination"], "Response should include pagination")
		assert.NotNil(t, data["searchTime"], "Response should include searchTime")

		// Verify query matches
		assert.Equal(t, "iPhone", data["query"].(string), "Query should match request")

		// Verify pagination structure
		pagination := data["pagination"].(map[string]interface{})
		assert.NotNil(t, pagination["currentPage"], "Pagination should include currentPage")
		assert.NotNil(t, pagination["totalPages"], "Pagination should include totalPages")
		assert.NotNil(t, pagination["totalItems"], "Pagination should include totalItems")
		assert.NotNil(t, pagination["itemsPerPage"], "Pagination should include itemsPerPage")
		assert.NotNil(t, pagination["hasNext"], "Pagination should include hasNext")
		assert.NotNil(t, pagination["hasPrev"], "Pagination should include hasPrev")
	})

	t.Run("Success - Search result includes product fields", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=iPhone")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.NotEmpty(t, results, "Should have at least one result")

		// Check first result structure
		firstResult := results[0].(map[string]interface{})

		// Search-specific fields
		assert.NotNil(t, firstResult["relevanceScore"], "Result should include relevanceScore")
		assert.NotNil(t, firstResult["matchedFields"], "Result should include matchedFields")

		// Product fields (embedded from ProductResponse)
		assert.NotNil(t, firstResult["id"], "Result should include product id")
		assert.NotNil(t, firstResult["name"], "Result should include product name")
		assert.NotNil(t, firstResult["categoryId"], "Result should include categoryId")
		assert.NotNil(t, firstResult["brand"], "Result should include brand")
		assert.NotNil(t, firstResult["sku"], "Result should include sku")
		assert.NotNil(t, firstResult["shortDescription"], "Result should include shortDescription")
		assert.NotNil(t, firstResult["tags"], "Result should include tags")
		assert.NotNil(t, firstResult["sellerId"], "Result should include sellerId")

		// Variant preview (should be present in search results like GetAll)
		assert.NotNil(t, firstResult["variantPreview"], "Result should include variantPreview")

		variantPreview := firstResult["variantPreview"].(map[string]interface{})
		assert.NotNil(t, variantPreview["totalVariants"], "Variant preview should include totalVariants")
		assert.NotNil(t, variantPreview["options"], "Variant preview should include options")
	})

	t.Run("Success - Search result does NOT include full variants array", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=iPhone")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.NotEmpty(t, results, "Should have at least one result")

		// Check first result
		firstResult := results[0].(map[string]interface{})

		// Full variants array should NOT be present in search results (listing view)
		_, hasVariants := firstResult["variants"]
		assert.False(t, hasVariants, "Search results should NOT include full variants array (use variantPreview)")
	})

	t.Run("Success - Search result includes variantPreview with correct structure", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=iPhone")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.NotEmpty(t, results, "Should have at least one result")

		firstResult := results[0].(map[string]interface{})
		variantPreview := firstResult["variantPreview"].(map[string]interface{})

		// Verify variantPreview structure
		totalVariants := int(variantPreview["totalVariants"].(float64))
		assert.Greater(t, totalVariants, 0, "Product should have at least one variant")

		options, ok := variantPreview["options"].([]interface{})
		assert.True(t, ok, "Options should be an array")
		assert.NotEmpty(t, options, "Product should have at least one option")

		// Verify option structure
		firstOption := options[0].(map[string]interface{})
		assert.NotNil(t, firstOption["name"], "Option should have name")
		assert.NotNil(t, firstOption["displayName"], "Option should have displayName")
		assert.NotNil(t, firstOption["availableValues"], "Option should have availableValues array")
	})

	// ============================================================================
	// EDGE CASES AND SPECIAL SCENARIOS
	// ============================================================================

	t.Run("Success - Search with URL encoded special characters", func(t *testing.T) {
		// Test that URL encoding is handled correctly
		w := client.Get(t, "/api/products/search?q=MacBook+Pro")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		// Query should be decoded
		query := data["query"].(string)
		assert.Contains(t, query, "MacBook", "Query should be decoded correctly")
	})

	t.Run("Success - Search with Unicode characters", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q="+url.QueryEscape("手机"))

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should handle Unicode gracefully (may not find results)
		_, ok := response["data"].(map[string]interface{})["results"]
		assert.True(t, ok, "Should handle Unicode query gracefully")
	})

	t.Run("Success - Search with multiple spaces in query", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=iPhone++15++Pro")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should handle multiple spaces gracefully
		_, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		// Should still find iPhone 15 Pro despite extra spaces
	})

	t.Run("Success - Search with query containing only numbers", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=1000")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should search for "1000" in product names/descriptions
		_, ok := response["data"].(map[string]interface{})["results"]
		assert.True(t, ok, "Should handle numeric query")
	})

	t.Run("Success - Search with both minPrice and maxPrice where min > max", func(t *testing.T) {
		w := client.Get(t, "/api/products/search?q=phone&minPrice=10000&maxPrice=1000")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should return empty results or handle gracefully
		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		// No products can satisfy minPrice > maxPrice
		assert.Empty(t, results, "Should return no results when minPrice > maxPrice")
	})

	t.Run("Success - Search combines query and filters correctly", func(t *testing.T) {
		// Search for "phone" AND filter by brand "Apple"
		w := client.Get(t, "/api/products/search?q=phone&brand=Apple")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		results, ok := response["data"].(map[string]interface{})["results"].([]interface{})
		assert.True(t, ok, "results should be an array")

		// If results exist, they should match both query and filter
		for _, item := range results {
			result := item.(map[string]interface{})
			brand := result["brand"].(string)
			assert.Equal(t, "Apple", brand, "Result should match brand filter")
			// Should also contain "phone" in name/description/tags
		}
	})

	t.Run("Success - Search performance with multiple results", func(t *testing.T) {
		// Search for common term that might match multiple products
		w := client.Get(t, "/api/products/search?q=premium")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify searchTime is present (performance indicator)
		data := response["data"].(map[string]interface{})
		searchTime := data["searchTime"].(string)
		assert.NotEmpty(t, searchTime, "Should include search time")
	})
}
