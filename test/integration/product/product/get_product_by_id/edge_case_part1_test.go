package get_product_by_id

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetProductByID_EdgeCases_Part1 tests edge case scenarios (EDGE-01 to EDGE-05)
func TestGetProductByID_EdgeCases_Part1(t *testing.T) {
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
	// EDGE-01: Product ID with Special Characters (XSS attempt)
	// ============================================================================
	t.Run("EDGE-01: Product ID with Special Characters", func(t *testing.T) {
		// Try to get product with XSS payload in ID
		xssPayload := "101<script>alert('xss')</script>"
		w := client.Get(t, "/api/products/"+xssPayload)

		// API treats this as a valid ID format that doesn't exist (not a malformed request)
		// Note: API returns 404 status code, verifying it handles special characters safely
		assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 for non-existent product")
	})

	// ============================================================================
	// EDGE-02: Product ID with SQL Injection Attempt
	// ============================================================================
	t.Run("EDGE-02: Product ID with SQL Injection Attempt", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		// SQL injection payload in product ID
		w := client.Get(t, "/api/products/101' OR '1'='1")

		// Verify 400 response (parsing error)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")

		response := helpers.ParseResponse(t, w.Body)

		// Verify SQL injection is prevented
		assert.False(t, response["success"].(bool), "Response should not be successful")

		// Verify no data leak
		assert.Nil(t, response["data"], "No data should be returned")
	})

	// ============================================================================
	// EDGE-03: Product with Extremely Long Product ID
	// ============================================================================
	t.Run("EDGE-03: Product with Extremely Long Product ID", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		// Max uint32 boundary
		w := client.Get(t, "/api/products/4294967295")

		// Should return 404 (product not found) or handle gracefully
		assert.True(t,
			w.Code == http.StatusNotFound || w.Code == http.StatusBadRequest,
			"Should return 404 or 400")

		response := helpers.ParseResponse(t, w.Body)

		// Verify no crash or overflow
		assert.False(t, response["success"].(bool), "Response should not be successful")

		// Verify system handles large numbers gracefully
		assert.NotNil(t, response, "Response should be returned")
	})

	// ============================================================================
	// EDGE-04: Product with Unicode Characters in Data
	// ============================================================================
	t.Run("EDGE-04: Product with Unicode Characters in Data", func(t *testing.T) {
		// Product 104 has Unicode characters
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/104")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify Unicode characters are preserved
		name := product["name"].(string)
		assert.Contains(t, name, "中文", "Chinese characters should be preserved")
		assert.Contains(t, name, "😀", "Emojis should be preserved")

		brand := product["brand"].(string)
		assert.Contains(t, brand, "العربية", "Arabic characters should be preserved")

		// Verify description has Unicode
		longDesc := product["longDescription"].(string)
		assert.Contains(t, longDesc, "中文", "Unicode should be in description")
		assert.Contains(t, longDesc, "العربية", "Arabic should be in description")
		assert.Contains(t, longDesc, "😀", "Emojis should be in description")

		// Verify tags have Unicode
		tags := product["tags"].([]interface{})
		hasUnicodeTag := false
		for _, tag := range tags {
			tagStr := tag.(string)
			if tagStr == "中文" || tagStr == "العربية" || tagStr == "emoji😀" {
				hasUnicodeTag = true
				break
			}
		}
		assert.True(t, hasUnicodeTag, "Tags should contain Unicode")

		// Verify UTF-8 encoding in response
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"),
			"Response should be UTF-8 encoded")
	})

	// ============================================================================
	// EDGE-05: Product with Empty Optional Fields
	// ============================================================================
	t.Run("EDGE-05: Product with Empty Optional Fields", func(t *testing.T) {
		// Product 102 has empty optional fields
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/102")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify required fields are present
		assert.NotNil(t, product["id"], "ID should be present")
		assert.Equal(t, "Minimal Product", product["name"], "Name should be present")
		assert.NotNil(t, product["categoryId"], "Category ID should be present")

		// Verify empty optional fields
		brand := product["brand"].(string)
		assert.Empty(t, brand, "Brand should be empty string")

		shortDesc := product["shortDescription"].(string)
		assert.Empty(t, shortDesc, "Short description should be empty")

		longDesc := product["longDescription"].(string)
		assert.Empty(t, longDesc, "Long description should be empty")

		tags := product["tags"].([]interface{})
		assert.Empty(t, tags, "Tags should be empty array")

		// Verify response structure is still valid
		assert.NotNil(t, product["variants"], "Variants should be present")
		assert.NotNil(t, product["options"], "Options should be present")

		// Verify no null pointer errors
		assert.NotNil(t, product["hasVariants"], "hasVariants should be present")
		assert.NotNil(t, product["allowPurchase"], "allowPurchase should be present")
	})
}
