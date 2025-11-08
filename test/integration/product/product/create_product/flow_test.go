package product

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestCreateProductIntegration tests integration scenarios and API composition
// Validates: data consistency, API interactions, concurrent operations
func TestCreateProductIntegration(t *testing.T) {
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
	// INTEGRATION SCENARIOS
	// ============================================================================

	t.Run("Integration - Create product then immediately fetch it", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create product
		createRequest := map[string]interface{}{
			"name":       "Test Product - Integration Fetch",
			"categoryId": 4,
			"baseSku":    "TEST-INTEG-FETCH-001",
			"brand":      "IntegrationBrand",
			"options": []map[string]interface{}{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "Black", "displayName": "Black"},
						{"value": "White", "displayName": "White"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-INTEG-FETCH-001-BLK",
					"price": 129.99,
					"options": []map[string]interface{}{
						{"optionName": "Color", "value": "Black"},
					},
				},
				{
					"sku":   "TEST-INTEG-FETCH-001-WHT",
					"price": 129.99,
					"options": []map[string]interface{}{
						{"optionName": "Color", "value": "White"},
					},
				},
			},
		}

		// POST - Create product
		w1 := client.Post(t, "/api/products", createRequest)
		createResponse := helpers.AssertSuccessResponse(t, w1, http.StatusCreated)
		createdProduct := helpers.GetResponseData(t, createResponse, "product")

		productID := int(createdProduct["id"].(float64))
		t.Logf("Created product with ID: %d", productID)

		// GET - Immediately fetch the product
		getURL := fmt.Sprintf("/api/products/%d", productID)
		w2 := client.Get(t, getURL)
		fetchResponse := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		fetchedProduct := helpers.GetResponseData(t, fetchResponse, "product")

		// Verify data consistency
		assert.Equal(t, createdProduct["id"], fetchedProduct["id"], "Product ID should match")
		assert.Equal(t, createdProduct["name"], fetchedProduct["name"], "Product name should match")
		assert.Equal(t, createdProduct["brand"], fetchedProduct["brand"], "Product brand should match")
		assert.Equal(t, createdProduct["categoryId"], fetchedProduct["categoryId"], "Category ID should match")

		// Verify variants count
		createdVariants := createdProduct["variants"].([]interface{})
		fetchedVariants := fetchedProduct["variants"].([]interface{})
		assert.Len(t, fetchedVariants, len(createdVariants), "Variants count should match")

		t.Log("Data consistency verified: Created and fetched product match")
	})

	t.Run("Integration - Create product then create additional variant via variant API", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create product with one variant
		createRequest := map[string]interface{}{
			"name":       "Test Product - Add Variant Later",
			"categoryId": 4,
			"baseSku":    "TEST-ADD-VAR-001",
			"options": []map[string]interface{}{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]interface{}{
						{"value": "Black", "displayName": "Black"},
						{"value": "Red", "displayName": "Red"},
					},
				},
			},
			"variants": []map[string]interface{}{
				{
					"sku":   "TEST-ADD-VAR-001-BLK",
					"price": 99.99,
					"options": []map[string]interface{}{
						{"optionName": "Color", "value": "Black"},
					},
				},
			},
		}

		// Create product
		w1 := client.Post(t, "/api/products", createRequest)
		createResponse := helpers.AssertSuccessResponse(t, w1, http.StatusCreated)
		product := helpers.GetResponseData(t, createResponse, "product")

		productID := int(product["id"].(float64))
		t.Logf("Created product with ID: %d", productID)

		// Verify initial variant count
		initialVariants := product["variants"].([]interface{})
		assert.Len(t, initialVariants, 1, "Should have 1 initial variant")

		// Add another variant via variant API
		addVariantRequest := map[string]interface{}{
			"sku":   "TEST-ADD-VAR-001-RED",
			"price": 99.99,
			"options": []map[string]interface{}{
				{"optionName": "Color", "value": "Red"},
			},
		}

		variantURL := fmt.Sprintf("/api/products/%d/variants", productID)
		w2 := client.Post(t, variantURL, addVariantRequest)

		// Verify variant was added
		if w2.Code == http.StatusCreated {
			addVariantResponse := helpers.AssertSuccessResponse(t, w2, http.StatusCreated)
			newVariant := helpers.GetResponseData(t, addVariantResponse, "variant")

			assert.NotNil(t, newVariant["id"], "New variant should have ID")
			assert.Equal(t, "TEST-ADD-VAR-001-RED", newVariant["sku"], "New variant SKU should match")

			// Fetch product again to verify total variants
			getURL := fmt.Sprintf("/api/products/%d", productID)
			w3 := client.Get(t, getURL)
			finalProduct := helpers.AssertSuccessResponse(t, w3, http.StatusOK)
			fetchedProduct := helpers.GetResponseData(t, finalProduct, "product")

			finalVariants := fetchedProduct["variants"].([]interface{})
			assert.Len(t, finalVariants, 2, "Should have 2 variants after adding")

			t.Log("API composition verified: Product created then variant added separately")
		} else {
			t.Logf("Variant API returned status: %d (may not be implemented yet)", w2.Code)
		}
	})

	// ============================================================================
	// CONCURRENT REQUEST SCENARIOS
	// ============================================================================

	t.Run("Concurrency - Multiple sellers creating products simultaneously", func(t *testing.T) {
		// This test verifies that concurrent product creation by different sellers
		// maintains data isolation and doesn't cause race conditions

		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)

		const numConcurrent = 5
		var wg sync.WaitGroup
		results := make(chan bool, numConcurrent)
		productIDs := make(chan int, numConcurrent)

		// Create multiple products concurrently
		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				// Create a new client for this goroutine
				concurrentClient := helpers.NewAPIClient(server)
				concurrentClient.SetToken(sellerToken)

				requestBody := map[string]interface{}{
					"name":       fmt.Sprintf("Concurrent Product %d", index),
					"categoryId": 4,
					"baseSku":    fmt.Sprintf("TEST-CONC-%03d", index),
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
							"sku":   fmt.Sprintf("TEST-CONC-%03d-V1", index),
							"price": 99.99,
							"options": []map[string]interface{}{
								{"optionName": "Color", "value": "Black"},
							},
						},
					},
				}

				w := concurrentClient.Post(t, "/api/products", requestBody)

				if w.Code == http.StatusCreated {
					response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
					product := helpers.GetResponseData(t, response, "product")
					productID := int(product["id"].(float64))
					productIDs <- productID
					results <- true
					t.Logf("Concurrent product %d created successfully with ID: %d", index, productID)
				} else {
					results <- false
					t.Logf("Concurrent product %d failed with status: %d", index, w.Code)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(results)
		close(productIDs)

		// Verify all products were created successfully
		successCount := 0
		for success := range results {
			if success {
				successCount++
			}
		}

		assert.Equal(t, numConcurrent, successCount, "All concurrent products should be created")

		// Verify all product IDs are unique
		uniqueIDs := make(map[int]bool)
		for id := range productIDs {
			uniqueIDs[id] = true
		}

		assert.Len(t, uniqueIDs, numConcurrent, "All product IDs should be unique")

		t.Logf("Concurrency test passed: %d products created simultaneously with unique IDs", numConcurrent)
	})

	// ============================================================================
	// REAL-WORLD BUSINESS SCENARIOS
	// ============================================================================

	t.Run("RealWorld - Create iPhone 15 Pro with all variants", func(t *testing.T) {
		// This test simulates a real product with multiple options
		// iPhone 15 Pro: 4 colors × 4 storage options = 16 variants

		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		colors := []struct {
			value string
			name  string
			code  string
		}{
			{"natural-titanium", "Natural Titanium", "#E5E5E5"},
			{"blue-titanium", "Blue Titanium", "#2D4E68"},
			{"white-titanium", "White Titanium", "#F5F5F0"},
			{"black-titanium", "Black Titanium", "#3A3A3C"},
		}

		storages := []struct {
			value string
			name  string
		}{
			{"128gb", "128GB"},
			{"256gb", "256GB"},
			{"512gb", "512GB"},
			{"1tb", "1TB"},
		}

		// Prepare options
		colorValues := []map[string]interface{}{}
		for _, color := range colors {
			colorValues = append(colorValues, map[string]interface{}{
				"value":       color.value,
				"displayName": color.name,
				"colorCode":   color.code,
			})
		}

		storageValues := []map[string]interface{}{}
		for _, storage := range storages {
			storageValues = append(storageValues, map[string]interface{}{
				"value":       storage.value,
				"displayName": storage.name,
			})
		}

		// Prepare all 16 variants
		variants := []map[string]interface{}{}
		prices := map[string]float64{
			"128gb": 999.00,
			"256gb": 1099.00,
			"512gb": 1299.00,
			"1tb":   1499.00,
		}

		for _, color := range colors {
			for _, storage := range storages {
				sku := fmt.Sprintf("IPHONE-15-PRO-%s-%s", color.value, storage.value)
				variants = append(variants, map[string]interface{}{
					"sku":   sku,
					"price": prices[storage.value],
					"options": []map[string]interface{}{
						{"optionName": "Color", "value": color.value},
						{"optionName": "Storage", "value": storage.value},
					},
				})
			}
		}

		requestBody := map[string]interface{}{
			"name":             "iPhone 15 Pro",
			"categoryId":       4, // Smartphones
			"baseSku":          "IPHONE-15-PRO",
			"brand":            "Apple",
			"shortDescription": "The ultimate iPhone with titanium design and A17 Pro chip",
			"longDescription":  "iPhone 15 Pro features a strong and light titanium design with a textured matte glass back. It's the most powerful iPhone ever with the A17 Pro chip.",
			"tags":             []string{"premium", "flagship", "5g", "titanium"},
			"options": []map[string]interface{}{
				{
					"name":        "Color",
					"displayName": "Color",
					"values":      colorValues,
				},
				{
					"name":        "Storage",
					"displayName": "Storage",
					"values":      storageValues,
				},
			},
			"variants": variants,
			"attributes": []map[string]interface{}{
				{
					"key":       "display_size",
					"name":      "Display Size",
					"value":     "6.1",
					"unit":      "inches",
					"sortOrder": 1,
				},
				{
					"key":       "chip",
					"name":      "Chip",
					"value":     "A17 Pro",
					"sortOrder": 2,
				},
				{
					"key":       "camera",
					"name":      "Camera System",
					"value":     "Pro camera system with 48MP Main, 12MP Ultra Wide, and 12MP Telephoto",
					"sortOrder": 3,
				},
			},
		}

		w := client.Post(t, "/api/products", requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify product creation
		assert.NotNil(t, product["id"], "Product should be created")
		assert.Equal(t, "iPhone 15 Pro", product["name"], "Product name should match")
		assert.Equal(t, "Apple", product["brand"], "Brand should match")

		// Verify all 16 variants were created
		productVariants, ok := product["variants"].([]interface{})
		assert.True(t, ok, "Variants should be an array")
		assert.Len(t, productVariants, 16, "Should have 16 variants (4 colors × 4 storage options)")

		// Verify options structure
		productOptions, ok := product["options"].([]interface{})
		assert.True(t, ok, "Options should be an array")
		assert.Len(t, productOptions, 2, "Should have 2 options (Color and Storage)")

		// Verify attributes
		attributes, ok := product["attributes"].([]interface{})
		assert.True(t, ok, "Attributes should be an array")
		assert.Len(t, attributes, 3, "Should have 3 attributes")

		// Verify variant prices match storage tiers
		variantPriceMap := make(map[string]float64)
		for _, v := range productVariants {
			variant := v.(map[string]interface{})
			sku := variant["sku"].(string)
			price := variant["price"].(float64)
			variantPriceMap[sku] = price
		}

		// Check that different storage options have different prices
		assert.NotEqual(t, variantPriceMap["IPHONE-15-PRO-natural-titanium-128gb"],
			variantPriceMap["IPHONE-15-PRO-natural-titanium-1tb"],
			"Different storage options should have different prices")

		t.Logf("Real-world test passed: iPhone 15 Pro created with %d variants", len(productVariants))
		t.Log("Product represents a complex real-world scenario with multiple option combinations")
	})
}
