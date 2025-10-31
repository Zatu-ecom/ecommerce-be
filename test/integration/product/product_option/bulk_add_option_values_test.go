package product_option

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestBulkAddOptionValues - Comprehensive tests for BulkAddOptionValues API
func TestBulkAddOptionValues(t *testing.T) {
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

	/*
		Seed Data Reference:
		- Product 5: "Classic Cotton T-Shirt" (seller_id: 3 - seller@example.com)
		  - Has existing options (Size, Color)
		- Product 6: "Summer Dress" (seller_id: 3 - seller@example.com)
		- Product 7: "Running Shoes" (seller_id: 3 - seller@example.com)
		- Product 1: "iPhone 15 Pro" (seller_id: 2 - john.seller@example.com)
	*/

	// Helper function to create an option
	createOption := func(productID int, name string, displayName string, position int) map[string]interface{} {
		requestBody := map[string]interface{}{
			"name":        name,
			"displayName": displayName,
			"position":    position,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		return helpers.GetResponseData(t, response, "option")
	}

	// Helper function to bulk add option values
	bulkAddOptionValues := func(productID int, optionID int, values []map[string]interface{}) *httptest.ResponseRecorder {
		requestBody := map[string]interface{}{
			"values": values,
		}

		url := fmt.Sprintf("/api/products/%d/options/%d/values/bulk", productID, optionID)
		return client.Post(t, url, requestBody)
	}

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Bulk add multiple values with all fields", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 5
		option := createOption(5, "material", "Material", 3)
		optionID := int(option["id"].(float64))

		// Prepare bulk values
		values := []map[string]interface{}{
			{
				"value":       "cotton",
				"displayName": "Cotton",
				"colorCode":   "#FFFFFF",
				"position":    1,
			},
			{
				"value":       "polyester",
				"displayName": "Polyester",
				"colorCode":   "#E0E0E0",
				"position":    2,
			},
			{
				"value":       "silk",
				"displayName": "Silk",
				"colorCode":   "#F5F5DC",
				"position":    3,
			},
		}

		w := bulkAddOptionValues(5, optionID, values)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Verify response structure
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok, "Response should have data field")

		optionValues, ok := data["optionValues"].([]interface{})
		assert.True(t, ok, "Data should have optionValues array")
		assert.Equal(t, 3, len(optionValues), "Should have 3 values")

		addedCount, ok := data["addedCount"].(float64)
		assert.True(t, ok, "Data should have addedCount")
		assert.Equal(t, float64(3), addedCount, "Added count should be 3")

		// Verify first value
		firstValue := optionValues[0].(map[string]interface{})
		assert.Equal(t, "cotton", firstValue["value"])
		assert.Equal(t, "Cotton", firstValue["displayName"])
		assert.Equal(t, "#FFFFFF", firstValue["colorCode"])
	})

	t.Run("Bulk add values without optional fields (colorCode)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 6
		option := createOption(6, "size", "Size", 1)
		optionID := int(option["id"].(float64))

		// Prepare bulk values without colorCode
		values := []map[string]interface{}{
			{"value": "xs", "displayName": "Extra Small", "position": 1},
			{"value": "s", "displayName": "Small", "position": 2},
			{"value": "m", "displayName": "Medium", "position": 3},
			{"value": "l", "displayName": "Large", "position": 4},
		}

		w := bulkAddOptionValues(6, optionID, values)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		data := response["data"].(map[string]interface{})
		optionValues := data["optionValues"].([]interface{})
		assert.Equal(t, 4, len(optionValues))
		assert.Equal(t, float64(4), data["addedCount"])
	})

	t.Run("Bulk add values with mixed - some with colorCode, some without", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 7
		option := createOption(7, "style", "Style", 1)
		optionID := int(option["id"].(float64))

		// Mixed values
		values := []map[string]interface{}{
			{"value": "casual", "displayName": "Casual", "position": 1},
			{"value": "sport", "displayName": "Sport", "colorCode": "#FF5733", "position": 2},
			{"value": "formal", "displayName": "Formal", "position": 3},
			{"value": "athletic", "displayName": "Athletic", "colorCode": "#00FF00", "position": 4},
		}

		w := bulkAddOptionValues(7, optionID, values)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(4), data["addedCount"])
	})

	t.Run("Bulk add with same position for multiple values", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 5
		option := createOption(5, "fit", "Fit", 4)
		optionID := int(option["id"].(float64))

		// All values with position 1
		values := []map[string]interface{}{
			{"value": "slim", "displayName": "Slim Fit", "position": 1},
			{"value": "regular", "displayName": "Regular Fit", "position": 1},
			{"value": "relaxed", "displayName": "Relaxed Fit", "position": 1},
		}

		w := bulkAddOptionValues(5, optionID, values)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["addedCount"])
	})

	t.Run("Bulk add single value (edge case)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 6
		option := createOption(6, "length", "Length", 2)
		optionID := int(option["id"].(float64))

		// Single value in bulk
		values := []map[string]interface{}{
			{"value": "short", "displayName": "Short", "position": 1},
		}

		w := bulkAddOptionValues(6, optionID, values)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["addedCount"])
	})

	t.Run("Bulk add many values (10-15 values)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 7
		option := createOption(7, "shoe_size", "Shoe Size", 2)
		optionID := int(option["id"].(float64))

		// Create 12 values
		values := []map[string]interface{}{}
		sizes := []string{
			"6",
			"6.5",
			"7",
			"7.5",
			"8",
			"8.5",
			"9",
			"9.5",
			"10",
			"10.5",
			"11",
			"11.5",
		}
		for i, size := range sizes {
			values = append(values, map[string]interface{}{
				"value":       size,
				"displayName": fmt.Sprintf("Size %s", size),
				"position":    i + 1,
			})
		}

		w := bulkAddOptionValues(7, optionID, values)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		data := response["data"].(map[string]interface{})
		optionValues := data["optionValues"].([]interface{})
		assert.Equal(t, 12, len(optionValues))
		assert.Equal(t, float64(12), data["addedCount"])
	})

	t.Run("Bulk add values with special characters", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 5
		option := createOption(5, "special_size", "Special Size", 5)
		optionID := int(option["id"].(float64))

		// Values with special characters
		values := []map[string]interface{}{
			{"value": "m/l", "displayName": "Medium/Large", "position": 1},
			{"value": "36½", "displayName": "36 and Half", "position": 2},
			{"value": "bleu", "displayName": "Bleu Français", "position": 3},
		}

		w := bulkAddOptionValues(5, optionID, values)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["addedCount"])
	})

	// ============================================================================
	// AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Unauthorized - No token", func(t *testing.T) {
		client.SetToken("")

		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		w := bulkAddOptionValues(5, 8, values)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Forbidden - Customer role", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		w := bulkAddOptionValues(5, 8, values)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Forbidden - Different seller's product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Product 1 belongs to john.seller@example.com, not seller@example.com
		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		w := bulkAddOptionValues(1, 1, values)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	// ============================================================================
	// VALIDATION ERRORS - INVALID IDs
	// ============================================================================

	t.Run("Invalid product ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		requestBody := map[string]interface{}{"values": values}
		w := client.Post(t, "/api/products/invalid/options/8/values/bulk", requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid option ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		requestBody := map[string]interface{}{"values": values}
		w := client.Post(t, "/api/products/5/options/abc/values/bulk", requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Non-existent product ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		w := bulkAddOptionValues(99999, 8, values)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Non-existent option ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		w := bulkAddOptionValues(5, 99999, values)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// VALIDATION ERRORS - REQUEST BODY
	// ============================================================================

	t.Run("Empty values array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option first
		option := createOption(5, "test_empty", "Test Empty", 10)
		optionID := int(option["id"].(float64))

		requestBody := map[string]interface{}{
			"values": []map[string]interface{}{},
		}

		url := fmt.Sprintf("/api/products/5/options/%d/values/bulk", optionID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing values field", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option first
		option := createOption(5, "test_missing", "Test Missing", 11)
		optionID := int(option["id"].(float64))

		requestBody := map[string]interface{}{}

		url := fmt.Sprintf("/api/products/5/options/%d/values/bulk", optionID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid request structure - values not an array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option first
		option := createOption(5, "test_not_array", "Test Not Array", 12)
		optionID := int(option["id"].(float64))

		requestBody := map[string]interface{}{
			"values": "not-an-array",
		}

		url := fmt.Sprintf("/api/products/5/options/%d/values/bulk", optionID)
		w := client.Post(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing required field in one value - no 'value'", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option first
		option := createOption(6, "test_no_value", "Test No Value", 10)
		optionID := int(option["id"].(float64))

		values := []map[string]interface{}{
			{"displayName": "Red", "position": 1},
		}

		w := bulkAddOptionValues(6, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing required field - no 'displayName'", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option first
		option := createOption(6, "test_no_display", "Test No Display", 11)
		optionID := int(option["id"].(float64))

		values := []map[string]interface{}{
			{"value": "red", "position": 1},
		}

		w := bulkAddOptionValues(6, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Value too short in batch", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option first
		option := createOption(7, "test_short", "Test Short", 10)
		optionID := int(option["id"].(float64))

		values := []map[string]interface{}{
			{"value": "", "displayName": "Empty Value", "position": 1},
		}

		w := bulkAddOptionValues(7, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Value too long in batch", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option first
		option := createOption(7, "test_long", "Test Long", 11)
		optionID := int(option["id"].(float64))

		longValue := strings.Repeat("a", 101)
		values := []map[string]interface{}{
			{"value": longValue, "displayName": "Long Value", "position": 1},
		}

		w := bulkAddOptionValues(7, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid colorCode format in batch", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option first
		option := createOption(5, "test_invalid_color", "Test Invalid Color", 13)
		optionID := int(option["id"].(float64))

		values := []map[string]interface{}{
			{"value": "red", "displayName": "Red", "colorCode": "#FFF", "position": 1},
		}

		w := bulkAddOptionValues(5, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// VALIDATION ERRORS - RELATIONSHIPS
	// ============================================================================

	t.Run("Option doesn't belong to product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 5
		option := createOption(5, "test_mismatch", "Test Mismatch", 14)
		optionID := int(option["id"].(float64))

		// Try to add values using product 6
		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		w := bulkAddOptionValues(6, optionID, values)
		helpers.AssertStatusCodeOneOf(t, w, http.StatusBadRequest, http.StatusNotFound)
	})

	t.Run("Product exists but option doesn't exist", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		values := []map[string]interface{}{
			{"value": "test", "displayName": "Test", "position": 1},
		}

		// Use a non-existent option ID for existing product
		w := bulkAddOptionValues(6, 88888, values)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// BUSINESS LOGIC - DUPLICATE DETECTION
	// ============================================================================

	t.Run("Duplicate value - conflicts with existing values in DB", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option and add initial values
		option := createOption(5, "dup_test1", "Duplicate Test 1", 20)
		optionID := int(option["id"].(float64))

		// Add first batch
		initialValues := []map[string]interface{}{
			{"value": "small", "displayName": "Small", "position": 1},
		}
		w := bulkAddOptionValues(5, optionID, initialValues)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Try to add batch with duplicate
		duplicateValues := []map[string]interface{}{
			{"value": "medium", "displayName": "Medium", "position": 2},
			{"value": "small", "displayName": "Small Again", "position": 3},
			{"value": "large", "displayName": "Large", "position": 4},
		}

		w = bulkAddOptionValues(5, optionID, duplicateValues)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("Duplicate value - case insensitive conflict", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option and add initial value
		option := createOption(6, "dup_test2", "Duplicate Test 2", 20)
		optionID := int(option["id"].(float64))

		// Add first value
		initialValues := []map[string]interface{}{
			{"value": "Red", "displayName": "Red", "position": 1},
		}
		w := bulkAddOptionValues(6, optionID, initialValues)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Try to add with different case
		duplicateValues := []map[string]interface{}{
			{"value": "blue", "displayName": "Blue", "position": 2},
			{"value": "RED", "displayName": "Red Again", "position": 3},
		}

		w = bulkAddOptionValues(6, optionID, duplicateValues)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("Duplicate within batch - same value twice", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option
		option := createOption(7, "dup_test3", "Duplicate Test 3", 20)
		optionID := int(option["id"].(float64))

		// Try to add batch with duplicate within
		values := []map[string]interface{}{
			{"value": "red", "displayName": "Red", "position": 1},
			{"value": "blue", "displayName": "Blue", "position": 2},
			{"value": "red", "displayName": "Red Again", "position": 3},
		}

		w := bulkAddOptionValues(7, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("Duplicate within batch - case insensitive", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option
		option := createOption(5, "dup_test4", "Duplicate Test 4", 21)
		optionID := int(option["id"].(float64))

		// Batch with case-insensitive duplicate
		values := []map[string]interface{}{
			{"value": "red", "displayName": "Red", "position": 1},
			{"value": "green", "displayName": "Green", "position": 2},
			{"value": "RED", "displayName": "Red Uppercase", "position": 3},
		}

		w := bulkAddOptionValues(5, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("Duplicate within batch - with whitespace", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option
		option := createOption(6, "dup_test5", "Duplicate Test 5", 21)
		optionID := int(option["id"].(float64))

		// Batch with whitespace duplicate
		values := []map[string]interface{}{
			{"value": "red", "displayName": "Red", "position": 1},
			{"value": "blue", "displayName": "Blue", "position": 2},
			{"value": " red ", "displayName": "Red With Spaces", "position": 3},
		}

		w := bulkAddOptionValues(6, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("Partial batch failure - first 2 values valid, 3rd is duplicate", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option and add initial value
		option := createOption(7, "dup_test6", "Duplicate Test 6", 21)
		optionID := int(option["id"].(float64))

		// Add initial value
		initialValues := []map[string]interface{}{
			{"value": "existing", "displayName": "Existing Value", "position": 1},
		}
		w := bulkAddOptionValues(7, optionID, initialValues)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Try to add batch where 3rd value conflicts
		values := []map[string]interface{}{
			{"value": "new1", "displayName": "New Value 1", "position": 2},
			{"value": "new2", "displayName": "New Value 2", "position": 3},
			{"value": "existing", "displayName": "Duplicate", "position": 4},
		}

		w = bulkAddOptionValues(7, optionID, values)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)

		// Verify transaction rollback - check that new1 and new2 were NOT created
		// by trying to add them again (should succeed if rollback worked)
		verifyValues := []map[string]interface{}{
			{"value": "new1", "displayName": "New Value 1 Verify", "position": 5},
		}
		wVerify := bulkAddOptionValues(7, optionID, verifyValues)
		// If rollback worked, new1 shouldn't exist and this should succeed
		helpers.AssertSuccessResponse(t, wVerify, http.StatusCreated)
	})
}
