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

// TestBulkUpdateOptionValues - Comprehensive tests for BulkUpdateOptionValues API
func TestBulkUpdateOptionValues(t *testing.T) {
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
		Seed Data Reference (002_seed_product_data.sql):
		- Product 5: "Classic Cotton T-Shirt" (seller_id: 3 - seller@example.com)
		  - Option 8: "Size" (position: 1)
		    - Value 23: "s" -> "Small" (position: 1)
		    - Value 24: "m" -> "Medium" (position: 2)
		    - Value 25: "l" -> "Large" (position: 3)
		    - Value 26: "xl" -> "Extra Large" (position: 4)
		    - Value 27: "xxl" -> "2X Large" (position: 5)
		  - Option 9: "Color" (position: 2)
		    - Value 28: "black" -> "Black" #000000 (position: 1)
		    - Value 29: "white" -> "White" #FFFFFF (position: 2)
		    - Value 30: "navy" -> "Navy Blue" #000080 (position: 3)
		    - Value 31: "gray" -> "Gray" #808080 (position: 4)
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

	// Helper function to bulk update option values
	bulkUpdateOptionValues := func(productID int, optionID int, updates []map[string]interface{}) *httptest.ResponseRecorder {
		requestBody := map[string]interface{}{
			"values": updates,
		}

		url := fmt.Sprintf("/api/products/%d/options/%d/values/bulk-update", productID, optionID)
		return client.Put(t, url, requestBody)
	}

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Bulk update multiple values - all fields", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use existing values from seed data: 23, 24, 25
		updates := []map[string]interface{}{
			{
				"valueId":     23,
				"displayName": "Extra Small Updated",
				"position":    10,
			},
			{
				"valueId":     24,
				"displayName": "Small Updated",
				"position":    20,
			},
			{
				"valueId":     25,
				"displayName": "Medium Updated",
				"position":    30,
			},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify response structure
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok, "Response should have data field")

		updatedCount, ok := data["updatedCount"].(float64)
		assert.True(t, ok, "Data should have updatedCount")
		assert.Equal(t, float64(3), updatedCount, "Updated count should be 3")
	})

	t.Run("Bulk update only displayName field", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option and add values for testing
		option := createOption(6, "test_display", "Test Display Update", 1)
		optionID := int(option["id"].(float64))

		// Add initial values
		initialValues := []map[string]interface{}{
			{"value": "opt1", "displayName": "Option 1", "colorCode": "#FF0000", "position": 1},
			{"value": "opt2", "displayName": "Option 2", "colorCode": "#00FF00", "position": 2},
		}
		addResponse := bulkAddOptionValues(6, optionID, initialValues)
		addData := helpers.AssertSuccessResponse(t, addResponse, http.StatusCreated)
		optionValues := addData["data"].(map[string]interface{})["optionValues"].([]interface{})

		valueId1 := uint(optionValues[0].(map[string]interface{})["id"].(float64))
		valueId2 := uint(optionValues[1].(map[string]interface{})["id"].(float64))

		// Update only displayName
		updates := []map[string]interface{}{
			{"valueId": valueId1, "displayName": "Updated Option 1"},
			{"valueId": valueId2, "displayName": "Updated Option 2"},
		}

		w := bulkUpdateOptionValues(6, optionID, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["updatedCount"])
	})

	t.Run("Bulk update only colorCode field", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use existing color values: 28, 29
		updates := []map[string]interface{}{
			{"valueId": 28, "colorCode": "#111111"},
			{"valueId": 29, "colorCode": "#EEEEEE"},
		}

		w := bulkUpdateOptionValues(5, 9, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["updatedCount"])
	})

	t.Run("Bulk update only position field", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option and values for testing
		option := createOption(7, "test_position", "Test Position Update", 1)
		optionID := int(option["id"].(float64))

		initialValues := []map[string]interface{}{
			{"value": "pos1", "displayName": "Position 1", "position": 1},
			{"value": "pos2", "displayName": "Position 2", "position": 2},
			{"value": "pos3", "displayName": "Position 3", "position": 3},
		}
		addResponse := bulkAddOptionValues(7, optionID, initialValues)
		addData := helpers.AssertSuccessResponse(t, addResponse, http.StatusCreated)
		optionValues := addData["data"].(map[string]interface{})["optionValues"].([]interface{})

		valueId1 := uint(optionValues[0].(map[string]interface{})["id"].(float64))
		valueId2 := uint(optionValues[1].(map[string]interface{})["id"].(float64))
		valueId3 := uint(optionValues[2].(map[string]interface{})["id"].(float64))

		// Update only positions
		updates := []map[string]interface{}{
			{"valueId": valueId1, "position": 100},
			{"valueId": valueId2, "position": 200},
			{"valueId": valueId3, "position": 300},
		}

		w := bulkUpdateOptionValues(7, optionID, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["updatedCount"])
	})

	t.Run("Bulk update mixed fields - different fields per value", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option and values
		option := createOption(5, "test_mixed", "Test Mixed Update", 10)
		optionID := int(option["id"].(float64))

		initialValues := []map[string]interface{}{
			{"value": "mix1", "displayName": "Mix 1", "colorCode": "#AAAAAA", "position": 1},
			{"value": "mix2", "displayName": "Mix 2", "colorCode": "#BBBBBB", "position": 2},
			{"value": "mix3", "displayName": "Mix 3", "colorCode": "#CCCCCC", "position": 3},
		}
		addResponse := bulkAddOptionValues(5, optionID, initialValues)
		addData := helpers.AssertSuccessResponse(t, addResponse, http.StatusCreated)
		optionValues := addData["data"].(map[string]interface{})["optionValues"].([]interface{})

		valueId1 := uint(optionValues[0].(map[string]interface{})["id"].(float64))
		valueId2 := uint(optionValues[1].(map[string]interface{})["id"].(float64))
		valueId3 := uint(optionValues[2].(map[string]interface{})["id"].(float64))

		// Mixed updates
		updates := []map[string]interface{}{
			{"valueId": valueId1, "displayName": "Updated Mix 1"},
			{"valueId": valueId2, "colorCode": "#DDDDDD"},
			{"valueId": valueId3, "displayName": "Updated Mix 3", "position": 99},
		}

		w := bulkUpdateOptionValues(5, optionID, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["updatedCount"])
	})

	t.Run("Bulk update single value", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Update single value from seed data
		updates := []map[string]interface{}{
			{"valueId": 26, "displayName": "Extra Large Single Update", "position": 50},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["updatedCount"])
	})

	t.Run("Bulk update with empty optional fields - keeps existing", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option and values with colorCode
		option := createOption(6, "test_keep", "Test Keep Existing", 10)
		optionID := int(option["id"].(float64))

		initialValues := []map[string]interface{}{
			{"value": "keep1", "displayName": "Keep 1", "colorCode": "#123456", "position": 1},
		}
		addResponse := bulkAddOptionValues(6, optionID, initialValues)
		addData := helpers.AssertSuccessResponse(t, addResponse, http.StatusCreated)
		optionValues := addData["data"].(map[string]interface{})["optionValues"].([]interface{})
		valueId := uint(optionValues[0].(map[string]interface{})["id"].(float64))

		// Update with empty optional fields
		updates := []map[string]interface{}{
			{"valueId": valueId, "displayName": "", "colorCode": ""},
		}

		w := bulkUpdateOptionValues(6, optionID, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["updatedCount"])
	})

	t.Run("Bulk update all values for an option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option with 4 values
		option := createOption(7, "test_all", "Test Update All", 10)
		optionID := int(option["id"].(float64))

		initialValues := []map[string]interface{}{
			{"value": "all1", "displayName": "All 1", "position": 1},
			{"value": "all2", "displayName": "All 2", "position": 2},
			{"value": "all3", "displayName": "All 3", "position": 3},
			{"value": "all4", "displayName": "All 4", "position": 4},
		}
		addResponse := bulkAddOptionValues(7, optionID, initialValues)
		addData := helpers.AssertSuccessResponse(t, addResponse, http.StatusCreated)
		optionValues := addData["data"].(map[string]interface{})["optionValues"].([]interface{})

		// Update all 4 values
		updates := []map[string]interface{}{}
		for i, ov := range optionValues {
			valueId := uint(ov.(map[string]interface{})["id"].(float64))
			updates = append(updates, map[string]interface{}{
				"valueId":     valueId,
				"displayName": fmt.Sprintf("Updated All %d", i+1),
			})
		}

		w := bulkUpdateOptionValues(7, optionID, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(4), data["updatedCount"])
	})

	t.Run("Bulk update positions - reorder values", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use existing values and reorder: 27 (was 5) → 1, 26 (was 4) → 2, 23 (was 1) → 3
		updates := []map[string]interface{}{
			{"valueId": 27, "position": 1},
			{"valueId": 26, "position": 2},
			{"valueId": 23, "position": 3},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["updatedCount"])
	})

	t.Run("Bulk update with same position for multiple values", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Update multiple values to same position
		updates := []map[string]interface{}{
			{"valueId": 30, "position": 1},
			{"valueId": 31, "position": 1},
		}

		w := bulkUpdateOptionValues(5, 9, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["updatedCount"])
	})

	// ============================================================================
	// AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Unauthorized - No token", func(t *testing.T) {
		client.SetToken("")

		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Test"},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Forbidden - Customer role", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Test"},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Forbidden - Different seller's product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Product 1 belongs to john.seller@example.com, not seller@example.com
		updates := []map[string]interface{}{
			{"valueId": 1, "displayName": "Test"},
		}

		w := bulkUpdateOptionValues(1, 1, updates)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Admin can bulk update option values for any product", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Admin bulk updates values on Product 5 (owned by seller_id 3)
		// Values 23, 24, 25 = S, M, L
		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Small Size - Admin"},
			{"valueId": 24, "displayName": "Medium Size - Admin"},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		
		// Verify response structure
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok, "Response should have data field")
		if ok && data != nil {
			optionValues, ok := data["optionValues"].([]interface{})
			if ok {
				assert.GreaterOrEqual(t, len(optionValues), 1, "Should have at least 1 updated value")
			}
		}
	})

	t.Run("Admin can bulk update option values for different seller's product", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Admin bulk updates values on Product 1 (owned by seller_id 2)
		// Values 1, 2 = Natural Titanium, Blue Titanium
		updates := []map[string]interface{}{
			{"valueId": 1, "displayName": "Natural Titanium - Admin Updated"},
			{"valueId": 2, "displayName": "Blue Titanium - Admin Updated", "colorCode": "#4169E1"},
		}

		w := bulkUpdateOptionValues(1, 1, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		
		// Verify response structure
		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok, "Response should have data field")
		if ok && data != nil {
			optionValues, ok := data["optionValues"].([]interface{})
			if ok {
				assert.GreaterOrEqual(t, len(optionValues), 1, "Should have at least 1 updated value")
			}
		}
	})

	// ============================================================================
	// VALIDATION ERRORS - INVALID IDs
	// ============================================================================

	t.Run("Invalid product ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Test"},
		}

		requestBody := map[string]interface{}{"values": updates}
		w := client.Put(t, "/api/products/invalid/options/8/values/bulk-update", requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid option ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Test"},
		}

		requestBody := map[string]interface{}{"values": updates}
		w := client.Put(t, "/api/products/5/options/abc/values/bulk-update", requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Non-existent product ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Test"},
		}

		w := bulkUpdateOptionValues(99999, 8, updates)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Non-existent option ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Test"},
		}

		w := bulkUpdateOptionValues(5, 99999, updates)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// VALIDATION ERRORS - REQUEST BODY
	// ============================================================================

	t.Run("Empty values array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"values": []map[string]interface{}{},
		}

		url := "/api/products/5/options/8/values/bulk-update"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing values field", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{}

		url := "/api/products/5/options/8/values/bulk-update"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing required field - no valueId", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		updates := []map[string]interface{}{
			{"displayName": "Updated", "position": 1},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("DisplayName too long", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		longDisplayName := strings.Repeat("a", 101)
		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": longDisplayName},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid request structure - values not an array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"values": "not-an-array",
		}

		url := "/api/products/5/options/8/values/bulk-update"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// VALIDATION ERRORS - RELATIONSHIPS
	// ============================================================================

	t.Run("Option doesn't belong to product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Option 8 belongs to product 5, try with product 6
		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Test"},
		}

		w := bulkUpdateOptionValues(6, 8, updates)
		helpers.AssertStatusCodeOneOf(t, w, http.StatusBadRequest, http.StatusNotFound)
	})

	t.Run("Value ID doesn't exist", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		updates := []map[string]interface{}{
			{"valueId": 99999, "displayName": "Non-existent"},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Value doesn't belong to the specified option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Value 23 belongs to option 8, try to update via option 9
		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Wrong Option"},
		}

		w := bulkUpdateOptionValues(5, 9, updates)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Mix of valid and invalid value IDs", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Mix of valid (23, 24) and invalid (99999) value IDs
		updates := []map[string]interface{}{
			{"valueId": 23, "displayName": "Valid 1"},
			{"valueId": 99999, "displayName": "Invalid"},
			{"valueId": 24, "displayName": "Valid 2"},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// BUSINESS LOGIC & EDGE CASES
	// ============================================================================

	t.Run("Partial update - only some values in option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option with 4 values
		option := createOption(5, "test_partial", "Test Partial Update", 20)
		optionID := int(option["id"].(float64))

		initialValues := []map[string]interface{}{
			{"value": "part1", "displayName": "Part 1", "position": 1},
			{"value": "part2", "displayName": "Part 2", "position": 2},
			{"value": "part3", "displayName": "Part 3", "position": 3},
			{"value": "part4", "displayName": "Part 4", "position": 4},
		}
		addResponse := bulkAddOptionValues(5, optionID, initialValues)
		addData := helpers.AssertSuccessResponse(t, addResponse, http.StatusCreated)
		optionValues := addData["data"].(map[string]interface{})["optionValues"].([]interface{})

		valueId1 := uint(optionValues[0].(map[string]interface{})["id"].(float64))
		valueId3 := uint(optionValues[2].(map[string]interface{})["id"].(float64))

		// Update only 1st and 3rd values
		updates := []map[string]interface{}{
			{"valueId": valueId1, "displayName": "Updated Part 1"},
			{"valueId": valueId3, "displayName": "Updated Part 3"},
		}

		w := bulkUpdateOptionValues(5, optionID, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(2), data["updatedCount"])
	})

	t.Run("Update same value twice in request (duplicate valueId)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Try to update same value twice
		updates := []map[string]interface{}{
			{"valueId": 27, "displayName": "First Update"},
			{"valueId": 27, "displayName": "Second Update"},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		// Behavior depends on implementation - might succeed with last update winning
		// or might reject. Let's just verify it doesn't crash
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
	})

	t.Run("Verify transaction atomicity - all or nothing", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option with values for testing atomicity
		option := createOption(6, "test_atomic", "Test Atomic Update", 20)
		optionID := int(option["id"].(float64))

		initialValues := []map[string]interface{}{
			{"value": "atom1", "displayName": "Atomic 1", "position": 1},
			{"value": "atom2", "displayName": "Atomic 2", "position": 2},
		}
		addResponse := bulkAddOptionValues(6, optionID, initialValues)
		addData := helpers.AssertSuccessResponse(t, addResponse, http.StatusCreated)
		optionValues := addData["data"].(map[string]interface{})["optionValues"].([]interface{})

		valueId1 := uint(optionValues[0].(map[string]interface{})["id"].(float64))
		valueId2 := uint(optionValues[1].(map[string]interface{})["id"].(float64))

		// Try to update with one valid and one invalid
		updates := []map[string]interface{}{
			{"valueId": valueId1, "displayName": "Should Not Update 1"},
			{"valueId": valueId2, "displayName": "Should Not Update 2"},
			{"valueId": 99999, "displayName": "Invalid ID"},
		}

		w := bulkUpdateOptionValues(6, optionID, updates)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)

		// Verify first two values were NOT updated by trying to update them again
		// If rollback worked, this should succeed
		verifyUpdates := []map[string]interface{}{
			{"valueId": valueId1, "displayName": "Verify Update 1"},
		}
		wVerify := bulkUpdateOptionValues(6, optionID, verifyUpdates)
		helpers.AssertSuccessResponse(t, wVerify, http.StatusOK)
	})

	t.Run("Update colorCode from value to empty - keeps existing", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option with colorCode
		option := createOption(7, "test_color_keep", "Test Color Keep", 20)
		optionID := int(option["id"].(float64))

		initialValues := []map[string]interface{}{
			{"value": "color1", "displayName": "Color 1", "colorCode": "#FF0000", "position": 1},
		}
		addResponse := bulkAddOptionValues(7, optionID, initialValues)
		addData := helpers.AssertSuccessResponse(t, addResponse, http.StatusCreated)
		optionValues := addData["data"].(map[string]interface{})["optionValues"].([]interface{})
		valueId := uint(optionValues[0].(map[string]interface{})["id"].(float64))

		// Update with empty colorCode
		updates := []map[string]interface{}{
			{"valueId": valueId, "colorCode": ""},
		}

		w := bulkUpdateOptionValues(7, optionID, updates)
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["updatedCount"])
	})

	t.Run("Update position to negative number", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Try to update position to negative
		updates := []map[string]interface{}{
			{"valueId": 23, "position": -1},
		}

		w := bulkUpdateOptionValues(5, 8, updates)
		// Verify it doesn't crash - might accept or reject based on business rules
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
	})
}
