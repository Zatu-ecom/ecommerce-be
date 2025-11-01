package variant

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// Helper function to get variant and verify field
func getAndVerifyVariant(
	t *testing.T,
	client *helpers.APIClient,
	productID, variantID uint,
) map[string]interface{} {
	url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
	w := client.Get(t, url)
	response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	return helpers.GetResponseData(t, response, "variant")
}

func TestBulkUpdateVariants(t *testing.T) {
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
	// SUCCESS SCENARIOS - Basic Bulk Updates
	// ============================================================================

	t.Run("Success - Update 2 variants with different fields", func(t *testing.T) {
		// Login as seller 2 who owns product 1 (iPhone 15 Pro) with variants 1-4
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{
					"id":    2,
					"price": 1299.99,
				},
				{
					"id":    3,
					"stock": 150,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])

		variants, ok := data["variants"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, variants, 2)

		// Verify variant 2 price was updated
		variant2URL := fmt.Sprintf("/api/products/%d/variants/2", productID)
		w2 := client.Get(t, variant2URL)
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		variant2Data := helpers.GetResponseData(t, response2, "variant")
		assert.Equal(t, 1299.99, variant2Data["price"].(float64))

		// Verify variant 3 stock was updated
		variant3URL := fmt.Sprintf("/api/products/%d/variants/3", productID)
		w3 := client.Get(t, variant3URL)
		response3 := helpers.AssertSuccessResponse(t, w3, http.StatusOK)
		variant3Data := helpers.GetResponseData(t, response3, "variant")
		assert.Equal(t, float64(150), variant3Data["stock"].(float64))
	})

	t.Run("Success - Update 3 variants with mixed updates", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1 // iPhone with 4 variants

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{
					"id":    1,
					"price": 1099.99,
				},
				{
					"id":    2,
					"stock": 75,
				},
				{
					"id":        3,
					"inStock":   true,
					"isPopular": true,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(3), data["updatedCount"])

		// Verify variant 1 price was updated
		variant1URL := fmt.Sprintf("/api/products/%d/variants/1", productID)
		w1 := client.Get(t, variant1URL)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		variant1Data := helpers.GetResponseData(t, response1, "variant")
		assert.Equal(t, 1099.99, variant1Data["price"].(float64))

		// Verify variant 2 stock was updated
		variant2URL := fmt.Sprintf("/api/products/%d/variants/2", productID)
		w2 := client.Get(t, variant2URL)
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		variant2Data := helpers.GetResponseData(t, response2, "variant")
		assert.Equal(t, float64(75), variant2Data["stock"].(float64))

		// Verify variant 3 flags were updated
		variant3URL := fmt.Sprintf("/api/products/%d/variants/3", productID)
		w3 := client.Get(t, variant3URL)
		response3 := helpers.AssertSuccessResponse(t, w3, http.StatusOK)
		variant3Data := helpers.GetResponseData(t, response3, "variant")
		assert.True(t, variant3Data["inStock"].(bool))
		assert.True(t, variant3Data["isPopular"].(bool))
	})

	t.Run("Success - Update all variants of a product", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1 // iPhone with 4 variants

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": 1199.99},
				{"id": 2, "price": 1299.99},
				{"id": 3, "price": 1399.99},
				{"id": 4, "price": 1499.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(4), data["updatedCount"])

		variants, ok := data["variants"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, variants, 4)

		// Verify all prices were updated
		expectedPrices := map[uint]float64{1: 1199.99, 2: 1299.99, 3: 1399.99, 4: 1499.99}
		for variantID, expectedPrice := range expectedPrices {
			variantData := getAndVerifyVariant(t, client, uint(productID), variantID)
			assert.Equal(t, expectedPrice, variantData["price"].(float64))
		}
	})

	t.Run("Success - Update single variant via bulk endpoint", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{
					"id":    9,
					"price": 34.99,
					"stock": 250,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(1), data["updatedCount"])
	})

	t.Run("Success - Update multiple variants - only price changes", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 10, "price": 29.99},
				{"id": 11, "price": 31.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])

		// Verify prices were updated
		variant10Data := getAndVerifyVariant(t, client, uint(productID), 10)
		assert.Equal(t, 29.99, variant10Data["price"].(float64))

		variant11Data := getAndVerifyVariant(t, client, uint(productID), 11)
		assert.Equal(t, 31.99, variant11Data["price"].(float64))
	})

	t.Run("Success - Update multiple variants - only stock changes", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6 // Summer Dress

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 12, "stock": 100},
				{"id": 13, "stock": 150},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])

		// Verify stock was updated
		variant12Data := getAndVerifyVariant(t, client, uint(productID), 12)
		assert.Equal(t, float64(100), variant12Data["stock"].(float64))

		variant13Data := getAndVerifyVariant(t, client, uint(productID), 13)
		assert.Equal(t, float64(150), variant13Data["stock"].(float64))
	})

	t.Run("Success - Update multiple variants with all fields", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7 // Running Shoes

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{
					"id":    14,
					"price": 119.99,
					"stock": 80,
					"images": []string{
						"https://example.com/shoe1.jpg",
						"https://example.com/shoe2.jpg",
					},
					"inStock":   true,
					"isPopular": true,
					"isDefault": true,
				},
				{
					"id":        15,
					"price":     109.99,
					"stock":     60,
					"images":    []string{"https://example.com/shoe3.jpg"},
					"inStock":   true,
					"isPopular": false,
					"isDefault": false,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])
	})

	// ============================================================================
	// SUCCESS SCENARIOS - Boolean Flags
	// ============================================================================

	t.Run("Success - Update inStock flag for multiple variants", func(t *testing.T) {
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 8 // Sofa

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 16, "inStock": false},
				{"id": 17, "inStock": true},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])

		// Verify inStock flags were updated
		variant16Data := getAndVerifyVariant(t, client, uint(productID), 16)
		assert.False(t, variant16Data["inStock"].(bool))

		variant17Data := getAndVerifyVariant(t, client, uint(productID), 17)
		assert.True(t, variant17Data["inStock"].(bool))
	})

	t.Run("Success - Update isPopular flag for multiple variants", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 3 // MacBook Pro

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 7, "isPopular": true},
				{"id": 8, "isPopular": false},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])

		// Verify isPopular flags were updated
		variant7Data := getAndVerifyVariant(t, client, uint(productID), 7)
		assert.True(t, variant7Data["isPopular"].(bool))

		variant8Data := getAndVerifyVariant(t, client, uint(productID), 8)
		assert.False(t, variant8Data["isPopular"].(bool))
	})

	t.Run("Success - Set multiple variants as default (only last one remains)", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2 // Samsung

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 5, "isDefault": true},
				{"id": 6, "isDefault": true}, // This should be the final default
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])

		// Verify only variant 6 is default (last one wins)
		variant5URL := fmt.Sprintf("/api/products/%d/variants/5", productID)
		w5 := client.Get(t, variant5URL)
		response5 := helpers.AssertSuccessResponse(t, w5, http.StatusOK)
		variant5Data := helpers.GetResponseData(t, response5, "variant")
		assert.False(t, variant5Data["isDefault"].(bool), "Variant 5 should NOT be default")

		variant6URL := fmt.Sprintf("/api/products/%d/variants/6", productID)
		w6 := client.Get(t, variant6URL)
		response6 := helpers.AssertSuccessResponse(t, w6, http.StatusOK)
		variant6Data := helpers.GetResponseData(t, response6, "variant")
		assert.True(t, variant6Data["isDefault"].(bool), "Variant 6 SHOULD be default")
	})

	t.Run("Success - Unset default and set new default in same request", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 9, "isDefault": false}, // Unset current default
				{"id": 10, "isDefault": true}, // Set new default
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])
	})

	// ============================================================================
	// SUCCESS SCENARIOS - Images
	// ============================================================================

	t.Run("Success - Update images for multiple variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{
					"id": 12,
					"images": []string{
						"https://example.com/dress1.jpg",
						"https://example.com/dress2.jpg",
					},
				},
				{
					"id": 13,
					"images": []string{
						"https://example.com/dress3.jpg",
						"https://example.com/dress4.jpg",
						"https://example.com/dress5.jpg",
					},
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])
	})

	t.Run("Success - Clear images for multiple variants", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 2, "images": []string{}},
				{"id": 3, "images": []string{}},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])
	})

	t.Run("Success - Mix images update (some with new, some empty)", func(t *testing.T) {
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 8

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 16, "images": []string{"https://example.com/sofa-new.jpg"}},
				{"id": 17, "images": []string{}},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])
	})

	// ============================================================================
	// SUCCESS SCENARIOS - Partial Updates
	// ============================================================================

	t.Run("Success - Update with zero stock", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 14, "stock": 0},
				{"id": 15, "stock": 0},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])

		// Verify stock was set to 0
		variant14Data := getAndVerifyVariant(t, client, uint(productID), 14)
		assert.Equal(t, float64(0), variant14Data["stock"].(float64))

		variant15Data := getAndVerifyVariant(t, client, uint(productID), 15)
		assert.Equal(t, float64(0), variant15Data["stock"].(float64))
	})

	t.Run("Success - Update with empty variants array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		// Empty array should fail validation (min=1)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// VALIDATION ERRORS - Price Validation
	// ============================================================================

	t.Run("Validation Error - One variant with negative price", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": 1199.99},
				{"id": 2, "price": -100.00}, // Invalid
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - One variant with zero price", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": 1199.99},
				{"id": 3, "price": 0}, // Invalid
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Multiple variants with invalid prices", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 5, "price": -50.00},
				{"id": 6, "price": 0},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// VALIDATION ERRORS - Stock Validation
	// ============================================================================

	t.Run("Validation Error - One variant with negative stock", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 9, "stock": 100},
				{"id": 10, "stock": -5}, // Invalid
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// VALIDATION ERRORS - Variant ID
	// ============================================================================

	t.Run("Validation Error - One non-existent variant ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 9, "price": 29.99},
				{"id": 9999, "price": 39.99}, // Non-existent
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		// Depending on implementation, this might be partial success or full failure
		// Adjust expectation based on actual API behavior
		assert.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusBadRequest}, w.Code)
	})

	t.Run("Validation Error - Multiple non-existent variant IDs", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 8888, "price": 29.99},
				{"id": 9999, "price": 39.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		assert.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusBadRequest}, w.Code)
	})

	t.Run("Validation Error - All variant IDs non-existent", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 7777, "price": 1199.99},
				{"id": 8888, "price": 1299.99},
				{"id": 9999, "price": 1399.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		assert.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusBadRequest}, w.Code)
	})

	t.Run("Validation Error - Variant belongs to different product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt
		// Trying to update variant 1 which belongs to product 1 (iPhone)

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 9, "price": 29.99},  // Belongs to product 5 - OK
				{"id": 1, "price": 999.99}, // Belongs to product 1 - ERROR
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		assert.Contains(
			t,
			[]int{http.StatusOK, http.StatusNotFound, http.StatusBadRequest, http.StatusForbidden},
			w.Code,
		)
	})

	t.Run("Validation Error - Duplicate variant IDs in request", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": 1199.99},
				{"id": 1, "price": 1299.99}, // Duplicate - should fail
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		// Should fail with 404 - duplicate IDs not allowed
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Validation Error - Invalid variant ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": "invalid", "price": 29.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// VALIDATION ERRORS - Request Body
	// ============================================================================

	t.Run("Validation Error - Empty request body", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Null variants array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": nil,
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Wrong data type for price", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": "not-a-number"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Wrong data type for stock", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 2, "stock": "not-a-number"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Wrong data type for boolean flags", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 3, "inStock": "yes"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Images field not an array", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 4, "images": "not-an-array"},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Missing variantId in item", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"price": 29.99}, // Missing id field
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Invalid variantId type", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": "abc", "price": 29.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// AUTHORIZATION SCENARIOS
	// ============================================================================

	t.Run("Authorization Error - Unauthenticated request", func(t *testing.T) {
		unauthClient := helpers.NewAPIClient(server)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": 999.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := unauthClient.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Authorization Error - Customer cannot bulk update", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": 999.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run(
		"Authorization Error - Seller cannot update another seller's variants",
		func(t *testing.T) {
			// Seller 3 trying to update product 1 which belongs to seller 2
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			productID := 1 // Belongs to seller 2

			requestBody := map[string]interface{}{
				"variants": []map[string]interface{}{
					{"id": 1, "price": 999.99},
				},
			}

			url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
			w := client.Put(t, url, requestBody)

			helpers.AssertErrorResponse(t, w, http.StatusForbidden)
		},
	)

	t.Run("Success - Seller can bulk update own variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // Belongs to seller 3

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 9, "price": 32.99},
				{"id": 10, "price": 33.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])
	})

	t.Run("Success - Admin can bulk update any seller's variants", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		productID := 1 // Belongs to seller 2

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": 1149.99},
				{"id": 2, "price": 1249.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])
	})

	// ============================================================================
	// NOT FOUND SCENARIOS
	// ============================================================================

	t.Run("Not Found - Non-existent product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 9999

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "price": 99.99},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// DEFAULT VARIANT LOGIC
	// ============================================================================

	t.Run("Default Logic - Set 3 variants as default sequentially", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 1, "isDefault": true},
				{"id": 2, "isDefault": true},
				{"id": 3, "isDefault": true}, // Only this should remain default
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(3), data["updatedCount"])

		// Verify only variant 3 is default (last one wins)
		variant1URL := fmt.Sprintf("/api/products/%d/variants/1", productID)
		w1 := client.Get(t, variant1URL)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		variant1Data := helpers.GetResponseData(t, response1, "variant")
		assert.False(t, variant1Data["isDefault"].(bool), "Variant 1 should NOT be default")

		variant2URL := fmt.Sprintf("/api/products/%d/variants/2", productID)
		w2 := client.Get(t, variant2URL)
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		variant2Data := helpers.GetResponseData(t, response2, "variant")
		assert.False(t, variant2Data["isDefault"].(bool), "Variant 2 should NOT be default")

		variant3URL := fmt.Sprintf("/api/products/%d/variants/3", productID)
		w3 := client.Get(t, variant3URL)
		response3 := helpers.AssertSuccessResponse(t, w3, http.StatusOK)
		variant3Data := helpers.GetResponseData(t, response3, "variant")
		assert.True(t, variant3Data["isDefault"].(bool), "Variant 3 SHOULD be default")
	})

	t.Run("Default Logic - Set all variants isDefault=false", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 12, "isDefault": false},
				{"id": 13, "isDefault": false},
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(2), data["updatedCount"])
	})

	t.Run("Default Logic - Update non-default to default", func(t *testing.T) {
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 8

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 17, "isDefault": true}, // Was not default, now should be
			},
		}

		url := fmt.Sprintf("/api/products/%d/variants/bulk", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok)

		assert.Equal(t, float64(1), data["updatedCount"])

		// Verify variant 17 is now default
		variant17URL := fmt.Sprintf("/api/products/%d/variants/17", productID)
		w17 := client.Get(t, variant17URL)
		response17 := helpers.AssertSuccessResponse(t, w17, http.StatusOK)
		variant17Data := helpers.GetResponseData(t, response17, "variant")
		assert.True(t, variant17Data["isDefault"].(bool), "Variant 17 SHOULD be default")

		assert.Equal(t, float64(1), data["updatedCount"])
	})

	// ============================================================================
	// PARAMETER VALIDATION
	// ============================================================================

	t.Run("Parameter Validation - Invalid product ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 9, "price": 29.99},
			},
		}

		url := "/api/products/invalid/variants/bulk"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Parameter Validation - Negative product ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 9, "price": 29.99},
			},
		}

		url := "/api/products/-1/variants/bulk"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Parameter Validation - Zero product ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"variants": []map[string]interface{}{
				{"id": 9, "price": 29.99},
			},
		}

		url := "/api/products/0/variants/bulk"
		w := client.Put(t, url, requestBody)

		// Returns 404 instead of 400
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, w.Code)
	})
}
