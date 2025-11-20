package product_option

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestDeleteProductOption(t *testing.T) {
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

	// Helper function to create an option with values
	createOptionWithValues := func(productID int, name string, displayName string, position int, values []map[string]interface{}) map[string]interface{} {
		requestBody := map[string]interface{}{
			"name":        name,
			"displayName": displayName,
			"position":    position,
			"values":      values,
		}

		url := fmt.Sprintf("/api/products/%d/options", productID)
		w := client.Post(t, url, requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		return helpers.GetResponseData(t, response, "option")
	}

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Successfully delete option", func(t *testing.T) {
		// Login as seller (Jane - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's Classic Cotton T-Shirt)
		productID := 5

		// Create option
		option := createOption(productID, "material", "Material", 1)
		optionID := int(option["id"].(float64))

		// Delete the option
		url := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify success message
		assert.Contains(
			t,
			response["message"],
			"deleted",
			"Response should contain success message",
		)
	})

	t.Run("Delete option and verify it's gone", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		// Create option
		option := createOption(productID, "fabric", "Fabric Type", 1)
		optionID := int(option["id"].(float64))

		// Delete the option
		deleteURL := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, deleteURL)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Try to delete again - should return 404
		w2 := client.Delete(t, deleteURL)
		helpers.AssertErrorResponse(t, w2, http.StatusNotFound)
	})

	t.Run("Delete option with multiple values - cascade delete", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		// First, get initial count of options
		getURL := fmt.Sprintf("/api/products/%d/options", productID)
		wGetInitial := client.Get(t, getURL)
		getInitialResponse := helpers.AssertSuccessResponse(t, wGetInitial, http.StatusOK)
		initialOptionsData := helpers.GetResponseData(t, getInitialResponse, "options")
		initialOptions := initialOptionsData["options"].([]interface{})
		initialCount := len(initialOptions)

		// Create option with 3 values
		values := []map[string]interface{}{
			{
				"value":       "red",
				"displayName": "Red",
				"position":    1,
			},
			{
				"value":       "blue",
				"displayName": "Blue",
				"position":    2,
			},
			{
				"value":       "green",
				"displayName": "Green",
				"position":    3,
			},
		}

		option := createOptionWithValues(productID, "color_variant", "Color Variant", 10, values)
		optionID := int(option["id"].(float64))

		// Verify option was created with values
		assert.NotNil(t, option["values"])
		optionValues := option["values"].([]interface{})
		assert.Equal(t, 3, len(optionValues), "Should have 3 values")

		// Delete the option
		deleteURL := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, deleteURL)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify option is deleted - count should be back to initial
		wGet := client.Get(t, getURL)
		getResponse := helpers.AssertSuccessResponse(t, wGet, http.StatusOK)

		optionsData := helpers.GetResponseData(t, getResponse, "options")
		options := optionsData["options"].([]interface{})

		// Should be back to initial count (our option was deleted)
		assert.Equal(
			t,
			initialCount,
			len(options),
			"Option count should be back to initial after deletion",
		)

		// Verify the deleted option is not in the list
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] != nil {
				optID := int(optMap["id"].(float64))
				assert.NotEqual(t, optionID, optID, "Deleted option should not be in list")
			}
		}
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

		option := createOption(productID, "pattern", "Pattern", 1)
		optionID := int(option["id"].(float64))

		// Clear token
		client.SetToken("")

		url := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Unauthorized - Invalid token", func(t *testing.T) {
		// First login as seller to create option
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "texture", "Texture", 1)
		optionID := int(option["id"].(float64))

		// Set invalid token
		client.SetToken("invalid.token.here")

		url := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Forbidden - Not a seller (customer trying to delete)", func(t *testing.T) {
		// First login as seller to create option
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "weight", "Weight", 1)
		optionID := int(option["id"].(float64))

		// Login as customer
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		url := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Forbidden - Wrong seller", func(t *testing.T) {
		// Login as seller (Jane) to create option on her product
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product - seller_id 3)
		productID := 5

		option := createOption(productID, "style", "Style", 1)
		optionID := int(option["id"].(float64))

		// Try to delete using product 1 which doesn't belong to Jane (belongs to seller_id 2)
		// Product 1 is owned by John Smith (seller_id 2)
		otherProductID := 1

		url := fmt.Sprintf("/api/products/%d/options/%d", otherProductID, optionID)
		w := client.Delete(t, url)

		// Should return 400, 403 Forbidden or 404 Not Found
		assert.True(
			t,
			w.Code == http.StatusBadRequest || w.Code == http.StatusForbidden ||
				w.Code == http.StatusNotFound,
			"Expected 400, 403 or 404, got %d",
			w.Code,
		)
	})

	t.Run("Admin can delete option from any product", func(t *testing.T) {
		// First login as seller to create option
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "finish", "Finish Type", 1)
		optionID := int(option["id"].(float64))

		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		url := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, url)

		// Admin should be able to delete options
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Admin can delete option from different seller's product", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Admin deletes option from Product 1 (owned by seller_id 2)
		// First create an option as admin
		productID := 1
		option := createOption(productID, "certification", "Certification", 1)
		optionID := int(option["id"].(float64))

		url := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, url)

		// Admin should be able to delete
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	// ============================================================================
	// FAILURE SCENARIOS - VALIDATION
	// ============================================================================

	t.Run("Invalid product ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use invalid product ID
		url := "/api/products/invalid/options/1"
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid option ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		url := fmt.Sprintf("/api/products/%d/options/invalid", productID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Product not found", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use non-existent product ID
		url := "/api/products/99999/options/1"
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Option not found", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		// Use non-existent option ID
		url := fmt.Sprintf("/api/products/%d/options/99999", productID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Option doesn't belong to product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create option for product 5
		productID1 := 5
		option := createOption(productID1, "sleeve", "Sleeve Type", 1)
		optionID := int(option["id"].(float64))

		// Try to delete it using product 6's URL
		productID2 := 6

		url := fmt.Sprintf("/api/products/%d/options/%d", productID2, optionID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Already deleted option", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		option := createOption(productID, "length", "Length", 1)
		optionID := int(option["id"].(float64))

		url := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)

		// Delete first time - should succeed
		w1 := client.Delete(t, url)
		helpers.AssertSuccessResponse(t, w1, http.StatusOK)

		// Try to delete again - should fail
		w2 := client.Delete(t, url)
		helpers.AssertErrorResponse(t, w2, http.StatusNotFound)
	})

	// ============================================================================
	// FAILURE SCENARIOS - BUSINESS LOGIC
	// ============================================================================

	t.Run("Cannot delete option in use by variants", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes) which has existing variants in seed data
		// Variants 14 and 15 use option 12 (Size) and option 13 (Color)
		productID := 7
		optionToDelete := 12 // Size option used by variants
		deleteURL := fmt.Sprintf("/api/products/%d/options/%d", productID, optionToDelete)
		w := client.Delete(t, deleteURL)

		// Should return 400 or 409 (Conflict)
		assert.True(
			t,
			w.Code == http.StatusBadRequest || w.Code == http.StatusConflict,
			"Expected 400 or 409, got %d. Body: %s",
			w.Code,
			w.Body.String(),
		)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Delete one option doesn't affect other options", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Get initial count
		getURL := fmt.Sprintf("/api/products/%d/options", productID)
		wGetInitial := client.Get(t, getURL)
		getInitialResponse := helpers.AssertSuccessResponse(t, wGetInitial, http.StatusOK)
		initialOptionsData := helpers.GetResponseData(t, getInitialResponse, "options")
		initialOptions := initialOptionsData["options"].([]interface{})
		initialCount := len(initialOptions)

		// Create multiple options
		createOption(productID, "neckline", "Neckline", 10)
		option2 := createOption(productID, "fit_type", "Fit Type", 11)
		createOption(productID, "care", "Care Instructions", 12)

		option2ID := int(option2["id"].(float64))

		// Delete the second option
		deleteURL := fmt.Sprintf("/api/products/%d/options/%d", productID, option2ID)
		w := client.Delete(t, deleteURL)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify we have the correct count: initial + 2 (created 3, deleted 1)
		wGet := client.Get(t, getURL)
		getResponse := helpers.AssertSuccessResponse(t, wGet, http.StatusOK)

		optionsData := helpers.GetResponseData(t, getResponse, "options")
		options := optionsData["options"].([]interface{})

		// Should have initial + 2 (we created 3 and deleted 1)
		expectedCount := initialCount + 2
		assert.Equal(
			t,
			expectedCount,
			len(options),
			"Should have correct number of options after delete",
		)

		// Verify the deleted option is not in the list
		for _, opt := range options {
			optMap := opt.(map[string]interface{})
			if optMap["id"] != nil {
				optID := int(optMap["id"].(float64))
				assert.NotEqual(t, option2ID, optID, "Deleted option should not be in list")
			}
		}
	})

	t.Run("Delete option from one product doesn't affect other products", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create same option name on two different products
		productID1 := 5 // Jane's T-Shirt
		productID2 := 6 // Jane's Summer Dress

		option1 := createOption(productID1, "fit_style", "Fit Style", 20)
		createOption(productID2, "fit_style", "Fit Style", 20)

		option1ID := int(option1["id"].(float64))

		// Get initial count for product 2
		getURL2 := fmt.Sprintf("/api/products/%d/options", productID2)
		wGetInitial2 := client.Get(t, getURL2)
		getInitialResponse2 := helpers.AssertSuccessResponse(t, wGetInitial2, http.StatusOK)
		initialOptionsData2 := helpers.GetResponseData(t, getInitialResponse2, "options")
		initialOptions2 := initialOptionsData2["options"].([]interface{})
		initialCount2 := len(initialOptions2)

		// Delete option from product 1
		deleteURL := fmt.Sprintf("/api/products/%d/options/%d", productID1, option1ID)
		w := client.Delete(t, deleteURL)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify product 2 still has the same count
		wGet := client.Get(t, getURL2)
		getResponse := helpers.AssertSuccessResponse(t, wGet, http.StatusOK)

		optionsData := helpers.GetResponseData(t, getResponse, "options")
		options := optionsData["options"].([]interface{})

		// Count should be the same as before (we only deleted from product 1)
		assert.Equal(
			t,
			initialCount2,
			len(options),
			"Product 2 option count should remain unchanged",
		)
	})

	t.Run("Deleted product - option delete should fail", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create option first
		option := createOption(productID, "collar", "Collar Type", 1)
		optionID := int(option["id"].(float64))

		// Soft delete the product
		deleteProductURL := fmt.Sprintf("/api/products/%d", productID)
		wDelete := client.Delete(t, deleteProductURL)
		helpers.AssertSuccessResponse(t, wDelete, http.StatusOK)

		// Try to delete option from soft-deleted product
		deleteOptionURL := fmt.Sprintf("/api/products/%d/options/%d", productID, optionID)
		w := client.Delete(t, deleteOptionURL)

		// Should return 404 since product is soft-deleted
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})
}
