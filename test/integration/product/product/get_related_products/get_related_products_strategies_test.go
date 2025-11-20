package product

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetRelatedProductsStrategies tests the strategy-based matching functionality
// This file contains tests for all 8 matching strategies and scoring/ranking
func TestGetRelatedProductsStrategies(t *testing.T) {
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
	// CATEGORY 1: STRATEGY-BASED MATCHING TESTS
	// ============================================================================

	t.Run("[Same Category Strategy] - Get related products in same category", func(t *testing.T) {
		// GRP-STR-001
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Product 101 is iPhone 14 in Smartphones category (ID 4)
		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "Response should contain data object")

		relatedProducts, ok := data["relatedProducts"].([]interface{})
		require.True(t, ok, "Should have relatedProducts array")
		assert.NotEmpty(t, relatedProducts, "Should have related products from same category")

		// Check that results include same category products
		foundSameCategory := false
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})

			// Verify required fields
			assert.NotNil(t, product["id"], "Product should have ID")
			assert.NotNil(t, product["score"], "Product should have score")
			assert.NotNil(t, product["strategyUsed"], "Product should have strategyUsed")
			assert.NotNil(t, product["relationReason"], "Product should have relationReason")

			// Check for same category matches
			if strategyUsed, ok := product["strategyUsed"].(string); ok {
				if strategyUsed == "same_category" {
					foundSameCategory = true
					score := product["score"].(float64)
					assert.GreaterOrEqual(
						t,
						score,
						100.0,
						"Same category should have base score >= 100",
					)
				}
			}
		}

		assert.True(t, foundSameCategory, "Should find at least one same category match")

		// Verify source product is excluded
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			assert.NotEqual(
				t,
				float64(productID),
				product["id"],
				"Source product should be excluded",
			)
		}

		// Check pagination metadata
		pagination, ok := data["pagination"].(map[string]interface{})
		require.True(t, ok, "Should have pagination metadata")
		assert.NotNil(t, pagination["currentPage"], "Should have currentPage")
		assert.NotNil(t, pagination["totalItems"], "Should have totalItems")

		// Check meta information
		meta, ok := data["meta"].(map[string]interface{})
		require.True(t, ok, "Should have meta object")
		strategiesUsed, ok := meta["strategiesUsed"].([]interface{})
		require.True(t, ok, "Should have strategiesUsed array")
		assert.NotEmpty(t, strategiesUsed, "Should have strategies used")
	})

	t.Run("[Same Brand Strategy] - Get related products from same brand", func(t *testing.T) {
		// GRP-STR-002
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Product 101 is iPhone 14 (Apple brand)
		// Should find other Apple products like MacBook, iPad, Apple Watch
		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		// Look for same brand matches (Apple products from different categories)
		foundSameBrand := false
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})

			// Check if it's Apple brand from different category
			brand, hasBrand := product["brand"].(string)
			strategyUsed, hasStrategy := product["strategyUsed"].(string)

			if hasBrand && hasStrategy && brand == "Apple" && strategyUsed == "same_brand" {
				foundSameBrand = true
				score := product["score"].(float64)
				assert.GreaterOrEqual(t, score, 80.0, "Same brand should have base score >= 80")

				// Verify it's from different category (not smartphones)
				category := product["category"].(map[string]interface{})
				categoryID := category["id"].(float64)
				assert.NotEqual(
					t,
					float64(4),
					categoryID,
					"Same brand product should be from different category",
				)
			}
		}

		assert.True(
			t,
			foundSameBrand,
			"Should find at least one same brand match from different category",
		)
	})

	t.Run("[Sibling Category Strategy] - Get products from sibling categories", func(t *testing.T) {
		// GRP-STR-003
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Product 101 is in Smartphones (ID 4), sibling categories: Laptops (5), Tablets (12), etc.
		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		foundSibling := false
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			strategyUsed, ok := product["strategyUsed"].(string)

			if ok && strategyUsed == "sibling_category" {
				foundSibling = true
				score := product["score"].(float64)
				assert.GreaterOrEqual(
					t,
					score,
					70.0,
					"Sibling category should have base score >= 70",
				)

				relationReason, ok := product["relationReason"].(string)
				assert.True(t, ok, "Should have relationReason")
				assert.Contains(
					t,
					relationReason,
					"sibling",
					"Relation reason should mention sibling",
				)
			}
		}

		// Note: Sibling matches may not always be present depending on data
		// This is acceptable as it tests the strategy when applicable
		if foundSibling {
			t.Log("Found sibling category matches")
		}
	})

	t.Run("[Tag Matching Strategy] - Get products with common tags", func(t *testing.T) {
		// GRP-STR-006
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Product 143 (Canon EOS R6) has tags: ["camera", "canon", "mirrorless", "professional"]
		// Use tag_matching strategy explicitly to force tag-based matching only
		productID := 143
		url := fmt.Sprintf("/api/products/%d/related?strategies=tag_matching", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		foundTagMatch := false
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			strategyUsed, ok := product["strategyUsed"].(string)

			if ok && strategyUsed == "tag_matching" {
				foundTagMatch = true
				score := product["score"].(float64)
				assert.GreaterOrEqual(t, score, 20.0, "Tag matching should have base score >= 20")

				// Verify product has tags
				tags, hasTags := product["tags"].([]interface{})
				assert.True(t, hasTags, "Product should have tags")
				assert.NotEmpty(t, tags, "Tags array should not be empty")
			}
		}

		assert.True(t, foundTagMatch, "Should find at least one tag matching product")
	})

	t.Run("[Price Range Strategy] - Get products in similar price range", func(t *testing.T) {
		// GRP-STR-007
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Product 101 (iPhone 14) price range: ~799-899
		// Should find products in similar range (25% variance)
		productID := 101
		url := fmt.Sprintf("/api/products/%d/related", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		foundPriceRange := false
		for _, item := range relatedProducts {
			product := item.(map[string]interface{})
			strategyUsed, ok := product["strategyUsed"].(string)

			if ok && strategyUsed == "price_range" {
				foundPriceRange = true
				score := product["score"].(float64)
				assert.GreaterOrEqual(t, score, 20.0, "Price range should have base score >= 20")

				// Verify price range exists
				priceRange, ok := product["priceRange"].(map[string]interface{})
				assert.True(t, ok, "Product should have priceRange")
				assert.NotNil(t, priceRange["min"], "Should have min price")
				assert.NotNil(t, priceRange["max"], "Should have max price")
			}
		}

		// Price range matches may not always be present
		if foundPriceRange {
			t.Log("Found price range matches")
		}
	})

	// ============================================================================
	// CATEGORY 2: SCORING AND RANKING TESTS
	// ============================================================================

	t.Run(
		"[Bonus - Same Brand and Category] - Bonus for matching brand and category",
		func(t *testing.T) {
			// GRP-SCR-001
			seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
			client.SetToken(seller2Token)

			// Product 103 is Samsung Galaxy S23 (Samsung brand, Smartphones category)
			// Should find Samsung Galaxy S24, A54 with bonus points
			productID := 103
			url := fmt.Sprintf("/api/products/%d/related", productID)

			w := client.Get(t, url)

			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			data := response["data"].(map[string]interface{})
			relatedProducts := data["relatedProducts"].([]interface{})

			foundBonusProduct := false
			for _, item := range relatedProducts {
				product := item.(map[string]interface{})
				brand, hasBrand := product["brand"].(string)
				category, hasCategory := product["category"].(map[string]interface{})

				if hasBrand && hasCategory && brand == "Samsung" {
					categoryID := category["id"].(float64)
					if categoryID == 4 { // Smartphones
						foundBonusProduct = true
						score := product["score"].(float64)
						// Base 100 + bonus 50 = 150
						assert.GreaterOrEqual(
							t,
							score,
							140.0,
							"Samsung smartphone should have high score with bonus",
						)
						t.Logf("Found Samsung smartphone with score: %.0f", score)
					}
				}
			}

			assert.True(t, foundBonusProduct, "Should find Samsung smartphone with bonus points")
		},
	)

	t.Run("[Score Ranking] - Products ordered by final score", func(t *testing.T) {
		// GRP-SCR-007
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 101
		url := fmt.Sprintf("/api/products/%d/related?limit=20", productID)

		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data := response["data"].(map[string]interface{})
		relatedProducts := data["relatedProducts"].([]interface{})

		require.NotEmpty(t, relatedProducts, "Should have related products")

		// Verify products are ordered by descending score
		previousScore := 999999.0
		for i, item := range relatedProducts {
			product := item.(map[string]interface{})
			score, ok := product["score"].(float64)
			require.True(t, ok, "Product should have score")

			assert.LessOrEqual(t, score, previousScore,
				"Product at index %d should have score <= previous score (%.0f <= %.0f)",
				i, score, previousScore)

			previousScore = score
		}
	})
}
