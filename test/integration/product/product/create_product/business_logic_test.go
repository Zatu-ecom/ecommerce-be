package product

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestCreateProductBusinessLogic tests business logic validation scenarios
// Validates: variant constraints, option validation, attribute validation, package options
func TestCreateProductBusinessLogic(t *testing.T) {
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

	// ============================================================================
	// BUSINESS LOGIC ERROR SCENARIOS
	// ============================================================================

	t.Run("Success - Multiple variants marked as default (last one wins)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		isDefault := true
		requestBody := map[string]interface{}{
			"name":       "Test Product - Multiple Defaults",
			"categoryId": 4,
			"baseSku":    "TEST-MULTI-DEFAULT-001",
			"options": []map[string]interface{}{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "black", "displayName": "Black"},
						{"value": "white", "displayName": "White"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":       "TEST-MULTI-DEFAULT-001-BLK",
					"price":     99.99,
					"isDefault": isDefault, // First default
					"options": []map[string]interface{}{
						{"optionName": "color", "value": "black"},
					},
				},
				{
					"sku":       "TEST-MULTI-DEFAULT-001-WHT",
					"price":     99.99,
					"isDefault": isDefault, // Second default - last one wins
					"options": []map[string]interface{}{
						{"optionName": "color", "value": "white"},
					},
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should accept with "last one wins" rule - only White variant is default
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		variants, ok := product["variants"].([]interface{})
		assert.True(t, ok, "variants should be an array")
		assert.Len(t, variants, 2, "Should have 2 variants")

		// Verify only one variant is marked as default (the last one - White)
		defaultCount := 0
		var defaultVariant map[string]interface{}
		for _, v := range variants {
			variant := v.(map[string]interface{})
			if isDefault, ok := variant["isDefault"].(bool); ok && isDefault {
				defaultCount++
				defaultVariant = variant
			}
		}

		assert.Equal(t, 1, defaultCount, "Should have exactly one default variant")
		assert.Equal(
			t,
			"TEST-MULTI-DEFAULT-001-WHT",
			defaultVariant["sku"],
			"Last variant (White) should be default",
		)
	})

	t.Run("Error - Variant with invalid option combination", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product - Invalid Option Combo",
			"categoryId": 4,
			"baseSku":    "TEST-INVALID-COMBO-001",
			"options": []map[string]interface{}{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "black", "displayName": "Black"},
						{"value": "white", "displayName": "White"},
					},
				},
				{
					"name":        "size",
					"displayName": "Size",
					"values": []map[string]interface{}{
						{"value": "m", "displayName": "Medium"},
						{"value": "l", "displayName": "Large"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-INVALID-COMBO-001-V1",
					"price": 99.99,
					"options": []map[string]interface{}{
						{
							"optionName": "color",
							"value":      "red",
						}, // Invalid: Red not in defined values
						{"optionName": "size", "value": "m"},
					},
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for invalid option value
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Variant missing required option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product - Missing Option",
			"categoryId": 4,
			"baseSku":    "TEST-MISSING-OPT-001",
			"options": []map[string]interface{}{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "black", "displayName": "Black"},
					},
				},
				{
					"name":        "size",
					"displayName": "Size",
					"values": []map[string]interface{}{
						{"value": "m", "displayName": "Medium"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-MISSING-OPT-001-V1",
				"price": 299.99,
				"options": []map[string]interface{}{
					{"optionName": "color", "value": "black"},
						// Missing Size option - should fail validation
					},
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for missing required option
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Package option with zero price", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product - Zero Price Package",
			"categoryId": 4,
			"baseSku":    "TEST-ZERO-PKG-001",
			"options": []map[string]interface{}{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "Black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-ZERO-PKG-001-V1",
					"price": 99.99,
					"options": []map[string]interface{}{
						{"optionName": "Color", "value": "Black"},
					},
				},
			},
			"packageOptions": []map[string]interface{}{
				{
					"name":        "Free Warranty",
					"description": "Extended warranty",
					"price":       0, // Invalid: zero price
					"quantity":    1,
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for zero price
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Package option with negative price", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product - Negative Price Package",
			"categoryId": 4,
			"baseSku":    "TEST-NEG-PKG-001",
			"options": []map[string]interface{}{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "Black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-NEG-PKG-001-V1",
					"price": 99.99,
					"options": []map[string]interface{}{
						{"optionName": "Color", "value": "Black"},
					},
				},
			},
			"packageOptions": []map[string]interface{}{
				{
					"name":        "Discount Package",
					"description": "Special discount",
					"price":       -10, // Invalid: negative price
					"quantity":    1,
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for negative price
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Package option with zero quantity", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product - Zero Quantity Package",
			"categoryId": 4,
			"baseSku":    "TEST-ZERO-QTY-001",
			"options": []map[string]interface{}{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "Black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-ZERO-QTY-001-V1",
					"price": 99.99,
					"options": []map[string]interface{}{
						{"optionName": "Color", "value": "Black"},
					},
				},
			},
			"packageOptions": []map[string]interface{}{
				{
					"name":        "Empty Package",
					"description": "Package with no items",
					"price":       49.99,
					"quantity":    0, // Invalid: zero quantity
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for zero quantity
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Package option with negative quantity", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":       "Test Product - Negative Quantity Package",
			"categoryId": 4,
			"baseSku":    "TEST-NEG-QTY-001",
			"options": []map[string]interface{}{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "Black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-NEG-QTY-001-V1",
					"price": 99.99,
					"options": []map[string]interface{}{
						{"optionName": "Color", "value": "Black"},
					},
				},
			},
			"packageOptions": []map[string]interface{}{
				{
					"name":        "Negative Package",
					"description": "Invalid package",
					"price":       49.99,
					"quantity":    -5, // Invalid: negative quantity
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		// Should return 400 Bad Request for negative quantity
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}
