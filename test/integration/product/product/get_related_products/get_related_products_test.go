package product

// // NOTE: This file has been split into multiple test files for better organization:
// //
// // 1. get_related_products_strategies_test.go - Strategy-based matching tests
// //    - Same Category, Same Brand, Sibling Category, Tag Matching, Price Range
// //    - Scoring and Ranking tests
// //
// // 2. get_related_products_features_test.go - Feature tests
// //    - Pagination tests (default, custom, navigation, validation)
// //    - Strategy filtering tests
// //    - Multi-tenant isolation tests
// //    - Error handling tests
// //    - Edge case tests
// //
// // 3. get_related_products_validation_test.go - Validation and security tests
// //    - Response structure validation
// //    - Data validation tests
// //    - Security tests
// //    - Integration tests
// //
// // All test files share the same setup using setupTestEnvironment() helper function.

// import (
// 	"fmt"
// 	"net/http"
// 	"testing"

// 	"ecommerce-be/test/integration/helpers"
// 	"ecommerce-be/test/integration/setup"

// 	"github.com/stretchr/testify/require"
// )

// // setupTestEnvironment creates and configures the test environment with database, Redis, and API client
// // This is shared across all get_related_products test files
// func setupTestEnvironment(t *testing.T) (*setup.TestContainers, *helpers.APIClient) {
// 	// Setup test containers (PostgreSQL + Redis)
// 	containers := setup.SetupTestContainers(t)

// 	// Run all migrations including the stored procedure
// 	containers.RunAllMigrations(t)

// 	// Run seeds with comprehensive test data
// 	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")
// 	containers.RunSeeds(t, "migrations/seeds/002_seed_product_data.sql")
// 	containers.RunSeeds(t, "migrations/seeds/003_seed_related_products_test_data.sql")

// 	// Setup test server with real database and Redis
// 	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

// 	// Create API client
// 	client := helpers.NewAPIClient(server)


// // Note: All test cases have been moved to separate files:
// // - get_related_products_strategies_test.go
// // - get_related_products_features_test.go
// // - get_related_products_validation_test.go
// 	t.Run("[Same Brand Strategy] - Get related products from same brand", func(t *testing.T) {
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 101 is iPhone 14 in Smartphones category (ID 4)
// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data, ok := response["data"].(map[string]interface{})
// 		require.True(t, ok, "Response should contain data object")

// 		relatedProducts, ok := data["relatedProducts"].([]interface{})
// 		require.True(t, ok, "Should have relatedProducts array")
// 		helpers.AssertCategoryFields.NotEmpty(t, relatedProducts, "Should have related products from same category")

// 		// Check that results include same category products
// 		foundSameCategory := false
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})

// 			// Verify required fields
// 			assert.NotNil(t, product["id"], "Product should have ID")
// 			assert.NotNil(t, product["score"], "Product should have score")
// 			assert.NotNil(t, product["strategyUsed"], "Product should have strategyUsed")
// 			assert.NotNil(t, product["relationReason"], "Product should have relationReason")

// 			// Check for same category matches
// 			if strategyUsed, ok := product["strategyUsed"].(string); ok {
// 				if strategyUsed == "same_category" {
// 					foundSameCategory = true
// 					score := product["score"].(float64)
// 					assert.GreaterOrEqual(
// 						t,
// 						score,
// 						100.0,
// 						"Same category should have base score >= 100",
// 					)
// 				}
// 			}
// 		}

// 		assert.True(t, foundSameCategory, "Should find at least one same category match")

// 		// Verify source product is excluded
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			assert.NotEqual(
// 				t,
// 				float64(productID),
// 				product["id"],
// 				"Source product should be excluded",
// 			)
// 		}

// 		// Check pagination metadata
// 		pagination, ok := data["pagination"].(map[string]interface{})
// 		require.True(t, ok, "Should have pagination metadata")
// 		assert.NotNil(t, pagination["currentPage"], "Should have currentPage")
// 		assert.NotNil(t, pagination["totalItems"], "Should have totalItems")

// 		// Check meta information
// 		meta, ok := data["meta"].(map[string]interface{})
// 		require.True(t, ok, "Should have meta object")
// 		strategiesUsed, ok := meta["strategiesUsed"].([]interface{})
// 		require.True(t, ok, "Should have strategiesUsed array")
// 		assert.NotEmpty(t, strategiesUsed, "Should have strategies used")
// 	})

// 	t.Run("[Same Brand Strategy] - Get related products from same brand", func(t *testing.T) {
// 		// GRP-STR-002
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 101 is iPhone 14 (Apple brand)
// 		// Should find other Apple products like MacBook, iPad, Apple Watch
// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// Look for same brand matches (Apple products from different categories)
// 		foundSameBrand := false
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})

// 			// Check if it's Apple brand from different category
// 			brand, hasBrand := product["brand"].(string)
// 			strategyUsed, hasStrategy := product["strategyUsed"].(string)

