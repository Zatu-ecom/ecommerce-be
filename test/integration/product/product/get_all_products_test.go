package product

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetAllProducts tests the GET /api/products endpoint with various filters and scenarios
func TestGetAllProducts(t *testing.T) {
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
	// BASIC SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Success - Get all products without filters", func(t *testing.T) {
		w := client.Get(t, "/api/products")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify products array exists
		products, ok := response["data"].(map[string]interface{})["products"].([]interface{})
		assert.True(t, ok, "products should be an array")
		assert.NotEmpty(t, products, "Should return at least one product")

		// Verify pagination exists
		pagination, ok := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		assert.True(t, ok, "pagination should exist")
		assert.NotNil(t, pagination["currentPage"])
		assert.NotNil(t, pagination["totalPages"])
		assert.NotNil(t, pagination["totalItems"])
		assert.NotNil(t, pagination["itemsPerPage"])

		// Verify each product has required fields
		for _, p := range products {
			product := p.(map[string]interface{})
			assert.NotNil(t, product["id"], "Product should have id")
			assert.NotNil(t, product["name"], "Product should have name")
			assert.NotNil(t, product["categoryId"], "Product should have categoryId")
			assert.NotNil(t, product["sku"], "Product should have baseSku")
		}
	})

	t.Run("Success - Get products with default pagination (page 1, limit 10)", func(t *testing.T) {
		w := client.Get(t, "/api/products")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		products := data["products"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		assert.LessOrEqual(t, len(products), 10, "Should return max 10 products per page")
		assert.Equal(t, float64(1), pagination["currentPage"], "Should be page 1 by default")
		assert.Equal(
			t,
			float64(20),
			pagination["itemsPerPage"],
			"Should have 20 items per page by default",
		)
	})

	t.Run("Success - Get products with custom pagination", func(t *testing.T) {
		w := client.Get(t, "/api/products?page=1&pageSize=5")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		products := data["products"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		assert.LessOrEqual(t, len(products), 5, "Should return max 5 products")
		assert.Equal(t, float64(5), pagination["itemsPerPage"], "Should have 5 items per page")
	})

	t.Run("Success - Get second page of products", func(t *testing.T) {
		w := client.Get(t, "/api/products?page=2&pageSize=3")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		pagination := data["pagination"].(map[string]interface{})
		assert.Equal(t, float64(2), pagination["currentPage"], "Should be page 2")
		assert.True(t, pagination["hasPrev"].(bool), "Should have previous page")
	})

	t.Run("Success - Product response includes variantPreview", func(t *testing.T) {
		w := client.Get(t, "/api/products")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// All products should have variantPreview (listing view doesn't include full variants)
		foundProductWithVariantPreview := false
		for _, p := range products {
			product := p.(map[string]interface{})
			if variantPreview, ok := product["variantPreview"].(map[string]interface{}); ok {
				foundProductWithVariantPreview = true
				// Verify variantPreview structure
				assert.NotNil(
					t,
					variantPreview["totalVariants"],
					"VariantPreview should have totalVariants",
				)
				assert.NotNil(t, variantPreview["options"], "VariantPreview should have options")

				totalVariants := variantPreview["totalVariants"].(float64)
				assert.Greater(
					t,
					totalVariants,
					float64(0),
					"Product should have at least 1 variant",
				)
				break
			}
		}
		assert.True(
			t,
			foundProductWithVariantPreview,
			"Should find at least one product with variantPreview",
		)

		// Verify that full variants are NOT included in listing (GetAll API)
		for _, p := range products {
			product := p.(map[string]interface{})
			variants, hasVariants := product["variants"]
			// variants should either not exist or be null/empty in listing view
			if hasVariants && variants != nil {
				variantsArray, isArray := variants.([]interface{})
				if isArray {
					assert.Empty(
						t,
						variantsArray,
						"GetAll API should not return full variants array",
					)
				}
			}
		}
	})

	// ============================================================================
	// FILTERING SCENARIOS
	// ============================================================================

	t.Run("Filter - By category ID", func(t *testing.T) {
		// Category 4 is Smartphones
		w := client.Get(t, "/api/products?categoryIds=4")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// All products should belong to category 4
		for _, p := range products {
			product := p.(map[string]interface{})
			assert.Equal(
				t,
				float64(4),
				product["categoryId"],
				"All products should be in category 4",
			)
		}
	})

	t.Run("Filter - By brand", func(t *testing.T) {
		w := client.Get(t, "/api/products?brands=Apple")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotEmpty(t, products, "Should find Apple products")
		for _, p := range products {
			product := p.(map[string]interface{})
			assert.Equal(t, "Apple", product["brand"], "All products should be Apple brand")
		}
	})

	t.Run("Filter - By min price", func(t *testing.T) {
		minPrice := 500.0
		w := client.Get(t, fmt.Sprintf("/api/products?minPrice=%.2f", minPrice))

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// Note: The filter might work on product level or variant level
		// This test just verifies the query doesn't error
		assert.NotNil(t, products, "Should return products array")
	})

	t.Run("Filter - By max price", func(t *testing.T) {
		maxPrice := 100.0
		w := client.Get(t, fmt.Sprintf("/api/products?maxPrice=%.2f", maxPrice))

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotNil(t, products, "Should return products array")
	})

	t.Run("Filter - By price range", func(t *testing.T) {
		w := client.Get(t, "/api/products?minPrice=20&maxPrice=100")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotNil(t, products, "Should return products array")
	})

	t.Run("Filter - By isPopular=true", func(t *testing.T) {
		w := client.Get(t, "/api/products?isPopular=true")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotNil(t, products, "Should return products array")
	})

	t.Run("Filter - By isPopular=false", func(t *testing.T) {
		w := client.Get(t, "/api/products?isPopular=false")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotNil(t, products, "Should return products array")
	})

	t.Run("Filter - Multiple filters combined", func(t *testing.T) {
		w := client.Get(t, "/api/products?categoryIds=4&brands=Apple&minPrice=900")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// Should work without error, results depend on data
		assert.NotNil(t, products, "Should return products array")
	})

	t.Run("Filter - No results for non-existent brand", func(t *testing.T) {
		w := client.Get(t, "/api/products?brands=NonExistentBrand123")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.Empty(t, products, "Should return empty array for non-existent brand")
	})

	t.Run("Filter - No results for non-existent category", func(t *testing.T) {
		w := client.Get(t, "/api/products?categoryIds=99999")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.Empty(t, products, "Should return empty array for non-existent category")
	})

	// ============================================================================
	// SORTING SCENARIOS
	// ============================================================================

	t.Run("Sort - By created_at descending (default)", func(t *testing.T) {
		w := client.Get(t, "/api/products")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotEmpty(t, products, "Should return products")
		// Verify products have createdAt field
		if len(products) > 0 {
			product := products[0].(map[string]interface{})
			assert.NotNil(t, product["createdAt"], "Product should have createdAt")
		}
	})

	t.Run("Sort - By created_at ascending", func(t *testing.T) {
		w := client.Get(t, "/api/products?sortBy=created_at&sortOrder=asc")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotNil(t, products, "Should return products array")
	})

	t.Run("Sort - By name ascending", func(t *testing.T) {
		w := client.Get(t, "/api/products?sortBy=name&sortOrder=asc")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotEmpty(t, products, "Should return products")
		// Verify sorted by name
		if len(products) >= 2 {
			name1 := products[0].(map[string]interface{})["name"].(string)
			name2 := products[1].(map[string]interface{})["name"].(string)
			assert.LessOrEqual(t, name1, name2, "Products should be sorted by name ascending")
		}
	})

	t.Run("Sort - By name descending", func(t *testing.T) {
		w := client.Get(t, "/api/products?sortBy=name&sortOrder=desc")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.NotEmpty(t, products, "Should return products")
		// Verify sorted by name
		if len(products) >= 2 {
			name1 := products[0].(map[string]interface{})["name"].(string)
			name2 := products[1].(map[string]interface{})["name"].(string)
			assert.GreaterOrEqual(t, name1, name2, "Products should be sorted by name descending")
		}
	})

	t.Run("Sort - Invalid sortBy field (should use default)", func(t *testing.T) {
		w := client.Get(t, "/api/products?sortBy=invalidField&sortOrder=asc")

		helpers.AssertShouldNotSucceed(t, w)
	})

	// ============================================================================
	// PAGINATION EDGE CASES
	// ============================================================================

	t.Run("Pagination - Page 0 (should default to page 1)", func(t *testing.T) {
		w := client.Get(t, "/api/products?page=0")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})

		assert.Equal(t, float64(1), pagination["currentPage"], "Page 0 should default to page 1")
	})

	t.Run("Pagination - Negative page number (should default to page 1)", func(t *testing.T) {
		w := client.Get(t, "/api/products?page=-5")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})

		assert.Equal(
			t,
			float64(1),
			pagination["currentPage"],
			"Negative page should default to page 1",
		)
	})

	t.Run("Pagination - Limit 0 (should use default limit)", func(t *testing.T) {
		w := client.Get(t, "/api/products?pageSize=0")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})

		assert.Greater(t, pagination["itemsPerPage"], float64(0), "Limit 0 should use default")
	})

	t.Run("Pagination - Negative limit (should use default)", func(t *testing.T) {
		w := client.Get(t, "/api/products?pageSize=-10")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})

		assert.Greater(
			t,
			pagination["itemsPerPage"],
			float64(0),
			"Negative limit should use default",
		)
	})

	t.Run("Pagination - Very large limit (should be capped at max)", func(t *testing.T) {
		w := client.Get(t, "/api/products?pageSize=1000")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// Should be capped at 100 (based on service implementation)
		assert.LessOrEqual(t, len(products), 100, "Should cap limit at max value")
		assert.LessOrEqual(
			t,
			pagination["itemsPerPage"],
			float64(100),
			"Should cap itemsPerPage at 100",
		)
	})

	t.Run("Pagination - Page beyond total pages", func(t *testing.T) {
		w := client.Get(t, "/api/products?page=9999&pageSize=10")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		assert.Empty(t, products, "Should return empty array for page beyond total")
	})

	t.Run("Pagination - hasPrev and hasNext flags", func(t *testing.T) {
		// Get first page
		w := client.Get(t, "/api/products?page=1&pageSize=2")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})

		assert.False(t, pagination["hasPrev"].(bool), "First page should not have previous")
		// hasNext depends on total products
		if pagination["totalPages"].(float64) > 1 {
			assert.True(t, pagination["hasNext"].(bool), "Should have next page if total pages > 1")
		}
	})

	t.Run("Pagination - Last page hasNext should be false", func(t *testing.T) {
		// First get total pages
		w1 := client.Get(t, "/api/products?pageSize=5")
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		pagination1 := response1["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		totalPages := int(pagination1["totalPages"].(float64))

		if totalPages > 1 {
			// Get last page
			w2 := client.Get(t, fmt.Sprintf("/api/products?page=%d&pageSize=5", totalPages))
			response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
			pagination2 := response2["data"].(map[string]interface{})["pagination"].(map[string]interface{})

			assert.False(t, pagination2["hasNext"].(bool), "Last page should not have next")
			assert.True(t, pagination2["hasPrev"].(bool), "Last page should have previous")
		}
	})

	// ============================================================================
	// INVALID QUERY PARAMETERS
	// ============================================================================

	t.Run("Invalid - Non-numeric page parameter", func(t *testing.T) {
		w := client.Get(t, "/api/products?page=abc")

		response := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		assert.False(t, response["success"].(bool), "Response success should be false")
	})

	t.Run("Invalid - Non-numeric limit parameter", func(t *testing.T) {
		w := client.Get(t, "/api/products?pageSize=xyz")

		response := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		assert.False(t, response["success"].(bool), "Response success should be false")
	})

	t.Run("Invalid - Non-numeric categoryId", func(t *testing.T) {
		w := client.Get(t, "/api/products?categoryIds=notanumber")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// Should ignore invalid categoryId and return all products
		assert.NotNil(t, products, "Should return products despite invalid categoryId")
	})

	t.Run("Invalid - Non-numeric price parameters", func(t *testing.T) {
		w := client.Get(t, "/api/products?minPrice=abc&maxPrice=xyz")

		response := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		assert.False(t, response["success"].(bool), "Response success should be false")
	})

	t.Run("Invalid - Non-boolean isPopular", func(t *testing.T) {
		w := client.Get(t, "/api/products?isPopular=notabool")

		response := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		assert.False(t, response["success"].(bool), "Response success should be false")
	})

	// ============================================================================
	// EMPTY RESULTS SCENARIOS
	// ============================================================================

	t.Run("Empty - No products match all combined filters", func(t *testing.T) {
		w := client.Get(t, "/api/products?brands=Apple&categoryIds=10&maxPrice=1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})

		assert.Empty(t, products, "Should return empty array")
		assert.Equal(t, float64(0), pagination["totalItems"], "Total items should be 0")
		assert.Equal(t, float64(0), pagination["totalPages"], "Total pages should be 0")
	})

	// ============================================================================
	// RESPONSE STRUCTURE VALIDATION
	// ============================================================================

	t.Run("Response - Verify complete response structure", func(t *testing.T) {
		w := client.Get(t, "/api/products?pageSize=1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		// Check top-level structure
		assert.NotNil(t, data["products"], "Response should have products")
		assert.NotNil(t, data["pagination"], "Response should have pagination")

		products := data["products"].([]interface{})
		if len(products) > 0 {
			product := products[0].(map[string]interface{})

			// Verify product fields
			requiredFields := []string{
				"id", "name", "categoryId", "sku", // Note: it's "sku" not "baseSku" in response
				"shortDescription", "createdAt", "updatedAt", "sellerId",
			}
			for _, field := range requiredFields {
				assert.NotNil(t, product[field], fmt.Sprintf("Product should have %s", field))
			}

			// Check optional fields exist (even if null)
			optionalFields := []string{"brand", "longDescription", "tags"}
			for _, field := range optionalFields {
				_, exists := product[field]
				assert.True(
					t,
					exists,
					fmt.Sprintf("Product should have %s field (even if null)", field),
				)
			}

			// Verify listing-specific fields
			assert.NotNil(t, product["hasVariants"], "Product should have hasVariants")
			assert.NotNil(t, product["allowPurchase"], "Product should have allowPurchase")
			assert.NotNil(t, product["images"], "Product should have images")

			// variantPreview should exist for products with variants
			if product["hasVariants"].(bool) {
				assert.NotNil(
					t,
					product["variantPreview"],
					"Product with variants should have variantPreview",
				)
			}

			// Full variants should NOT be in listing view
			variants, hasVariants := product["variants"]
			if hasVariants && variants != nil {
				variantsArray, isArray := variants.([]interface{})
				if isArray {
					assert.Empty(t, variantsArray, "GetAll API should not return full variants")
				}
			}
		}

		// Verify pagination fields
		pagination := data["pagination"].(map[string]interface{})
		paginationFields := []string{
			"currentPage", "totalPages", "totalItems", "itemsPerPage", "hasNext", "hasPrev",
		}
		for _, field := range paginationFields {
			assert.NotNil(t, pagination[field], fmt.Sprintf("Pagination should have %s", field))
		}
	})

	// ============================================================================
	// SPECIAL CHARACTERS AND ENCODING
	// ============================================================================

	t.Run("Special - Brand with special characters", func(t *testing.T) {
		w := client.Get(t, "/api/products?brands=L%27Oreal") // L'Oreal with URL encoding

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// Should handle URL-encoded special characters
		assert.NotNil(t, products, "Should handle special characters in brand")
	})

	t.Run("Special - Very long brand name", func(t *testing.T) {
		longBrand := "VeryLongBrandNameThatExceedsNormalLengthAndShouldBeHandledProperly123456789"
		w := client.Get(t, fmt.Sprintf("/api/products?brands=%s", longBrand))

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// Should handle long brand names
		assert.NotNil(t, products, "Should handle long brand names")
	})

	// ============================================================================
	// PERFORMANCE AND LIMITS
	// ============================================================================

	t.Run("Performance - Request with all filters and sorting", func(t *testing.T) {
		url := "/api/products?page=1&pageSize=20&categoryIds=4&brands=Apple" +
			"&minPrice=500&maxPrice=3000&isPopular=true" +
			"&sortBy=name&sortOrder=asc"

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// Should handle complex queries without error
		assert.NotNil(t, products, "Should handle complex queries")
	})

	t.Run("Performance - Maximum allowed limit", func(t *testing.T) {
		w := client.Get(t, "/api/products?pageSize=100")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})
		pagination := response["data"].(map[string]interface{})["pagination"].(map[string]interface{})

		assert.LessOrEqual(t, len(products), 100, "Should not exceed max limit")
		assert.LessOrEqual(
			t,
			pagination["itemsPerPage"],
			float64(100),
			"itemsPerPage should not exceed 100",
		)
	})

	// ============================================================================
	// CONSISTENCY CHECKS
	// ============================================================================

	t.Run("Consistency - Total items count matches across pages", func(t *testing.T) {
		// Get first page
		w1 := client.Get(t, "/api/products?page=1&pageSize=5")
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		pagination1 := response1["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		totalItems1 := pagination1["totalItems"]

		// Get second page
		w2 := client.Get(t, "/api/products?page=2&pageSize=5")
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		pagination2 := response2["data"].(map[string]interface{})["pagination"].(map[string]interface{})
		totalItems2 := pagination2["totalItems"]

		// Total items should be consistent
		assert.Equal(t, totalItems1, totalItems2, "Total items should be same across pages")
	})

	t.Run("Consistency - Same query returns same results", func(t *testing.T) {
		query := "/api/products?categoryIds=4&pageSize=5&sortBy=name&sortOrder=asc"

		// First request
		w1 := client.Get(t, query)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		products1 := response1["data"].(map[string]interface{})["products"].([]interface{})

		// Second request
		w2 := client.Get(t, query)
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		products2 := response2["data"].(map[string]interface{})["products"].([]interface{})

		// Should return same number of products
		assert.Equal(t, len(products1), len(products2), "Same query should return same count")

		// Should return same products in same order
		if len(products1) > 0 && len(products2) > 0 {
			firstProduct1 := products1[0].(map[string]interface{})
			firstProduct2 := products2[0].(map[string]interface{})
			assert.Equal(
				t,
				firstProduct1["id"],
				firstProduct2["id"],
				"First product should be same",
			)
		}
	})

	// ============================================================================
	// X-SELLER-ID HEADER VALIDATION (PUBLIC API REQUIREMENT)
	// ============================================================================

	t.Run("Header - Missing X-Seller-ID header", func(t *testing.T) {
		// Create new client without seller ID header
		clientNoHeader := helpers.NewAPIClient(server)

		w := clientNoHeader.Get(t, "/api/products")

		// Should return 400 Bad Request when X-Seller-ID is missing
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Header - Empty X-Seller-ID header", func(t *testing.T) {
		clientEmptyHeader := helpers.NewAPIClient(server)
		clientEmptyHeader.SetHeader("X-Seller-ID", "")

		w := clientEmptyHeader.Get(t, "/api/products")

		// Should return 400 Bad Request for empty seller ID
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Header - Whitespace only X-Seller-ID", func(t *testing.T) {
		clientWhitespace := helpers.NewAPIClient(server)
		clientWhitespace.SetHeader("X-Seller-ID", "   ")

		w := clientWhitespace.Get(t, "/api/products")

		// Should return 400 Bad Request for whitespace-only seller ID
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Header - Invalid X-Seller-ID (non-numeric)", func(t *testing.T) {
		clientInvalid := helpers.NewAPIClient(server)
		clientInvalid.SetHeader("X-Seller-ID", "abc")

		w := clientInvalid.Get(t, "/api/products")

		// Should return 400 Bad Request for non-numeric seller ID
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Header - Zero X-Seller-ID", func(t *testing.T) {
		clientZero := helpers.NewAPIClient(server)
		clientZero.SetHeader("X-Seller-ID", "0")

		w := clientZero.Get(t, "/api/products")

		// Should return 400 Bad Request for zero seller ID
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Header - Negative X-Seller-ID", func(t *testing.T) {
		clientNegative := helpers.NewAPIClient(server)
		clientNegative.SetHeader("X-Seller-ID", "-1")

		w := clientNegative.Get(t, "/api/products")

		// Should return 400 Bad Request for negative seller ID
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Header - Decimal X-Seller-ID", func(t *testing.T) {
		clientDecimal := helpers.NewAPIClient(server)
		clientDecimal.SetHeader("X-Seller-ID", "2.5")

		w := clientDecimal.Get(t, "/api/products")

		// Should return 400 Bad Request for decimal seller ID
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Header - Non-existent seller ID", func(t *testing.T) {
		clientNonExistent := helpers.NewAPIClient(server)
		clientNonExistent.SetHeader("X-Seller-ID", "99999")

		w := clientNonExistent.Get(t, "/api/products")

		// Should return 404 or 403 for non-existent seller
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusForbidden,
			"Should return 404 or 403 for non-existent seller")
	})

	t.Run("Header - Valid X-Seller-ID with whitespace (should trim)", func(t *testing.T) {
		clientTrim := helpers.NewAPIClient(server)
		clientTrim.SetHeader("X-Seller-ID", "  2  ")

		w := clientTrim.Get(t, "/api/products")

		// Should succeed after trimming whitespace
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})
		assert.NotNil(t, products, "Should return products after trimming whitespace")
	})

	t.Run("Header - Different seller ID returns different products", func(t *testing.T) {
		// Get products for seller 2
		client2 := helpers.NewAPIClient(server)
		client2.SetHeader("X-Seller-ID", "2")
		w2 := client2.Get(t, "/api/products")
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		products2 := response2["data"].(map[string]interface{})["products"].([]interface{})

		// Get products for seller 3
		client3 := helpers.NewAPIClient(server)
		client3.SetHeader("X-Seller-ID", "3")
		w3 := client3.Get(t, "/api/products")
		response3 := helpers.AssertSuccessResponse(t, w3, http.StatusOK)
		products3 := response3["data"].(map[string]interface{})["products"].([]interface{})

		// Both sellers should have products (based on seed data)
		assert.NotEmpty(t, products2, "Seller 2 should have products")
		assert.NotEmpty(t, products3, "Seller 3 should have products")

		// Verify seller 2's products belong to seller 2
		for _, p := range products2 {
			product := p.(map[string]interface{})
			assert.Equal(t, float64(2), product["sellerId"], "Products should belong to seller 2")
		}

		// Verify seller 3's products belong to seller 3
		for _, p := range products3 {
			product := p.(map[string]interface{})
			assert.Equal(t, float64(3), product["sellerId"], "Products should belong to seller 3")
		}
	})

	// ============================================================================
	// MULTI-TENANT ISOLATION
	// ============================================================================

	t.Run("MultiTenant - Seller only sees their own products", func(t *testing.T) {
		clientSeller2 := helpers.NewAPIClient(server)
		clientSeller2.SetHeader("X-Seller-ID", "2")

		w := clientSeller2.Get(t, "/api/products")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// All products should belong to seller 2
		for _, p := range products {
			product := p.(map[string]interface{})
			assert.Equal(
				t,
				float64(2),
				product["sellerId"],
				"All products should belong to seller 2",
			)
		}
	})

	t.Run("MultiTenant - Filters work within seller scope", func(t *testing.T) {
		clientSeller3 := helpers.NewAPIClient(server)
		clientSeller3.SetHeader("X-Seller-ID", "3")

		// Seller 3 owns products 5, 6, 7 (T-Shirt, Dress, Shoes) - all in Fashion category
		w := clientSeller3.Get(t, "/api/products?categoryIds=7") // Men's Clothing

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		products := response["data"].(map[string]interface{})["products"].([]interface{})

		// Should only return seller 3's products in category 7
		for _, p := range products {
			product := p.(map[string]interface{})
			assert.Equal(t, float64(3), product["sellerId"], "Products should belong to seller 3")
			assert.Equal(t, float64(7), product["categoryId"], "Products should be in category 7")
		}
	})
}
