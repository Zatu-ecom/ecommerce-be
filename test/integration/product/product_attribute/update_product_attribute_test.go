package product_attribute

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestUpdateProductAttribute(t *testing.T) {
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

	t.Run("Seller updates attribute value for own product", func(t *testing.T) {
		// Login as seller (Jane - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5      // Jane's T-Shirt
		attributeDefID := 1 // color - has allowed_values

		// Create attribute first (use valid color value)
		attribute := createAttribute(productID, attributeDefID, "Red", 0)
		attributeID := int(attribute["id"].(float64))

		// Update the attribute
		updateBody := map[string]interface{}{
			"value":     "Blue", // Another valid color
			"sortOrder": 1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Put(t, url, updateBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusOK,
		)

		updatedAttribute := helpers.GetResponseData(t, response, "attribute")

		// Assert updated fields
		assert.Equal(t, float64(attributeID), updatedAttribute["id"])
		assert.Equal(t, "Blue", updatedAttribute["value"])
		assert.Equal(t, float64(1), updatedAttribute["sortOrder"])
		// Note: updatedAt might be the same if update happens very fast (within same millisecond)
		assert.NotNil(t, updatedAttribute["updatedAt"])
	})

	t.Run("Seller updates another product attribute", func(t *testing.T) {
		// Login as seller to create attribute (Jane)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6      // Jane's Summer Dress
		attributeDefID := 7 // processor - no allowed_values constraints

		attribute := createAttribute(productID, attributeDefID, "Intel", 0)
		attributeID := int(attribute["id"].(float64))

		// Update the attribute with same seller
		updateBody := map[string]interface{}{
			"value":     "AMD",
			"sortOrder": 2,
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Put(t, url, updateBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusOK,
		)

		updatedAttribute := helpers.GetResponseData(t, response, "attribute")
		assert.Equal(t, "AMD", updatedAttribute["value"])
		assert.Equal(t, float64(2), updatedAttribute["sortOrder"])
	})

	t.Run("Update attribute sort order only", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7      // Jane's Running Shoes
		attributeDefID := 8 // battery - no allowed_values constraints

		attribute := createAttribute(productID, attributeDefID, "5000mAh", 1)
		attributeID := int(attribute["id"].(float64))

		// Update only sort order
		updateBody := map[string]interface{}{
			"value":     "5000mAh", // Keep same value
			"sortOrder": 10,        // Change sort order
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Put(t, url, updateBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedAttribute := helpers.GetResponseData(t, response, "attribute")

		assert.Equal(t, "5000mAh", updatedAttribute["value"])
		assert.Equal(t, float64(10), updatedAttribute["sortOrder"])
	})

	t.Run("Update attribute value only", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5       // Jane's T-Shirt
		attributeDefID := 11 // dimensions - no allowed_values

		attribute := createAttribute(productID, attributeDefID, "10x20x5", 5)
		attributeID := int(attribute["id"].(float64))
		originalSortOrder := attribute["sortOrder"]

		// Update only value
		updateBody := map[string]interface{}{
			"value":     "15x25x8",
			"sortOrder": int(originalSortOrder.(float64)), // Keep same sort order
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Put(t, url, updateBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		updatedAttribute := helpers.GetResponseData(t, response, "attribute")

		assert.Equal(t, "15x25x8", updatedAttribute["value"])
		assert.Equal(t, originalSortOrder, updatedAttribute["sortOrder"])
	})

	// ============================================================================
	// FAILURE SCENARIOS - AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Update attribute without authentication", func(t *testing.T) {
		// First login as seller to create attribute
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6       // Jane's Summer Dress
		attributeDefID := 12 // weight_capacity - no allowed_values

		attribute := createAttribute(productID, attributeDefID, "100kg", 0)
		attributeID := int(attribute["id"].(float64))

		// Clear token
		client.SetToken("")

		updateBody := map[string]interface{}{
			"value":     "Updated",
			"sortOrder": 1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Put(t, url, updateBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Seller tries to update other seller's product attribute", func(t *testing.T) {
		// Login as Jane (seller) to create attribute
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5      // Jane's T-Shirt
		attributeDefID := 4 // screen_size

		attribute := createAttribute(productID, attributeDefID, "6.5 inches", 0)
		attributeID := int(attribute["id"].(float64))

		// Try to update using a product that doesn't belong to Jane
		// Product 1 belongs to John (seller_id=1)
		otherProductID := 1

		updateBody := map[string]interface{}{
			"value":     "Updated",
			"sortOrder": 1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", otherProductID, attributeID)
		w := client.Put(t, url, updateBody)

		// Should return 403 Forbidden or 404 Not Found
		assert.True(
			t,
			w.Code == http.StatusForbidden || w.Code == http.StatusNotFound,
			"Expected 403 or 404, got %d",
			w.Code,
		)
	})

	// ============================================================================
	// FAILURE SCENARIOS - VALIDATION
	// ============================================================================

	t.Run("Update with invalid product ID", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7      // Jane's Running Shoes
		attributeDefID := 1 // color

		attribute := createAttribute(productID, attributeDefID, "Black", 0)
		attributeID := int(attribute["id"].(float64))

		updateBody := map[string]interface{}{
			"value":     "Updated",
			"sortOrder": 1,
		}

		// Use non-existent product ID
		url := fmt.Sprintf("/api/products/99999/attributes/%d", attributeID)
		w := client.Put(t, url, updateBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Update with invalid attribute ID", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // Jane's product

		updateBody := map[string]interface{}{
			"value":     "Updated",
			"sortOrder": 1,
		}

		// Use non-existent attribute ID
		url := fmt.Sprintf("/api/products/%d/attributes/99999", productID)
		w := client.Put(t, url, updateBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Update attribute that doesn't belong to the product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create attribute for product 5
		productID1 := 5                                      // Jane's T-Shirt
		attribute := createAttribute(productID1, 9, "XL", 0) // size
		attributeID := int(attribute["id"].(float64))

		// Try to update it using product 6's URL
		productID2 := 6 // Jane's Summer Dress

		updateBody := map[string]interface{}{
			"value":     "Updated",
			"sortOrder": 1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID2, attributeID)
		w := client.Put(t, url, updateBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Update with empty value", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6      // Jane's Summer Dress
		attributeDefID := 8 // battery

		attribute := createAttribute(productID, attributeDefID, "3000mAh", 0)
		attributeID := int(attribute["id"].(float64))

		// Try to update with empty value
		updateBody := map[string]interface{}{
			"value":     "",
			"sortOrder": 1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Put(t, url, updateBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Update with missing required fields", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7       // Jane's Running Shoes
		attributeDefID := 11 // dimensions

		attribute := createAttribute(productID, attributeDefID, "30x20x10", 0)
		attributeID := int(attribute["id"].(float64))

		// Missing value field
		updateBody := map[string]interface{}{
			"sortOrder": 1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Put(t, url, updateBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Update with invalid attribute ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // Jane's product

		updateBody := map[string]interface{}{
			"value":     "Updated",
			"sortOrder": 1,
		}

		url := fmt.Sprintf("/api/products/%d/attributes/invalid", productID)
		w := client.Put(t, url, updateBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// EDGE CASE - MULTIPLE ATTRIBUTES
	// ============================================================================

	t.Run("Update multiple attributes independently", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // Jane's T-Shirt

		// Create multiple attributes (avoid conflicts with seeds: brand=2, material=3, fit=10)
		// Use attributes without allowed_values constraints
		attr1 := createAttribute(productID, 7, "Intel i5", 1) // processor - no constraints
		attr2 := createAttribute(productID, 8, "4000mAh", 2)  // battery - no constraints
		attr3 := createAttribute(productID, 12, "500kg", 3)   // weight_capacity - no constraints

		attr1ID := int(attr1["id"].(float64))
		attr2ID := int(attr2["id"].(float64))
		attr3ID := int(attr3["id"].(float64))

		// Update first attribute
		updateBody1 := map[string]interface{}{
			"value":     "Intel i7",
			"sortOrder": 10,
		}
		url1 := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attr1ID)
		w1 := client.Put(t, url1, updateBody1)
		helpers.AssertSuccessResponse(t, w1, http.StatusOK)

		// Update second attribute
		updateBody2 := map[string]interface{}{
			"value":     "5000mAh",
			"sortOrder": 20,
		}
		url2 := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attr2ID)
		w2 := client.Put(t, url2, updateBody2)
		helpers.AssertSuccessResponse(t, w2, http.StatusOK)

		// Verify third attribute is unchanged (GET request requires X-Seller-ID header)
		client.SetToken("")
		client.SetHeader("X-Seller-ID", "3") // Jane's seller ID

		getURL := fmt.Sprintf("/api/products/%d/attributes", productID)
		wGet := client.Get(t, getURL)
		getResponse := helpers.AssertSuccessResponse(t, wGet, http.StatusOK)

		productAttributes := helpers.GetResponseData(t, getResponse, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		// Find and verify each attribute
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			attrID := int(attrMap["id"].(float64))

			if attrID == attr1ID {
				assert.Equal(t, "Intel i7", attrMap["value"])
				assert.Equal(t, float64(10), attrMap["sortOrder"])
			} else if attrID == attr2ID {
				assert.Equal(t, "5000mAh", attrMap["value"])
				assert.Equal(t, float64(20), attrMap["sortOrder"])
			} else if attrID == attr3ID {
				assert.Equal(t, "500kg", attrMap["value"])
				assert.Equal(t, float64(3), attrMap["sortOrder"])
			}
		}
	})
}