// 			if hasBrand && hasStrategy && brand == "Apple" && strategyUsed == "same_brand" {
// 				foundSameBrand = true
// 				score := product["score"].(float64)
// 				assert.GreaterOrEqual(t, score, 80.0, "Same brand should have base score >= 80")

// 				// Verify it's from different category (not smartphones)
// 				category := product["category"].(map[string]interface{})
// 				categoryID := category["id"].(float64)
// 				assert.NotEqual(
// 					t,
// 					float64(4),
// 					categoryID,
// 					"Same brand product should be from different category",
// 				)
// 			}
// 		}

// 		assert.True(
// 			t,
// 			foundSameBrand,
// 			"Should find at least one same brand match from different category",
// 		)
// 	})

// 	t.Run("[Sibling Category Strategy] - Get products from sibling categories", func(t *testing.T) {
// 		// GRP-STR-003
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 101 is in Smartphones (ID 4), sibling categories: Laptops (5), Tablets (12), etc.
// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		foundSibling := false
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			strategyUsed, ok := product["strategyUsed"].(string)

// 			if ok && strategyUsed == "sibling_category" {
// 				foundSibling = true
// 				score := product["score"].(float64)
// 				assert.GreaterOrEqual(
// 					t,
// 					score,
// 					70.0,
// 					"Sibling category should have base score >= 70",
// 				)

// 				relationReason, ok := product["relationReason"].(string)
// 				assert.True(t, ok, "Should have relationReason")
// 				assert.Contains(
// 					t,
// 					relationReason,
// 					"sibling",
// 					"Relation reason should mention sibling",
// 				)
// 			}
// 		}

// 		// Note: Sibling matches may not always be present depending on data
// 		// This is acceptable as it tests the strategy when applicable
// 		if foundSibling {
// 			t.Log("Found sibling category matches")
// 		}
// 	})

// 	t.Run("[Tag Matching Strategy] - Get products with common tags", func(t *testing.T) {
// 		// GRP-STR-006
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 143 (Canon EOS R6) has tags: ["camera", "canon", "mirrorless", "professional"]
// 		// Use tag_matching strategy explicitly to force tag-based matching only
// 		productID := 143
// 		url := fmt.Sprintf("/api/products/%d/related?strategies=tag_matching", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		foundTagMatch := false
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			strategyUsed, ok := product["strategyUsed"].(string)

// 			if ok && strategyUsed == "tag_matching" {
// 				foundTagMatch = true
// 				score := product["score"].(float64)
// 				assert.GreaterOrEqual(t, score, 20.0, "Tag matching should have base score >= 20")

// 				// Verify product has tags
// 				tags, hasTags := product["tags"].([]interface{})
// 				assert.True(t, hasTags, "Product should have tags")
// 				assert.NotEmpty(t, tags, "Tags array should not be empty")
// 			}
// 		}

// 		assert.True(t, foundTagMatch, "Should find at least one tag matching product")
// 	})

// 	t.Run("[Price Range Strategy] - Get products in similar price range", func(t *testing.T) {
// 		// GRP-STR-007
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 101 (iPhone 14) price range: ~799-899
// 		// Should find products in similar range (25% variance)
// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		foundPriceRange := false
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			strategyUsed, ok := product["strategyUsed"].(string)

// 			if ok && strategyUsed == "price_range" {
// 				foundPriceRange = true
// 				score := product["score"].(float64)
// 				assert.GreaterOrEqual(t, score, 20.0, "Price range should have base score >= 20")

// 				// Verify price range exists
// 				priceRange, ok := product["priceRange"].(map[string]interface{})
// 				assert.True(t, ok, "Product should have priceRange")
// 				assert.NotNil(t, priceRange["min"], "Should have min price")
// 				assert.NotNil(t, priceRange["max"], "Should have max price")
// 			}
// 		}

// 		// Price range matches may not always be present
// 		if foundPriceRange {
// 			t.Log("Found price range matches")
// 		}
// 	})

// 	// ============================================================================
// 	// CATEGORY 2: SCORING AND RANKING TESTS
// 	// ============================================================================

// 	t.Run(
// 		"[Bonus - Same Brand and Category] - Bonus for matching brand and category",
// 		func(t *testing.T) {
// 			// GRP-SCR-001
// 			seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 			client.SetToken(seller2Token)

// 			// Product 103 is Samsung Galaxy S23 (Samsung brand, Smartphones category)
// 			// Should find Samsung Galaxy S24, A54 with bonus points
// 			productID := 103
// 			url := fmt.Sprintf("/api/products/%d/related", productID)

// 			w := client.Get(t, url)

// 			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 			data := response["data"].(map[string]interface{})
// 			relatedProducts := data["relatedProducts"].([]interface{})

