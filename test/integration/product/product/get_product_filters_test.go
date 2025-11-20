package product

import (
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestGetProductFilters tests the GET /api/products/filters endpoint
// Requirements:
// - migrations/seeds/001_seed_user_data.sql (for authentication)
// - migrations/seeds/002_seed_product_data.sql (for products with categories, brands, variants)
func TestGetProductFilters(t *testing.T) {
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
	// HAPPY PATH SCENARIOS
	// ============================================================================

	t.Run("Success - Retrieve all filters with products available", func(t *testing.T) {
		// Seller 2 has products: iPhone, Samsung, MacBook, Sony (4 products)
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify filters object exists
		data := response["data"].(map[string]interface{})
		filters, ok := data["filters"].(map[string]interface{})
		assert.True(t, ok, "Should have filters object")

		// Verify all filter categories are present
		assert.NotNil(t, filters["categories"], "Should have categories")
		assert.NotNil(t, filters["brands"], "Should have brands")
		assert.NotNil(t, filters["attributes"], "Should have attributes")
		assert.NotNil(t, filters["priceRange"], "Should have priceRange")
		assert.NotNil(t, filters["variantTypes"], "Should have variantTypes")
		assert.NotNil(t, filters["stockStatus"], "Should have stockStatus")

		// Verify categories structure
		categories, ok := filters["categories"].([]interface{})
		assert.True(t, ok, "Categories should be an array")
		assert.NotEmpty(t, categories, "Should have at least one category")

		// Verify brands structure
		brands, ok := filters["brands"].([]interface{})
		assert.True(t, ok, "Brands should be an array")
		assert.NotEmpty(t, brands, "Should have at least one brand")

		// Verify attributes structure
		attributes, ok := filters["attributes"].([]interface{})
		assert.True(t, ok, "Attributes should be an array")
		assert.NotNil(t, attributes, "Attributes should not be nil")

		// Verify price range structure
		priceRange, ok := filters["priceRange"].(map[string]interface{})
		assert.True(t, ok, "Price range should be an object")
		assert.NotNil(t, priceRange["min"], "Should have min")
		assert.NotNil(t, priceRange["max"], "Should have max")
		assert.NotNil(t, priceRange["productCount"], "Should have productCount")

		// Verify variant types structure
		variantTypes, ok := filters["variantTypes"].([]interface{})
		assert.True(t, ok, "Variant types should be an array")
		assert.NotNil(t, variantTypes, "Variant types should not be nil")

		// Verify stock status structure
		stockStatus, ok := filters["stockStatus"].(map[string]interface{})
		assert.True(t, ok, "Stock status should be an object")
		assert.NotNil(t, stockStatus["inStock"], "Should have inStock count")
		assert.NotNil(t, stockStatus["outOfStock"], "Should have outOfStock count")
		assert.NotNil(t, stockStatus["totalProducts"], "Should have totalProducts count")
	})

	t.Run("Success - Verify category hierarchy structure", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		categories := filters["categories"].([]interface{})

		// Verify at least one category exists
		assert.NotEmpty(t, categories, "Should have categories")

		// Check first category structure
		firstCategory := categories[0].(map[string]interface{})
		assert.NotNil(t, firstCategory["id"], "Category should have ID")
		assert.NotNil(t, firstCategory["name"], "Category should have name")
		assert.NotNil(t, firstCategory["productCount"], "Category should have product count")

		// Verify product count is positive
		productCount := int(firstCategory["productCount"].(float64))
		assert.Greater(t, productCount, 0, "Product count should be positive")

		// Verify children field exists (should be present, may be empty or nil for leaf categories)
		_, hasChildren := firstCategory["children"]
		assert.True(t, hasChildren, "Category should have children field")

		// If children is not nil and is an array, verify structure
		if children, ok := firstCategory["children"].([]interface{}); ok && len(children) > 0 {
			firstChild := children[0].(map[string]interface{})
			assert.NotNil(t, firstChild["id"], "Child category should have ID")
			assert.NotNil(t, firstChild["name"], "Child category should have name")
			assert.NotNil(t, firstChild["productCount"], "Child category should have product count")
			assert.NotNil(t, firstChild["children"], "Child category should have children array")
		}
	})

	t.Run("Success - Verify brands array structure", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		brands := filters["brands"].([]interface{})

		assert.NotEmpty(t, brands, "Should have brands")

		// Check first brand structure
		firstBrand := brands[0].(map[string]interface{})
		assert.NotNil(t, firstBrand["brand"], "Brand should have name")
		assert.NotNil(t, firstBrand["productCount"], "Brand should have product count")

		// Verify brand name is not empty
		brandName := firstBrand["brand"].(string)
		assert.NotEmpty(t, brandName, "Brand name should not be empty")

		// Verify product count is positive
		productCount := int(firstBrand["productCount"].(float64))
		assert.Greater(t, productCount, 0, "Product count should be positive")

		// Verify seller 2 has expected brands (Apple, Samsung, Sony)
		brandNames := make(map[string]bool)
		for _, brand := range brands {
			b := brand.(map[string]interface{})
			brandNames[b["brand"].(string)] = true
		}

		// Seller 2 should have Apple brand (iPhone, MacBook)
		assert.True(t, brandNames["Apple"] || brandNames["Samsung"] || brandNames["Sony"],
			"Seller 2 should have at least one of: Apple, Samsung, or Sony")
	})

	t.Run("Success - Verify price range structure", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		priceRange := filters["priceRange"].(map[string]interface{})

		// Verify price range fields
		assert.NotNil(t, priceRange["min"], "Should have min")
		assert.NotNil(t, priceRange["max"], "Should have max")
		assert.NotNil(t, priceRange["productCount"], "Should have productCount")

		minPrice := priceRange["min"].(float64)
		maxPrice := priceRange["max"].(float64)

		// Verify min price <= max price
		assert.LessOrEqual(t, minPrice, maxPrice, "Min price should be <= max price")

		// Verify prices are positive
		assert.Greater(t, minPrice, 0.0, "Min price should be positive")
		assert.Greater(t, maxPrice, 0.0, "Max price should be positive")
	})

	t.Run("Success - Verify variant types structure", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		variantTypes, ok := filters["variantTypes"].([]interface{})
		assert.True(t, ok, "Variant types should be an array")

		// If variant types exist, verify structure
		if len(variantTypes) > 0 {
			firstType := variantTypes[0].(map[string]interface{})
			assert.NotNil(t, firstType["name"], "Variant type should have name")
			assert.NotNil(t, firstType["values"], "Variant type should have values array")

			// Verify values is an array
			values, ok := firstType["values"].([]interface{})
			assert.True(t, ok, "Values should be an array")
			assert.NotEmpty(t, values, "Values should not be empty")

			// Verify first value is an object with required fields
			firstValue := values[0].(map[string]interface{})
			assert.NotNil(t, firstValue["value"], "Value should have value field")
			assert.NotNil(t, firstValue["displayName"], "Value should have displayName")
			assert.NotNil(t, firstValue["productCount"], "Value should have productCount")
		}
	})

	t.Run("Success - Verify stock status structure", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		stockStatus := filters["stockStatus"].(map[string]interface{})

		// Verify stock status fields
		assert.NotNil(t, stockStatus["inStock"], "Should have inStock count")
		assert.NotNil(t, stockStatus["outOfStock"], "Should have outOfStock count")
		assert.NotNil(t, stockStatus["totalProducts"], "Should have totalProducts count")

		inStock := int(stockStatus["inStock"].(float64))
		outOfStock := int(stockStatus["outOfStock"].(float64))
		totalProducts := int(stockStatus["totalProducts"].(float64))

		// Verify counts are non-negative
		assert.GreaterOrEqual(t, inStock, 0, "InStock count should be >= 0")
		assert.GreaterOrEqual(t, outOfStock, 0, "Out of stock count should be >= 0")
		assert.Equal(t, inStock+outOfStock, totalProducts, "inStock + outOfStock should equal totalProducts")
	})

	t.Run("Success - Verify attributes structure", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		attributes, ok := filters["attributes"].([]interface{})
		assert.True(t, ok, "Attributes should be an array")

		// If attributes exist, verify structure
		if len(attributes) > 0 {
			firstAttr := attributes[0].(map[string]interface{})
			assert.NotNil(t, firstAttr["key"], "Attribute should have key")
			assert.NotNil(t, firstAttr["name"], "Attribute should have name")
			assert.NotNil(t, firstAttr["productCount"], "Attribute should have productCount")

			// Verify allowedValues exists and is an array (may be empty for some attributes like Brand)
			_, hasAllowedValues := firstAttr["allowedValues"]
			assert.True(t, hasAllowedValues, "Attribute should have allowedValues field")
		}
	})

	// ============================================================================
	// MULTI-TENANT ISOLATION TESTS
	// ============================================================================

	t.Run("Success - Different sellers get different filter results", func(t *testing.T) {
		// Get filters for seller 2 (Electronics - iPhone, Samsung, MacBook, Sony)
		client.SetHeader("X-Seller-ID", "2")
		w2 := client.Get(t, "/api/products/filters")
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusOK)
		filters2 := response2["data"].(map[string]interface{})["filters"].(map[string]interface{})

		// Get filters for seller 3 (Fashion - T-Shirt, Dress, Shoes)
		client.SetHeader("X-Seller-ID", "3")
		w3 := client.Get(t, "/api/products/filters")
		response3 := helpers.AssertSuccessResponse(t, w3, http.StatusOK)
		filters3 := response3["data"].(map[string]interface{})["filters"].(map[string]interface{})

		// Verify both have filters but they are different
		assert.NotNil(t, filters2, "Seller 2 should have filters")
		assert.NotNil(t, filters3, "Seller 3 should have filters")

		// Get brand names for comparison
		brands2 := filters2["brands"].([]interface{})
		brands3 := filters3["brands"].([]interface{})

		// Create brand name sets
		brandNames2 := make(map[string]bool)
		for _, brand := range brands2 {
			b := brand.(map[string]interface{})
			brandNames2[b["brand"].(string)] = true
		}

		brandNames3 := make(map[string]bool)
		for _, brand := range brands3 {
			b := brand.(map[string]interface{})
			brandNames3[b["brand"].(string)] = true
		}

		// Verify sellers have different brands (unless they share brands)
		// Seller 2: Apple, Samsung, Sony
		// Seller 3: Nike, Zara, Adidas
		// These should not overlap
		seller2HasElectronicsBrand := brandNames2["Apple"] || brandNames2["Samsung"] || brandNames2["Sony"]
		seller3HasFashionBrand := brandNames3["Nike"] || brandNames3["Zara"] || brandNames3["Adidas"]

		assert.True(t, seller2HasElectronicsBrand, "Seller 2 should have electronics brands")
		assert.True(t, seller3HasFashionBrand, "Seller 3 should have fashion brands")
	})

	t.Run("Success - Seller 2 filters include only their products", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		brands := filters["brands"].([]interface{})

		// Seller 2 should only have their brands: Apple, Samsung, Sony
		expectedBrands := map[string]bool{"Apple": true, "Samsung": true, "Sony": true}
		for _, brand := range brands {
			b := brand.(map[string]interface{})
			brandName := b["brand"].(string)
			assert.True(t, expectedBrands[brandName],
				"Seller 2 should only have brands: Apple, Samsung, or Sony, got: %s", brandName)
		}
	})

	t.Run("Success - Seller 3 filters include only their products", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "3")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		brands := filters["brands"].([]interface{})

		// Seller 3 should only have their brands: Nike, Zara, Adidas
		expectedBrands := map[string]bool{"Nike": true, "Zara": true, "Adidas": true}
		for _, brand := range brands {
			b := brand.(map[string]interface{})
			brandName := b["brand"].(string)
			assert.True(t, expectedBrands[brandName],
				"Seller 3 should only have brands: Nike, Zara, or Adidas, got: %s", brandName)
		}
	})

	t.Run("Success - Seller 4 filters include only their products", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "4")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		brands := filters["brands"].([]interface{})

		// Seller 4 should only have their brands: IKEA, Casper
		expectedBrands := map[string]bool{"IKEA": true, "Casper": true}
		for _, brand := range brands {
			b := brand.(map[string]interface{})
			brandName := b["brand"].(string)
			assert.True(t, expectedBrands[brandName],
				"Seller 4 should only have brands: IKEA or Casper, got: %s", brandName)
		}
	})

	t.Run("Success - Seller cannot see other sellers' categories", func(t *testing.T) {
		// Seller 3 should see Fashion categories, not Electronics
		client.SetHeader("X-Seller-ID", "3")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		categories := filters["categories"].([]interface{})

		// Check that category names are Fashion-related
		if len(categories) > 0 {
			categoryNames := make(map[string]bool)
			for _, cat := range categories {
				c := cat.(map[string]interface{})
				categoryNames[c["name"].(string)] = true
			}

			// Seller 3 should NOT have Electronics categories
			assert.False(t, categoryNames["Electronics"], "Seller 3 should not see Electronics category")
			assert.False(t, categoryNames["Smartphones"], "Seller 3 should not see Smartphones category")
			assert.False(t, categoryNames["Laptops"], "Seller 3 should not see Laptops category")
		}
	})

	// ============================================================================
	// ERROR HANDLING TESTS
	// ============================================================================

	t.Run("Error - Missing X-Seller-ID header", func(t *testing.T) {
		clientNoHeader := helpers.NewAPIClient(server)
		// Don't set X-Seller-ID header

		w := clientNoHeader.Get(t, "/api/products/filters")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Error - Invalid X-Seller-ID header format", func(t *testing.T) {
		clientInvalidHeader := helpers.NewAPIClient(server)
		clientInvalidHeader.SetHeader("X-Seller-ID", "invalid")

		w := clientInvalidHeader.Get(t, "/api/products/filters")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Error - Zero X-Seller-ID header", func(t *testing.T) {
		clientZeroHeader := helpers.NewAPIClient(server)
		clientZeroHeader.SetHeader("X-Seller-ID", "0")

		w := clientZeroHeader.Get(t, "/api/products/filters")

		helpers.AssertShouldNotSucceed(t, w)
	})

	t.Run("Error - Negative X-Seller-ID header", func(t *testing.T) {
		clientNegativeHeader := helpers.NewAPIClient(server)
		clientNegativeHeader.SetHeader("X-Seller-ID", "-1")

		w := clientNegativeHeader.Get(t, "/api/products/filters")

		helpers.AssertShouldNotSucceed(t, w)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	// Note: Skipping empty filters test since all sellers in seed data have products
	// To test this scenario, you would need to add a new seller with no products to seed data
	t.Run("Success - Seller with few products returns valid filters", func(t *testing.T) {
		// Seller 4 has furniture products (fewer products than seller 2)
		client.SetHeader("X-Seller-ID", "4")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})

		// Verify arrays exist and are properly formatted (even if small)
		categories, ok := filters["categories"].([]interface{})
		assert.True(t, ok, "Categories should be an array")
		assert.NotNil(t, categories, "Categories should not be nil")

		brands, ok := filters["brands"].([]interface{})
		assert.True(t, ok, "Brands should be an array")
		assert.NotNil(t, brands, "Brands should not be nil")

		attributes, ok := filters["attributes"].([]interface{})
		assert.True(t, ok, "Attributes should be an array")
		assert.NotNil(t, attributes, "Attributes should not be nil")
	})

	t.Run("Success - Response structure is valid JSON", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify top-level structure
		assert.NotNil(t, response["success"], "Should have success field")
		assert.NotNil(t, response["data"], "Should have data")

		// Verify success is true
		assert.True(t, response["success"].(bool), "Success should be true")

		// Verify data contains filters
		data := response["data"].(map[string]interface{})
		assert.NotNil(t, data["filters"], "Data should contain filters")
	})

	// ============================================================================
	// DATA VALIDATION TESTS
	// ============================================================================

	t.Run("Validation - Product counts are accurate", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		brands := filters["brands"].([]interface{})

		// Calculate total product count from brands
		totalFromBrands := 0
		for _, brand := range brands {
			b := brand.(map[string]interface{})
			count := int(b["productCount"].(float64))
			totalFromBrands += count
			assert.Greater(t, count, 0, "Each brand should have at least one product")
		}

		// Seller 2 has 4 products (iPhone, Samsung, MacBook, Sony)
		// Total from brands should equal this
		assert.Equal(t, 4, totalFromBrands, "Total brand product count should match seller's total products")
	})

	t.Run("Validation - Price range reflects variant prices", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		priceRange := filters["priceRange"].(map[string]interface{})

		minPrice := priceRange["min"].(float64)
		maxPrice := priceRange["max"].(float64)

		// Verify reasonable price range (based on seed data)
		// Electronics products should have prices > 0
		assert.Greater(t, minPrice, 0.0, "Min price should be positive")
		assert.Greater(t, maxPrice, minPrice, "Max price should be greater than min price")
	})

	t.Run("Validation - No duplicate brands in filter", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		brands := filters["brands"].([]interface{})

		// Check for duplicates
		brandNames := make(map[string]int)
		for _, brand := range brands {
			b := brand.(map[string]interface{})
			brandName := b["brand"].(string)
			brandNames[brandName]++
		}

		// Verify no brand appears more than once
		for brandName, count := range brandNames {
			assert.Equal(t, 1, count, "Brand %s should appear only once, appeared %d times", brandName, count)
		}
	})

	t.Run("Validation - Variant types have unique values", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		variantTypes, ok := filters["variantTypes"].([]interface{})
		assert.True(t, ok, "Variant types should be an array")

		// Check each variant type for unique values
		for _, vt := range variantTypes {
			variantType := vt.(map[string]interface{})
			values := variantType["values"].([]interface{})

			// Check for duplicates by value field
			valueSet := make(map[string]int)
			for _, val := range values {
				valObj := val.(map[string]interface{})
				valStr := valObj["value"].(string)
				valueSet[valStr]++
			}

			// Verify no value appears more than once
			for value, count := range valueSet {
				assert.Equal(t, 1, count, "Variant value %s should appear only once, appeared %d times", value, count)
			}
		}
	})

	// ============================================================================
	// SECURITY TESTS
	// ============================================================================

	t.Run("Security - SQL injection in seller ID prevented", func(t *testing.T) {
		clientSQLInjection := helpers.NewAPIClient(server)
		clientSQLInjection.SetHeader("X-Seller-ID", "1; DROP TABLE product; --")

		w := clientSQLInjection.Get(t, "/api/products/filters")

		// Should return 400 Bad Request, not execute SQL
		assert.Equal(t, http.StatusBadRequest, w.Code, "SQL injection should be rejected")
	})

	t.Run("Security - No data leakage in error messages", func(t *testing.T) {
		clientNoHeader := helpers.NewAPIClient(server)

		w := clientNoHeader.Get(t, "/api/products/filters")

		// Verify error message doesn't expose internal details
		assert.NotEqual(t, http.StatusOK, w.Code, "Should return error")
		
		// Error response should not contain sensitive information
		// like database structure, file paths, or stack traces
		bodyStr := w.Body.String()
		assert.NotContains(t, bodyStr, "SELECT", "Error should not contain SQL queries")
		assert.NotContains(t, bodyStr, "FROM", "Error should not contain SQL keywords")
		assert.NotContains(t, bodyStr, "database", "Error should not contain database references")
		assert.NotContains(t, bodyStr, "/home/", "Error should not contain file paths")
	})

	// ============================================================================
	// PERFORMANCE TESTS (Basic)
	// ============================================================================

	t.Run("Performance - Response contains reasonable amount of data", func(t *testing.T) {
		client.SetHeader("X-Seller-ID", "2")

		w := client.Get(t, "/api/products/filters")

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify response is not excessively large
		bodySize := w.Body.Len()
		assert.Less(t, bodySize, 1024*1024, "Response should be less than 1MB")

		// Verify filters are present
		filters := response["data"].(map[string]interface{})["filters"].(map[string]interface{})
		assert.NotNil(t, filters, "Filters should be present")
	})
}
