package product_attribute

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestDeleteProductAttribute(t *testing.T) {
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

	t.Run("Seller deletes attribute from own product", func(t *testing.T) {
		// Login as seller (Jane - seller_id 3, owns products 5, 6, 7)
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5
		// Use color attribute (1) which doesn't exist on product 5
		attributeDefID := 1

		// Create attribute first
		attribute := createAttribute(productID, attributeDefID, "Red", 0)
		attributeID := int(attribute["id"].(float64))

		// Delete the attribute
		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusOK,
		)
	})

	t.Run("Seller deletes another attribute from own product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6
		// Use color attribute (1) which was already created in previous test
		// Let's create a new one with size attribute (9)
		attributeDefID := 9

		attribute := createAttribute(productID, attributeDefID, "L", 0)
		attributeID := int(attribute["id"].(float64))

		// Delete the attribute (still logged in as same seller)
		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Delete(t, url)

		helpers.AssertSuccessResponse(
			t,
			w,
			http.StatusOK,
		)
	})

	t.Run("Delete attribute and verify it's gone", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7
		// Use color attribute (1) which doesn't exist on product 7
		attributeDefID := 1

		// Create attribute
		attribute := createAttribute(productID, attributeDefID, "Black", 0)
		attributeID := int(attribute["id"].(float64))

		// Delete the attribute
		deleteURL := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Delete(t, deleteURL)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Try to delete again - should return 404
		w2 := client.Delete(t, deleteURL)
		helpers.AssertErrorResponse(t, w2, http.StatusNotFound)

		// Verify it's not in the list
		getURL := fmt.Sprintf("/api/products/%d/attributes", productID)
		wGet := client.Get(t, getURL)
		getResponse := helpers.AssertSuccessResponse(t, wGet, http.StatusOK)

		productAttributes := helpers.GetResponseData(t, getResponse, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		// Verify the deleted attribute is not in the list
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			attrID := int(attrMap["id"].(float64))
			assert.NotEqual(t, attributeID, attrID, "Deleted attribute should not be in list")
		}
	})

	// ============================================================================
	// FAILURE SCENARIOS - AUTHENTICATION & AUTHORIZATION
	// ============================================================================

	t.Run("Delete attribute without authentication", func(t *testing.T) {
		// First login as seller to create attribute
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt)
		productID := 5
		// Use size attribute (9) which doesn't exist on product 5
		attributeDefID := 9

		attribute := createAttribute(productID, attributeDefID, "M", 0)
		attributeID := int(attribute["id"].(float64))

		// Clear token
		client.SetToken("")

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("Seller tries to delete other seller's product attribute", func(t *testing.T) {
		// Login as seller (Jane) to create attribute on her product
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's product)
		productID := 6
		// Use size attribute (9) which doesn't exist on product 6
		attributeDefID := 9

		attribute := createAttribute(productID, attributeDefID, "S", 0)
		attributeID := int(attribute["id"].(float64))

		// Try to delete using product 1 which doesn't belong to Jane (belongs to seller_id 2)
		otherProductID := 1

		url := fmt.Sprintf("/api/products/%d/attributes/%d", otherProductID, attributeID)
		w := client.Delete(t, url)

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

	t.Run("Delete with invalid product ID", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 7 (Jane's Running Shoes)
		productID := 7
		// Use size attribute (9) which doesn't exist on product 7
		attributeDefID := 9

		attribute := createAttribute(productID, attributeDefID, "XL", 0)
		attributeID := int(attribute["id"].(float64))

		// Use non-existent product ID
		url := fmt.Sprintf("/api/products/99999/attributes/%d", attributeID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Delete with invalid attribute ID", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		// Use non-existent attribute ID
		url := fmt.Sprintf("/api/products/%d/attributes/99999", productID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Delete attribute that doesn't belong to the product", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create attribute for product 5 (Jane's T-Shirt)
		productID1 := 5
		// Use dimensions attribute (11) which doesn't exist on product 5
		attribute := createAttribute(productID1, 11, "150x80x10", 0)
		attributeID := int(attribute["id"].(float64))

		// Try to delete it using product 6's URL (Jane's Summer Dress)
		productID2 := 6

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID2, attributeID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Delete already deleted attribute", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 6 (Jane's Summer Dress)
		productID := 6
		// Use dimensions attribute (11) which doesn't exist on product 6
		attributeDefID := 11

		attribute := createAttribute(productID, attributeDefID, "100x50x5", 0)
		attributeID := int(attribute["id"].(float64))

		url := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attributeID)

		// Delete first time - should succeed
		w1 := client.Delete(t, url)
		helpers.AssertSuccessResponse(t, w1, http.StatusOK)

		// Try to delete again - should fail
		w2 := client.Delete(t, url)
		helpers.AssertErrorResponse(t, w2, http.StatusNotFound)
	})

	t.Run("Delete with invalid attribute ID format", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's product)
		productID := 5

		url := fmt.Sprintf("/api/products/%d/attributes/invalid", productID)
		w := client.Delete(t, url)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// EDGE CASE - MULTIPLE ATTRIBUTES
	// ============================================================================

	t.Run("Delete one attribute doesn't affect others", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Use product 5 (Jane's T-Shirt) for better isolation
		productID := 5

		// Create multiple attributes that don't exist on product 5 yet
		// Product 5 has: brand(2), material(3), fit(10) from seeds
		// Use electronics attributes which won't conflict with a T-Shirt:
		// screen_size(4), storage(5), ram(6)
		attr1 := createAttribute(
			productID,
			4,
			"6.5",
			1,
		) // screen_size (silly for T-shirt but won't conflict)
		attr2 := createAttribute(productID, 5, "128", 2) // storage
		attr3 := createAttribute(productID, 6, "8", 3)   // ram

		attr1ID := int(attr1["id"].(float64))
		attr2ID := int(attr2["id"].(float64))
		attr3ID := int(attr3["id"].(float64))

		// Delete the second attribute
		deleteURL := fmt.Sprintf("/api/products/%d/attributes/%d", productID, attr2ID)
		w := client.Delete(t, deleteURL)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify other attributes still exist
		getURL := fmt.Sprintf("/api/products/%d/attributes", productID)
		wGet := client.Get(t, getURL)
		getResponse := helpers.AssertSuccessResponse(t, wGet, http.StatusOK)

		productAttributes := helpers.GetResponseData(t, getResponse, "productAttributes")
		attributes := productAttributes["attributes"].([]interface{})

		// Product 5 has 3 from seeds (brand, material, fit) + 2 remaining (attr1, attr3) = 5 total
		assert.GreaterOrEqual(
			t,
			len(attributes),
			5,
			"Should have at least 5 attributes (3 from seeds + 2 remaining)",
		)

		// Verify attr1 and attr3 are still there, attr2 is gone
		foundAttr1 := false
		foundAttr2 := false
		foundAttr3 := false

		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			attrID := int(attrMap["id"].(float64))

			if attrID == attr1ID {
				foundAttr1 = true
				assert.Equal(t, "6.5", attrMap["value"])
			} else if attrID == attr2ID {
				foundAttr2 = true
			} else if attrID == attr3ID {
				foundAttr3 = true
				assert.Equal(t, "8", attrMap["value"])
			}
		}

		assert.True(t, foundAttr1, "Attribute 1 should still exist")
		assert.False(t, foundAttr2, "Attribute 2 should be deleted")
		assert.True(t, foundAttr3, "Attribute 3 should still exist")
	})
}