// 			foundBonusProduct := false
// 			for _, item := range relatedProducts {
// 				product := item.(map[string]interface{})
// 				brand, hasBrand := product["brand"].(string)
// 				category, hasCategory := product["category"].(map[string]interface{})

// 				if hasBrand && hasCategory && brand == "Samsung" {
// 					categoryID := category["id"].(float64)
// 					if categoryID == 4 { // Smartphones
// 						foundBonusProduct = true
// 						score := product["score"].(float64)
// 						// Base 100 + bonus 50 = 150
// 						assert.GreaterOrEqual(
// 							t,
// 							score,
// 							140.0,
// 							"Samsung smartphone should have high score with bonus",
// 						)
// 						t.Logf("Found Samsung smartphone with score: %.0f", score)
// 					}
// 				}
// 			}

// 			assert.True(t, foundBonusProduct, "Should find Samsung smartphone with bonus points")
// 		},
// 	)

// 	t.Run("[Score Ranking] - Products ordered by final score", func(t *testing.T) {
// 		// GRP-SCR-007
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=20", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		require.NotEmpty(t, relatedProducts, "Should have related products")

// 		// Verify products are ordered by descending score
// 		previousScore := 999999.0
// 		for i, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			score, ok := product["score"].(float64)
// 			require.True(t, ok, "Product should have score")

// 			assert.LessOrEqual(t, score, previousScore,
// 				"Product at index %d should have score <= previous score (%.0f <= %.0f)",
// 				i, score, previousScore)

// 			previousScore = score
// 		}
// 	})

// 	// ============================================================================
// 	// CATEGORY 3: PAGINATION TESTS
// 	// ============================================================================

// 	t.Run("[Pagination] - Default page and limit", func(t *testing.T) {
// 		// GRP-PAG-001
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})

// 		relatedProducts := data["relatedProducts"].([]interface{})
// 		pagination := data["pagination"].(map[string]interface{})

// 		// Default limit is 10
// 		assert.LessOrEqual(
// 			t,
// 			len(relatedProducts),
// 			10,
// 			"Should return at most 10 products by default",
// 		)

// 		assert.Equal(t, float64(1), pagination["currentPage"], "Default page should be 1")
// 		assert.Equal(
// 			t,
// 			float64(10),
// 			pagination["itemsPerPage"],
// 			"Default items per page should be 10",
// 		)
// 		assert.Equal(t, false, pagination["hasPrev"], "First page should not have previous")

// 		totalItems := pagination["totalItems"].(float64)
// 		assert.Greater(t, totalItems, float64(0), "Should have total items count")
// 	})

// 	t.Run("[Pagination] - Custom limit parameter", func(t *testing.T) {
// 		// GRP-PAG-002
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=25", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})

// 		relatedProducts := data["relatedProducts"].([]interface{})
// 		pagination := data["pagination"].(map[string]interface{})

// 		assert.LessOrEqual(t, len(relatedProducts), 25, "Should return at most 25 products")
// 		assert.Equal(t, float64(25), pagination["itemsPerPage"], "Items per page should be 25")
// 	})

// 	t.Run("[Pagination] - Navigate to specific page", func(t *testing.T) {
// 		// GRP-PAG-003
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101

// 		// Get page 1
// 		url1 := fmt.Sprintf("/api/products/%d/related?page=1&limit=10", productID)
// 		w1 := client.Get(t, url1)
// 		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
// 		data1 := response1["data"].(map[string]interface{})
// 		products1 := data1["relatedProducts"].([]interface{})
// 		pagination1 := data1["pagination"].(map[string]interface{})

// 		assert.Equal(t, float64(1), pagination1["currentPage"], "Should be page 1")
// 		assert.Equal(t, false, pagination1["hasPrev"], "Page 1 should not have previous")

// 		// Get page 2 if available
// 		if pagination1["hasNext"].(bool) {
// 			url2 := fmt.Sprintf("/api/products/%d/related?page=2&limit=10", productID)
// 			w2 := client.Get(t, url2)
// 			response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
// 			data2 := response2["data"].(map[string]interface{})
// 			products2 := data2["relatedProducts"].([]interface{})
// 			pagination2 := data2["pagination"].(map[string]interface{})

// 			assert.Equal(t, float64(2), pagination2["currentPage"], "Should be page 2")
// 			assert.Equal(t, true, pagination2["hasPrev"], "Page 2 should have previous")

// 			// Verify different products
// 			if len(products1) > 0 && len(products2) > 0 {
// 				firstProduct1 := products1[0].(map[string]interface{})
// 				firstProduct2 := products2[0].(map[string]interface{})
// 				assert.NotEqual(t, firstProduct1["id"], firstProduct2["id"],
// 					"Different pages should have different products")
// 			}
// 		}
// 	})

// 	t.Run("[Pagination] - Last page", func(t *testing.T) {
// 		// GRP-PAG-004
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101

