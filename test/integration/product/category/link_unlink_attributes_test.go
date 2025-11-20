package category

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestLinkUnlinkAttributes(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunSeeds(t, "migrations/seeds/001_seed_user_data.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// LINK ATTRIBUTE TESTS - Basic Functionality (P0)
	// ============================================================================

	t.Run("Admin links attribute to global category", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create global category
		categoryReq := map[string]interface{}{
			"name":        "Electronics Link Test",
			"description": "Electronics category for linking",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create attribute
		attrReq := map[string]interface{}{
			"key":           "brand_link",
			"name":          "Brand",
			"description":   "Product brand",
			"allowedValues": []string{"Apple", "Samsung", "Sony"},
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		// Link attribute to category
		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		linkW := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		linkResponse := helpers.AssertSuccessResponse(t, linkW, http.StatusCreated)

		// Verify response structure
		data := linkResponse["data"].(map[string]interface{})
		assert.NotNil(t, data, "Response data should not be nil")
		assert.Equal(
			t,
			float64(categoryID),
			data["categoryId"].(float64),
			"Should return correct category ID",
		)
		assert.Equal(
			t,
			float64(attributeID),
			data["attributeDefinitionId"].(float64),
			"Should return correct attribute ID",
		)
		assert.NotNil(t, data["createdAt"], "Should have createdAt timestamp")
		assert.Contains(t, data["createdAt"].(string), "T", "createdAt should be in ISO8601 format")

		// Verify the link by getting category attributes
		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		getResponse := helpers.AssertSuccessResponse(t, getW, http.StatusOK)
		getData := getResponse["data"].(map[string]interface{})
		attributes := getData["attributes"].([]interface{})
		assert.GreaterOrEqual(t, len(attributes), 1, "Should have at least 1 linked attribute")

		// Find our linked attribute
		found := false
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			if uint(attrMap["id"].(float64)) == attributeID {
				found = true
				assert.Equal(t, "brand_link", attrMap["key"], "Should have correct attribute key")
				break
			}
		}
		assert.True(t, found, "Linked attribute should be present in category attributes")
	})

	t.Run("Seller links attribute to own category", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create seller category
		categoryReq := map[string]interface{}{
			"name":        "Seller Products Link",
			"description": "Seller category for linking",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create attribute (seller can create attributes)
		attrReq := map[string]interface{}{
			"key":         "seller_color_link",
			"name":        "Color",
			"description": "Product color",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		// Link attribute to category
		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		linkW := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		linkResponse := helpers.AssertSuccessResponse(t, linkW, http.StatusCreated)

		// Verify response
		data := linkResponse["data"].(map[string]interface{})
		assert.Equal(t, float64(categoryID), data["categoryId"].(float64))
		assert.Equal(t, float64(attributeID), data["attributeDefinitionId"].(float64))
	})

	t.Run("Link multiple attributes to same category", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Multi Attribute Category",
			"description": "Category with multiple attributes",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create and link first attribute
		attr1Req := map[string]interface{}{
			"key":         "size_multi",
			"name":        "Size",
			"description": "Product size",
		}
		attr1W := client.Post(t, "/api/attributes", attr1Req)
		attr1Response := helpers.AssertSuccessResponse(t, attr1W, http.StatusCreated)
		attr1 := helpers.GetResponseData(t, attr1Response, "attribute")
		attr1ID := uint(attr1["id"].(float64))

		link1Req := map[string]interface{}{
			"attributeDefinitionId": attr1ID,
		}
		link1W := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), link1Req)
		helpers.AssertSuccessResponse(t, link1W, http.StatusCreated)

		// Create and link second attribute
		attr2Req := map[string]interface{}{
			"key":         "weight_multi",
			"name":        "Weight",
			"description": "Product weight",
		}
		attr2W := client.Post(t, "/api/attributes", attr2Req)
		attr2Response := helpers.AssertSuccessResponse(t, attr2W, http.StatusCreated)
		attr2 := helpers.GetResponseData(t, attr2Response, "attribute")
		attr2ID := uint(attr2["id"].(float64))

		link2Req := map[string]interface{}{
			"attributeDefinitionId": attr2ID,
		}
		link2W := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), link2Req)
		helpers.AssertSuccessResponse(t, link2W, http.StatusCreated)

		// Verify both attributes are linked
		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		getResponse := helpers.AssertSuccessResponse(t, getW, http.StatusOK)
		getData := getResponse["data"].(map[string]interface{})
		attributes := getData["attributes"].([]interface{})
		assert.GreaterOrEqual(t, len(attributes), 2, "Should have at least 2 linked attributes")
	})

	// ============================================================================
	// LINK ATTRIBUTE VALIDATION TESTS (P0)
	// ============================================================================

	t.Run("Link with invalid category ID", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		linkReq := map[string]interface{}{
			"attributeDefinitionId": 1,
		}
		linkW := client.Post(t, "/api/categories/99999/attributes", linkReq)
		errorResponse := helpers.AssertErrorResponse(t, linkW, http.StatusNotFound)

		// Assert error code
		assert.Equal(
			t,
			"CATEGORY_NOT_FOUND",
			errorResponse["code"],
			"Should return CATEGORY_NOT_FOUND error code",
		)
	})

	t.Run("Link with invalid attribute ID", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Invalid Attr Link Test",
			"description": "Testing invalid attribute",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Try to link non-existent attribute
		linkReq := map[string]interface{}{
			"attributeDefinitionId": 99999,
		}
		linkW := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		errorResponse := helpers.AssertErrorResponse(t, linkW, http.StatusNotFound)

		// Assert error code
		assert.Equal(
			t,
			"ATTRIBUTE_DEFINITION_NOT_FOUND",
			errorResponse["code"],
			"Should return ATTRIBUTE_DEFINITION_NOT_FOUND error code",
		)
	})

	t.Run("Link with missing attributeDefinitionId", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Missing Attr ID Test",
			"description": "Testing missing attribute ID",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Try to link without attributeDefinitionId
		linkReq := map[string]interface{}{}
		linkW := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		helpers.AssertErrorResponse(t, linkW, http.StatusBadRequest)
	})

	t.Run("Link duplicate attribute to same category", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Duplicate Link Test",
			"description": "Testing duplicate link",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create attribute
		attrReq := map[string]interface{}{
			"key":         "duplicate_link_test",
			"name":        "Duplicate Test",
			"description": "For duplicate link testing",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		// Link first time - should succeed
		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		link1W := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		helpers.AssertSuccessResponse(t, link1W, http.StatusCreated)

		// Link second time - should fail
		link2W := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		errorResponse := helpers.AssertErrorResponse(t, link2W, http.StatusConflict)

		// Assert error code
		assert.Equal(
			t,
			"ATTRIBUTE_ALREADY_LINKED",
			errorResponse["code"],
			"Should return ATTRIBUTE_ALREADY_LINKED error code",
		)
	})

	// ============================================================================
	// LINK ATTRIBUTE AUTHORIZATION TESTS (P0)
	// ============================================================================

	t.Run("Unauthorized user cannot link attribute", func(t *testing.T) {
		// No token
		client.SetToken("")

		linkReq := map[string]interface{}{
			"attributeDefinitionId": 1,
		}
		linkW := client.Post(t, "/api/categories/1/attributes", linkReq)
		helpers.AssertErrorResponse(t, linkW, http.StatusUnauthorized)
	})

	t.Run("Customer cannot link attribute", func(t *testing.T) {
		// Login as customer
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		linkReq := map[string]interface{}{
			"attributeDefinitionId": 1,
		}
		linkW := client.Post(t, "/api/categories/1/attributes", linkReq)
		helpers.AssertErrorResponse(t, linkW, http.StatusForbidden)
	})

	t.Run("Seller cannot link attribute to global category", func(t *testing.T) {
		// Admin creates global category
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		categoryReq := map[string]interface{}{
			"name":        "Global for Seller Link Test",
			"description": "Global category",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create attribute
		attrReq := map[string]interface{}{
			"key":         "seller_unauthorized_link",
			"name":        "Unauthorized",
			"description": "Seller shouldn't link this",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		// Seller tries to link to global category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		linkW := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		errorResponse := helpers.AssertErrorResponse(t, linkW, http.StatusForbidden)

		// Assert error code
		assert.Equal(
			t,
			"UNAUTHORIZED_CATEGORY_UPDATE",
			errorResponse["code"],
			"Should return UNAUTHORIZED_CATEGORY_UPDATE error code",
		)
	})

	t.Run("Seller cannot link attribute to other seller's category", func(t *testing.T) {
		// Seller 1 creates category
		seller1Token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(seller1Token)

		categoryReq := map[string]interface{}{
			"name":        "Seller1 Category Link",
			"description": "Seller 1's category",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create attribute
		attrReq := map[string]interface{}{
			"key":         "cross_seller_link",
			"name":        "Cross Seller",
			"description": "Cross seller test",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		// Seller 2 tries to link to Seller 1's category
		seller2Token := helpers.Login(t, client, "bob.store@example.com", "seller123")
		client.SetToken(seller2Token)

		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		linkW := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		errorResponse := helpers.AssertErrorResponse(t, linkW, http.StatusForbidden)

		// Assert error code
		assert.Equal(
			t,
			"UNAUTHORIZED_CATEGORY_UPDATE",
			errorResponse["code"],
			"Should return UNAUTHORIZED_CATEGORY_UPDATE error code",
		)
	})

	// ============================================================================
	// UNLINK ATTRIBUTE TESTS - Basic Functionality (P0)
	// ============================================================================

	t.Run("Admin unlinks attribute from global category", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Unlink Test Category",
			"description": "Category for unlink testing",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create attribute
		attrReq := map[string]interface{}{
			"key":         "unlink_test_attr",
			"name":        "Unlink Test",
			"description": "For unlink testing",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		// Link attribute
		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		linkW := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		helpers.AssertSuccessResponse(t, linkW, http.StatusCreated)

		// Verify link exists
		getW1 := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		getResponse1 := helpers.AssertSuccessResponse(t, getW1, http.StatusOK)
		getData1 := getResponse1["data"].(map[string]interface{})
		attributes1 := getData1["attributes"].([]interface{})
		assert.GreaterOrEqual(t, len(attributes1), 1, "Should have linked attribute")

		// Unlink attribute
		unlinkW := client.Delete(
			t,
			fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID),
		)
		helpers.AssertSuccessResponse(t, unlinkW, http.StatusOK)

		// Verify link no longer exists
		getW2 := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		getResponse2 := helpers.AssertSuccessResponse(t, getW2, http.StatusOK)
		getData2 := getResponse2["data"].(map[string]interface{})

		// Check that our attribute is not in the list
		found := false
		if getData2["attributes"] != nil {
			attributes2 := getData2["attributes"].([]interface{})
			for _, attr := range attributes2 {
				attrMap := attr.(map[string]interface{})
				if uint(attrMap["id"].(float64)) == attributeID {
					found = true
					break
				}
			}
		}
		assert.False(t, found, "Unlinked attribute should not be present")
	})

	t.Run("Seller unlinks attribute from own category", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Seller Unlink Category",
			"description": "Seller category for unlink",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create and link attribute
		attrReq := map[string]interface{}{
			"key":         "seller_unlink_attr",
			"name":        "Seller Unlink",
			"description": "For seller unlink testing",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)

		// Unlink attribute
		unlinkW := client.Delete(
			t,
			fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID),
		)
		helpers.AssertSuccessResponse(t, unlinkW, http.StatusOK)
	})

	// ============================================================================
	// UNLINK ATTRIBUTE VALIDATION TESTS (P0)
	// ============================================================================

	t.Run("Unlink with invalid category ID", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		unlinkW := client.Delete(t, "/api/categories/99999/attributes/1")
		errorResponse := helpers.AssertErrorResponse(t, unlinkW, http.StatusNotFound)

		// Assert error code
		assert.Equal(
			t,
			"CATEGORY_NOT_FOUND",
			errorResponse["code"],
			"Should return CATEGORY_NOT_FOUND error code",
		)
	})

	t.Run("Unlink non-existent link", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Unlink Non-existent Test",
			"description": "Testing unlink of non-existent link",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Try to unlink non-existent attribute (attribute ID that doesn't exist)
		unlinkW := client.Delete(t, fmt.Sprintf("/api/categories/%d/attributes/99999", categoryID))
		errorResponse := helpers.AssertErrorResponse(t, unlinkW, http.StatusNotFound)

		// Assert error code
		assert.Equal(
			t,
			"ATTRIBUTE_NOT_LINKED",
			errorResponse["code"],
			"Should return ATTRIBUTE_NOT_LINKED error code",
		)
	})

	t.Run("Unlink already unlinked attribute", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Double Unlink Test",
			"description": "Testing double unlink",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create and link attribute
		attrReq := map[string]interface{}{
			"key":         "double_unlink_attr",
			"name":        "Double Unlink",
			"description": "For double unlink testing",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)

		// Unlink first time - should succeed
		unlink1W := client.Delete(
			t,
			fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID),
		)
		helpers.AssertSuccessResponse(t, unlink1W, http.StatusOK)

		// Unlink second time - should fail
		unlink2W := client.Delete(
			t,
			fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID),
		)
		errorResponse := helpers.AssertErrorResponse(t, unlink2W, http.StatusNotFound)

		// Assert error code
		assert.Equal(
			t,
			"ATTRIBUTE_NOT_LINKED",
			errorResponse["code"],
			"Should return ATTRIBUTE_NOT_LINKED error code",
		)
	})

	// ============================================================================
	// UNLINK ATTRIBUTE AUTHORIZATION TESTS (P0)
	// ============================================================================

	t.Run("Unauthorized user cannot unlink attribute", func(t *testing.T) {
		client.SetToken("")

		unlinkW := client.Delete(t, "/api/categories/1/attributes/1")
		helpers.AssertErrorResponse(t, unlinkW, http.StatusUnauthorized)
	})

	t.Run("Customer cannot unlink attribute", func(t *testing.T) {
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)

		unlinkW := client.Delete(t, "/api/categories/1/attributes/1")
		helpers.AssertErrorResponse(t, unlinkW, http.StatusForbidden)
	})

	t.Run("Seller cannot unlink from global category", func(t *testing.T) {
		// Admin creates and links attribute to global category
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		categoryReq := map[string]interface{}{
			"name":        "Global Seller Unlink Test",
			"description": "Global category for seller unlink test",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		attrReq := map[string]interface{}{
			"key":         "global_unlink_test",
			"name":        "Global Unlink",
			"description": "For global unlink test",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)

		// Seller tries to unlink
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		unlinkW := client.Delete(
			t,
			fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID),
		)
		errorResponse := helpers.AssertErrorResponse(t, unlinkW, http.StatusForbidden)

		// Assert error code
		assert.Equal(
			t,
			"UNAUTHORIZED_CATEGORY_UPDATE",
			errorResponse["code"],
			"Should return UNAUTHORIZED_CATEGORY_UPDATE error code",
		)
	})

	t.Run("Seller cannot unlink from other seller's category", func(t *testing.T) {
		// Seller 1 creates and links
		seller1Token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(seller1Token)

		categoryReq := map[string]interface{}{
			"name":        "Seller1 Unlink Protection",
			"description": "Seller 1's category",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		attrReq := map[string]interface{}{
			"key":         "cross_seller_unlink",
			"name":        "Cross Seller Unlink",
			"description": "Cross seller unlink test",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)

		// Seller 2 tries to unlink
		seller2Token := helpers.Login(t, client, "bob.store@example.com", "seller123")
		client.SetToken(seller2Token)

		unlinkW := client.Delete(
			t,
			fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID),
		)
		errorResponse := helpers.AssertErrorResponse(t, unlinkW, http.StatusForbidden)

		// Assert error code
		assert.Equal(
			t,
			"UNAUTHORIZED_CATEGORY_UPDATE",
			errorResponse["code"],
			"Should return UNAUTHORIZED_CATEGORY_UPDATE error code",
		)
	})

	// ============================================================================
	// EDGE CASES & DATA INTEGRITY (P1)
	// ============================================================================

	t.Run("Link and unlink same attribute multiple times", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Toggle Link Test",
			"description": "Testing multiple link/unlink cycles",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create attribute
		attrReq := map[string]interface{}{
			"key":         "toggle_attr",
			"name":        "Toggle",
			"description": "For toggle testing",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}

		// Cycle 1: Link -> Unlink
		client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		client.Delete(t, fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID))

		// Cycle 2: Link -> Unlink
		client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		client.Delete(t, fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID))

		// Cycle 3: Link (leave linked)
		link3W := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		helpers.AssertSuccessResponse(t, link3W, http.StatusCreated)

		// Verify final state
		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		getResponse := helpers.AssertSuccessResponse(t, getW, http.StatusOK)
		getData := getResponse["data"].(map[string]interface{})
		attributes := getData["attributes"].([]interface{})

		found := false
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			if uint(attrMap["id"].(float64)) == attributeID {
				found = true
				break
			}
		}
		assert.True(t, found, "Attribute should be linked after final link operation")
	})

	t.Run("Admin can manage links for any category", func(t *testing.T) {
		// Seller creates category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		categoryReq := map[string]interface{}{
			"name":        "Seller Category Admin Management",
			"description": "Seller category that admin will manage",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(t, categoryW, http.StatusCreated)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Admin links attribute to seller's category
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		attrReq := map[string]interface{}{
			"key":         "admin_manage_attr",
			"name":        "Admin Manage",
			"description": "Admin managing seller category",
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(t, attrW, http.StatusCreated)
		attribute := helpers.GetResponseData(t, attrResponse, "attribute")
		attributeID := uint(attribute["id"].(float64))

		linkReq := map[string]interface{}{
			"attributeDefinitionId": attributeID,
		}
		linkW := client.Post(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID), linkReq)
		helpers.AssertSuccessResponse(t, linkW, http.StatusCreated)

		// Admin unlinks from seller's category
		unlinkW := client.Delete(
			t,
			fmt.Sprintf("/api/categories/%d/attributes/%d", categoryID, attributeID),
		)
		helpers.AssertSuccessResponse(t, unlinkW, http.StatusOK)
	})
}
