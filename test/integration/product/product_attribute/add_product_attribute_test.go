package product_attribute

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestAddProductAttribute(t *testing.T) {
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

	t.Run("Seller adds valid attribute to own product", func(t *testing.T) {
		// Login as seller (Jane Merchant - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Product 5 is owned by seller_id 3 (Jane) - Classic Cotton T-Shirt
		productID := 5

		// Use color attribute (ID 1) which doesn't exist on product 5 yet
		attributeDefID := 1

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "Red",
			"sortOrder":             1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
			"Product attribute added successfully",
		)

		attribute := helpers.GetResponseData(t, response, "attribute")

		// Assert response fields
		assert.NotNil(t, attribute["id"])
		assert.Equal(t, float64(productID), attribute["productId"])
		assert.Equal(t, float64(attributeDefID), attribute["attributeDefinitionId"])
		assert.Equal(t, "Red", attribute["value"])
		assert.Equal(t, float64(1), attribute["sortOrder"])
		assert.NotNil(t, attribute["attributeKey"])
		assert.NotNil(t, attribute["attributeName"])
		assert.NotNil(t, attribute["createdAt"])
		assert.NotNil(t, attribute["updatedAt"])
	})

	t.Run("Seller adds another attribute to own product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 owned by Jane (seller_id 3)
		productID := 5

		// Use dimensions attribute (ID 11) which doesn't exist on product 5 yet
		attributeDefID := 11

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "28 x 18",
			"sortOrder":             0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
			"Product attribute added successfully",
		)

		attribute := helpers.GetResponseData(t, response, "attribute")

		// Assert response fields
		assert.NotNil(t, attribute["id"])
		assert.Equal(t, float64(productID), attribute["productId"])
		assert.Equal(t, "28 x 18", attribute["value"])
	})

	t.Run("Add attribute with sort order", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Product 6 is owned by seller_id 3 (Jane) - Summer Dress
		productID := 6
		// Use size attribute (ID 9) which doesn't exist on product 6 yet
		attributeDefID := 9

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "M",
			"sortOrder":             5,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
			"",
		)

		attribute := helpers.GetResponseData(t, response, "attribute")
		assert.Equal(t, float64(5), attribute["sortOrder"])
	})

	t.Run("Add attribute with sort order zero", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Product 7 is owned by seller_id 3 (Jane) - Running Shoes
		productID := 7
		// Use size attribute (ID 9) with valid value from allowed_values
		attributeDefID := 9

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "L",
			"sortOrder":             0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
			"",
		)

		attribute := helpers.GetResponseData(t, response, "attribute")
		assert.Equal(t, float64(0), attribute["sortOrder"])
	})

	// ============================================================================
	// FAILURE SCENARIOS - AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Add attribute without authentication", func(t *testing.T) {
		// Clear token
		client.SetToken("")

		productID := 1
		attributeDefID := 1

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "Test",
			"sortOrder":             0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized, "")
	})

	t.Run("Seller tries to add attribute to other seller's product", func(t *testing.T) {
		// Login as seller (Jane - seller_id 3)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Try to add attribute to product 1 which belongs to seller_id 2
		productID := 1
		attributeDefID := 1

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "Test",
			"sortOrder":             0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		// Should return either 403 Forbidden or 404 Not Found depending on implementation
		assert.True(
			t,
			w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d",
			w.Code,
		)
	})

	// ============================================================================
	// FAILURE SCENARIOS - VALIDATION
	// ============================================================================

	t.Run("Add attribute with invalid product ID", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 99999 // Non-existent product
		attributeDefID := 1

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "Test",
			"sortOrder":             0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound, "Product not found")
	})

	t.Run("Add attribute with invalid attribute definition ID", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 owned by Jane (seller_id 3)
		productID := 5
		attributeDefID := 99999 // Non-existent attribute definition

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "Test",
			"sortOrder":             0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound, "")
	})

	t.Run("Add duplicate attribute to same product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 owned by Jane (seller_id 3)
		productID := 5
		// Use weight_capacity attribute (ID 12) which doesn't exist on product 5 yet
		attributeDefID := 12

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "150",
			"sortOrder":             0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)

		// Add first time - should succeed
		w1 := client.Post(t, url, requestBody)
		helpers.AssertSuccessResponse(t, w1, http.StatusCreated, "")

		// Try to add same attribute again - should fail
		requestBody["value"] = "200"
		w2 := client.Post(t, url, requestBody)
		helpers.AssertErrorResponse(
			t,
			w2,
			http.StatusConflict,
			"Product already has this attribute assigned",
		)
	})

	t.Run("Add attribute with empty value", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 owned by Jane (seller_id 3)
		productID := 6
		// Use fit attribute (ID 10)
		attributeDefID := 10

		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 "",
			"sortOrder":             0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("Add attribute with missing required fields", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 1

		// Missing attributeDefinitionId
		requestBody := map[string]interface{}{
			"value":     "Test",
			"sortOrder": 0,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("Add attribute with invalid product ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"attributeDefinitionId": 1,
			"value":                 "Test",
			"sortOrder":             0,
		}

		url := "/api/products/invalid/attributes"
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid productId")
	})

	// ============================================================================
	// EDGE CASE - MULTIPLE ATTRIBUTES
	// ============================================================================

	t.Run("Add multiple different attributes to same product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 owned by Jane (seller_id 3) - Running Shoes
		productID := 7

		// Add first attribute - color (doesn't exist on product 7)
		requestBody1 := map[string]interface{}{
			"attributeDefinitionId": 1, // color
			"value":                     "Black",
			"sortOrder":                 1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w1 := client.Post(t, url, requestBody1)
		helpers.AssertSuccessResponse(t, w1, http.StatusCreated, "")

		// Add second attribute - fit (doesn't exist on product 7)
		requestBody2 := map[string]interface{}{
			"attributeDefinitionId": 10, // fit
			"value":                     "Regular",
			"sortOrder":                 2,
		}

		w2 := client.Post(t, url, requestBody2)
		helpers.AssertSuccessResponse(t, w2, http.StatusCreated, "")

		// Add third attribute - dimensions (doesn't exist on product 7)
		requestBody3 := map[string]interface{}{
			"attributeDefinitionId": 11, // dimensions
			"value":                     "28 x 18 x 10",
			"sortOrder":                 3,
		}

		w3 := client.Post(t, url, requestBody3)
		helpers.AssertSuccessResponse(t, w3, http.StatusCreated, "")

		// Verify all attributes exist by getting them
		getURL := fmt.Sprintf("/api/products/%d/attributes", productID)
		wGet := client.Get(t, getURL)
		getResponse := helpers.AssertSuccessResponse(t, wGet, http.StatusOK, "")

		productAttributes := helpers.GetResponseData(t, getResponse, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		// Should have at least the 3 we just added plus the 2 from seed data (brand, material)
		assert.GreaterOrEqual(t, len(attributes), 5, "Should have at least 5 attributes")
	})
}
