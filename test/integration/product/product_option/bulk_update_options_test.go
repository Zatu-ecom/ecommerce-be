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

func TestBulkUpdateProductOptions(t *testing.T) {
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

	// Helper function to get options for a product
	getOptions := func(productID int) []interface{} {
		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Get(t, url)
		// Check response status
		if w.Code != http.StatusOK {
			t.Fatalf("Failed to get options: %d", w.Code)
		}
		response := helpers.ParseResponse(t, w.Body)

		// Check if response is successful
		if success, ok := response["success"].(bool); !ok || !success {
			t.Fatal("GetOptions response not successful")
		}

		// Extract options data
		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Invalid data structure in GetOptions response")
		}

		optionsObj, ok := data["options"].(map[string]interface{})
		if !ok {
			t.Fatal("Invalid options structure")
		}

		optionsArray, ok := optionsObj["options"].([]interface{})
		if !ok {
			t.Fatal("Options is not an array")
		}

		return optionsArray
	}

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Bulk update multiple options - display name and position", func(t *testing.T) {
		// Login as seller (Jane - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create 3 options
		option1 := createOption(productID, "material_type", "Material Type", 1)
		option2 := createOption(productID, "collar_style", "Collar Style", 2)
		option3 := createOption(productID, "sleeve_length", "Sleeve Length", 3)

		option1ID := int(option1["id"].(float64))
		option2ID := int(option2["id"].(float64))
		option3ID := int(option3["id"].(float64))

		// Bulk update all 3 options
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    option1ID,
					"displayName": "Fabric Material",
					"position":    10,
				},
				{
					"optionId":    option2ID,
					"displayName": "Neckline Style",
					"position":    20,
				},
				{
					"optionId":    option3ID,
					"displayName": "Sleeve Type",
					"position":    30,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify response contains updatedCount
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["updatedCount"], "Should update 3 options")

		// Verify options are actually updated
		options := getOptions(productID)
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] == nil {
				continue
			}
			optID := int(optMap["id"].(float64))

			switch optID {
			case option1ID:
				assert.Equal(t, "Fabric Material", optMap["displayName"])
				assert.Equal(t, float64(10), optMap["position"])
			case option2ID:
				assert.Equal(t, "Neckline Style", optMap["displayName"])
				assert.Equal(t, float64(20), optMap["position"])
			case option3ID:
				assert.Equal(t, "Sleeve Type", optMap["displayName"])
				assert.Equal(t, float64(30), optMap["position"])
			}
		}
	})

	t.Run("Update only positions", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		// Create 2 options
		option1 := createOption(productID, "dress_length", "Dress Length", 1)
		option2 := createOption(productID, "waist_style", "Waist Style", 2)

		option1ID := int(option1["id"].(float64))
		option2ID := int(option2["id"].(float64))
		originalDisplayName1 := option1["displayName"].(string)
		originalDisplayName2 := option2["displayName"].(string)

		// Update only positions (empty displayName should keep existing)
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId": option1ID,
					"position": 100,
				},
				{
					"optionId": option2ID,
					"position": 50,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify positions changed but displayNames remained
		options := getOptions(productID)
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] == nil {
				continue
			}
			optID := int(optMap["id"].(float64))

			switch optID {
			case option1ID:
				assert.Equal(
					t,
					originalDisplayName1,
					optMap["displayName"],
					"Display name should remain unchanged",
				)
				assert.Equal(t, float64(100), optMap["position"])
			case option2ID:
				assert.Equal(
					t,
					originalDisplayName2,
					optMap["displayName"],
					"Display name should remain unchanged",
				)
				assert.Equal(t, float64(50), optMap["position"])
			}
		}
	})

	t.Run("Update only display names", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		// Create 2 options
		option1 := createOption(productID, "cushioning", "Cushioning", 1)
		option2 := createOption(productID, "terrain", "Terrain", 2)

		option1ID := int(option1["id"].(float64))
		option2ID := int(option2["id"].(float64))
		originalPosition1 := int(option1["position"].(float64))
		originalPosition2 := int(option2["position"].(float64))

		// Update only display names
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    option1ID,
					"displayName": "Cushion Type",
					"position":    0, // Should not change position if 0
				},
				{
					"optionId":    option2ID,
					"displayName": "Surface Type",
					"position":    0,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify display names changed
		options := getOptions(productID)
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] == nil {
				continue
			}
			optID := int(optMap["id"].(float64))

			switch optID {
			case option1ID:
				assert.Equal(t, "Cushion Type", optMap["displayName"])
				// Position might be 0 or original depending on implementation
				pos := int(optMap["position"].(float64))
				assert.True(
					t,
					pos == 0 || pos == originalPosition1,
					"Position should be 0 or unchanged",
				)
			case option2ID:
				assert.Equal(t, "Surface Type", optMap["displayName"])
				pos := int(optMap["position"].(float64))
				assert.True(
					t,
					pos == 0 || pos == originalPosition2,
					"Position should be 0 or unchanged",
				)
			}
		}
	})

	t.Run("Update single option", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create 1 option
		option := createOption(productID, "fabric_weight", "Fabric Weight", 1)
		optionID := int(option["id"].(float64))

		// Bulk update with single option
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "Material Weight",
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify updatedCount is 1
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["updatedCount"], "Should update 1 option")

		// The bulk update should have worked, so we're done
		// We can't easily verify via GET because it returns only options with values
	})

	t.Run("Empty display name uses existing", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		// Create option
		option := createOption(productID, "pattern_type", "Pattern Type", 1)
		optionID := int(option["id"].(float64))
		originalDisplayName := option["displayName"].(string)

		// Update with empty displayName
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "", // Empty string
					"position":    15,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify displayName unchanged, position updated
		options := getOptions(productID)
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] != nil && int(optMap["id"].(float64)) == optionID {
				assert.Equal(
					t,
					originalDisplayName,
					optMap["displayName"],
					"Display name should remain unchanged",
				)
				assert.Equal(t, float64(15), optMap["position"], "Position should be updated")
				break
			}
		}
	})

	t.Run("Update all product options", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		// Create some options first
		option1 := createOption(productID, "all_opt1", "All Option 1", 1)
		option2 := createOption(productID, "all_opt2", "All Option 2", 2)
		option3 := createOption(productID, "all_opt3", "All Option 3", 3)

		option1ID := int(option1["id"].(float64))
		option2ID := int(option2["id"].(float64))
		option3ID := int(option3["id"].(float64))

		// Create bulk update request for all options
		updates := []map[string]interface{}{
			{
				"optionId":    option1ID,
				"displayName": "Updated All Option 1",
				"position":    101,
			},
			{
				"optionId":    option2ID,
				"displayName": "Updated All Option 2",
				"position":    102,
			},
			{
				"optionId":    option3ID,
				"displayName": "Updated All Option 3",
				"position":    103,
			},
		}

		requestBody := map[string]interface{}{
			"options": updates,
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify all options were updated
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["updatedCount"], "Should update all 3 options")
	})

	// ============================================================================
	// FAILURE SCENARIOS - AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Unauthorized - No token", func(t *testing.T) {
		// First login as seller to create option
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_option", "Test Option", 1)
		optionID := int(option["id"].(float64))

		// Clear token
		client.SetToken("")

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Unauthorized - Invalid token", func(t *testing.T) {
		// First login as seller to create option
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_option2", "Test Option 2", 1)
		optionID := int(option["id"].(float64))

		// Set invalid token
		client.SetToken("invalid.token.here")

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Forbidden - Not a seller (customer)", func(t *testing.T) {
		// First login as seller to create option
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_option3", "Test Option 3", 1)
		optionID := int(option["id"].(float64))

		// Login as customer
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Forbidden - Wrong seller", func(t *testing.T) {
		// Login as seller (Jane) to create option on her product
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product - seller_id 3)
		productID := 5

		option := createOption(productID, "test_option4", "Test Option 4", 1)
		optionID := int(option["id"].(float64))

		// Try to update using product 1 which doesn't belong to Jane (belongs to seller_id 2)
		otherProductID := 1

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", otherProductID)
		w := client.Put(t, url, requestBody)

		// Should return 400, 403 Forbidden or 404 Not Found
		assert.True(
			t,
			w.Code == http.StatusBadRequest || w.Code == http.StatusForbidden ||
				w.Code == http.StatusNotFound,
			"Expected 400, 403 or 404, got %d",
			w.Code,
		)
	})

	// ============================================================================
	// FAILURE SCENARIOS - VALIDATION
	// ============================================================================

	t.Run("Invalid product ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    1,
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := "/api/products/invalid/options/bulk-update"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Product not found", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    1,
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := "/api/products/99999/options/bulk-update"
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Empty options array", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing required field - optionId", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					// Missing optionId
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid display name - too short", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_short", "Test Short", 1)
		optionID := int(option["id"].(float64))

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "ab", // Only 2 characters (min is 3)
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid display name - too long", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_long", "Test Long", 1)
		optionID := int(option["id"].(float64))

		// Create string longer than 100 characters
		longName := strings.Repeat("a", 101)

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": longName,
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Option ID not found", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    99999, // Non-existent option ID
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Option doesn't belong to product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 5
		productID1 := 5
		option := createOption(productID1, "test_mismatch", "Test Mismatch", 1)
		optionID := int(option["id"].(float64))

		// Try to update it using product 6's URL
		productID2 := 6

		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "New Name",
					"position":    5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID2)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Duplicate option IDs in request", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_duplicate", "Test Duplicate", 1)
		optionID := int(option["id"].(float64))

		// Same option ID appears twice
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "First Update",
					"position":    5,
				},
				{
					"optionId":    optionID,
					"displayName": "Second Update",
					"position":    10,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		// Should either succeed (last one wins) or return error
		// Based on implementation, adjust the assertion
		if w.Code == http.StatusOK {
			// If it succeeds, verify the last update was applied
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			data := response["data"].(map[string]interface{})
			// updatedCount could be 1 or 2 depending on implementation
			assert.NotNil(t, data["updatedCount"])
		} else {
			// If it fails, should return 400
			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		}
	})

	t.Run("Invalid request body structure", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		// Invalid structure - options is not an array
		requestBody := map[string]interface{}{
			"options": "not an array",
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing options field", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		requestBody := map[string]interface{}{
			// Missing "options" field
			"something": "else",
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Large batch update", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create 10 options
		updates := []map[string]interface{}{}
		for i := 1; i <= 10; i++ {
			option := createOption(
				productID,
				fmt.Sprintf("option_%d", i),
				fmt.Sprintf("Option %d", i),
				i,
			)
			optionID := int(option["id"].(float64))
			updates = append(updates, map[string]interface{}{
				"optionId":    optionID,
				"displayName": fmt.Sprintf("Updated Option %d", i),
				"position":    i * 10,
			})
		}

		requestBody := map[string]interface{}{
			"options": updates,
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify all 10 were updated
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(10), data["updatedCount"], "Should update 10 options")
	})

	t.Run("Same values as existing", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		// Create option
		option := createOption(productID, "same_value", "Same Value", 5)
		optionID := int(option["id"].(float64))
		displayName := option["displayName"].(string)
		position := int(option["position"].(float64))

		// Update with same values
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": displayName,
					"position":    position,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should still return updatedCount = 1 even though values didn't change
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["updatedCount"], "Should report 1 update")
	})

	t.Run("Mixed valid and invalid option IDs", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create valid option
		validOption := createOption(productID, "valid_mixed", "Valid Mixed", 1)
		validOptionID := int(validOption["id"].(float64))

		// Mix valid and invalid option IDs
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    validOptionID,
					"displayName": "Valid Update",
					"position":    10,
				},
				{
					"optionId":    99999, // Invalid
					"displayName": "Invalid Update",
					"position":    20,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		// Should fail entire operation
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)

		// Verify the valid option was NOT updated (no partial updates)
		options := getOptions(productID)
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] != nil && int(optMap["id"].(float64)) == validOptionID {
				// Should still have original display name
				assert.Equal(
					t,
					"Valid Mixed",
					optMap["displayName"],
					"Should not be partially updated",
				)
				break
			}
		}
	})

	t.Run("Special characters in display name", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		option := createOption(productID, "special_chars", "Special Chars", 1)
		optionID := int(option["id"].(float64))

		// Test various special characters, unicode, emojis
		testCases := []string{
			"Color & Style 🎨",
			"Size (US/EU)",
			"Material - Cotton 100%",
			"Design: Modern™",
			"Taille (Fr/Eu)",
			"サイズ選択", // Japanese
			"옵션 선택", // Korean
		}

		for _, displayName := range testCases {
			requestBody := map[string]interface{}{
				"options": []map[string]interface{}{
					{
						"optionId":    optionID,
						"displayName": displayName,
						"position":    5,
					},
				},
			}

			url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
			w := client.Put(t, url, requestBody)

			helpers.AssertSuccessResponse(t, w, http.StatusOK)

			// Verify the special characters are preserved
			options := getOptions(productID)
			for _, opt := range options {
				optMap := opt.(map[string]interface{})
				if optMap["id"] != nil && int(optMap["id"].(float64)) == optionID {
					assert.Equal(
						t,
						displayName,
						optMap["displayName"],
						"Special characters should be preserved",
					)
					break
				}
			}
		}
	})

	t.Run("Reordering with position conflicts", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		// Create 3 options
		option1 := createOption(productID, "conflict1", "Conflict 1", 1)
		option2 := createOption(productID, "conflict2", "Conflict 2", 2)
		option3 := createOption(productID, "conflict3", "Conflict 3", 3)

		option1ID := int(option1["id"].(float64))
		option2ID := int(option2["id"].(float64))
		option3ID := int(option3["id"].(float64))

		// Assign same position to all
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    option1ID,
					"displayName": "Conflict One",
					"position":    10, // Same position
				},
				{
					"optionId":    option2ID,
					"displayName": "Conflict Two",
					"position":    10, // Same position
				},
				{
					"optionId":    option3ID,
					"displayName": "Conflict Three",
					"position":    10, // Same position
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should succeed - database allows duplicate positions
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(3), data["updatedCount"], "Should update all 3 options")

		// Verify all have position 10
		options := getOptions(productID)
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] == nil {
				continue
			}
			optID := int(optMap["id"].(float64))

			if optID == option1ID || optID == option2ID || optID == option3ID {
				assert.Equal(t, float64(10), optMap["position"], "All should have position 10")
			}
		}
	})

	t.Run("Zero position value", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "zero_position", "Zero Position", 5)
		optionID := int(option["id"].(float64))

		// Set position to 0
		requestBody := map[string]interface{}{
			"options": []map[string]interface{}{
				{
					"optionId":    optionID,
					"displayName": "Zero Pos",
					"position":    0, // Zero position
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/options/bulk-update", productID)
		w := client.Put(t, url, requestBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Should succeed
		data := response["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["updatedCount"], "Should update 1 option")

		// Verify position is 0
		options := getOptions(productID)
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] != nil && int(optMap["id"].(float64)) == optionID {
				assert.Equal(t, float64(0), optMap["position"], "Position should be 0")
				assert.Equal(t, "Zero Pos", optMap["displayName"])
				break
			}
		}
	})
}
