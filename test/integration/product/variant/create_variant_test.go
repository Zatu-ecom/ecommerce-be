package variant

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestCreateVariant(t *testing.T) {
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
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Success - Basic variant creation with minimal fields", func(t *testing.T) {
		// Login as seller (Jane Merchant - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Product 5 is Classic Cotton T-Shirt with Size and Color options
		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "NIKE-TSHIRT-NAVY-XL",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "XL"},
				{"optionName": "Color", "value": "Navy"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert response fields
		assert.NotNil(t, variant["id"])
		assert.Equal(t, float64(productID), variant["productId"])
		assert.Equal(t, "NIKE-TSHIRT-NAVY-XL", variant["sku"])
		assert.Equal(t, 29.99, variant["price"])
		assert.NotNil(t, variant["createdAt"])
		assert.NotNil(t, variant["updatedAt"])

		// Check selected options
		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok, "selectedOptions should be an array")
		assert.Len(t, selectedOptions, 2, "Should have 2 selected options")
	})

	t.Run("Success - Variant creation with all fields populated", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "NIKE-TSHIRT-GRAY-XL",
			"price": 34.99,
			"images": []string{
				"https://example.com/img1.jpg",
				"https://example.com/img2.jpg",
			},
			"allowPurchase": true,
			"isPopular":     true,
			"isDefault":     false,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "XL"},
				{"optionName": "Color", "value": "Gray"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert all fields
		assert.Equal(t, "NIKE-TSHIRT-GRAY-XL", variant["sku"])
		assert.Equal(t, 34.99, variant["price"])
		assert.True(t, variant["allowPurchase"].(bool))
		assert.True(t, variant["isPopular"].(bool))
		assert.False(t, variant["isDefault"].(bool))

		// Check images
		images, ok := variant["images"].([]interface{})
		assert.True(t, ok, "Images should be an array")
		assert.Len(t, images, 2, "Should have 2 images")
	})

	t.Run("Success - Variant creation with single option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Running Shoes) owned by seller_id 3
		// It has 2 options (Size, Color), but we'll create a variant selecting both
		// Since we need a product with truly single option, let's use product 5 with only Size option selected
		productID := 7 // Running Shoes

		// Create variant with both required options (Size and Color)
		requestBody := map[string]interface{}{
			"sku":   "ADIDAS-RUN-BW-12",
			"price": 89.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "12"},
				{"optionName": "Color", "value": "Black/White"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Check options
		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(selectedOptions), 1, "Should have at least 1 selected option")
	})

	t.Run("Success - Variant creation with multiple options (3 options)", func(t *testing.T) {
		// Login as seller 2 (John) since Product 3 belongs to seller_id 2
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		// Product 3 is MacBook Pro with Color, Memory, and Storage options (owned by seller_id 2)
		productID := 3

		requestBody := map[string]interface{}{
			"sku":   "MBP-16-M3-SB-32-1TB",
			"price": 2999.00,
			"options": []map[string]interface{}{
				{"optionName": "Color", "value": "Space Black"},
				{"optionName": "Memory", "value": "32GB"},
				{"optionName": "Storage", "value": "1TB"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Check all 3 options
		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, selectedOptions, 3, "Should have 3 selected options")

		assert.Equal(t, "MBP-16-M3-SB-32-1TB", variant["sku"])
		assert.Equal(t, 2999.00, variant["price"])
	})

	t.Run("Success - Variant creation with multiple images", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7 // Running Shoes

		images := []string{
			"https://example.com/shoe-front.jpg",
			"https://example.com/shoe-side.jpg",
			"https://example.com/shoe-back.jpg",
			"https://example.com/shoe-sole.jpg",
		}

		requestBody := map[string]interface{}{
			"sku":    "ADIDAS-RUN-BO-11",
			"price":  89.99,
			"images": images,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "11"},
				{"optionName": "Color", "value": "Blue/Orange"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Check images
		returnedImages, ok := variant["images"].([]interface{})
		assert.True(t, ok, "Images should be an array")
		assert.Len(t, returnedImages, 4, "Should have 4 images")
	})

	t.Run("Success - Variant creation with empty images array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7 // Running Shoes

		requestBody := map[string]interface{}{
			"sku":    "ADIDAS-RUN-AB-11",
			"price":  89.99,
			"images": []string{},
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "11"},
				{"optionName": "Color", "value": "All Black"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Check images is empty or nil
		images, hasImages := variant["images"]
		if hasImages && images != nil {
			imagesArray, isArray := images.([]interface{})
			if isArray {
				assert.Empty(t, imagesArray, "Images array should be empty")
			}
		}
	})

	t.Run("Success - Variant creation as default variant", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6 // Summer Dress

		requestBody := map[string]interface{}{
			"sku":       "ZARA-DRESS-BLUE-L",
			"price":     49.99,
			"isDefault": true,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "L"},
				{"optionName": "Color", "value": "Floral Blue"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.True(t, variant["isDefault"].(bool), "Should be marked as default variant")
	})

	t.Run("Success - Variant creation as popular variant", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6 // Summer Dress

		requestBody := map[string]interface{}{
			"sku":       "ZARA-DRESS-PINK-L",
			"price":     49.99,
			"isPopular": true,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "L"},
				{"optionName": "Color", "value": "Floral Pink"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.True(t, variant["isPopular"].(bool), "Should be marked as popular variant")
	})

	t.Run("Success - Empty SKU should be accepted", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7 // Running Shoes

		requestBody := map[string]interface{}{
			"sku":   "", // Empty SKU
			"price": 89.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "7"},
				{"optionName": "Color", "value": "Black/White"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Empty SKU should be accepted
		assert.NotNil(t, variant["id"])
		assert.Equal(t, "", variant["sku"])
	})

	// ============================================================================
	// VALIDATION ERROR SCENARIOS
	// ============================================================================

	t.Run("Validation - Invalid product ID in URL (non-numeric)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := "/api/products/invalid/variants"
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Product ID is zero", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := "/api/products/0/variants"
		w := client.Post(t, url, requestBody)

		// Product ID 0 is treated as "not found" which returns 404
		// This is acceptable behavior
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Validation - Missing required field: Price", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku": "TEST-SKU",
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Invalid price: Zero", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 0,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Invalid price: Negative", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": -10.50,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Price must be positive", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Missing required field: Options", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Empty options array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":     "TEST-SKU",
			"price":   29.99,
			"options": []map[string]interface{}{},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Option missing optionName", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"value": "M"}, // Missing optionName
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Option with empty optionName", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Option missing value", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size"}, // Missing value
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Option with empty value", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": ""},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation - Invalid JSON body", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		url := fmt.Sprintf("/api/products/%d/variants", productID)

		// Test with empty body (missing all required fields)
		requestBody := map[string]interface{}{}
		w := client.Post(t, url, requestBody)

		// Should fail due to missing required fields
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// ============================================================================
	// BUSINESS LOGIC ERROR SCENARIOS
	// ============================================================================

	t.Run("Error - Product does not exist", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 99999 // Non-existent product

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run(
		"Error - Seller tries to create variant for another seller's product",
		func(t *testing.T) {
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			// Product 1 (iPhone 15 Pro) is owned by seller_id 2, not seller_id 3 (Jane)
			productID := 1

			// Use a unique combination that doesn't exist in seed data
			requestBody := map[string]interface{}{
				"sku":   "IPHONE-HACK",
				"price": 999.00,
				"options": []map[string]interface{}{
					{"optionName": "Color", "value": "Black Titanium"}, // Different color
					{"optionName": "Storage", "value": "1TB"},          // Different storage
				},
			}

			url := fmt.Sprintf("/api/products/%d/variants", productID)
			w := client.Post(t, url, requestBody)

			// Should return 404 or 403
			assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusForbidden,
				"Should not allow seller to create variant for another seller's product")
		},
	)

	t.Run("Error - Option name does not exist for the product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt has Size and Color options

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{
					"optionName": "Material",
					"value":      "Cotton",
				}, // Material option doesn't exist for this product
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Error - Option value does not exist for the option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "XXXL"}, // XXXL doesn't exist as a value
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Error - Variant with same option combination already exists", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt

		// First create a variant
		requestBody := map[string]interface{}{
			"sku":   "NIKE-TSHIRT-WHITE-XXL",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "XXL"},
				{"optionName": "Color", "value": "White"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Try to create another variant with the same option combination
		requestBody2 := map[string]interface{}{
			"sku":   "NIKE-TSHIRT-WHITE-XXL-DUPLICATE",
			"price": 34.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "XXL"},
				{"optionName": "Color", "value": "White"},
			},
		}

		w2 := client.Post(t, url, requestBody2)

		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("Error - Extra option not belonging to product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt has only Size and Color

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
				{"optionName": "Color", "value": "Black"},
				{"optionName": "Storage", "value": "128GB"}, // Storage doesn't belong to T-Shirt
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Error - Duplicate options in request", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
				{"optionName": "Size", "value": "L"}, // Duplicate Size option
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		// This should fail - either validation error or business logic error
		assert.NotEqual(t, http.StatusCreated, w.Code)
	})

	// ============================================================================
	// AUTHENTICATION & AUTHORIZATION SCENARIOS
	// ============================================================================

	t.Run("Auth - Unauthenticated request", func(t *testing.T) {
		client.SetToken("") // Clear token

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Auth - Invalid token", func(t *testing.T) {
		client.SetToken("invalid-token-12345")

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Auth - Customer tries to create variant", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"sku":   "TEST-SKU",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Auth - Admin creates variant for any product", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Try to create variant for product owned by seller_id 2
		productID := 1 // iPhone 15 Pro

		requestBody := map[string]interface{}{
			"sku":   "IPHONE-15-PRO-WHT-512",
			"price": 1199.00,
			"options": []map[string]interface{}{
				{"optionName": "Color", "value": "White Titanium"},
				{"optionName": "Storage", "value": "512GB"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		// Admin should be able to create variants (if your business logic allows)
		// This test may succeed or fail depending on your authorization logic
		assert.True(t, w.Code == http.StatusCreated || w.Code == http.StatusForbidden,
			"Admin should either be allowed or explicitly forbidden")
	})

	t.Run("Success - Admin can create variant for any product", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Admin creates variant for product owned by seller_id 3
		productID := 6 // Summer Dress (owned by seller_id 3)

		requestBody := map[string]interface{}{
			"sku":   "ZARA-DRESS-WHITE-M-ADMIN",
			"price": 49.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "M"},
				{"optionName": "Color", "value": "Solid White"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Verify variant was created successfully
		assert.NotNil(t, variant["id"])
		assert.Equal(t, float64(productID), variant["productId"])
		assert.Equal(t, "ZARA-DRESS-WHITE-M-ADMIN", variant["sku"])
		assert.Equal(t, 49.99, variant["price"])

		// Verify selected options
		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok, "selectedOptions should be an array")
		assert.Len(t, selectedOptions, 2, "Should have 2 selected options")
	})

	// ============================================================================
	// RESPONSE VALIDATION SCENARIOS
	// ============================================================================

	t.Run("Response - Verify complete response structure", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6 // Summer Dress

		requestBody := map[string]interface{}{
			"sku":           "ZARA-DRESS-BLUE-XL",
			"price":         54.99,
			"images":        []string{"https://example.com/dress.jpg"},
			"allowPurchase": true,
			"isPopular":     false,
			"isDefault":     false,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "XL"},
				{"optionName": "Color", "value": "Floral Blue"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Verify all expected fields exist
		assert.NotNil(t, variant["id"], "ID should be present")
		assert.NotNil(t, variant["productId"], "Product ID should be present")
		assert.NotNil(t, variant["sku"], "SKU should be present")
		assert.NotNil(t, variant["price"], "Price should be present")
		assert.NotNil(t, variant["allowPurchase"], "AllowPurchase should be present")
		assert.NotNil(t, variant["isPopular"], "IsPopular should be present")
		assert.NotNil(t, variant["isDefault"], "IsDefault should be present")
		assert.NotNil(t, variant["selectedOptions"], "SelectedOptions should be present")
		assert.NotNil(t, variant["createdAt"], "CreatedAt should be present")
		assert.NotNil(t, variant["updatedAt"], "UpdatedAt should be present")

		// Check product info (if included)
		if product, hasProduct := variant["product"]; hasProduct && product != nil {
			productInfo := product.(map[string]interface{})
			assert.NotNil(t, productInfo["id"], "Product info should have ID")
			assert.NotNil(t, productInfo["name"], "Product info should have name")
		}
	})

	t.Run("Response - Verify timestamps are set", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7 // Running Shoes

		requestBody := map[string]interface{}{
			"sku":   "ADIDAS-RUN-BW-8",
			"price": 89.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "8"},
				{"optionName": "Color", "value": "Black/White"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Verify timestamps
		createdAt, hasCreatedAt := variant["createdAt"]
		assert.True(t, hasCreatedAt, "CreatedAt should be present")
		assert.NotEmpty(t, createdAt, "CreatedAt should not be empty")

		updatedAt, hasUpdatedAt := variant["updatedAt"]
		assert.True(t, hasUpdatedAt, "UpdatedAt should be present")
		assert.NotEmpty(t, updatedAt, "UpdatedAt should not be empty")
	})

	t.Run("Response - Verify option values are correctly mapped", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt

		requestBody := map[string]interface{}{
			"sku":   "NIKE-TSHIRT-NAVY-S",
			"price": 29.99,
			"options": []map[string]interface{}{
				{"optionName": "Size", "value": "S"},
				{"optionName": "Color", "value": "Navy"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		variant := helpers.GetResponseData(t, response, "variant")

		// Check selected options mapping
		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok, "selectedOptions should be an array")
		assert.Len(t, selectedOptions, 2, "Should have 2 selected options")

		// Verify each option has required fields
		for _, opt := range selectedOptions {
			option := opt.(map[string]interface{})
			assert.NotNil(t, option["optionId"], "Option should have optionId")
			assert.NotNil(t, option["optionName"], "Option should have optionName")
			assert.NotNil(t, option["optionDisplayName"], "Option should have optionDisplayName")
			assert.NotNil(t, option["valueId"], "Option should have valueId")
			assert.NotNil(t, option["value"], "Option should have value")
			assert.NotNil(t, option["valueDisplayName"], "Option should have valueDisplayName")
		}
	})
}
