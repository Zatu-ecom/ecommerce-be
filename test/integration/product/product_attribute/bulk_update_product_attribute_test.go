package product_attribute

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestBulkUpdateProductAttributes(t *testing.T) {
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
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated, "")
		return helpers.GetResponseData(t, response, "attribute")
	}

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Bulk update multiple attributes successfully", func(t *testing.T) {
		// Login as seller (Jane - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5 // Jane's T-Shirt

		// Create three attributes (avoid seed data: brand=2, material=3, fit=10)
		attr1 := createAttribute(productID, 7, "Intel i5", 1) // processor
		attr2 := createAttribute(productID, 8, "4000mAh", 2)  // battery
		attr3 := createAttribute(productID, 11, "10x20x5", 3) // dimensions

		attr1ID := int(attr1["id"].(float64))
		attr2ID := int(attr2["id"].(float64))
		attr3ID := int(attr3["id"].(float64))

		// Bulk update all three attributes
		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": attr1ID,
					"value":       "Intel i7",
					"sortOrder":   10,
				},
				{
					"attributeId": attr2ID,
					"value":       "5000mAh",
					"sortOrder":   20,
				},
				{
					"attributeId": attr3ID,
					"value":       "15x25x8",
					"sortOrder":   30,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		response := helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusOK,
			"Product attributes bulk updated successfully",
		)

		result := helpers.GetResponseData(t, response, "result")

		// Assert update count
		assert.Equal(t, float64(3), result["updatedCount"])

		// Assert updated attributes
		attributes := result["attributes"].([]interface{})
		assert.Equal(t, 3, len(attributes))

		// Check each attribute was updated
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
				assert.Equal(t, "15x25x8", attrMap["value"])
				assert.Equal(t, float64(30), attrMap["sortOrder"])
			}
		}
	})

	t.Run("Bulk update with only sort order changes", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6 // Jane's Summer Dress

		// Create attributes
		attr1 := createAttribute(productID, 7, "AMD Ryzen", 1) // processor
		attr2 := createAttribute(productID, 8, "3000mAh", 2)   // battery

		attr1ID := int(attr1["id"].(float64))
		attr2ID := int(attr2["id"].(float64))

		// Bulk update - only change sort orders
		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": attr1ID,
					"value":       "AMD Ryzen", // Keep same value
					"sortOrder":   100,         // Change order
				},
				{
					"attributeId": attr2ID,
					"value":       "3000mAh", // Keep same value
					"sortOrder":   200,       // Change order
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK, "")
		result := helpers.GetResponseData(t, response, "result")

		assert.Equal(t, float64(2), result["updatedCount"])

		attributes := result["attributes"].([]interface{})
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			attrID := int(attrMap["id"].(float64))

			if attrID == attr1ID {
				assert.Equal(t, float64(100), attrMap["sortOrder"])
			} else if attrID == attr2ID {
				assert.Equal(t, float64(200), attrMap["sortOrder"])
			}
		}
	})

	t.Run("Bulk update single attribute", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7 // Jane's Running Shoes

		// Create one attribute
		attr := createAttribute(productID, 12, "500kg", 1) // weight_capacity
		attrID := int(attr["id"].(float64))

		// Bulk update single attribute
		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": attrID,
					"value":       "750kg",
					"sortOrder":   5,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK, "")
		result := helpers.GetResponseData(t, response, "result")

		assert.Equal(t, float64(1), result["updatedCount"])

		attributes := result["attributes"].([]interface{})
		assert.Equal(t, 1, len(attributes))

		firstAttr := attributes[0].(map[string]interface{})
		assert.Equal(t, "750kg", firstAttr["value"])
		assert.Equal(t, float64(5), firstAttr["sortOrder"])
	})

	// ============================================================================
	// FAILURE SCENARIOS - AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Bulk update without authentication", func(t *testing.T) {
		// Login as seller to create attributes first
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		attr := createAttribute(productID, 4, "6.5 inches", 0) // screen_size
		attrID := int(attr["id"].(float64))

		// Clear token
		client.SetToken("")

		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": attrID,
					"value":       "7 inches",
					"sortOrder":   1,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized, "")
	})

	t.Run("Seller tries to bulk update other seller's product attributes", func(t *testing.T) {
		// Login as Jane (seller)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5
		attr := createAttribute(productID, 1, "Red", 0) // color
		attrID := int(attr["id"].(float64))

		// Try to bulk update using a product that doesn't belong to Jane
		// Product 1 belongs to John (seller_id=1)
		otherProductID := 1

		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": attrID,
					"value":       "Blue",
					"sortOrder":   1,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", otherProductID)
		w := client.Put(t, url, bulkUpdateBody)

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

	t.Run("Bulk update with invalid product ID", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": 999,
					"value":       "Test",
					"sortOrder":   1,
				},
			},
		}

		// Use non-existent product ID
		url := "/api/products/99999/attributes/bulk"
		w := client.Put(t, url, bulkUpdateBody)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound, "Product not found")
	})

	t.Run("Bulk update with empty attributes array", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("Bulk update with empty value", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6
		attr := createAttribute(productID, 11, "100x50x20", 0) // dimensions
		attrID := int(attr["id"].(float64))

		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": attrID,
					"value":       "", // Empty value
					"sortOrder":   1,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("Bulk update with invalid attribute value (allowed_values)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 7
		// color attribute has allowed_values: Red, Blue, Green, Black, White, Silver, Gold
		attr := createAttribute(productID, 1, "Red", 0)
		attrID := int(attr["id"].(float64))

		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": attrID,
					"value":       "Purple", // Not in allowed_values
					"sortOrder":   1,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest, "Invalid attribute value")
	})

	t.Run("Bulk update with missing required fields", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 5

		// Missing value field
		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": 1,
					"sortOrder":   1,
					// Missing "value"
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Bulk update with non-existent attribute IDs (skipped)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		productID := 6 // Use product 6 to avoid conflicts with product 5

		// Create one valid attribute (avoid previously used: processor=7, battery=8, dimensions=11)
		attr := createAttribute(productID, 12, "500kg", 0) // weight_capacity
		validAttrID := int(attr["id"].(float64))

		// Bulk update with mix of valid and invalid attribute IDs
		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": validAttrID,
					"value":       "750kg",
					"sortOrder":   1,
				},
				{
					"attributeId": 99999, // Non-existent
					"value":       "Test",
					"sortOrder":   2,
				},
			},
		}

		url := fmt.Sprintf("/api/products/%d/attributes/bulk", productID)
		w := client.Put(t, url, bulkUpdateBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK, "")
		result := helpers.GetResponseData(t, response, "result")

		// Only 1 attribute should be updated (the valid one)
		assert.Equal(t, float64(1), result["updatedCount"])

		attributes := result["attributes"].([]interface{})
		assert.Equal(t, 1, len(attributes))
	})

	t.Run("Bulk update attributes from different products (skipped)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create attributes on different products (avoid conflicts with earlier tests)
		// Product 5 already has: processor(7), battery(8), dimensions(11), screen_size(4)
		// Product 6 already has: processor(7), battery(8), dimensions(11)
		// Product 7 already has: weight_capacity(12), color(1), dimensions(11)
		// Use fresh attributes: screen_size(4) on product 6, battery(8) on product 7
		attr6 := createAttribute(6, 4, "5.5 inches", 0) // product 6, screen_size
		attr7 := createAttribute(7, 8, "3500mAh", 0)    // product 7, battery

		attr6ID := int(attr6["id"].(float64))
		attr7ID := int(attr7["id"].(float64))

		// Try to bulk update both using product 6's URL
		bulkUpdateBody := map[string]interface{}{
			"attributes": []map[string]interface{}{
				{
					"attributeId": attr6ID,
					"value":       "6 inches",
					"sortOrder":   1,
				},
				{
					"attributeId": attr7ID, // Belongs to product 7, not 6
					"value":       "4000mAh",
					"sortOrder":   2,
				},
			},
		}

		url := fmt.Sprintf("/api/products/6/attributes/bulk")
		w := client.Put(t, url, bulkUpdateBody)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK, "")
		result := helpers.GetResponseData(t, response, "result")

		// Only 1 attribute should be updated (attr6, not attr7)
		assert.Equal(t, float64(1), result["updatedCount"])
	})
}