// 		// Get first page to know total pages
// 		url1 := fmt.Sprintf("/api/products/%d/related?limit=10", productID)
// 		w1 := client.Get(t, url1)
// 		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
// 		data1 := response1["data"].(map[string]interface{})
// 		pagination1 := data1["pagination"].(map[string]interface{})
// 		totalPages := int(pagination1["totalPages"].(float64))

// 		if totalPages > 1 {
// 			// Get last page
// 			urlLast := fmt.Sprintf(
// 				"/api/products/%d/related?page=%d&limit=10",
// 				productID,
// 				totalPages,
// 			)
// 			wLast := client.Get(t, urlLast)
// 			responseLast := helpers.AssertSuccessResponse(t, wLast, http.StatusOK)
// 			dataLast := responseLast["data"].(map[string]interface{})
// 			paginationLast := dataLast["pagination"].(map[string]interface{})

// 			assert.Equal(
// 				t,
// 				float64(totalPages),
// 				paginationLast["currentPage"],
// 				"Should be last page",
// 			)
// 			assert.Equal(t, false, paginationLast["hasNext"], "Last page should not have next")
// 			assert.Equal(t, true, paginationLast["hasPrev"], "Last page should have previous")
// 		}
// 	})

// 	t.Run("[Pagination] - Limit validation minimum", func(t *testing.T) {
// 		// GRP-PAG-005
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=0", productID)

// 		w := client.Get(t, url)

// 		// Should return 400 Bad Request for invalid limit
// 		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
// 	})

// 	t.Run("[Pagination] - Limit validation maximum", func(t *testing.T) {
// 		// GRP-PAG-006
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=101", productID)

// 		w := client.Get(t, url)

// 		// Should return 400 Bad Request for limit exceeding maximum
// 		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
// 	})

// 	t.Run("[Pagination] - Page beyond available results", func(t *testing.T) {
// 		// GRP-PAG-007
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?page=999&limit=10", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})
// 		pagination := data["pagination"].(map[string]interface{})

// 		// Should return empty array but valid response
// 		assert.Empty(t, relatedProducts, "Should return empty array for out-of-range page")
// 		assert.Equal(t, float64(999), pagination["currentPage"], "Should show requested page")
// 		assert.NotNil(t, pagination["totalItems"], "Should have totalItems")
// 	})

// 	// ============================================================================
// 	// CATEGORY 4: STRATEGY SELECTION TESTS
// 	// ============================================================================

