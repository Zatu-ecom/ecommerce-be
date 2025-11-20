package product_attribute

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestGetProductAttributes(t *testing.T) {
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

	// Helper function to create an attribute
	createAttribute := func(productID, attributeDefID int, value string, sortOrder int) map[string]interface{} {
		requestBody := map[string]interface{}{
			"attributeDefinitionId": attributeDefID,
			"value":                 value,
			"sortOrder":             sortOrder,
		}

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Post(t, url, requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		return helpers.GetResponseData(t, response, "attribute")
	}

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Get all attributes for product with attributes", func(t *testing.T) {
		// Login as seller to create attributes (Jane - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Create multiple attributes (avoid conflicts with seeds: brand=2, material=3, fit=10)
		createAttribute(productID, 7, "Intel i7", 1)  // processor
		createAttribute(productID, 8, "5000", 2)      // battery
		createAttribute(productID, 11, "100x50x2", 3) // dimensions

		// Get all attributes (public API requires X-Seller-ID header)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3") // Jane's seller ID

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusOK,
		)

		productAttributes := helpers.GetResponseData(t, response, "productAttributes")

		// Assert response structure
		assert.Equal(t, float64(productID), productAttributes["productId"])
		assert.NotNil(t, productAttributes["attributes"])
		assert.NotNil(t, productAttributes["total"])

		attributes := productAttributes["attributes"].([]interface{})
		total := int(productAttributes["total"].(float64))

		assert.Equal(t, len(attributes), total, "Total should match attributes length")
		// Product 5 has 3 from seeds + 3 just created = 6 total
		assert.GreaterOrEqual(t, len(attributes), 6, "Should have at least 6 attributes")
	})

	t.Run("Get attributes for product without attributes", func(t *testing.T) {
		// Use a product that exists but might not have many attributes
		productID := 9 // Product 9 exists from seeds

		client.SetHeader("X-Seller-ID", "3") // Required for public API

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, url)

		// Should succeed with list (might have seed data)
		if w.Code == http.StatusOK {
			response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

			productAttributes := helpers.GetResponseData(t, response, "productAttributes")
			attributes := productAttributes["attributes"].([]interface{})

			// Just verify we can get attributes (might be 0 or have seed data)
			assert.NotNil(t, attributes, "Should return attributes array")
			assert.Equal(t, float64(productID), productAttributes["productId"])
		} else if w.Code == http.StatusNotFound {
			// Product doesn't exist, which is also acceptable
			helpers.AssertErrorResponse(t, w, http.StatusNotFound)
		}
	})

	t.Run("Public access - requires X-Seller-ID header", func(t *testing.T) {
		// Login as seller to create attributes
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6
		createAttribute(productID, 7, "AMD Ryzen", 0) // processor

		// Clear token for public access but add required X-Seller-ID header
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3") // Jane's seller ID

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, url)

		// Should succeed with X-Seller-ID header
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("Verify attributes include definition details", func(t *testing.T) {
		// Login as seller to create attribute
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7
		attributeDefID := 4 // screen_size

		createAttribute(productID, attributeDefID, "6.5 inches", 0)

		// Get attributes with X-Seller-ID header
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		productAttributes := helpers.GetResponseData(t, response, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		assert.Greater(t, len(attributes), 0, "Should have at least one attribute")

		// Check first attribute has definition details
		firstAttr := attributes[0].(map[string]interface{})

		assert.NotNil(t, firstAttr["id"])
		assert.NotNil(t, firstAttr["productId"])
		assert.NotNil(t, firstAttr["attributeDefinitionId"])
		assert.NotNil(t, firstAttr["attributeKey"], "Should include attribute key")
		assert.NotNil(t, firstAttr["attributeName"], "Should include attribute name")
		assert.NotNil(t, firstAttr["value"])
		assert.NotNil(t, firstAttr["sortOrder"])
		assert.NotNil(t, firstAttr["createdAt"])
		assert.NotNil(t, firstAttr["updatedAt"])

		// Unit can be null or string
		_, hasUnit := firstAttr["unit"]
		assert.True(t, hasUnit, "Should have unit field")
	})

	t.Run("Verify attributes are sorted by sort order", func(t *testing.T) {
		// Login as seller to create attributes
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes) to avoid conflicts with other tests on product 6
		productID := 7

		// Create attributes with different sort orders (not in order)
		// Avoid conflicts: product 7 has brand(2), material(3) from seeds, and screen_size(4) from previous test
		// Use attributes without allowed_values constraints
		createAttribute(productID, 8, "Third-5000mAh", 30) // battery - no allowed_values
		createAttribute(productID, 7, "First-Intel", 10)   // processor - no allowed_values
		createAttribute(productID, 12, "Second-500kg", 20) // weight_capacity - no allowed_values

		// Get attributes with X-Seller-ID header
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		productAttributes := helpers.GetResponseData(t, response, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		// Product 7 has 2 from seeds + 3 we just added = 5 total
		assert.GreaterOrEqual(t, len(attributes), 5, "Should have at least 5 attributes")

		// Verify attributes are sorted by sort order
		for i := 0; i < len(attributes)-1; i++ {
			currentAttr := attributes[i].(map[string]interface{})
			nextAttr := attributes[i+1].(map[string]interface{})

			currentOrder := currentAttr["sortOrder"].(float64)
			nextOrder := nextAttr["sortOrder"].(float64)

			assert.LessOrEqual(
				t,
				currentOrder,
				nextOrder,
				"Attributes should be sorted by sort order ascending",
			)
		}

		// Verify the values are in correct order (first non-seed attribute by sort order)
		// Find the first attribute with sortOrder 10
		var firstAttr map[string]interface{}
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			if attrMap["sortOrder"].(float64) == 10 {
				firstAttr = attrMap
				break
			}
		}

		if firstAttr != nil {
			assert.Equal(t, "First-Intel", firstAttr["value"])
			assert.Equal(t, float64(10), firstAttr["sortOrder"])
		}
	})

	// ============================================================================
	// FAILURE SCENARIOS
	// ============================================================================

	t.Run("Get attributes with invalid product ID", func(t *testing.T) {
		// Invalid product ID format
		client.SetHeader("X-Seller-ID", "3")

		url := "/api/products/invalid/attributes"
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Get attributes for non-existent product", func(t *testing.T) {
		// Non-existent product ID
		productID := 99999

		client.SetHeader("X-Seller-ID", "3")

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Get attributes after adding multiple", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5

		// Add 3 attributes (avoiding already used ones)
		createAttribute(productID, 1, "Blue", 1) // color
		createAttribute(productID, 9, "M", 2)    // size
		createAttribute(productID, 12, "200", 3) // weight_capacity

		// Get all attributes with X-Seller-ID header
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		url := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, url)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		productAttributes := helpers.GetResponseData(t, response, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		assert.GreaterOrEqual(t, len(attributes), 3, "Should have at least 3 attributes")
	})

	t.Run("Get attributes after update", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7

		// Create attribute (using one not yet on product 7)
		// fit attribute (10) has allowed values: Slim, Regular, Loose
		attr := createAttribute(productID, 10, "Slim", 0) // fit
		attributeID := int(attr["id"].(float64))

		// Update the attribute to another valid value
		updateBody := map[string]interface{}{
			"value":     "Regular",
			"sortOrder": 5,
		}
		updateURL := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		wUpdate := client.Put(t, updateURL, updateBody)
		helpers.AssertSuccessResponse(t, wUpdate, http.StatusOK)

		// Get attributes with X-Seller-ID header
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		getURL := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, getURL)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		productAttributes := helpers.GetResponseData(t, response, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		// Find the updated attribute
		found := false
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			if int(attrMap["id"].(float64)) == attributeID {
				assert.Equal(t, "Regular", attrMap["value"])
				assert.Equal(t, float64(5), attrMap["sortOrder"])
				found = true
				break
			}
		}

		assert.True(t, found, "Updated attribute should be in the list")
	})

	t.Run("Get attributes after delete", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6

		// Create multiple attributes (avoiding conflicts with what's already there)
		attr1 := createAttribute(productID, 11, "50x30x10", 1) // dimensions
		createAttribute(productID, 12, "150", 2)               // weight_capacity

		attr1ID := int(attr1["id"].(float64))

		// Delete first attribute
		deleteURL := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attr1ID)
		wDelete := client.Delete(t, deleteURL)
		helpers.AssertSuccessResponse(t, wDelete, http.StatusOK)

		// Get attributes with X-Seller-ID header
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3")

		getURL := fmt.Sprintf("/api/products/%d/attributes", productID)
		w := client.Get(t, getURL)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		productAttributes := helpers.GetResponseData(t, response, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		// Verify deleted attribute is not in list
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			attrID := int(attrMap["id"].(float64))
			assert.NotEqual(t, attr1ID, attrID, "Deleted attribute should not be in list")
		}
	})
}
