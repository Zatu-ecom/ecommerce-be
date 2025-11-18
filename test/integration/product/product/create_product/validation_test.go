package product

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"
)

// TestCreateProductValidation tests input validation scenarios for product creation
// Validates: data types, required fields, and format validation
func TestCreateProductValidation(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// VALIDATION ERROR SCENARIOS - Invalid Data Types
	// ============================================================================

	t.Run("Error - Invalid categoryId data type (string instead of number)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product",
			"categoryId": "abc", // Invalid: string instead of number
			"baseSku":    "TEST-INVALID-TYPE-001",
			"options": []map[string]interface{}{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-INVALID-TYPE-001-V1",
					"price": 99.99,
					"options": []map[string]interface{}{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for invalid data type
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Invalid price data type (string instead of number)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-INVALID-PRICE-001",
			"options": []map[string]interface{}{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-INVALID-PRICE-001-V1",
					"price": "not-a-number", // Invalid: string instead of number
					"options": []map[string]interface{}{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for invalid data type
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Invalid variants data type (not an array)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-INVALID-VARIANTS-001",
			"variants":   "not-an-array", // Invalid: string instead of array
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for invalid data type
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Invalid options data type (not an array)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-INVALID-OPTIONS-001",
			"options":    "not-an-array", // Invalid: string instead of array
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-INVALID-OPTIONS-001-V1",
					"price": 99.99,
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for invalid data type
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Invalid isDefault data type (string instead of boolean)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-INVALID-BOOL-001",
			"options": []map[string]interface{}{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":       "TEST-INVALID-BOOL-001-V1",
					"price":     99.99,
					"isDefault": "yes", // Invalid: string instead of boolean
					"options": []map[string]interface{}{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for invalid data type
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}
