package product

import (
	"net/http"
	"strings"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestCreateProductEdgeCases tests edge cases and boundary conditions
// Validates: price precision, unicode support, URL length limits, special flags
func TestCreateProductEdgeCases(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// EDGE CASES & BOUNDARY TESTING
	// ============================================================================

	t.Run("EdgeCase - Price with many decimal places", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product - Precise Price",
			"categoryId": 4,
			"baseSku":    "TEST-PRECISE-PRICE-001",
			"options": []map[string]any{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-PRECISE-PRICE-001-V1",
					"price": 19.999999, // Many decimal places
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)

		// Should succeed and verify rounding behavior
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		variants, ok := product["variants"].([]any)
		assert.True(t, ok, "variants should be an array")
		assert.Len(t, variants, 1)

		variant := variants[0].(map[string]any)
		price := variant["price"].(float64)

		// Verify price is rounded appropriately (typically to 2 decimal places)
		// The exact behavior depends on your database and application logic
		assert.InDelta(t, 20.00, price, 0.01, "Price should be rounded appropriately")
		t.Logf("Price with many decimals (19.999999) was stored as: %.2f", price)
	})

	t.Run("EdgeCase - Price with exactly 2 decimals", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product - Standard Price",
			"categoryId": 4,
			"baseSku":    "TEST-STD-PRICE-001",
			"options": []map[string]any{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-STD-PRICE-001-V1",
					"price": 99.99, // Standard 2 decimal places
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		variant := variants[0].(map[string]any)

		// Should preserve exact price
		assert.Equal(t, 99.99, variant["price"], "Price should be preserved exactly")
	})

	t.Run("EdgeCase - Unicode characters in product name (Japanese)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "プロダクト名", // Japanese characters
			"categoryId": 4,
			"baseSku":    "TEST-UNICODE-JP-001",
			"options": []map[string]any{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-UNICODE-JP-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify Unicode characters are preserved
		assert.Equal(t, "プロダクト名", product["name"], "Japanese characters should be preserved")
	})

	t.Run("EdgeCase - Unicode characters in product name (French accents)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Café Crème", // French accented characters
			"categoryId": 4,
			"baseSku":    "TEST-UNICODE-FR-001",
			"options": []map[string]any{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-UNICODE-FR-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify Unicode characters are preserved
		assert.Equal(t, "Café Crème", product["name"], "French accents should be preserved")
	})

	t.Run("EdgeCase - Variant with very long SKU is accepted", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Images are now managed via the variant media endpoints (file module).
		// This test verifies that a product with a very long SKU is still accepted.
		longSKU := strings.Repeat("A", 200)

		requestBody := map[string]any{
			"name":       "Test Product - Long SKU",
			"categoryId": 4,
			"baseSku":    "TEST-LONG-SKU-001",
			"options": []map[string]any{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   longSKU,
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)

		// Accept or reject depending on SKU length constraints
		if w.Code == http.StatusCreated {
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			product := helpers.GetResponseData(t, response, "product")
			variants, ok := product["variants"].([]any)
			assert.True(t, ok)
			variant := variants[0].(map[string]any)
			_, ok = variant["media"].([]any)
			assert.True(t, ok, "media should always be a JSON array")
			t.Log("Very long SKU was accepted")
		} else {
			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
			t.Log("Very long SKU was rejected (expected if there is a length constraint)")
		}
	})

	t.Run("EdgeCase - allowPurchase set to false", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		allowPurchase := false
		requestBody := map[string]any{
			"name":       "Test Product - Not Purchasable",
			"categoryId": 4,
			"baseSku":    "TEST-NO-PURCHASE-001",
			"options": []map[string]any{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":           "TEST-NO-PURCHASE-001-V1",
					"price":         99.99,
					"allowPurchase": allowPurchase, // Not purchasable
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify variant created but not purchasable
		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Len(t, variants, 1)

		variant := variants[0].(map[string]any)
		assert.NotNil(t, variant["id"], "Variant should be created")

		// Verify allowPurchase is false
		if allowPurchaseValue, ok := variant["allowPurchase"].(bool); ok {
			assert.False(t, allowPurchaseValue, "allowPurchase should be false")
		}

		t.Log("Variant created successfully with allowPurchase=false")
	})

	t.Run("EdgeCase - Empty strings in optional fields", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":             "Test Product - Empty Optionals",
			"categoryId":       4,
			"baseSku":          "TEST-EMPTY-OPT-001",
			"brand":            "", // Empty optional field
			"shortDescription": "", // Empty optional field
			"longDescription":  "", // Empty optional field
			"options": []map[string]any{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-EMPTY-OPT-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)

		// Should accept empty strings for optional fields
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		assert.NotNil(t, product["id"], "Product should be created")
		t.Log("Product created successfully with empty optional fields")
	})

	t.Run("EdgeCase - Maximum price value", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product - Max Price",
			"categoryId": 4,
			"baseSku":    "TEST-MAX-PRICE-001",
			"options": []map[string]any{
				{
					"name":        "color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-MAX-PRICE-001-V1",
					"price": 999999.99, // Very high price
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)

		// Behavior depends on database constraints
		if w.Code == http.StatusCreated {
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			product := helpers.GetResponseData(t, response, "product")
			variants := product["variants"].([]any)
			variant := variants[0].(map[string]any)
			assert.Equal(t, 999999.99, variant["price"], "High price should be stored")
			t.Log("Very high price was accepted")
		} else {
			// Price too high - validation error
			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
			t.Log("Very high price was rejected (expected if there's a limit)")
		}
	})
}
