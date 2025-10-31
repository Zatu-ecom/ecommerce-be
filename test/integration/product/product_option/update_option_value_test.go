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

// TestUpdateOptionValueSimple - Simplified version using only seed data to avoid bugs in CreateOption APIs
func TestUpdateOptionValueSimple(t *testing.T) {
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
		- Product 5: "Classic Cotton T-Shirt" (seller_id: 3 - Jane/seller@example.com)
		  - Option 8: "Size" (position: 1)
		    - Value 23: "S" -> "Small" (position: 1)
		    - Value 24: "M" -> "Medium" (position: 2)
		    - Value 25: "L" -> "Large" (position: 3)
		    - Value 26: "XL" -> "Extra Large" (position: 4)
		    - Value 27: "XXL" -> "2X Large" (position: 5)
		  - Option 9: "Color" (position: 2)
		    - Value 28: "Black" -> "Black" #000000 (position: 1)
		    - Value 29: "White" -> "White" #FFFFFF (position: 2)
		    - Value 30: "Navy" -> "Navy Blue" #000080 (position: 3)
		    - Value 31: "Gray" -> "Gray" #808080 (position: 4)

		- Product 1: "iPhone 15 Pro" (seller_id: 2 - John Seller/john.seller@example.com)
		  - Option 1: "Color" (position: 1)
		    - Value 1: "Natural Titanium" -> "Natural Titanium" #F5E6D3 (position: 1)
	*/

	// Helper function to update an option value
	updateOptionValue := func(productID int, optionID, valueID int, body map[string]interface{}) *httptest.ResponseRecorder {
		url := fmt.Sprintf("/api/products/%d/options/%d/values/%d", productID, optionID, valueID)
		return client.Put(t, url, body)
	}

	// ============================================================================
	// SUCCESS CASES
	// ============================================================================

	t.Run("Update value display name only", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use: Product 5, Option 8, Value 24 (Medium)
		w := updateOptionValue(5, 8, 24, map[string]interface{}{
			"displayName": "Medium Size Updated",
		})
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "Medium Size Updated", optionValue["displayName"])
		assert.Equal(t, "M", optionValue["value"]) // Value is "M" not "m" in seed data
	})

	t.Run("Update value color code only", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use: Product 5, Option 9, Value 28 (Black)
		w := updateOptionValue(5, 9, 28, map[string]interface{}{
			"colorCode": "#FF0000",
		})
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "Black", optionValue["displayName"])
		assert.Equal(t, "#FF0000", optionValue["colorCode"])
	})

	t.Run("Update value position only", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use: Product 5, Option 8, Value 23 (Small)
		w := updateOptionValue(5, 8, 23, map[string]interface{}{
			"position": 10,
		})
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "Small", optionValue["displayName"])
		assert.Equal(t, float64(10), optionValue["position"])
	})

	t.Run("Update all fields together", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use: Product 5, Option 9, Value 29 (White)
		w := updateOptionValue(5, 9, 29, map[string]interface{}{
			"displayName": "Pure White",
			"colorCode":   "#F0F0F0",
			"position":    5,
		})
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "Pure White", optionValue["displayName"])
		assert.Equal(t, "#F0F0F0", optionValue["colorCode"])
		assert.Equal(t, float64(5), optionValue["position"])
	})

	t.Run("Update with valid hex color code uppercase", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use: Product 5, Option 9, Value 30 (Navy)
		w := updateOptionValue(5, 9, 30, map[string]interface{}{
			"colorCode": "#ABCDEF",
		})
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "#ABCDEF", optionValue["colorCode"])
	})

	t.Run("Update with minimum valid displayName (1 char)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use: Product 5, Option 8, Value 25 (Large)
		w := updateOptionValue(5, 8, 25, map[string]interface{}{
			"displayName": "L",
		})
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "L", optionValue["displayName"])
	})

	t.Run("Update with maximum valid displayName (100 chars)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use: Product 5, Option 8, Value 26 (XL)
		longName := strings.Repeat("A", 100)
		w := updateOptionValue(5, 8, 26, map[string]interface{}{
			"displayName": longName,
		})
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, longName, optionValue["displayName"])
	})

	// ============================================================================
	// AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Update without authentication", func(t *testing.T) {
		client.SetToken("")

		w := updateOptionValue(5, 8, 24, map[string]interface{}{
			"displayName": "Updated",
		})

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Update with customer role", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		w := updateOptionValue(5, 8, 24, map[string]interface{}{
			"displayName": "Updated by Customer",
		})

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Update another seller's product option value", func(t *testing.T) {
		// Product 5 belongs to seller_id 3 (seller@example.com)
		// Try with seller_id 2 (john.seller@example.com)
		anotherSellerToken := helpers.Login(t, client, "john.seller@example.com", "seller123")
		client.SetToken(anotherSellerToken)

		w := updateOptionValue(5, 8, 24, map[string]interface{}{
			"displayName": "Updated by Another Seller",
		})

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// ============================================================================
	// VALIDATION ERRORS - INVALID IDs
	// ============================================================================

	t.Run("Invalid product ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := client.Put(t, "/api/products/invalid/options/8/values/24", map[string]interface{}{
			"displayName": "Updated",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid option ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := client.Put(t, "/api/products/5/options/invalid/values/24", map[string]interface{}{
			"displayName": "Updated",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid value ID format", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := client.Put(t, "/api/products/5/options/8/values/invalid", map[string]interface{}{
			"displayName": "Updated",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Non-existent product ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := updateOptionValue(99999, 8, 24, map[string]interface{}{
			"displayName": "Updated",
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// ============================================================================
	// VALIDATION ERRORS - RELATIONSHIPS
	// ============================================================================

	t.Run("Non-existent option ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := updateOptionValue(5, 99999, 24, map[string]interface{}{
			"displayName": "Updated",
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Non-existent value ID", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := updateOptionValue(5, 8, 99999, map[string]interface{}{
			"displayName": "Updated",
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Option doesn't belong to product", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Option 8 belongs to product 5, try with product 6
		w := updateOptionValue(6, 8, 24, map[string]interface{}{
			"displayName": "Updated",
		})

		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound)
	})

	t.Run("Value doesn't belong to option", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Value 28 belongs to option 9, try with option 8
		w := updateOptionValue(5, 8, 28, map[string]interface{}{
			"displayName": "Updated",
		})

		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound)
	})

	// ============================================================================
	// VALIDATION ERRORS - REQUEST BODY
	// ============================================================================

	t.Run("Empty request body - No fields provided", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := updateOptionValue(5, 8, 24, map[string]interface{}{})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid displayName - too short (empty string)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := updateOptionValue(5, 8, 24, map[string]interface{}{
			"displayName": "",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid displayName - too long (>100 chars)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		longName := strings.Repeat("A", 101)
		w := updateOptionValue(5, 8, 24, map[string]interface{}{
			"displayName": longName,
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Update position to 0", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := updateOptionValue(5, 8, 27, map[string]interface{}{
			"position": 0,
		})

		if w.Code == http.StatusOK {
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			optionValue := helpers.GetResponseData(t, response, "optionValue")
			position := optionValue["position"].(float64)
			t.Logf("Position after update to 0: %v (original was 5)", position)
		}
	})

	t.Run("Update position to negative", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := updateOptionValue(5, 8, 23, map[string]interface{}{
			"position": -1,
		})

		if w.Code == http.StatusOK {
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			optionValue := helpers.GetResponseData(t, response, "optionValue")
			position := optionValue["position"].(float64)
			t.Logf("Position after update to -1: %v", position)
		}
	})

	t.Run("Update with special characters in displayName", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		w := updateOptionValue(5, 8, 24, map[string]interface{}{
			"displayName": "Test™ Value® with Special©",
		})

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.NotEmpty(t, optionValue["displayName"])
	})

	t.Run("Clear color code", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Value 28 has color code #000000
		w := updateOptionValue(5, 9, 28, map[string]interface{}{
			"colorCode": "",
		})

		t.Logf("Clear color code response status: %d", w.Code)
		if w.Code == http.StatusOK {
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			optionValue := helpers.GetResponseData(t, response, "optionValue")
			colorCode := optionValue["colorCode"]
			t.Logf("ColorCode after clearing: %v", colorCode)
		}
	})

	t.Run("Update same value multiple times", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// First update
		w1 := updateOptionValue(5, 9, 31, map[string]interface{}{
			"displayName": "Updated Once",
		})
		helpers.AssertSuccessResponse(t, w1, http.StatusOK)

		// Second update
		w2 := updateOptionValue(5, 9, 31, map[string]interface{}{
			"displayName": "Updated Twice",
		})
		helpers.AssertSuccessResponse(t, w2, http.StatusOK)

		// Third update
		w3 := updateOptionValue(5, 9, 31, map[string]interface{}{
			"displayName": "Updated Three Times",
			"position":    10,
		})
		response := helpers.AssertSuccessResponse(t, w3, http.StatusOK)

		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "Updated Three Times", optionValue["displayName"])
		assert.Equal(t, float64(10), optionValue["position"])
	})
}