// 	t.Run("[Strategy Filter] - Request specific strategies", func(t *testing.T) {
// 		// GRP-STF-001
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf(
// 			"/api/products/%d/related?strategies=same_category,same_brand",
// 			productID,
// 		)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})
// 		meta := data["meta"].(map[string]interface{})

// 		// Verify only requested strategies are used
// 		strategiesUsed := meta["strategiesUsed"].([]interface{})
// 		for _, strategy := range strategiesUsed {
// 			strategyName := strategy.(string)
// 			assert.Contains(t, []string{"same_category", "same_brand"}, strategyName,
// 				"Only requested strategies should be used")
// 		}

// 		// Verify products match requested strategies
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			strategyUsed := product["strategyUsed"].(string)
// 			assert.Contains(t, []string{"same_category", "same_brand"}, strategyUsed,
// 				"Product strategy should match filter")
// 		}
// 	})

// 	t.Run("[Strategy Filter] - Request single strategy", func(t *testing.T) {
// 		// GRP-STF-002
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 105 // Has tags
// 		url := fmt.Sprintf("/api/products/%d/related?strategies=tag_matching", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// All products should use tag_matching strategy
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			strategyUsed := product["strategyUsed"].(string)
// 			assert.Equal(t, "tag_matching", strategyUsed, "Should only use tag_matching strategy")
// 		}
// 	})

// 	t.Run("[Strategy Filter] - Invalid strategy name", func(t *testing.T) {
// 		// GRP-STF-003
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?strategies=invalid_strategy", productID)

// 		w := client.Get(t, url)

// 		// Should return 400 Bad Request for invalid strategy
// 		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
// 	})

// 	t.Run("[Strategy Filter] - All strategies explicitly", func(t *testing.T) {
// 		// GRP-STF-004
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?strategies=all", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		meta := data["meta"].(map[string]interface{})

// 		totalStrategies := meta["totalStrategies"].(float64)
// 		assert.Equal(t, float64(8), totalStrategies, "Should have 8 total strategies available")
// 	})

// 	// ============================================================================
// 	// CATEGORY 5: MULTI-TENANT ISOLATION TESTS
// 	// ============================================================================

// 	t.Run("[Multi-Tenant] - Seller sees only their products", func(t *testing.T) {
// 		// GRP-MT-001
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101 // Seller 2's product
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// All returned products should belong to Seller 2
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			sellerID := product["sellerId"].(float64)
// 			assert.Equal(t, float64(2), sellerID,
// 				"All products should belong to Seller 2 (sellerId=2)")
// 		}
// 	})

// 	t.Run("[Multi-Tenant] - Cannot access another seller's product", func(t *testing.T) {
// 		// GRP-MT-003
// 		seller3Token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
// 		client.SetToken(seller3Token)

// 		// Try to access Seller 2's product with Seller 3's credentials
// 		productID := 101 // Seller 2's product
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		// Should return 404 Not Found
// 		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
// 	})

// 	t.Run(
// 		"[Multi-Tenant] - Seller with no related products gets empty results",
// 		func(t *testing.T) {
// 			// GRP-MT-004
// 			// This test requires a seller with only one product
// 			// If we don't have such data, we can skip or use Seller 4
// 			seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
// 			client.SetToken(seller4Token)

// 			// Product 8 or 9 belongs to Seller 4 (Home & Living)
// 			// They might have limited related products
// 			productID := 8
// 			url := fmt.Sprintf("/api/products/%d/related", productID)

// 			w := client.Get(t, url)

// 			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 			data := response["data"].(map[string]interface{})
// 			relatedProducts := data["relatedProducts"].([]interface{})
// 			pagination := data["pagination"].(map[string]interface{})

// 			// May be empty if seller has only one product
// 			// Verify response structure is valid
// 			assert.NotNil(t, relatedProducts, "relatedProducts should not be nil")
// 			assert.NotNil(t, pagination, "pagination should not be nil")

// 			if len(relatedProducts) == 0 {
// 				totalItems := pagination["totalItems"].(float64)
// 				assert.Equal(t, float64(0), totalItems, "Total items should be 0 for empty results")
// 			}
// 		},
// 	)

// 	// ============================================================================
// 	// CATEGORY 6: ERROR HANDLING TESTS
// 	// ============================================================================

// 	t.Run("[Error] - Missing X-Seller-ID header", func(t *testing.T) {
// 		// GRP-ERR-001
// 		// Don't set token, so no X-Seller-ID header
// 		client.SetToken("")

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		// Should return 400 Bad Request (missing required header parameter)
// 		// X-Seller-ID is a required business parameter, not an auth credential
// 		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
// 	})

// 	t.Run("[Error] - Invalid product ID format", func(t *testing.T) {
// 		// GRP-ERR-003
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		url := "/api/products/invalid/related"

// 		w := client.Get(t, url)

// 		// Should return 400 Bad Request
// 		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
// 	})

// 	t.Run("[Error] - Product not found", func(t *testing.T) {
// 		// GRP-ERR-004
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 99999 // Non-existent product
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		// Should return 404 Not Found
// 		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
// 	})

// 	// ============================================================================
// 	// CATEGORY 7: EDGE CASE TESTS
// 	// ============================================================================

// 	t.Run("[Edge Case] - Product with no tags", func(t *testing.T) {
// 		// GRP-EC-002
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Use a product that might have no tags (if any exist in seed data)
// 		// For this test, we'll use any product and verify tag_matching strategy is not used
// 		productID := 142 // Anker headphones - might have minimal tags
// 		url := fmt.Sprintf("/api/products/%d/related?strategies=tag_matching", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// Should return empty or products without tag_matching if source has no tags
// 		// This validates the strategy handles missing tags gracefully
// 		assert.NotNil(t, relatedProducts, "Should return valid array even with no tag matches")
// 	})

// 	t.Run("[Edge Case] - Product with extreme price", func(t *testing.T) {
// 		// GRP-EC-004
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 150 is ultra budget (~$99)
// 		productID := 150
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// Should handle extreme prices without errors
// 		assert.NotNil(t, relatedProducts, "Should handle extreme price products")

// 		// Verify no math errors occurred
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			score := product["score"].(float64)
// 			assert.Greater(t, score, float64(0), "Score should be positive")
// 			assert.Less(t, score, float64(1000), "Score should be reasonable")
// 		}
// 	})

// 	// TODO: Re-enable when inventory service is integrated
// 	// t.Run("[Edge Case] - All related products out of stock", func(t *testing.T) {
// 	// 	// GRP-EC-005
// 	// 	seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 	// 	client.SetToken(seller2Token)

// 	// 	// Product 148 and 149 are out of stock/discontinued
// 	// 	productID := 148
// 	// 	url := fmt.Sprintf("/api/products/%d/related", productID)

// 	// 	w := client.Get(t, url)

// 	// 	response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 	// 	data := response["data"].(map[string]interface{})
// 	// 	relatedProducts := data["relatedProducts"].([]interface{})

// 	// 	// Should still return products even if they're out of stock
// 	// 	// They should have penalty applied to scores
// 	// 	for _, item := range relatedProducts {
// 	// 		product := item.(map[string]interface{})
// 	// 		score := product["score"].(float64)
// 	// 		// Score should still be present but may be low due to penalty
// 	// 		assert.NotNil(t, score, "Score should be present even for out-of-stock products")
// 	// 	}
// 	// })

// 	t.Run("[Edge Case] - Page number exceeds total pages", func(t *testing.T) {
// 		// GRP-EC-006
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?page=999", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})
// 		pagination := data["pagination"].(map[string]interface{})

// 		// Should return empty array without error
// 		assert.Empty(t, relatedProducts, "Should return empty array for out-of-range page")
// 		assert.Equal(
// 			t,
// 			float64(999),
// 			pagination["currentPage"],
// 			"Should show requested page number",
// 		)
// 	})

// 	t.Run("[Edge Case] - Duplicate products in multiple strategies", func(t *testing.T) {
// 		// GRP-EC-007
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 102 (iPhone 13) should match 101 via multiple strategies
// 		productID := 102
// 		url := fmt.Sprintf("/api/products/%d/related?limit=50", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// Verify no duplicate products
// 		seenIDs := make(map[float64]bool)
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			productID := product["id"].(float64)

// 			assert.False(t, seenIDs[productID],
// 				"Product ID %.0f should not appear twice", productID)
// 			seenIDs[productID] = true
// 		}
// 	})

// 	// ============================================================================
// 	// CATEGORY 8: RESPONSE STRUCTURE VALIDATION TESTS
// 	// ============================================================================

// 	t.Run("[Response] - Verify complete product structure", func(t *testing.T) {
// 		// GRP-RS-001
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		require.NotEmpty(t, relatedProducts, "Should have related products")

// 		// Verify first product has all required fields
// 		product := relatedProducts[0].(map[string]interface{})

// 		// Core fields
// 		assert.NotNil(t, product["id"], "Should have id")
// 		assert.NotNil(t, product["name"], "Should have name")
// 		assert.NotNil(t, product["categoryId"], "Should have categoryId")
// 		assert.NotNil(t, product["category"], "Should have category object")
// 		assert.NotNil(t, product["brand"], "Should have brand")
// 		assert.NotNil(t, product["sku"], "Should have sku")
// 		assert.NotNil(t, product["shortDescription"], "Should have shortDescription")
// 		assert.NotNil(t, product["tags"], "Should have tags array")
// 		assert.NotNil(t, product["sellerId"], "Should have sellerId")

// 		// New scoring fields
// 		assert.NotNil(t, product["score"], "Should have score")
// 		assert.NotNil(t, product["strategyUsed"], "Should have strategyUsed")
// 		assert.NotNil(t, product["relationReason"], "Should have relationReason")

// 		// Variant preview
// 		assert.NotNil(t, product["priceRange"], "Should have priceRange")
// 		priceRange := product["priceRange"].(map[string]interface{})
// 		assert.NotNil(t, priceRange["min"], "Should have min price")
// 		assert.NotNil(t, priceRange["max"], "Should have max price")
// 	})

// 	t.Run("[Response] - Verify scoring fields", func(t *testing.T) {
// 		// GRP-RS-002
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})

