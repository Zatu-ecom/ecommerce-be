package product_option

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestCreateProductOption(t *testing.T) {
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

	t.Run("Seller creates option for own product without values", func(t *testing.T) {
		// Login as seller (Jane Merchant - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Product 5 is owned by seller_id 3 (Jane) - Classic Cotton T-Shirt
		productID := 5

		requestBody := map[string]interface{}{
			"name":        "material",
			"displayName": "Material",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)

		option := helpers.GetResponseData(t, response, "option")

		// Assert response fields
		assert.NotNil(t, option["id"])
		assert.Equal(t, float64(productID), option["productId"])
		assert.Equal(t, "material", option["name"])
		assert.Equal(t, "Material", option["displayName"])
		assert.Equal(t, float64(1), option["position"])
		assert.NotNil(t, option["createdAt"])
		assert.NotNil(t, option["updatedAt"])

		// Values should be empty or nil when not provided
		values, hasValues := option["values"]
		if hasValues && values != nil {
			valuesArray, isArray := values.([]interface{})
			if isArray {
				assert.Empty(t, valuesArray, "Values array should be empty when no values provided")
			}
		}
	})

	t.Run("Seller creates option with 2 initial values", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 owned by Jane (seller_id 3) - Summer Dress
		productID := 6

		requestBody := map[string]interface{}{
			"name":        "size",
			"displayName": "Size",
			"position":    1,
			"values": []map[string]interface{}{
				{
					"value":       "small",
					"displayName": "Small",
					"position":    1,
				},
				{
					"value":       "medium",
					"displayName": "Medium",
					"position":    2,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)

		option := helpers.GetResponseData(t, response, "option")

		// Assert option fields
		assert.NotNil(t, option["id"])
		assert.Equal(t, float64(productID), option["productId"])
		assert.Equal(t, "size", option["name"])
		assert.Equal(t, "Size", option["displayName"])
		assert.Equal(t, float64(1), option["position"])

		// Assert values were created
		values, ok := option["values"].([]interface{})
		assert.True(t, ok, "Values should be an array")
		assert.Len(t, values, 2, "Should have 2 values")

		// Check first value
		value1 := values[0].(map[string]interface{})
		assert.NotNil(t, value1["id"])
		assert.NotNil(t, value1["optionId"])
		assert.Equal(t, "small", value1["value"])
		assert.Equal(t, "Small", value1["displayName"])
		assert.Equal(t, float64(1), value1["position"])
		assert.NotNil(t, value1["createdAt"])
		assert.NotNil(t, value1["updatedAt"])

		// Check second value
		value2 := values[1].(map[string]interface{})
		assert.Equal(t, "medium", value2["value"])
		assert.Equal(t, "Medium", value2["displayName"])
		assert.Equal(t, float64(2), value2["position"])
	})

	t.Run("Seller creates option with 3 initial values", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 owned by Jane (seller_id 3) - Running Shoes
		productID := 7

		requestBody := map[string]interface{}{
			"name":        "fit",
			"displayName": "Fit Type",
			"position":    2,
			"values": []map[string]interface{}{
				{
					"value":       "slim",
					"displayName": "Slim Fit",
					"position":    1,
				},
				{
					"value":       "regular",
					"displayName": "Regular Fit",
					"position":    2,
				},
				{
					"value":       "relaxed",
					"displayName": "Relaxed Fit",
					"position":    3,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)

		option := helpers.GetResponseData(t, response, "option")

		// Assert values were created
		values, ok := option["values"].([]interface{})
		assert.True(t, ok, "Values should be an array")
		assert.Len(t, values, 3, "Should have 3 values")
	})

	t.Run("Create color option with valid hex color codes", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 owned by Jane (seller_id 3)
		productID := 5

		requestBody := map[string]interface{}{
			"name":        "color",
			"displayName": "Color",
			"position":    1,
			"values": []map[string]interface{}{
				{
					"value":       "red",
					"displayName": "Red",
					"colorCode":   "#FF0000",
					"position":    1,
				},
				{
					"value":       "blue",
					"displayName": "Blue",
					"colorCode":   "#0000FF",
					"position":    2,
				},
				{
					"value":       "green",
					"displayName": "Green",
					"colorCode":   "#00FF00",
					"position":    3,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusCreated,
		)

		option := helpers.GetResponseData(t, response, "option")

		// Assert color codes are present
		values, ok := option["values"].([]interface{})
		assert.True(t, ok, "Values should be an array")
		assert.Len(t, values, 3, "Should have 3 color values")

		// Verify color codes
		value1 := values[0].(map[string]interface{})
		assert.Equal(t, "#FF0000", value1["colorCode"])

		value2 := values[1].(map[string]interface{})
		assert.Equal(t, "#0000FF", value2["colorCode"])

		value3 := values[2].(map[string]interface{})
		assert.Equal(t, "#00FF00", value3["colorCode"])
	})

	t.Run("Create multiple options with different positions", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 owned by Jane (seller_id 3)
		productID := 6

		// Create first option with position 3
		requestBody1 := map[string]interface{}{
			"name":        "sleeve",
			"displayName": "Sleeve Length",
			"position":    3,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w1 := client.Post(t, url, requestBody1)

		response1 := helpers.AssertSuccessResponse(
			t,
			w1,
			http.StatusCreated,
		)

		option1 := helpers.GetResponseData(t, response1, "option")
		assert.Equal(t, float64(3), option1["position"])

		// Create second option with position 1
		requestBody2 := map[string]interface{}{
			"name":        "neckline",
			"displayName": "Neckline",
			"position":    1,
		}

		w2 := client.Post(t, url, requestBody2)

		response2 := helpers.AssertSuccessResponse(
			t,
			w2,
			http.StatusCreated,
		)

		option2 := helpers.GetResponseData(t, response2, "option")
		assert.Equal(t, float64(1), option2["position"])

		// Create third option with position 2
		requestBody3 := map[string]interface{}{
			"name":        "length",
			"displayName": "Dress Length",
			"position":    2,
		}

		w3 := client.Post(t, url, requestBody3)

		response3 := helpers.AssertSuccessResponse(
			t,
			w3,
			http.StatusCreated,
		)

		option3 := helpers.GetResponseData(t, response3, "option")
		assert.Equal(t, float64(2), option3["position"])
	})

	// ============================================================================
	// VALIDATION ERROR SCENARIOS
	// ============================================================================

	t.Run("Create option with missing name field", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"displayName": "Color",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Create option with missing displayName field", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"name":     "color",
			"position": 1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Create option with name too short (< 2 chars)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "c", // Only 1 character
			"displayName": "Color",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Create option with name too long (> 50 chars)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		// Create a name longer than 50 characters
		longName := strings.Repeat("a", 51)

		requestBody := map[string]interface{}{
			"name":        longName,
			"displayName": "Very Long Option Name",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Create option with displayName too short (< 3 chars)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "color",
			"displayName": "Co", // Only 2 characters
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Create option with displayName too long (> 100 chars)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		// Create a displayName longer than 100 characters
		longDisplayName := strings.Repeat("a", 101)

		requestBody := map[string]interface{}{
			"name":        "color",
			"displayName": longDisplayName,
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Create option with invalid colorCode format (not 7 chars)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "shade",
			"displayName": "Shade",
			"position":    1,
			"values": []map[string]interface{}{
				{
					"value":       "red",
					"displayName": "Red",
					"colorCode":   "#FF00", // Only 5 characters, should be 7
					"position":    1,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Create option with missing required value fields", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "pattern",
			"displayName": "Pattern",
			"position":    1,
			"values": []map[string]interface{}{
				{
					// Missing "value" field
					"displayName": "Striped",
					"position":    1,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Create option with value missing displayName field", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "style",
			"displayName": "Style",
			"position":    1,
			"values": []map[string]interface{}{
				{
					"value": "casual",
					// Missing "displayName" field
					"position": 1,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// AUTHORIZATION ERROR SCENARIOS
	// ============================================================================

	t.Run("Create option without authentication token", func(t *testing.T) {
		// Clear any existing token
		client.SetToken("")

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "finish",
			"displayName": "Finish Type",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Create option with invalid authentication token", func(t *testing.T) {
		// Set an invalid token
		client.SetToken("invalid.token.here")

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "texture",
			"displayName": "Texture",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Seller creates option for another seller's product", func(t *testing.T) {
		// Login as seller (Jane Merchant - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Product 1 is owned by seller_id 1 (different seller)
		productID := 1

		requestBody := map[string]interface{}{
			"name":        "weight",
			"displayName": "Weight",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		// Should return 403 Forbidden (seller doesn't own this product)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Admin tries to create option (only sellers allowed)", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "quality",
			"displayName": "Quality Grade",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		// Should return 403 Forbidden (only sellers can create options)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Customer tries to create option (only sellers allowed)", func(t *testing.T) {
		// Login as customer
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		productID := 5

		requestBody := map[string]interface{}{
			"name":        "rating",
			"displayName": "Rating",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		// Should return 403 Forbidden (only sellers can create options)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	// ============================================================================
	// BUSINESS LOGIC ERROR SCENARIOS
	// ============================================================================

	t.Run("Create option for non-existent product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use a non-existent product ID
		productID := 99999

		requestBody := map[string]interface{}{
			"name":        "type",
			"displayName": "Type",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Create duplicate option name for same product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 owned by Jane (seller_id 3)
		productID := 7

		requestBody := map[string]interface{}{
			"name":        "brand",
			"displayName": "Brand",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)

		// Create first option - should succeed
		w1 := client.Post(t, url, requestBody)
		helpers.AssertSuccessResponse(
			t,
			w1,
			http.StatusCreated,
		)

		// Try to create duplicate option with same name - should fail
		requestBody2 := map[string]interface{}{
			"name":        "brand", // Same name
			"displayName": "Brand Name",
			"position":    2,
		}

		w2 := client.Post(t, url, requestBody2)

		// Should return 409 Conflict
		helpers.AssertErrorResponse(t, w2, http.StatusConflict)
	})

	t.Run("Create option with invalid product ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"name":        "dimension",
			"displayName": "Dimension",
			"position":    1,
		}

		// Use invalid product ID format
		url := "/api/products/invalid/options"
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}
