package get_product_by_id

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetProductByID_HappyPath tests successful scenarios for retrieving a product by ID
func TestGetProductByID_HappyPath(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")
	containers.RunSeeds(t, "migrations/seeds/002_seed_product_data.sql")
	containers.RunSeeds(t, "test/integration/product/product/get_product_by_id/test_seed_data.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// HP-01: Public User Gets Product Successfully with Valid Seller ID
	// ============================================================================
	t.Run("HP-01: Public User Gets Product Successfully with Valid Seller ID", func(t *testing.T) {
		// Product 1 (iPhone 15 Pro) belongs to seller 2
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Extract product data
		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify product basic fields
		assert.Equal(t, float64(1), product["id"], "Product ID should be 1")
		assert.Equal(t, "iPhone 15 Pro", product["name"], "Product name should match")
		assert.Equal(t, "Apple", product["brand"], "Brand should match")
		assert.NotEmpty(t, product["sku"], "SKU should not be empty")

		// Verify category hierarchy
		category, ok := product["category"].(map[string]interface{})
		assert.True(t, ok, "Category should be present")
		assert.NotNil(t, category["id"], "Category ID should be present")
		assert.NotNil(t, category["name"], "Category name should be present")

		// Verify variants are present
		assert.NotNil(t, product["hasVariants"], "hasVariants should be present")
		assert.True(t, product["hasVariants"].(bool), "Product should have variants")

		// Verify options array exists
		options, ok := product["options"].([]interface{})
		assert.True(t, ok, "Options should be an array")
		assert.NotEmpty(t, options, "Options array should not be empty")

		// Verify variants array exists
		variants, ok := product["variants"].([]interface{})
		assert.True(t, ok, "Variants should be an array")
		assert.NotEmpty(t, variants, "Variants array should not be empty")

		// Verify price range
		priceRange, ok := product["priceRange"].(map[string]interface{})
		assert.True(t, ok, "Price range should be present")
		assert.NotNil(t, priceRange["min"], "Price range min should be present")
		assert.NotNil(t, priceRange["max"], "Price range max should be present")

		// Verify timestamps
		assert.NotEmpty(t, product["createdAt"], "CreatedAt should be present")
		assert.NotEmpty(t, product["updatedAt"], "UpdatedAt should be present")
	})

	// ============================================================================
	// HP-02: Customer Gets Product Successfully
	// ============================================================================
	t.Run("HP-02: Customer Gets Product Successfully", func(t *testing.T) {
		// Login as customer (use credentials from test_data.go)
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)
		client.SetHeader("X-Seller-ID", "") // Clear seller ID header

		w := client.Get(t, "/api/products/1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Extract product data
		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify product details are accessible to customer
		assert.Equal(t, float64(1), product["id"], "Product ID should be 1")
		assert.Equal(t, "iPhone 15 Pro", product["name"], "Product name should match")

		// Verify all nested data is included
		assert.NotNil(t, product["variants"], "Variants should be present")
		assert.NotNil(t, product["options"], "Options should be present")
		assert.NotNil(t, product["category"], "Category should be present")
	})

	// ============================================================================
	// HP-03: Seller Gets Their Own Product
	// ============================================================================
	t.Run("HP-03: Seller Gets Their Own Product", func(t *testing.T) {
		// Login as seller (seller_id 2 owns products 1-4)
		sellerToken := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
		client.SetToken(sellerToken)

		w := client.Get(t, "/api/products/1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Extract product data
		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify product details
		assert.Equal(t, float64(1), product["id"], "Product ID should be 1")
		assert.Equal(
			t,
			float64(2),
			product["sellerId"],
			"Seller ID should match authenticated seller",
		)

		// Verify all variant information is accessible
		variants, ok := product["variants"].([]interface{})
		assert.True(t, ok, "Variants should be present")
		assert.NotEmpty(t, variants, "Variants should not be empty")

		// Verify first variant has complete details
		if len(variants) > 0 {
			variant := variants[0].(map[string]interface{})
			assert.NotNil(t, variant["id"], "Variant ID should be present")
			assert.NotNil(t, variant["sku"], "Variant SKU should be present")
			assert.NotNil(t, variant["price"], "Variant price should be present")
			assert.NotNil(t, variant["allowPurchase"], "Variant allowPurchase should be present")
		}
	})

	// ============================================================================
	// HP-04: Admin Gets Any Product
	// ============================================================================
	t.Run("HP-04: Admin Gets Any Product", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Admin should be able to get product from any seller
		w := client.Get(t, "/api/products/5") // Product 5 belongs to seller 3

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Extract product data
		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify admin can access product from different seller
		assert.Equal(t, float64(5), product["id"], "Product ID should be 5")
		assert.Equal(t, float64(3), product["sellerId"], "Product belongs to seller 3")

		// Verify complete product information is returned
		assert.NotNil(t, product["variants"], "Variants should be present")
		assert.NotNil(t, product["options"], "Options should be present")
	})

	// ============================================================================
	// HP-05: Product with Multiple Variants and Options Retrieved
	// ============================================================================
	t.Run("HP-05: Product with Multiple Variants and Options Retrieved", func(t *testing.T) {
		// Product 1 (iPhone) has 2 options (Color, Storage) and 4 variants
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("") // Clear token

		w := client.Get(t, "/api/products/1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify options
		options, ok := product["options"].([]interface{})
		assert.True(t, ok, "Options should be an array")
		assert.Len(t, options, 2, "iPhone should have 2 options (Color, Storage)")

		// Verify each option has values
		for _, opt := range options {
			option := opt.(map[string]interface{})
			assert.NotNil(t, option["optionId"], "Option should have ID")
			assert.NotNil(t, option["optionName"], "Option should have name")

			values, ok := option["values"].([]interface{})
			assert.True(t, ok, "Option should have values array")
			assert.NotEmpty(t, values, "Option should have at least one value")

			// Verify each value has required fields
			for _, val := range values {
				value := val.(map[string]interface{})
				assert.NotNil(t, value["valueId"], "Value should have ID")
				assert.NotNil(t, value["value"], "Value should have value")
				assert.NotNil(t, value["displayName"], "Value should have display name")
			}
		}

		// Verify variants
		variants, ok := product["variants"].([]interface{})
		assert.True(t, ok, "Variants should be an array")
		assert.Len(t, variants, 4, "iPhone should have 4 variants")

		// Verify each variant has selected options
		for _, v := range variants {
			variant := v.(map[string]interface{})
			assert.NotNil(t, variant["id"], "Variant should have ID")
			assert.NotNil(t, variant["sku"], "Variant should have SKU")
			assert.NotNil(t, variant["price"], "Variant should have price")

			selectedOptions, ok := variant["selectedOptions"].([]interface{})
			assert.True(t, ok, "Variant should have selectedOptions")
			assert.NotEmpty(t, selectedOptions, "Variant should have at least one selected option")
		}

		// Verify price range
		priceRange := product["priceRange"].(map[string]interface{})
		minPrice := priceRange["min"].(float64)
		maxPrice := priceRange["max"].(float64)
		assert.True(t, minPrice <= maxPrice, "Min price should be <= max price")
	})

	// ============================================================================
	// HP-06: Product with Attributes and Package Options
	// ============================================================================
	t.Run("HP-06: Product with Attributes and Package Options", func(t *testing.T) {
		// Product 1 (iPhone) has attributes and package options
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify attributes if present
		if attributes, ok := product["attributes"].([]interface{}); ok && len(attributes) > 0 {
			for _, attr := range attributes {
				attribute := attr.(map[string]interface{})
				assert.NotNil(t, attribute["key"], "Attribute should have key")
				assert.NotNil(t, attribute["name"], "Attribute should have name")
				assert.NotNil(t, attribute["value"], "Attribute should have value")
			}
		}

		// Verify package options if present
		if packageOptions, ok := product["packageOptions"].([]interface{}); ok &&
			len(packageOptions) > 0 {
			for _, pkg := range packageOptions {
				packageOption := pkg.(map[string]interface{})
				assert.NotNil(t, packageOption["id"], "Package option should have ID")
				assert.NotNil(t, packageOption["name"], "Package option should have name")
				assert.NotNil(t, packageOption["price"], "Package option should have price")
				assert.NotNil(t, packageOption["quantity"], "Package option should have quantity")

				// Verify price is positive
				price := packageOption["price"].(float64)
				assert.True(t, price > 0, "Package option price should be positive")
			}
		}
	})

	// ============================================================================
	// HP-07: Product with Category Hierarchy
	// ============================================================================
	t.Run("HP-07: Product with Category Hierarchy", func(t *testing.T) {
		// Product 1 belongs to Smartphones (4) which has parent Electronics (1)
		client.SetHeader("X-Seller-ID", "2")
		client.SetToken("")

		w := client.Get(t, "/api/products/1")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		product := response["data"].(map[string]interface{})["product"].(map[string]interface{})

		// Verify category
		category, ok := product["category"].(map[string]interface{})
		assert.True(t, ok, "Category should be present")
		assert.Equal(t, float64(4), category["id"], "Category ID should be 4 (Smartphones)")
		assert.Equal(t, "Smartphones", category["name"], "Category name should be Smartphones")

		// Verify parent category
		parent, ok := category["parent"].(map[string]interface{})
		assert.True(t, ok, "Parent category should be present")
		assert.Equal(t, float64(1), parent["id"], "Parent category ID should be 1 (Electronics)")
		assert.Equal(t, "Electronics", parent["name"], "Parent category name should be Electronics")

		// Verify categoryId field matches category.id
		assert.Equal(
			t,
			category["id"],
			product["categoryId"],
			"CategoryID should match category.id",
		)
	})
}