// 			// Verify score is positive integer
// 			score, ok := product["score"].(float64)
// 			require.True(t, ok, "Score should be a number")
// 			assert.Greater(t, score, float64(0), "Score should be positive")

// 			// Verify strategyUsed is valid strategy name
// 			strategyUsed, ok := product["strategyUsed"].(string)
// 			require.True(t, ok, "strategyUsed should be a string")
// 			assert.NotEmpty(t, strategyUsed, "strategyUsed should not be empty")

// 			validStrategies := []string{
// 				"same_category", "same_brand", "sibling_category",
// 				"parent_category", "child_category", "tag_matching",
// 				"price_range", "seller_popular",
// 			}
// 			assert.Contains(t, validStrategies, strategyUsed,
// 				"strategyUsed should be a valid strategy")

// 			// Verify relationReason describes the match
// 			relationReason, ok := product["relationReason"].(string)
// 			require.True(t, ok, "relationReason should be a string")
// 			assert.NotEmpty(t, relationReason, "relationReason should not be empty")
// 		}
// 	})

// 	t.Run("[Response] - Verify pagination structure", func(t *testing.T) {
// 		// GRP-RS-003
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=10&page=1", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})

// 		pagination, ok := data["pagination"].(map[string]interface{})
// 		require.True(t, ok, "Should have pagination object")

// 		// Verify all pagination fields
// 		assert.NotNil(t, pagination["currentPage"], "Should have currentPage")
// 		assert.NotNil(t, pagination["totalPages"], "Should have totalPages")
// 		assert.NotNil(t, pagination["totalItems"], "Should have totalItems")
// 		assert.NotNil(t, pagination["itemsPerPage"], "Should have itemsPerPage")
// 		assert.NotNil(t, pagination["hasNext"], "Should have hasNext")
// 		assert.NotNil(t, pagination["hasPrev"], "Should have hasPrev")

