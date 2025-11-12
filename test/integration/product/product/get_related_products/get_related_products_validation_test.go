package product

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetRelatedProductsValidation tests response structure, data validation, security, and integration
// This file contains tests for response validation, security concerns, and system integration
func TestGetRelatedProductsValidation(t *testing.T) {
	// Setup test containers (PostgreSQL + Redis)
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run all migrations including the stored procedure
	containers.RunAllMigrations(t)

	// Run seeds with comprehensive test data
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")
	containers.RunSeeds(t, "migrations/seeds/002_seed_product_data.sql")
	containers.RunSeeds(t, "migrations/seeds/003_seed_related_products_test_data.sql")

	// Setup test server with real database and Redis
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// CATEGORY 8: RESPONSE STRUCTURE TESTS
	// ============================================================================

	t.Run("[Response Structure] - Required fields present", func(t *testing.T) {
		// GRP-RS-001
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		// Verify top-level structure
		assert.Contains(t, data, "relatedProducts", "Response should have relatedProducts")
		assert.Contains(t, data, "pagination", "Response should have pagination")
		assert.Contains(t, data, "meta", "Response should have meta")

		// Verify pagination structure
		pagination := data["pagination"].(map[string]interface{})
		assert.Contains(t, pagination, "currentPage")
		assert.Contains(t, pagination, "totalPages")
		assert.Contains(t, pagination, "totalItems")
		assert.Contains(t, pagination, "itemsPerPage")
		assert.Contains(t, pagination, "hasNext")
		assert.Contains(t, pagination, "hasPrev")

		// Verify meta structure
		meta := data["meta"].(map[string]interface{})
		assert.Contains(t, meta, "strategiesUsed")
		assert.Contains(t, meta, "totalStrategies")
	})

	t.Run("[Response Structure] - Product object structure", func(t *testing.T) {
		// GRP-RS-002
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		if len(relatedProducts) > 0 {
			product := relatedProducts[0].(map[string]interface{})

			// Verify required product fields
			assert.Contains(t, product, "id")
			assert.Contains(t, product, "name")
			assert.Contains(
				t,
				product,
				"priceRange",
				"Should have priceRange instead of price for products with variants",
			)
			assert.Contains(t, product, "categoryId")
			assert.Contains(t, product, "brand", "Should have brand name instead of brandId")
			assert.Contains(t, product, "sellerId")
			assert.Contains(t, product, "strategyUsed")
			assert.Contains(t, product, "score")

			// Verify priceRange structure
			priceRange := product["priceRange"].(map[string]interface{})
			assert.Contains(t, priceRange, "min", "priceRange should have min price")
			assert.Contains(t, priceRange, "max", "priceRange should have max price")
		}
	})

	t.Run("[Response Structure] - Score field format", func(t *testing.T) {
		// GRP-RS-004
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			score := product["score"].(float64)

			// Score should be a positive number
			assert.Greater(t, score, float64(0), "Score should be positive")
			assert.IsType(t, float64(0), score, "Score should be numeric")
		}
	})

	t.Run("[Response Structure] - StrategyUsed field values", func(t *testing.T) {
		// GRP-RS-005
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		validStrategies := []string{
			"same_category",
			"same_brand",
			"sibling_category",
			"parent_category",
			"child_category",
			"tag_matching",
			"price_range",
			"seller_popular",
		}

		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			strategyUsed := product["strategyUsed"].(string)

			assert.Contains(t, validStrategies, strategyUsed,
				"Strategy '%s' should be a valid strategy", strategyUsed)
		}
	})

	t.Run("[Response Structure] - Pagination consistency", func(t *testing.T) {
		// GRP-RS-006
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?page=1&limit=10", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		currentPage := int(pagination["currentPage"].(float64))
		totalPages := int(pagination["totalPages"].(float64))
		totalItems := int(pagination["totalItems"].(float64))
		itemsPerPage := int(pagination["itemsPerPage"].(float64))
		hasNext := pagination["hasNext"].(bool)
		hasPrev := pagination["hasPrev"].(bool)

		// Verify math consistency
		if totalItems > 0 {
			expectedTotalPages := (totalItems + itemsPerPage - 1) / itemsPerPage
			assert.Equal(t, expectedTotalPages, totalPages,
				"Total pages should match calculation")
		}

		// Verify hasNext consistency
		if currentPage < totalPages {
			assert.True(t, hasNext, "hasNext should be true when not on last page")
		} else {
			assert.False(t, hasNext, "hasNext should be false on last page")
		}

		// Verify hasPrev consistency
		if currentPage > 1 {
			assert.True(t, hasPrev, "hasPrev should be true when not on first page")
		} else {
			assert.False(t, hasPrev, "hasPrev should be false on first page")
		}

		// Verify items count consistency
		assert.LessOrEqual(t, len(relatedProducts), itemsPerPage,
			"Returned items should not exceed limit")
	})

	// ============================================================================
	// CATEGORY 9: DATA VALIDATION TESTS
	// ============================================================================

	t.Run("[Data Validation] - No duplicate product IDs", func(t *testing.T) {
		// GRP-DV-001
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?limit=100", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// Track product IDs
		seenIDs := make(map[float64]bool)
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			productID := product["id"].(float64)

			assert.False(t, seenIDs[productID],
				"Product ID %.0f should only appear once", productID)
			seenIDs[productID] = true
		}
	})

	t.Run("[Data Validation] - Source product excluded from results", func(t *testing.T) {
		// GRP-DV-002
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		sourceProductID := 101
		url := fmt.Sprintf("/api/products/%d/related?limit=100", sourceProductID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// Verify source product is not in results
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			productID := product["id"].(float64)

			assert.NotEqual(t, float64(sourceProductID), productID,
				"Source product (ID: %d) should not appear in results", sourceProductID)
		}
	})

	t.Run("[Data Validation] - All products belong to same seller", func(t *testing.T) {
		// GRP-DV-003
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101 // Seller 2's product
		url := fmt.Sprintf("/api/products/%d/related?limit=100", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// All products should belong to Seller 2
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			sellerID := product["sellerId"].(float64)

			assert.Equal(t, float64(2), sellerID,
				"All products should belong to Seller 2 (sellerId=2)")
		}
	})

	// ============================================================================
	// CATEGORY 10: SECURITY TESTS
	// ============================================================================

	t.Run("[Security] - Missing X-Seller-ID Returns Wrong Status Code", func(t *testing.T) {
		// GRP-SEC-001
		// Remove authentication to test missing X-Seller-ID header
		client.SetToken("")

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		// Should return 400 Bad Request (not 401 Unauthorized)
		// X-Seller-ID is a required business parameter, not an auth credential
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("[Security] - Invalid seller token rejected", func(t *testing.T) {
		// GRP-SEC-003
		// Use an invalid/malformed token
		client.SetToken("invalid_token_12345")

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		// Should return 401 Unauthorized
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("[Security] - SQL injection in product ID", func(t *testing.T) {
		// GRP-SEC-004
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Try SQL injection patterns
		url := "/api/products/1' OR '1'='1/related"

		w := client.Get(t, url)

		// Should return 400 Bad Request (invalid ID format)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("[Security] - SQL injection in query parameters", func(t *testing.T) {
		// GRP-SEC-005
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf(
			"/api/products/%d/related?strategies=same_category' OR '1'='1",
			productID,
		)

		w := client.Get(t, url)

		// Should return 400 Bad Request (invalid strategy)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// CATEGORY 11: INTEGRATION TESTS
	// ============================================================================

	t.Run("[Integration] - Complete workflow with multiple pages", func(t *testing.T) {
		// GRP-INT-001
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101

		// Step 1: Get first page
		url1 := fmt.Sprintf("/api/products/%d/related?page=1&limit=10", productID)
		w1 := client.Get(t, url1)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		data1 := response1["data"].(map[string]interface{})
		pagination1 := data1["pagination"].(map[string]interface{})

		totalPages := int(pagination1["totalPages"].(float64))

		// Step 2: If multiple pages exist, get second page
		if totalPages > 1 {
			url2 := fmt.Sprintf("/api/products/%d/related?page=2&limit=10", productID)
			w2 := client.Get(t, url2)
			response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
			data2 := response2["data"].(map[string]interface{})
			products2 := data2["relatedProducts"].([]interface{})

			assert.NotEmpty(t, products2, "Second page should have products")
		}

		// Step 3: Get last page
		urlLast := fmt.Sprintf("/api/products/%d/related?page=%d&limit=10", productID, totalPages)
		wLast := client.Get(t, urlLast)
		responseLast := helpers.AssertSuccessResponse(t, wLast, http.StatusOK)
		dataLast := responseLast["data"].(map[string]interface{})
		paginationLast := dataLast["pagination"].(map[string]interface{})

		assert.Equal(t, false, paginationLast["hasNext"],
			"Last page should not have next")
	})

	t.Run("[Integration] - Strategy filtering workflow", func(t *testing.T) {
		// GRP-INT-002
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101

		// Step 1: Get all strategies
		urlAll := fmt.Sprintf("/api/products/%d/related", productID)
		wAll := client.Get(t, urlAll)
		responseAll := helpers.AssertSuccessResponse(t, wAll, http.StatusOK)
		dataAll := responseAll["data"].(map[string]interface{})
		productsAll := dataAll["relatedProducts"].([]interface{})
		metaAll := dataAll["meta"].(map[string]interface{})
		strategiesUsedAll := metaAll["strategiesUsed"].([]interface{})

		// Step 2: Filter by first available strategy
		if len(strategiesUsedAll) > 0 {
			firstStrategy := strategiesUsedAll[0].(string)
			urlFiltered := fmt.Sprintf("/api/products/%d/related?strategies=%s",
				productID, firstStrategy)
			wFiltered := client.Get(t, urlFiltered)
			responseFiltered := helpers.AssertSuccessResponse(t, wFiltered, http.StatusOK)
			dataFiltered := responseFiltered["data"].(map[string]interface{})
			productsFiltered := dataFiltered["relatedProducts"].([]interface{})

			// Filtered results should be subset
			assert.LessOrEqual(t, len(productsFiltered), len(productsAll),
				"Filtered results should not exceed total results")

			// All filtered products should use the selected strategy
			for _, item := range productsFiltered {
				product := item.(map[string]interface{})
				strategyUsed := product["strategyUsed"].(string)
				assert.Equal(t, firstStrategy, strategyUsed,
					"All products should use the selected strategy")
			}
		}
	})

	t.Run("[Integration] - Multi-seller isolation verification", func(t *testing.T) {
		// GRP-INT-003
		// Get products for Seller 2
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID1 := 101 // Seller 2's product
		url1 := fmt.Sprintf("/api/products/%d/related?limit=100", productID1)
		w1 := client.Get(t, url1)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		data1 := response1["data"].(map[string]interface{})
		products1 := data1["relatedProducts"].([]interface{})

		// Collect all product IDs from Seller 2
		seller2ProductIDs := make(map[float64]bool)
		for _, item := range products1 {
			product := item.(map[string]interface{})
			seller2ProductIDs[product["id"].(float64)] = true
		}

		// Get products for Seller 3
		seller3Token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(seller3Token)

		productID2 := 6 // Seller 3's product (if exists)
		url2 := fmt.Sprintf("/api/products/%d/related?limit=100", productID2)
		w2 := client.Get(t, url2)
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		data2 := response2["data"].(map[string]interface{})
		products2 := data2["relatedProducts"].([]interface{})

		// Verify no overlap between sellers
		for _, item := range products2 {
			product := item.(map[string]interface{})
			productID := product["id"].(float64)

			assert.False(t, seller2ProductIDs[productID],
				"Product %.0f from Seller 3 should not appear in Seller 2's results", productID)
		}
	})

	// ============================================================================
	// CATEGORY 12: PERFORMANCE & EDGE CASES
	// ============================================================================

	t.Run("[Edge Case] - Maximum limit boundary", func(t *testing.T) {
		// GRP-EC-005
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?limit=100", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		// Should handle maximum limit without error
		assert.LessOrEqual(t, len(relatedProducts), 100,
			"Should not exceed maximum limit")
		assert.Equal(t, float64(100), pagination["itemsPerPage"],
			"Items per page should match requested limit")
	})

	t.Run("[Edge Case] - Empty strategy result", func(t *testing.T) {
		// GRP-EC-008
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Use a product that might not have matches for a specific strategy
		productID := 150 // Budget product with possibly no price range matches
		url := fmt.Sprintf("/api/products/%d/related?strategies=price_range", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// Should return empty array without error
		assert.NotNil(t, relatedProducts, "relatedProducts should not be nil")
		// May be empty if no matches for this strategy
	})

	t.Run("[Performance] - Large result set handling", func(t *testing.T) {
		// GRP-PERF-001
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?limit=100", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// Should handle large result sets efficiently
		assert.LessOrEqual(t, len(relatedProducts), 100,
			"Should respect limit even with large dataset")

		// Verify all returned products are properly formatted
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			assert.Contains(t, product, "id")
			assert.Contains(t, product, "score")
			assert.Contains(t, product, "strategyUsed")
		}
	})
}
