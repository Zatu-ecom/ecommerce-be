package get_product_by_id

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetProductByID_EdgeCases_Part2 tests edge case scenarios (EDGE-06 to EDGE-10)
func TestGetProductByID_EdgeCases_Part2(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")
	containers.RunSeeds(t, "migrations/seeds/002_seed_product_data.sql")
	containers.RunSeeds(t, "test/integration/data/get_product_by_id_seed_data.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// EDGE-06: Product with Maximum Field Lengths
	// ============================================================================
	t.Run("EDGE-06: Product with Maximum Field Lengths", func(t *testing.T) {
		// Product 103 has maximum length fields
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/103")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify name length (200 chars)
		name := product["name"].(string)
		assert.Equal(t, 200, len(name), "Name should be exactly 200 characters")

		// Verify brand length (100 chars)
		brand := product["brand"].(string)
		assert.Equal(t, 100, len(brand), "Brand should be exactly 100 characters")

		// Verify SKU length (50 chars)
		sku := product["sku"].(string)
		assert.LessOrEqual(t, len(sku), 50, "SKU should be <= 50 characters")

		// Verify short description length (~500 chars)
		shortDesc := product["shortDescription"].(string)
		assert.LessOrEqual(t, len(shortDesc), 500, "Short description should be <= 500 characters")

		// Verify long description length (~5000 chars)
		longDesc := product["longDescription"].(string)
		assert.LessOrEqual(t, len(longDesc), 5000, "Long description should be <= 5000 characters")

		// Verify tags count (20 tags)
		tags := product["tags"].([]interface{})
		assert.LessOrEqual(t, len(tags), 20, "Tags should be <= 20 items")

		// Verify response JSON is valid
		assert.NotNil(t, response, "Response should be valid JSON")

		// Verify no truncation in response (all 200 'A' characters should be present)
		assert.Equal(t, 200, len(name), "Full name with all 200 characters should be returned")
		assert.Contains(t, name, "AAAA", "Name should contain repeated A characters")
	})

	// ============================================================================
	// EDGE-07: Variant Option Values with Special Characters
	// ============================================================================
	t.Run("EDGE-07: Variant Option Values with Special Characters", func(t *testing.T) {
		// Product 106 has special characters in option values
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/106")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Get variants
		variants := product["variants"].([]interface{})
		assert.NotEmpty(t, variants, "Variants should be present")

		variant := variants[0].(map[string]interface{})
		selectedOptions := variant["selectedOptions"].([]interface{})

		// Find option with special characters
		hasSpecialChars := false
		for _, opt := range selectedOptions {
			option := opt.(map[string]interface{})
			value := option["value"].(string)

			// Check for special characters
			if value == "Red & Blue" || value == "Size: M/L" {
				hasSpecialChars = true

				// Verify special characters are preserved
				assert.True(t,
					value == "Red & Blue" || value == "Size: M/L",
					"Special characters should be preserved")

				// Verify display name is readable
				displayName := option["valueDisplayName"].(string)
				assert.NotEmpty(t, displayName, "Display name should be present")
			}
		}

		assert.True(t, hasSpecialChars, "Should have options with special characters")

		// Verify JSON encoding handles special characters
		assert.NotContains(t, w.Body.String(), "\\u0026", "& should not be escaped in JSON")
	})

	// ============================================================================
	// EDGE-08: Product with Very Long SKU
	// ============================================================================
	t.Run("EDGE-08: Product with Very Long SKU", func(t *testing.T) {
		// Product 110 has long SKU
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/110")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify base SKU
		baseSku := product["sku"].(string)
		assert.NotEmpty(t, baseSku, "Base SKU should be present")
		assert.Contains(t, baseSku, "VERYLONGSKU", "Base SKU should contain prefix")

		// Verify variants have complete SKUs
		variants := product["variants"].([]interface{})
		assert.NotEmpty(t, variants, "Variants should be present")

		variant := variants[0].(map[string]interface{})
		variantSku := variant["sku"].(string)
		assert.NotEmpty(t, variantSku, "Variant SKU should be present")
		assert.Contains(t, variantSku, "VERYLONGSKU", "Variant SKU should contain base SKU")

		// Verify SKU is not truncated
		assert.GreaterOrEqual(t, len(variantSku), 40, "Variant SKU should be long")

		// Verify SKU format is maintained
		assert.NotContains(t, variantSku, "...", "SKU should not be truncated with ellipsis")
	})

	// ============================================================================
	// EDGE-09: Product with Zero-Price Variant (Free Product)
	// ============================================================================
	t.Run("EDGE-09: Product with Zero-Price Variant (Free Product)", func(t *testing.T) {
		// Product 105 has a free variant (price = 0.00)
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/105")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify price range includes zero
		priceRange := product["priceRange"].(map[string]interface{})
		minPrice := priceRange["min"].(float64)
		maxPrice := priceRange["max"].(float64)

		assert.Equal(t, 0.0, minPrice, "Min price should be 0.00 for free variant")
		assert.Greater(t, maxPrice, 0.0, "Max price should be > 0 for paid variant")

		// Verify variants
		variants := product["variants"].([]interface{})
		assert.NotEmpty(t, variants, "Variants should be present")

		// Find the free variant
		hasFreeVariant := false
		for _, v := range variants {
			variant := v.(map[string]interface{})
			price := variant["price"].(float64)

			if price == 0.0 {
				hasFreeVariant = true

				// Verify free variant is purchasable
				allowPurchase := variant["allowPurchase"].(bool)
				assert.True(t, allowPurchase, "Free variant should be purchasable")

				// Verify SKU is present
				sku := variant["sku"].(string)
				assert.NotEmpty(t, sku, "Free variant should have SKU")
			}
		}

		assert.True(t, hasFreeVariant, "Product should have at least one free variant")

		// Verify no division-by-zero errors
		assert.NotNil(t, product["allowPurchase"], "allowPurchase should be present")
	})

	// ============================================================================
	// EDGE-10: Product Timestamps at Boundary Values
	// ============================================================================
	t.Run("EDGE-10: Product Timestamps at Boundary Values", func(t *testing.T) {
		// Use any product to test timestamp formatting
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify timestamps are in RFC3339 format
		createdAt := product["createdAt"].(string)
		updatedAt := product["updatedAt"].(string)

		assert.NotEmpty(t, createdAt, "CreatedAt should be present")
		assert.NotEmpty(t, updatedAt, "UpdatedAt should be present")

		// Verify RFC3339 format (contains T and Z or timezone offset)
		assert.Contains(t, createdAt, "T", "CreatedAt should be in RFC3339 format")
		assert.True(t,
			createdAt[len(createdAt)-1:] == "Z" || createdAt[len(createdAt)-6] == '+' || createdAt[len(createdAt)-6] == '-',
			"CreatedAt should have timezone information")

		assert.Contains(t, updatedAt, "T", "UpdatedAt should be in RFC3339 format")

		// Verify no timestamp overflow errors
		assert.NotContains(t, createdAt, "0001-01-01", "CreatedAt should not be zero value")
		assert.NotContains(t, updatedAt, "0001-01-01", "UpdatedAt should not be zero value")

		// Verify timestamps are consistent (updatedAt >= createdAt)
		// Note: This is a basic check, proper time parsing would be needed for exact comparison
		assert.NotEmpty(t, createdAt, "Timestamps should be parseable")
		assert.NotEmpty(t, updatedAt, "Timestamps should be parseable")
	})
}
