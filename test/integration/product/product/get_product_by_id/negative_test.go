package get_product_by_id

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetProductByID_Negative tests error scenarios for retrieving a product by ID
func TestGetProductByID_Negative(t *testing.T) {
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
	// NEG-01: Product ID Does Not Exist
	// ============================================================================
	t.Run("NEG-01: Product ID Does Not Exist", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		// Product ID 99999 does not exist
		w := client.Get(t, "/api/products/99999")

		// Verify 404 response
		assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 Not Found")

		response := helpers.ParseResponse(t, w.Body)

		// Verify error response structure
		assert.False(t, response["success"].(bool), "Response should not be successful")

		// Verify error message
		message, ok := response["message"].(string)
		assert.True(t, ok, "Response should have error message")
		assert.Contains(t, message, "not found", "Error message should mention 'not found'")

		// Verify no product data is returned
		assert.Nil(t, response["data"], "No product data should be returned")
	})

	// ============================================================================
	// NEG-02: Invalid Product ID Format (Non-Numeric)
	// ============================================================================
	t.Run("NEG-02: Invalid Product ID Format (Non-Numeric)", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		// Non-numeric product ID
		w := client.Get(t, "/api/products/abc")

		// Verify 400 response
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")

		response := helpers.ParseResponse(t, w.Body)

		// Verify error response
		assert.False(t, response["success"].(bool), "Response should not be successful")

		message, ok := response["message"].(string)
		assert.True(t, ok, "Response should have error message")
		assert.Contains(t, message, "Invalid", "Error message should mention invalid")
	})

	// ============================================================================
	// NEG-03: Invalid Product ID (Negative Number)
	// ============================================================================
	t.Run("NEG-03: Invalid Product ID (Negative Number)", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		// Negative product ID
		w := client.Get(t, "/api/products/-5")

		// Verify 400 response
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")

		response := helpers.ParseResponse(t, w.Body)

		// Verify error response
		assert.False(t, response["success"].(bool), "Response should not be successful")

		message := response["message"].(string)
		assert.NotEmpty(t, message, "Error message should be present")
	})

	// ============================================================================
	// NEG-04: Invalid Product ID (Zero)
	// ============================================================================
	t.Run("NEG-04: Invalid Product ID (Zero)", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		// Zero product ID
		w := client.Get(t, "/api/products/0")

		// Verify 400 or 404 response
		assert.True(t,
			w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
			"Should return 400 or 404")

		response := helpers.ParseResponse(t, w.Body)

		// Verify error response
		assert.False(t, response["success"].(bool), "Response should not be successful")
	})

	// ============================================================================
	// NEG-05: Product ID Exceeds Maximum Integer Value
	// ============================================================================
	t.Run("NEG-05: Product ID Exceeds Maximum Integer Value", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		// Very large number that exceeds reasonable limits
		w := client.Get(t, "/api/products/999999999999999999999")

		// Verify 400 response (parsing error)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")

		response := helpers.ParseResponse(t, w.Body)

		// Verify error response
		assert.False(t, response["success"].(bool), "Response should not be successful")

		message := response["message"].(string)
		assert.NotEmpty(t, message, "Error message should be present")
	})

	// ============================================================================
	// NEG-06: Missing Product ID Parameter
	// ============================================================================
	t.Run("NEG-06: Missing Product ID Parameter", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		// Request to /api/products/ without ID
		w := client.Get(t, "/api/products/")

		// This might return 404 (route not found) or redirect to GetAllProducts
		// Accept either behavior
		assert.True(t,
			w.Code == http.StatusNotFound ||
				w.Code == http.StatusOK ||
				w.Code == http.StatusMovedPermanently ||
				w.Code == http.StatusBadRequest,
			"Should return 404, 400, or redirect to list endpoint")

		// If it's an error response, verify structure
		if w.Code >= 400 {
			response := helpers.ParseResponse(t, w.Body)
			assert.False(t, response["success"].(bool), "Response should not be successful")
		}
	})
}