// 		// Verify field types and values
// 		assert.Equal(t, float64(1), pagination["currentPage"], "currentPage should be 1")
// 		assert.Equal(t, float64(10), pagination["itemsPerPage"], "itemsPerPage should be 10")
// 		assert.Equal(t, false, pagination["hasPrev"], "hasPrev should be false on page 1")
// 	})

// 	t.Run("[Response] - Verify meta information", func(t *testing.T) {
// 		// GRP-RS-004
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})

// 		meta, ok := data["meta"].(map[string]interface{})
// 		require.True(t, ok, "Should have meta object")

// 		// Verify meta fields
// 		strategiesUsed, ok := meta["strategiesUsed"].([]interface{})
// 		require.True(t, ok, "Should have strategiesUsed array")
// 		assert.NotEmpty(t, strategiesUsed, "strategiesUsed should not be empty")

// 		avgScore, ok := meta["avgScore"].(float64)
// 		require.True(t, ok, "Should have avgScore")
// 		assert.Greater(t, avgScore, float64(0), "avgScore should be positive")

// 		totalStrategies, ok := meta["totalStrategies"].(float64)
// 		require.True(t, ok, "Should have totalStrategies")
// 		assert.Equal(t, float64(8), totalStrategies, "totalStrategies should be 8")
// 	})

// 	t.Run("[Response] - Verify empty response structure", func(t *testing.T) {
// 		// GRP-RS-005
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 150 might have no related products
// 		productID := 150
// 		url := fmt.Sprintf("/api/products/%d/related?strategies=child_category", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})

// 		relatedProducts := data["relatedProducts"].([]interface{})
// 		pagination := data["pagination"].(map[string]interface{})
// 		meta := data["meta"].(map[string]interface{})

// 		// Should have valid structure even if empty
// 		assert.NotNil(t, relatedProducts, "relatedProducts should not be nil")
// 		assert.NotNil(t, pagination, "pagination should not be nil")
// 		assert.NotNil(t, meta, "meta should not be nil")

// 		if len(relatedProducts) == 0 {
// 			assert.Equal(t, float64(0), pagination["totalItems"], "totalItems should be 0")

// 			strategiesUsed := meta["strategiesUsed"].([]interface{})
// 			// May be empty if no matches found
// 			assert.NotNil(t, strategiesUsed, "strategiesUsed should not be nil")
// 		}
// 	})

// 	// ============================================================================
// 	// CATEGORY 9: DATA VALIDATION TESTS
// 	// ============================================================================

// 	t.Run("[Validation] - Source product excluded from results", func(t *testing.T) {
// 		// GRP-DV-001
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=50", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// Verify source product is not in results
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			assert.NotEqual(t, float64(productID), product["id"],
// 				"Source product should be excluded from related products")
// 		}
// 	})

// 	t.Run("[Validation] - No duplicate products", func(t *testing.T) {
// 		// GRP-DV-004
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=50", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// Verify all product IDs are unique
// 		idCounts := make(map[float64]int)
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			id := product["id"].(float64)
// 			idCounts[id]++
// 		}

// 		for id, count := range idCounts {
// 			assert.Equal(t, 1, count,
// 				"Product ID %.0f should appear exactly once, found %d times", id, count)
// 		}
// 	})

// 	t.Run("[Validation] - Scores within valid range", func(t *testing.T) {
// 		// GRP-DV-005
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=50", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		require.NotEmpty(t, relatedProducts, "Should have products to validate scores")

// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			score := product["score"].(float64)

// 			// Base scores range from 15 to 100
// 			// With bonuses (+110 max) and penalties (-50 max)
// 			// Final score should be positive
// 			assert.Greater(t, score, float64(0), "Score should be positive")
// 			assert.Less(t, score, float64(250),
// 				"Score should be reasonable (base 100 + bonus 110 + margin)")
// 		}
// 	})

// 	// ============================================================================
// 	// CATEGORY 11: SECURITY TESTS
// 	// ============================================================================

// 	t.Run("[Security] - SQL injection in product ID", func(t *testing.T) {
// 		// GRP-SEC-001
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		url := "/api/products/1' OR '1'='1/related"

// 		w := client.Get(t, url)

// 		// Should return 400 Bad Request (invalid format)
// 		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
// 	})

// 	t.Run("[Security] - Path traversal attempt in product ID", func(t *testing.T) {
// 		// GRP-SEC-003
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		url := "/api/products/../../etc/passwd/related"

// 		w := client.Get(t, url)

// 		// Should return 400 or 404
// 		assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, w.Code,
// 			"Should reject path traversal attempts")
// 	})

// 	t.Run("[Security] - No sensitive data in error messages", func(t *testing.T) {
// 		// GRP-SEC-004
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Trigger various errors and check messages
// 		urls := []string{
// 			"/api/products/99999/related",                  // Product not found
// 			"/api/products/invalid/related",                // Invalid format
// 			"/api/products/101/related?limit=0",            // Invalid limit
// 			"/api/products/101/related?strategies=invalid", // Invalid strategy
// 		}

// 		for _, url := range urls {
// 			w := client.Get(t, url)

// 			// Parse error response
// 			response := helpers.ParseResponse(t, w.Body)
// 			errorMsg, ok := response["error"].(string)

// 			if ok && errorMsg != "" {
// 				// Verify no sensitive data leaked
// 				assert.NotContains(t, errorMsg, "database", "Should not expose database info")
// 				assert.NotContains(t, errorMsg, "postgres", "Should not expose DB type")
// 				assert.NotContains(t, errorMsg, "SELECT", "Should not expose SQL")
// 				assert.NotContains(t, errorMsg, "/home/", "Should not expose file paths")
// 				assert.NotContains(t, errorMsg, "panic", "Should not expose panic info")
// 			}
// 		}
// 	})

// 	t.Run("[Security] - Authorization bypass attempt", func(t *testing.T) {
// 		// GRP-SEC-006
// 		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
// 		client.SetToken(sellerToken)

// 		// Try to access Seller 2's product (Seller 3 trying to access Seller 2's data)
// 		productID := 101 // Seller 2's product
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		// Should return 404 (product not found for this seller)
// 		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
// 	})

// 	// ============================================================================
// 	// CATEGORY 12: INTEGRATION TESTS
// 	// ============================================================================

// 	t.Run("[Integration] - Stored procedure execution", func(t *testing.T) {
// 		// GRP-INT-001
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related?limit=20", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})
// 		meta := data["meta"].(map[string]interface{})

// 		// Verify stored procedure executed successfully
// 		assert.NotEmpty(t, relatedProducts, "Stored procedure should return results")

// 		// Verify multiple strategies were evaluated
// 		strategiesUsed := meta["strategiesUsed"].([]interface{})
// 		assert.NotEmpty(t, strategiesUsed, "Should have used at least one strategy")

// 		// Verify deduplication worked
// 		idMap := make(map[float64]bool)
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})
// 			id := product["id"].(float64)
// 			assert.False(t, idMap[id], "Product should appear only once (deduplication)")
// 			idMap[id] = true
// 		}
// 	})

