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

func TestAddOptionValue(t *testing.T) {
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

	// Helper function to add an option value
	addOptionValue := func(productID int, optionID int, value string, displayName string, colorCode string, position int) *httptest.ResponseRecorder {
		requestBody := map[string]interface{}{
			"value":       value,
			"displayName": displayName,
			"position":    position,
		}
		if colorCode != "" {
			requestBody["colorCode"] = colorCode
		}

		url := fmt.Sprintf("/api/products/%d/options/%d/values", productID, optionID)
		return client.Post(t, url, requestBody)
	}

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Add value with all fields", func(t *testing.T) {
		// Login as seller (Jane - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create an option
		option := createOption(productID, "color", "Color", 1)
		optionID := int(option["id"].(float64))

		// Add value with all fields
		w := addOptionValue(productID, optionID, "red", "Red", "#FF0000", 1)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Verify response
		valueData := helpers.GetResponseData(t, response, "optionValue")
		assertOptionValueFieldsWithColor(t, valueData, "red", "Red", "#FF0000", 1)
	})

	t.Run("Add value without colorCode", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		// Create an option
		option := createOption(productID, "size", "Size", 1)
		optionID := int(option["id"].(float64))

		// Add value without colorCode
		requestBody := map[string]interface{}{
			"value":       "small",
			"displayName": "Small",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options/%d/values", productID, optionID)
		w := client.Post(t, url, requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Verify response
		valueData := helpers.GetResponseData(t, response, "optionValue")
		assertOptionValueFields(t, valueData, "small", "Small")
	})

	t.Run("Add value without position", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		// Create an option
		option := createOption(productID, "material", "Material", 1)
		optionID := int(option["id"].(float64))

		// Add value without position
		requestBody := map[string]interface{}{
			"value":       "leather",
			"displayName": "Leather",
			"colorCode":   "#8B4513",
		}

		url := fmt.Sprintf("/api/products/%d/options/%d/values", productID, optionID)
		w := client.Post(t, url, requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Verify response
		valueData := helpers.GetResponseData(t, response, "optionValue")
		assertOptionValueFields(t, valueData, "leather", "Leather")
		assert.Equal(t, "#8B4513", valueData["colorCode"])
	})

	t.Run("Add multiple values to same option", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create an option
		option := createOption(productID, "pattern", "Pattern", 1)
		optionID := int(option["id"].(float64))

		// Add multiple values
		values := []struct {
			value       string
			displayName string
		}{
			{"solid", "Solid"},
			{"striped", "Striped"},
			{"checked", "Checked"},
		}

		for _, v := range values {
			w := addOptionValue(productID, optionID, v.value, v.displayName, "", 0)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			valueData := helpers.GetResponseData(t, response, "optionValue")
			assert.Equal(t, v.value, valueData["value"])
			assert.Equal(t, v.displayName, valueData["displayName"])
		}
	})

	t.Run("Add value with position 0", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		// Create an option
		option := createOption(productID, "fit", "Fit", 1)
		optionID := int(option["id"].(float64))

		// Add value with position 0
		w := addOptionValue(productID, optionID, "regular", "Regular Fit", "", 0)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Verify response
		valueData := helpers.GetResponseData(t, response, "optionValue")
		assertOptionValueFields(t, valueData, "regular", "Regular Fit")
		assert.Equal(t, float64(0), valueData["position"])
	})

	t.Run("Add value with valid hex color code", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		// Create an option
		option := createOption(productID, "accent_color", "Accent Color", 1)
		optionID := int(option["id"].(float64))

		// Test various valid hex codes
		validColors := []struct {
			value     string
			colorCode string
		}{
			{"black", "#000000"},
			{"white", "#FFFFFF"},
			{"blue", "#0000FF"},
			{"custom", "#AB12CD"},
		}

		for _, vc := range validColors {
			w := addOptionValue(
				productID,
				optionID,
				vc.value,
				strings.Title(vc.value),
				vc.colorCode,
				0,
			)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			valueData := helpers.GetResponseData(t, response, "optionValue")
			assert.Equal(t, vc.colorCode, valueData["colorCode"])
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

		option := createOption(productID, "test_unauth", "Test Unauth", 1)
		optionID := int(option["id"].(float64))

		// Clear token
		client.SetToken("")

		// Try to add value
		w := addOptionValue(productID, optionID, "value1", "Value 1", "", 0)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Unauthorized - Invalid token", func(t *testing.T) {
		// First login as seller to create option
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_invalid", "Test Invalid", 1)
		optionID := int(option["id"].(float64))

		// Set invalid token
		client.SetToken("invalid.token.here")

		// Try to add value
		w := addOptionValue(productID, optionID, "value1", "Value 1", "", 0)
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Forbidden - Not a seller (customer)", func(t *testing.T) {
		// First login as seller to create option
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_customer", "Test Customer", 1)
		optionID := int(option["id"].(float64))

		// Login as customer
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		// Try to add value
		w := addOptionValue(productID, optionID, "value1", "Value 1", "", 0)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Forbidden - Wrong seller", func(t *testing.T) {
		// Login as seller (Jane) to create option on her product
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product - seller_id 3)
		productID := 5

		option := createOption(productID, "test_wrong_seller", "Test Wrong Seller", 1)
		optionID := int(option["id"].(float64))

		// Try to add value using product 1 which doesn't belong to Jane (belongs to seller_id 2)
		otherProductID := 1

		w := addOptionValue(otherProductID, optionID, "value1", "Value 1", "", 0)

		helpers.AssertStatusCodeOneOf(
			t,
			w,
			http.StatusBadRequest,
			http.StatusForbidden,
			http.StatusNotFound,
		)
	})

	t.Run("Admin can add option value to any product", func(t *testing.T) {
		// First create option as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		option := createOption(productID, "test_admin_add", "Test Admin Add", 1)
		optionID := int(option["id"].(float64))

		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Admin adds option value
		w := addOptionValue(productID, optionID, "admin_val", "Admin Value", "#AAAAAA", 1)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "admin_val", optionValue["value"])
		assert.Equal(t, "Admin Value", optionValue["displayName"])
	})

	t.Run("Admin can add option value to different seller's product", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Admin adds value to Product 1 Option 1 (Color - owned by seller_id 2)
		productID := 1
		optionID := 1 // Color option from seed data

		w := addOptionValue(productID, optionID, "rose gold", "Rose Gold Titanium", "#B76E79", 5)

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		optionValue := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "rose gold", optionValue["value"])
		assert.Equal(t, "Rose Gold Titanium", optionValue["displayName"])
	})

	// ============================================================================
	// FAILURE SCENARIOS - VALIDATION
	// ============================================================================

	t.Run("Invalid product ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"value":       "test",
			"displayName": "Test",
			"position":    1,
		}

		url := "/api/products/invalid/options/1/values"
		w := client.Post(t, url, requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid option ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"value":       "test",
			"displayName": "Test",
			"position":    1,
		}

		url := "/api/products/5/options/abc/values"
		w := client.Post(t, url, requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Product not found", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]interface{}{
			"value":       "test",
			"displayName": "Test",
			"position":    1,
		}

		url := "/api/products/99999/options/1/values"
		w := client.Post(t, url, requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Option not found", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		requestBody := map[string]interface{}{
			"value":       "test",
			"displayName": "Test",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options/99999/values", productID)
		w := client.Post(t, url, requestBody)
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

		// Try to add value using product 6's ID
		productID2 := 6

		w := addOptionValue(productID2, optionID, "value1", "Value 1", "", 0)

		helpers.AssertStatusCodeOneOf(t, w, http.StatusBadRequest, http.StatusNotFound)
	})

	t.Run("Missing required field - value", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_no_value", "Test No Value", 1)
		optionID := int(option["id"].(float64))

		requestBody := map[string]interface{}{
			// Missing "value" field
			"displayName": "Test",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options/%d/values", productID, optionID)
		w := client.Post(t, url, requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing required field - displayName", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_no_display", "Test No Display", 1)
		optionID := int(option["id"].(float64))

		requestBody := map[string]interface{}{
			"value": "test",
			// Missing "displayName" field
			"position": 1,
		}

		url := fmt.Sprintf("/api/products/%d/options/%d/values", productID, optionID)
		w := client.Post(t, url, requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Value too short (empty string)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_empty_value", "Test Empty Value", 1)
		optionID := int(option["id"].(float64))

		w := addOptionValue(productID, optionID, "", "Test", "", 0)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Value too long", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_long_value", "Test Long Value", 1)
		optionID := int(option["id"].(float64))

		// Create string longer than 100 characters
		longValue := strings.Repeat("a", 101)

		w := addOptionValue(productID, optionID, longValue, "Test", "", 0)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("DisplayName too short", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_short_display", "Test Short Display", 1)
		optionID := int(option["id"].(float64))

		w := addOptionValue(productID, optionID, "test", "", "", 0)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("DisplayName too long", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_long_display", "Test Long Display", 1)
		optionID := int(option["id"].(float64))

		// Create string longer than 100 characters
		longDisplay := strings.Repeat("a", 101)

		w := addOptionValue(productID, optionID, "test", longDisplay, "", 0)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Invalid colorCode format - wrong length", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_color_len", "Test Color Length", 1)
		optionID := int(option["id"].(float64))

		// Test wrong lengths
		invalidColors := []string{"#FFF", "#FFFFFF00", "#FF"}

		for _, color := range invalidColors {
			w := addOptionValue(productID, optionID, "test_"+color, "Test", color, 0)
			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		}
	})

	t.Run("Invalid colorCode format - missing hash", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_no_hash", "Test No Hash", 1)
		optionID := int(option["id"].(float64))

		w := addOptionValue(productID, optionID, "test", "Test", "FF5733", 0)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Duplicate value for same option", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		option := createOption(productID, "test_duplicate", "Test Duplicate", 1)
		optionID := int(option["id"].(float64))

		// Add first value
		w := addOptionValue(productID, optionID, "red", "Red", "#FF0000", 1)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Try to add duplicate value
		w = addOptionValue(productID, optionID, "red", "Red Again", "#FF0000", 2)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("Duplicate value - case insensitive", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		option := createOption(productID, "test_case", "Test Case", 1)
		optionID := int(option["id"].(float64))

		// Add first value
		w := addOptionValue(productID, optionID, "red", "Red", "", 1)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Try to add with different case
		w = addOptionValue(productID, optionID, "RED", "RED", "", 2)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)

		// Try another variant
		w = addOptionValue(productID, optionID, "Red", "Red Again", "", 3)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("Duplicate value - with whitespace", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_whitespace", "Test Whitespace", 1)
		optionID := int(option["id"].(float64))

		// Add first value
		w := addOptionValue(productID, optionID, "red", "Red", "", 1)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Try to add with whitespace (should be trimmed)
		w = addOptionValue(productID, optionID, " red ", "Red with space", "", 2)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)

		w = addOptionValue(productID, optionID, "red ", "Red trailing", "", 3)
		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("Invalid request body structure", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_invalid_body", "Test Invalid Body", 1)
		optionID := int(option["id"].(float64))

		// Invalid structure - value is not a string
		requestBody := map[string]interface{}{
			"value":       123, // Should be string
			"displayName": "Test",
			"position":    1,
		}

		url := fmt.Sprintf("/api/products/%d/options/%d/values", productID, optionID)
		w := client.Post(t, url, requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Missing entire request body", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_empty_body", "Test Empty Body", 1)
		optionID := int(option["id"].(float64))

		requestBody := map[string]interface{}{}

		url := fmt.Sprintf("/api/products/%d/options/%d/values", productID, optionID)
		w := client.Post(t, url, requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Value with special characters", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		option := createOption(productID, "test_special", "Test Special", 1)
		optionID := int(option["id"].(float64))

		// Test various special characters
		specialValues := []struct {
			value       string
			displayName string
		}{
			{"red-emoji", "Red with Emoji 🔴"},
			{"grosse-xl", "Größe XL"},
			{"size_36/38", "Size 36/38"},
			{"korean-opt", "Korean Option 옵션"},
			{"japanese-size", "Japanese Size サイズ"},
		}

		for _, sv := range specialValues {
			w := addOptionValue(productID, optionID, sv.value, sv.displayName, "", 0)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			valueData := helpers.GetResponseData(t, response, "optionValue")
			// Verify value is saved (values are normalized - lowercased and trimmed)
			assert.NotEmpty(t, valueData["value"])
			assert.NotEmpty(t, valueData["displayName"])
		}
	})

	t.Run("DisplayName with special characters", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		option := createOption(productID, "test_display_special", "Test Display Special", 1)
		optionID := int(option["id"].(float64))

		// Test international characters in display name
		w := addOptionValue(productID, optionID, "val1", "Größe™ (EU/FR) 🎨", "", 0)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		valueData := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "Größe™ (EU/FR) 🎨", valueData["displayName"])
	})

	t.Run("Negative position value", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		option := createOption(productID, "test_negative", "Test Negative", 1)
		optionID := int(option["id"].(float64))

		w := addOptionValue(productID, optionID, "test", "Test", "", -1)

		// This might succeed or fail depending on business logic
		// If it succeeds, verify position is stored
		if w.Code == http.StatusCreated {
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			valueData := helpers.GetResponseData(t, response, "optionValue")
			assert.Equal(t, float64(-1), valueData["position"])
		}
		// If it fails, that's also acceptable behavior
	})

	t.Run("Very large position value", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		option := createOption(productID, "test_large_pos", "Test Large Position", 1)
		optionID := int(option["id"].(float64))

		w := addOptionValue(productID, optionID, "test", "Test", "", 999999)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		valueData := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, float64(999999), valueData["position"])
	})

	t.Run("ColorCode with lowercase hex", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		option := createOption(productID, "test_lowercase", "Test Lowercase", 1)
		optionID := int(option["id"].(float64))

		w := addOptionValue(productID, optionID, "test", "Test", "#ff5733", 0)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		valueData := helpers.GetResponseData(t, response, "optionValue")
		// Should accept both uppercase and lowercase
		assert.Equal(t, "#ff5733", valueData["colorCode"])
	})

	t.Run("Add value to option with existing values", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create an option
		option := createOption(productID, "existing_values", "Existing Values Test", 1)
		optionID := int(option["id"].(float64))

		// Add 5 values first
		existingValues := []string{"value1", "value2", "value3", "value4", "value5"}
		for i, val := range existingValues {
			w := addOptionValue(productID, optionID, val, fmt.Sprintf("Value %d", i+1), "", i+1)
			helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		}

		// Now add another value (6th value)
		w := addOptionValue(productID, optionID, "value6", "Value 6", "#ABCDEF", 6)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)

		// Verify the new value was added successfully
		valueData := helpers.GetResponseData(t, response, "optionValue")
		assert.Equal(t, "value6", valueData["value"])
		assert.Equal(t, "Value 6", valueData["displayName"])
		assert.Equal(t, "#ABCDEF", valueData["colorCode"])
		assert.Equal(t, float64(6), valueData["position"])

		// Add more values to test larger sets (7-10)
		for i := 7; i <= 10; i++ {
			val := fmt.Sprintf("value%d", i)
			display := fmt.Sprintf("Value %d", i)
			w := addOptionValue(productID, optionID, val, display, "", i)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			valueData := helpers.GetResponseData(t, response, "optionValue")
			assert.Equal(t, val, valueData["value"])
			assert.Equal(t, display, valueData["displayName"])
		}
	})
}

// Helper function to get minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
