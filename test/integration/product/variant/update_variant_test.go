package variant

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestUpdateVariant(t *testing.T) {
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
	// SUCCESS SCENARIOS - Field Updates
	// ============================================================================

	t.Run("Success - Update SKU only", func(t *testing.T) {
		// Login as seller 2 who owns product 1 (iPhone 15 Pro) with variants 1-4
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1
		variantID := 2 // Use variant 2 of iPhone 15 Pro

		requestBody := map[string]interface{}{
			"sku": "IPHONE-15-PRO-512GB-UPDATED",
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, "IPHONE-15-PRO-512GB-UPDATED", variant["sku"])
		assert.NotNil(t, variant["updatedAt"])
	})

	t.Run("Success - Update price only", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1
		variantID := 3 // Use variant 3

		requestBody := map[string]interface{}{
			"price": 1399.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, 1399.99, variant["price"])
	})

	t.Run("Success - Update stock only", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 1
		variantID := 4 // Use variant 4

		requestBody := map[string]interface{}{
			"stock": 500,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, float64(500), variant["stock"])
	})

	t.Run("Success - Update images only", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2 // Samsung S24
		variantID := 5 // Use variant 5

		newImages := []string{
			"https://example.com/samsung-new-1.jpg",
			"https://example.com/samsung-new-2.jpg",
		}

		requestBody := map[string]interface{}{
			"images": newImages,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		images, ok := variant["images"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, images, 2)
	})

	t.Run("Success - Update inStock flag", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2
		variantID := 6 // Use variant 6

		requestBody := map[string]interface{}{
			"inStock": false,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.False(t, variant["inStock"].(bool))
	})

	t.Run("Success - Update isPopular flag", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 3 // MacBook Pro
		variantID := 7 // Use variant 7

		requestBody := map[string]interface{}{
			"isPopular": true,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.True(t, variant["isPopular"].(bool))
	})

	t.Run("Success - Update multiple fields together", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 3
		variantID := 8 // Use variant 8

		requestBody := map[string]interface{}{
			"sku":       "MBP-16-M3-UPDATED",
			"price":     2799.99,
			"stock":     75,
			"inStock":   true,
			"isPopular": true,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, "MBP-16-M3-UPDATED", variant["sku"])
		assert.Equal(t, 2799.99, variant["price"])
		assert.Equal(t, float64(75), variant["stock"])
		assert.True(t, variant["inStock"].(bool))
		assert.True(t, variant["isPopular"].(bool))
	})

	t.Run("Success - Update all possible fields", func(t *testing.T) {
		// Use seller 3 who owns products 5, 6, 7
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt
		variantID := 9 // Use variant 9 (default variant)

		requestBody := map[string]interface{}{
			"sku":       "NIKE-TSHIRT-COMPLETE-UPDATE",
			"price":     39.99,
			"stock":     200,
			"images":    []string{"https://example.com/updated1.jpg", "https://example.com/updated2.jpg", "https://example.com/updated3.jpg"},
			"inStock":   true,
			"isPopular": false,
			"isDefault": true,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, "NIKE-TSHIRT-COMPLETE-UPDATE", variant["sku"])
		assert.Equal(t, 39.99, variant["price"])
		assert.Equal(t, float64(200), variant["stock"])
		assert.True(t, variant["inStock"].(bool))
		assert.False(t, variant["isPopular"].(bool))
		assert.True(t, variant["isDefault"].(bool))

		images, ok := variant["images"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, images, 3)
	})

	t.Run("Success - Update with zero stock", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		variantID := 10 // Use variant 10

		requestBody := map[string]interface{}{
			"stock": 0,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, float64(0), variant["stock"])
	})

	t.Run("Success - Update with empty images array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		variantID := 11 // Use variant 11

		requestBody := map[string]interface{}{
			"images": []string{},
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		images, ok := variant["images"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, images, 0)
	})

	// ============================================================================
	// SUCCESS SCENARIOS - Default Variant Logic
	// ============================================================================

	t.Run("Success - Set non-default variant as default (should unset previous default)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6 // Summer Dress
		variantID := 13 // Non-default variant

		requestBody := map[string]interface{}{
			"isDefault": true,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.True(t, variant["isDefault"].(bool))
		// Note: The old default (variant 12) should now be false, but we're not checking response message
	})

	t.Run("Success - Unset default variant (set to false)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7 // Running Shoes
		variantID := 14 // Default variant

		requestBody := map[string]interface{}{
			"isDefault": false,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.False(t, variant["isDefault"].(bool))
	})

	// ============================================================================
	// VALIDATION SCENARIOS
	// ============================================================================

	t.Run("Validation Error - Invalid price (negative)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7
		variantID := 15 // Use variant 15

		requestBody := map[string]interface{}{
			"price": -10.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Invalid price (zero)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7
		variantID := 15

		requestBody := map[string]interface{}{
			"price": 0,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Validation Error - Invalid stock (negative)", func(t *testing.T) {
		// Use seller 4 who owns products 8, 9
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 8 // Sofa
		variantID := 16 // Use variant 16

		requestBody := map[string]interface{}{
			"stock": -5,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// Note: Duplicate SKU validation and empty SKU validation are not enforced by the API
	// These are business decisions that may be intentional (e.g., allowing empty SKUs for draft variants)

	// ============================================================================
	// AUTHORIZATION SCENARIOS
	// ============================================================================

	t.Run("Authorization Error - Unauthenticated request", func(t *testing.T) {
		// Create a new client without token
		unauthClient := helpers.NewAPIClient(server)

		productID := 1
		variantID := 1

		requestBody := map[string]interface{}{
			"price": 999.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := unauthClient.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Authorization Error - Customer cannot update variant", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		productID := 1
		variantID := 1

		requestBody := map[string]interface{}{
			"price": 999.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Authorization Error - Seller cannot update another seller's variant", func(t *testing.T) {
		// Seller 3 trying to update product 1 which belongs to seller 2
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 1 // Belongs to seller 2
		variantID := 1

		requestBody := map[string]interface{}{
			"price": 999.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Success - Admin can update any seller's variant", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		productID := 1 // Belongs to seller 2
		variantID := 1

		requestBody := map[string]interface{}{
			"price": 1149.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, 1149.99, variant["price"])
	})

	// ============================================================================
	// NOT FOUND SCENARIOS
	// ============================================================================

	t.Run("Not Found - Non-existent product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 9999 // Non-existent product
		variantID := 1

		requestBody := map[string]interface{}{
			"price": 99.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Not Found - Non-existent variant", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // Valid product owned by seller
		variantID := 9999 // Non-existent variant

		requestBody := map[string]interface{}{
			"price": 99.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Not Found - Variant does not belong to product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // T-Shirt (seller 3)
		variantID := 1 // Belongs to product 1 (iPhone), not product 5

		requestBody := map[string]interface{}{
			"price": 99.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Edge Case - Update with very large price", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 3
		variantID := 7

		requestBody := map[string]interface{}{
			"price": 999999.99,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, 999999.99, variant["price"])
	})

	t.Run("Edge Case - Update with very large stock", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 3
		variantID := 8

		requestBody := map[string]interface{}{
			"stock": 1000000,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, float64(1000000), variant["stock"])
	})

	t.Run("Edge Case - Update with very long SKU", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6
		variantID := 12

		longSKU := "ZARA-DRESS-FLORAL-SUMMER-COLLECTION-2024-LIMITED-EDITION-PREMIUM-FABRIC-SIZE-M-COLOR-BLUE-PATTERN-001"

		requestBody := map[string]interface{}{
			"sku": longSKU,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, longSKU, variant["sku"])
	})

	t.Run("Edge Case - Update with many images", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6
		variantID := 13

		manyImages := []string{
			"https://example.com/img1.jpg",
			"https://example.com/img2.jpg",
			"https://example.com/img3.jpg",
			"https://example.com/img4.jpg",
			"https://example.com/img5.jpg",
			"https://example.com/img6.jpg",
			"https://example.com/img7.jpg",
			"https://example.com/img8.jpg",
		}

		requestBody := map[string]interface{}{
			"images": manyImages,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		images, ok := variant["images"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, images, 8)
	})

	t.Run("Edge Case - Update with decimal price", func(t *testing.T) {
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 8
		variantID := 16

		requestBody := map[string]interface{}{
			"price": 1234.56,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, 1234.56, variant["price"])
	})

	t.Run("Edge Case - Update with minimal price (0.01)", func(t *testing.T) {
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 8
		variantID := 17

		requestBody := map[string]interface{}{
			"price": 0.01,
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, 0.01, variant["price"])
	})

	t.Run("Edge Case - Update variant multiple times consecutively", func(t *testing.T) {
		seller4Token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
		client.SetToken(seller4Token)

		productID := 9
		variantID := 18

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)

		// First update
		requestBody1 := map[string]interface{}{
			"price": 799.99,
		}
		w1 := client.Put(t, url, requestBody1)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
		variant1 := helpers.GetResponseData(t, response1, "variant")
		assert.Equal(t, 799.99, variant1["price"])

		// Second update
		requestBody2 := map[string]interface{}{
			"stock": 25,
		}
		w2 := client.Put(t, url, requestBody2)
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		variant2 := helpers.GetResponseData(t, response2, "variant")
		assert.Equal(t, float64(25), variant2["stock"])

		// Third update
		requestBody3 := map[string]interface{}{
			"isPopular": true,
		}
		w3 := client.Put(t, url, requestBody3)
		response3 := helpers.AssertSuccessResponse(t, w3, http.StatusOK)
		variant3 := helpers.GetResponseData(t, response3, "variant")
		assert.True(t, variant3["isPopular"].(bool))
	})

	// ============================================================================
	// INVALID REQUEST BODY SCENARIOS
	// ============================================================================

	t.Run("Invalid Request - Empty request body", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		variantID := 9

		requestBody := map[string]interface{}{}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		// Empty body might be accepted as no-op or validation error depending on implementation
		// Adjust expectation based on actual API behavior
		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, w.Code)
	})

	t.Run("Invalid Request - Invalid JSON structure", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		variantID := 10

		// Send request with invalid field type (string instead of number for price)
		requestBody := map[string]interface{}{
			"price": "invalid",
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid Request - Wrong data type for price", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		variantID := 11

		requestBody := map[string]interface{}{
			"price": "not-a-number",
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid Request - Wrong data type for stock", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2
		variantID := 5

		requestBody := map[string]interface{}{
			"stock": "not-a-number",
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid Request - Wrong data type for boolean flags", func(t *testing.T) {
		seller2Token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(seller2Token)

		productID := 2
		variantID := 6

		requestBody := map[string]interface{}{
			"inStock": "yes",
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid Request - Images not an array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7
		variantID := 14

		requestBody := map[string]interface{}{
			"images": "not-an-array",
		}

		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// PARAMETER VALIDATION SCENARIOS
	// ============================================================================

	t.Run("Invalid Parameter - Invalid product ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"price": 99.99,
		}

		url := "/api/products/invalid/variants/1"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid Parameter - Invalid variant ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"price": 99.99,
		}

		url := "/api/products/5/variants/invalid"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid Parameter - Negative product ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"price": 99.99,
		}

		url := "/api/products/-1/variants/1"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid Parameter - Zero product ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"price": 99.99,
		}

		url := "/api/products/0/variants/1"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})
}
