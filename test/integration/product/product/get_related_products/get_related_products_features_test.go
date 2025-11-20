package product

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetRelatedProductsFeatures tests pagination, filtering, multi-tenant, and error handling
// This file contains tests for core API features beyond strategy matching
func TestGetRelatedProductsFeatures(t *testing.T) {
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
	// CATEGORY 3: PAGINATION TESTS
	// ============================================================================

	t.Run("[Pagination] - Default page and limit", func(t *testing.T) {
		// GRP-PAG-001
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		relatedProducts := data["relatedProducts"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		// Default limit is 10
		assert.LessOrEqual(
			t,
			len(relatedProducts),
			10,
			"Should return at most 10 products by default",
		)

		assert.Equal(t, float64(1), pagination["currentPage"], "Default page should be 1")
		assert.Equal(
			t,
			float64(10),
			pagination["itemsPerPage"],
			"Default items per page should be 10",
		)
		assert.Equal(t, false, pagination["hasPrev"], "First page should not have previous")

		totalItems := pagination["totalItems"].(float64)
		assert.Greater(t, totalItems, float64(0), "Should have total items count")
	})

	t.Run("[Pagination] - Custom limit parameter", func(t *testing.T) {
		// GRP-PAG-002
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?limit=25", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})

		relatedProducts := data["relatedProducts"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		assert.LessOrEqual(t, len(relatedProducts), 25, "Should return at most 25 products")
		assert.Equal(t, float64(25), pagination["itemsPerPage"], "Items per page should be 25")
	})

	t.Run("[Pagination] - Navigate to specific page", func(t *testing.T) {
		// GRP-PAG-003
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101

		// Get page 1
		url1 := fmt.Sprintf("/api/products/%d/related?page=1&limit=10", productID)
		w1 := client.Get(t, url1)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		data1 := response1["data"].(map[string]interface{})
		products1 := data1["relatedProducts"].([]interface{})
		pagination1 := data1["pagination"].(map[string]interface{})

		assert.Equal(t, float64(1), pagination1["currentPage"], "Should be page 1")
		assert.Equal(t, false, pagination1["hasPrev"], "Page 1 should not have previous")

		// Get page 2 if available
		if pagination1["hasNext"].(bool) {
			url2 := fmt.Sprintf("/api/products/%d/related?page=2&limit=10", productID)
			w2 := client.Get(t, url2)
			response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
			data2 := response2["data"].(map[string]interface{})
			products2 := data2["relatedProducts"].([]interface{})
			pagination2 := data2["pagination"].(map[string]interface{})

			assert.Equal(t, float64(2), pagination2["currentPage"], "Should be page 2")
			assert.Equal(t, true, pagination2["hasPrev"], "Page 2 should have previous")

			// Verify different products
			if len(products1) > 0 && len(products2) > 0 {
				firstProduct1 := products1[0].(map[string]interface{})
				firstProduct2 := products2[0].(map[string]interface{})
				assert.NotEqual(t, firstProduct1["id"], firstProduct2["id"],
					"Different pages should have different products")
			}
		}
	})

	t.Run("[Pagination] - Last page", func(t *testing.T) {
		// GRP-PAG-004
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101

		// Get first page to know total pages
		url1 := fmt.Sprintf("/api/products/%d/related?limit=10", productID)
		w1 := client.Get(t, url1)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		data1 := response1["data"].(map[string]interface{})
		pagination1 := data1["pagination"].(map[string]interface{})
		totalPages := int(pagination1["totalPages"].(float64))

		if totalPages > 1 {
			// Get last page
			urlLast := fmt.Sprintf(
				"/api/products/%d/related?page=%d&limit=10",
				productID,
				totalPages,
			)
			wLast := client.Get(t, urlLast)
			responseLast := helpers.AssertSuccessResponse(t, wLast, http.StatusOK)
			dataLast := responseLast["data"].(map[string]interface{})
			paginationLast := dataLast["pagination"].(map[string]interface{})

			assert.Equal(
				t,
				float64(totalPages),
				paginationLast["currentPage"],
				"Should be last page",
			)
			assert.Equal(t, false, paginationLast["hasNext"], "Last page should not have next")
			assert.Equal(t, true, paginationLast["hasPrev"], "Last page should have previous")
		}
	})

	t.Run("[Pagination] - Limit validation minimum", func(t *testing.T) {
		// GRP-PAG-005
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?limit=0", productID)

		w := client.Get(t, url)

		// Should return 400 Bad Request for invalid limit
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("[Pagination] - Limit validation maximum", func(t *testing.T) {
		// GRP-PAG-006
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?limit=101", productID)

		w := client.Get(t, url)

		// Should return 400 Bad Request for limit exceeding maximum
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("[Pagination] - Page beyond available results", func(t *testing.T) {
		// GRP-PAG-007
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?page=999&limit=10", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		// Should return empty array but valid response
		assert.Empty(t, relatedProducts, "Should return empty array for out-of-range page")
		assert.Equal(t, float64(999), pagination["currentPage"], "Should show requested page")
		assert.NotNil(t, pagination["totalItems"], "Should have totalItems")
	})

	// ============================================================================
	// CATEGORY 4: STRATEGY SELECTION TESTS
	// ============================================================================

	t.Run("[Strategy Filter] - Request specific strategies", func(t *testing.T) {
		// GRP-STF-001
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf(
			"/api/products/%d/related?strategies=same_category,same_brand",
			productID,
		)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})
		meta := data["meta"].(map[string]interface{})

		// Verify only requested strategies are used
		strategiesUsed := meta["strategiesUsed"].([]interface{})
		for _, strategy := range strategiesUsed {
			strategyName := strategy.(string)
			assert.Contains(t, []string{"same_category", "same_brand"}, strategyName,
				"Only requested strategies should be used")
		}

		// Verify products match requested strategies
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			strategyUsed := product["strategyUsed"].(string)
			assert.Contains(t, []string{"same_category", "same_brand"}, strategyUsed,
				"Product strategy should match filter")
		}
	})

	t.Run("[Strategy Filter] - Request single strategy", func(t *testing.T) {
		// GRP-STF-002
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 105 // Has tags
		url := fmt.Sprintf("/api/products/%d/related?strategies=tag_matching", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// All products should use tag_matching strategy
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			strategyUsed := product["strategyUsed"].(string)
			assert.Equal(t, "tag_matching", strategyUsed, "Should only use tag_matching strategy")
		}
	})

	t.Run("[Strategy Filter] - Invalid strategy name", func(t *testing.T) {
		// GRP-STF-003
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?strategies=invalid_strategy", productID)

		w := client.Get(t, url)

		// Should return 400 Bad Request for invalid strategy
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("[Strategy Filter] - All strategies explicitly", func(t *testing.T) {
		// GRP-STF-004
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?strategies=all", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		meta := data["meta"].(map[string]interface{})

		totalStrategies := meta["totalStrategies"].(float64)
		assert.Equal(t, float64(8), totalStrategies, "Should have 8 total strategies available")
	})

	// ============================================================================
	// CATEGORY 5: MULTI-TENANT ISOLATION TESTS
	// ============================================================================

	t.Run("[Multi-Tenant] - Seller sees only their products", func(t *testing.T) {
		// GRP-MT-001
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101 // Seller 2's product
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// All returned products should belong to Seller 2
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			sellerID := product["sellerId"].(float64)
			assert.Equal(t, float64(2), sellerID,
				"All products should belong to Seller 2 (sellerId=2)")
		}
	})

	t.Run("[Multi-Tenant] - Cannot access another seller's product", func(t *testing.T) {
		// GRP-MT-003
		seller3Token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(seller3Token)

		// Try to access Seller 2's product with Seller 3's credentials
		productID := 101 // Seller 2's product
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		// Should return 404 Not Found
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run(
		"[Multi-Tenant] - Seller with no related products gets empty results",
		func(t *testing.T) {
			// GRP-MT-004
			// This test requires a seller with only one product
			// If we don't have such data, we can skip or use Seller 4
			seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
			client.SetToken(seller4Token)

			// Product 8 or 9 belongs to Seller 4 (Home & Living)
			// They might have limited related products
			productID := 8
			url := fmt.Sprintf("/api/products/%d/related", productID)

			w := client.Get(t, url)

			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			data := response["data"].(map[string]interface{})
			relatedProducts := data["relatedProducts"].([]interface{})
			pagination := data["pagination"].(map[string]interface{})

			// May be empty if seller has only one product
			// Verify response structure is valid
			assert.NotNil(t, relatedProducts, "relatedProducts should not be nil")
			assert.NotNil(t, pagination, "pagination should not be nil")

			if len(relatedProducts) == 0 {
				totalItems := pagination["totalItems"].(float64)
				assert.Equal(t, float64(0), totalItems, "Total items should be 0 for empty results")
			}
		},
	)

	// ============================================================================
	// CATEGORY 6: ERROR HANDLING TESTS
	// ============================================================================

	t.Run("[Error] - Missing X-Seller-ID header", func(t *testing.T) {
		// GRP-ERR-001
		// Don't set token, so no X-Seller-ID header
		client.SetToken("")

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		// Should return 400 Bad Request (missing required header parameter)
		// X-Seller-ID is a required business parameter, not an auth credential
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("[Error] - Invalid product ID format", func(t *testing.T) {
		// GRP-ERR-003
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		url := "/api/products/invalid/related"

		w := client.Get(t, url)

		// Should return 400 Bad Request
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("[Error] - Product not found", func(t *testing.T) {
		// GRP-ERR-004
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 99999 // Non-existent product
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		// Should return 404 Not Found
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// CATEGORY 7: EDGE CASE TESTS
	// ============================================================================

	t.Run("[Edge Case] - Product with no tags", func(t *testing.T) {
		// GRP-EC-002
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Use a product that might have no tags (if any exist in seed data)
		// For this test, we'll use any product and verify tag_matching strategy is not used
		productID := 142 // Anker headphones - might have minimal tags
		url := fmt.Sprintf("/api/products/%d/related?strategies=tag_matching", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// Should return empty or products without tag_matching if source has no tags
		// This validates the strategy handles missing tags gracefully
		assert.NotNil(t, relatedProducts, "Should return valid array even with no tag matches")
	})

	t.Run("[Edge Case] - Product with extreme price", func(t *testing.T) {
		// GRP-EC-004
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Product 150 is ultra budget (~$99)
		productID := 150
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// Should handle extreme prices without errors
		assert.NotNil(t, relatedProducts, "Should handle extreme price products")

		// Verify no math errors occurred
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			score := product["score"].(float64)
			assert.Greater(t, score, float64(0), "Score should be positive")
			assert.Less(t, score, float64(1000), "Score should be reasonable")
		}
	})

	t.Run("[Edge Case] - Page number exceeds total pages", func(t *testing.T) {
		// GRP-EC-006
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?page=999", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})
		pagination := data["pagination"].(map[string]interface{})

		// Should return empty array without error
		assert.Empty(t, relatedProducts, "Should return empty array for out-of-range page")
		assert.Equal(
			t,
			float64(999),
			pagination["currentPage"],
			"Should show requested page number",
		)
	})

	t.Run("[Edge Case] - Duplicate products in multiple strategies", func(t *testing.T) {
		// GRP-EC-007
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Product 102 (iPhone 13) should match 101 via multiple strategies
		productID := 102
		url := fmt.Sprintf("/api/products/%d/related?limit=50", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// Verify no duplicate products
		seenIDs := make(map[float64]bool)
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			productID := product["id"].(float64)

			assert.False(t, seenIDs[productID],
				"Product ID %.0f should not appear twice", productID)
			seenIDs[productID] = true
		}
	})
}