// 	t.Run("[Integration] - Variant data integration", func(t *testing.T) {
// 		// GRP-INT-002
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product 101 has variants with options
// 		productID := 101
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		relatedProducts := data["relatedProducts"].([]interface{})

// 		// Check that products with variants have proper variant preview
// 		for _, item := range relatedProducts {
// 			product := item.(map[string]interface{})

// 			hasVariants, ok := product["hasVariants"].(bool)
// 			if ok && hasVariants {
// 				// Verify variant preview exists
// 				variantPreview, ok := product["variantPreview"].(map[string]interface{})
// 				assert.True(t, ok, "Product with variants should have variantPreview")

// 				if variantPreview != nil {
// 					// Check variant preview fields
// 					assert.NotNil(t, variantPreview["totalVariants"],
// 						"Should have totalVariants")

// 					options, ok := variantPreview["options"].([]interface{})
// 					if ok && len(options) > 0 {
// 						// Verify option structure
// 						option := options[0].(map[string]interface{})
// 						assert.NotNil(t, option["name"], "Option should have name")
// 						assert.NotNil(t, option["displayName"], "Option should have displayName")
// 						assert.NotNil(t, option["availableValues"],
// 							"Option should have availableValues")
// 					}
// 				}
// 			}
// 		}
// 	})

// 	t.Run("[Integration] - Category hierarchy navigation", func(t *testing.T) {
// 		// GRP-INT-003
// 		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
// 		client.SetToken(seller2Token)

// 		// Product in nested category (e.g., Android Phones under Smartphones under Electronics)
// 		productID := 149 // In Android Phones category
// 		url := fmt.Sprintf("/api/products/%d/related", productID)

// 		w := client.Get(t, url)

// 		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
// 		data := response["data"].(map[string]interface{})
// 		_ = data["relatedProducts"].([]interface{}) // Verify type
// 		meta := data["meta"].(map[string]interface{})

// 		// Verify category strategies work correctly
// 		strategiesUsed := meta["strategiesUsed"].([]interface{})

// 		// Check if any category-based strategy was used
// 		foundCategoryStrategy := false
// 		categoryStrategies := []string{
// 			"same_category", "sibling_category",
// 			"parent_category", "child_category",
// 		}

// 		for _, strategy := range strategiesUsed {
// 			strategyName := strategy.(string)
// 			for _, catStrategy := range categoryStrategies {
// 				if strategyName == catStrategy {
// 					foundCategoryStrategy = true
// 					break
// 				}
// 			}
// 		}

// 		// At least one category strategy should have been used
// 		assert.True(t, foundCategoryStrategy,
// 			"Should use at least one category-based strategy for nested categories")
// 	})
// }
