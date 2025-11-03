package variant

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestGetVariantByID(t *testing.T) {
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

	t.Run("Success - Get variant by ID with valid product and variant", func(t *testing.T) {
		// Product 5 (T-Shirt), Variant 9 (Black + M, default)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert basic fields
		assert.Equal(t, float64(variantID), variant["id"])
		assert.Equal(t, float64(productID), variant["productId"])
		assert.Equal(t, "NIKE-TSHIRT-BLK-M", variant["sku"])
		assert.Equal(t, 29.99, variant["price"])
		assert.NotNil(t, variant["product"], "Should include product info")
		assert.NotNil(t, variant["selectedOptions"])
		assert.NotNil(t, variant["createdAt"])
		assert.NotNil(t, variant["updatedAt"])
	})

	t.Run("Success - Get variant with all fields populated", func(t *testing.T) {
		// Product 1 (iPhone), Variant 1 with images, options
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		productID := 1
		variantID := 1
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Assert complete variant detail response
		assert.Equal(t, float64(variantID), variant["id"])
		assert.Equal(t, "IPHONE-15-PRO-NAT-128", variant["sku"])
		assert.Equal(t, 999.00, variant["price"])
		assert.True(t, variant["allowPurchase"].(bool))
		assert.True(t, variant["isPopular"].(bool))
		assert.True(t, variant["isDefault"].(bool))
		assert.NotNil(t, variant["images"])
		assert.NotNil(t, variant["product"])
	})

	t.Run("Success - Get variant with multiple options (3 options)", func(t *testing.T) {
		// Product 3 (MacBook), Variant 7 (Color + Memory + Storage)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		productID := 3
		variantID := 7
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Check selectedOptions has 3 options
		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok, "selectedOptions should be an array")
		assert.Len(t, selectedOptions, 3, "MacBook should have 3 options: Color, Memory, Storage")

		assert.Equal(t, "MBP-16-M3-SB-16-512", variant["sku"])
		assert.Equal(t, 2499.00, variant["price"])
	})

	t.Run("Success - Get default variant", func(t *testing.T) {
		// Product 5, Variant 9 (isDefault: true)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.True(t, variant["isDefault"].(bool), "Variant 9 should be the default")
		assert.Equal(t, "NIKE-TSHIRT-BLK-M", variant["sku"])
	})

	t.Run("Success - Get non-default variant", func(t *testing.T) {
		// Product 5, Variant 10 (White + M, isDefault: false)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 10
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.False(t, variant["isDefault"].(bool), "Variant 10 should not be default")
		assert.Equal(t, "NIKE-TSHIRT-WHT-M", variant["sku"])
	})

	t.Run("Success - Public access without authentication", func(t *testing.T) {
		// Clear any existing token
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.NotNil(t, variant["id"])
		assert.Equal(t, "NIKE-TSHIRT-BLK-M", variant["sku"])
	})

	t.Run("Success - Seller accessing their own product variant", func(t *testing.T) {
		// Seller 3 owns Product 5
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, float64(variantID), variant["id"])
		assert.Equal(t, "NIKE-TSHIRT-BLK-M", variant["sku"])
	})

	// The handler dereferences sellerID without checking for nil at line 58
	// When admin/customer users access (no seller ID), this causes panic
	t.Run("Admin accessing any product variant", func(t *testing.T) {
		// Admin can access any product
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		productID := 1
		variantID := 1
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		// This should succeed but currently panics with nil pointer dereference
		// Expected behavior: Admin should be able to view any variant
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, float64(variantID), variant["id"])
		assert.Equal(t, "IPHONE-15-PRO-NAT-128", variant["sku"])
	})

	// Same issue as admin test - nil pointer dereference when customer accesses
	t.Run("Customer accessing any variant", func(t *testing.T) {
		// Customer can view any product
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		productID := 2
		variantID := 5
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		// This should succeed but currently panics with nil pointer dereference
		// Expected behavior: Customer should be able to view any variant
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, float64(variantID), variant["id"])
		assert.Equal(t, "SAMSUNG-S24-BLK-128", variant["sku"])
	})

	t.Run("Success - Get variant with popular flag", func(t *testing.T) {
		// Product 1, Variant 1 (isPopular: true)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		productID := 1
		variantID := 1
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.True(t, variant["isPopular"].(bool), "Variant 1 should be popular")
	})

	t.Run("Success - Get variant with multiple images", func(t *testing.T) {
		// Product 1, Variant 1 with image URL
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		productID := 1
		variantID := 1
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		images, ok := variant["images"].([]interface{})
		assert.True(t, ok, "images should be an array")
		assert.GreaterOrEqual(t, len(images), 1, "iPhone variant should have at least 1 image")
	})

	// ============================================================================
	// ERROR SCENARIOS
	// ============================================================================

	t.Run("Error - Product does not exist", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 99999
		variantID := 1
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Error - Variant does not exist", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 99999
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Error - Variant belongs to different product", func(t *testing.T) {
		// Variant 9 belongs to Product 5, not Product 6
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 6 // Summer Dress
		variantID := 9 // T-Shirt variant
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Error - Seller accessing another seller's product", func(t *testing.T) {
		// Seller 3 trying to access Product 1 (owned by Seller 2)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 1 // iPhone (owned by seller 2)
		variantID := 1
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		// This is actually better security practice to distinguish authorization from not-found
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("Error - Missing X-Seller-ID header for public access", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "") // Clear the header

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Invalid X-Seller-ID header value", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "invalid")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("EdgeCase - Get variant with special characters in option values", func(t *testing.T) {
		// Product 7, Variant 14 with "Black/White" color
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 7
		variantID := 14
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		assert.Equal(t, "ADIDAS-RUN-BW-9", variant["sku"])

		// Check selectedOptions for the special character
		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok)

		// Find the Color option
		foundColorOption := false
		for _, opt := range selectedOptions {
			option := opt.(map[string]interface{})
			if option["optionName"] == "Color" {
				foundColorOption = true
				// Should have "Black/White" as value
				assert.Contains(
					t,
					option["value"].(string),
					"/",
					"Color value should contain forward slash",
				)
			}
		}
		assert.True(t, foundColorOption, "Should find Color option")
	})

	// ============================================================================
	// AUTHORIZATION SCENARIOS
	// ============================================================================

	t.Run("Authorization - Invalid/Malformed JWT token", func(t *testing.T) {
		client.SetToken("invalid-token-12345")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Authorization - Expired JWT token", func(t *testing.T) {
		expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1MTYyMzkwMjIsInVzZXJJZCI6MX0.expired"
		client.SetToken(expiredToken)

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	// ============================================================================
	// RESPONSE VALIDATION SCENARIOS
	// ============================================================================

	t.Run(
		"ResponseValidation - Verify VariantDetailResponse has all required fields",
		func(t *testing.T) {
			client.SetToken("")
			client.SetHeader("X-Seller-ID", "3")

			productID := 5
			variantID := 9
			url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
			w := client.Get(t, url)

			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			variant := helpers.GetResponseData(t, response, "variant")

			// Verify all required fields for VariantDetailResponse
			requiredFields := []string{
				"id", "productId", "product", "sku", "price", "images",
				"allowPurchase", "isPopular", "isDefault",
				"selectedOptions", "createdAt", "updatedAt",
			}

			for _, field := range requiredFields {
				assert.Contains(
					t,
					variant,
					field,
					fmt.Sprintf("Variant should have %s field", field),
				)
				assert.NotNil(t, variant[field], fmt.Sprintf("%s should not be nil", field))
			}
		},
	)

	t.Run("ResponseValidation - Verify product info in response", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Check product object
		product, ok := variant["product"].(map[string]interface{})
		assert.True(t, ok, "product should be an object")

		// Verify product fields
		assert.NotNil(t, product["id"], "product should have id")
		assert.NotNil(t, product["name"], "product should have name")
		assert.Equal(t, "Classic Cotton T-Shirt", product["name"], "product name should match")
		assert.NotNil(t, product["brand"], "product should have brand")
		assert.Equal(t, "Nike", product["brand"], "product brand should match")
	})

	t.Run("ResponseValidation - Verify selectedOptions structure", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "2")

		productID := 1
		variantID := 1
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok, "selectedOptions should be an array")
		assert.Greater(t, len(selectedOptions), 0, "Should have at least one option")

		// Verify structure of each option
		for _, opt := range selectedOptions {
			option := opt.(map[string]interface{})

			// Required fields in each option
			assert.NotNil(t, option["optionId"])
			assert.NotNil(t, option["optionName"])
			assert.NotNil(t, option["optionDisplayName"])
			assert.NotNil(t, option["valueId"])
			assert.NotNil(t, option["value"])
			assert.NotNil(t, option["valueDisplayName"])

			// Verify types
			assert.IsType(t, float64(0), option["optionId"])
			assert.IsType(t, "", option["optionName"])
			assert.IsType(t, "", option["value"])
		}
	})

	t.Run("ResponseValidation - Verify color option has colorCode", func(t *testing.T) {
		// Product 5: T-Shirt with Color option that has color codes
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9 // Black variant
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok)

		// Find the Color option and verify it has colorCode
		foundColorOption := false
		for _, opt := range selectedOptions {
			option := opt.(map[string]interface{})
			if option["optionName"] == "Color" {
				foundColorOption = true
				assert.NotNil(t, option["colorCode"], "Color option should have colorCode")
				assert.Equal(
					t,
					"#000000",
					option["colorCode"],
					"Black color should have #000000 code",
				)
			}
		}
		assert.True(t, foundColorOption, "Should find Color option in selectedOptions")
	})

	t.Run("ResponseValidation - Verify timestamps format", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Check timestamps exist and are non-empty strings
		createdAt, ok := variant["createdAt"].(string)
		assert.True(t, ok, "createdAt should be a string")
		assert.NotEmpty(t, createdAt, "createdAt should not be empty")

		updatedAt, ok := variant["updatedAt"].(string)
		assert.True(t, ok, "updatedAt should be a string")
		assert.NotEmpty(t, updatedAt, "updatedAt should not be empty")
	})

	t.Run("ResponseValidation - Verify price is float", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// Price should be float64 in JSON
		price, ok := variant["price"].(float64)
		assert.True(t, ok, "price should be a float64")
		assert.Equal(t, 29.99, price)
	})

	// ============================================================================
	// BUSINESS LOGIC TESTS
	// ============================================================================

	t.Run(
		"BusinessLogic - Different variants of same product return different data",
		func(t *testing.T) {
			client.SetToken("")
			client.SetHeader("X-Seller-ID", "3")

			productID := 5

			// Get Variant 9 (Black + M)
			url1 := fmt.Sprintf("/api/products/%d/variants/9", productID)
			w1 := client.Get(t, url1)
			response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
			variant1 := helpers.GetResponseData(t, response1, "variant")

			// Get Variant 10 (White + M)
			url2 := fmt.Sprintf("/api/products/%d/variants/10", productID)
			w2 := client.Get(t, url2)
			response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
			variant2 := helpers.GetResponseData(t, response2, "variant")

			// Should have different IDs and SKUs
			assert.NotEqual(t, variant1["id"], variant2["id"])
			assert.NotEqual(t, variant1["sku"], variant2["sku"])
			assert.Equal(t, "NIKE-TSHIRT-BLK-M", variant1["sku"])
			assert.Equal(t, "NIKE-TSHIRT-WHT-M", variant2["sku"])

			// selectedOptions should be different
			options1 := variant1["selectedOptions"].([]interface{})
			options2 := variant2["selectedOptions"].([]interface{})
			assert.Len(t, options1, 2)
			assert.Len(t, options2, 2)
		},
	)

	t.Run(
		"BusinessLogic - Product info is same for all variants of same product",
		func(t *testing.T) {
			client.SetToken("")
			client.SetHeader("X-Seller-ID", "3")

			productID := 5

			// Get two different variants of same product
			url1 := fmt.Sprintf("/api/products/%d/variants/9", productID)
			w1 := client.Get(t, url1)
			response1 := helpers.AssertSuccessResponse(t, w1, http.StatusOK)
			variant1 := helpers.GetResponseData(t, response1, "variant")

			url2 := fmt.Sprintf("/api/products/%d/variants/10", productID)
			w2 := client.Get(t, url2)
			response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
			variant2 := helpers.GetResponseData(t, response2, "variant")

			// Product info should be identical
			product1 := variant1["product"].(map[string]interface{})
			product2 := variant2["product"].(map[string]interface{})

			assert.Equal(t, product1["id"], product2["id"])
			assert.Equal(t, product1["name"], product2["name"])
			assert.Equal(t, product1["brand"], product2["brand"])
		},
	)

	t.Run("BusinessLogic - Variant belongs to correct product", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		// productId in response should match the productId in URL
		assert.Equal(t, float64(productID), variant["productId"])

		// product.id should also match
		product := variant["product"].(map[string]interface{})
		assert.Equal(t, float64(productID), product["id"])
	})

	t.Run("BusinessLogic - Option values match variant's actual configuration", func(t *testing.T) {
		// Get variant that should have "Size: M, Color: Black"
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		productID := 5
		variantID := 9 // Black + M
		url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		variant := helpers.GetResponseData(t, response, "variant")

		selectedOptions, ok := variant["selectedOptions"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, selectedOptions, 2, "Should have exactly 2 options")

		// Verify the options are Size and Color with correct values
		optionsMap := make(map[string]string)
		for _, opt := range selectedOptions {
			option := opt.(map[string]interface{})
			optionsMap[option["optionName"].(string)] = option["value"].(string)
		}

		assert.Equal(t, "M", optionsMap["Size"], "Size should be M")
		assert.Equal(t, "Black", optionsMap["Color"], "Color should be Black")
	})

	// ============================================================================
	// COMPARISON WITH FindVariantByOptions
	// ============================================================================

	t.Run(
		"Comparison - GetVariantByID returns more detailed response than FindVariantByOptions",
		func(t *testing.T) {
			client.SetToken("")
			client.SetHeader("X-Seller-ID", "3")

			productID := 5
			variantID := 9

			// Get variant by ID (returns VariantDetailResponse)
			url := fmt.Sprintf("/api/products/%d/variants/%d", productID, variantID)
			w := client.Get(t, url)
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
			variantDetail := helpers.GetResponseData(t, response, "variant")

			// VariantDetailResponse should have:
			// - product info (id, name, brand)
			// - timestamps (createdAt, updatedAt)
			// - productId field
			assert.Contains(
				t,
				variantDetail,
				"product",
				"VariantDetailResponse should have product info",
			)
			assert.Contains(
				t,
				variantDetail,
				"createdAt",
				"VariantDetailResponse should have createdAt",
			)
			assert.Contains(
				t,
				variantDetail,
				"updatedAt",
				"VariantDetailResponse should have updatedAt",
			)
			assert.Contains(
				t,
				variantDetail,
				"productId",
				"VariantDetailResponse should have productId",
			)

			// Verify product object exists and has required fields
			product, ok := variantDetail["product"].(map[string]interface{})
			assert.True(t, ok, "product should be an object")
			assert.NotNil(t, product["id"])
			assert.NotNil(t, product["name"])
			assert.NotNil(t, product["brand"])
		},
	)
}
