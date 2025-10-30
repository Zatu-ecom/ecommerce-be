package category

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/common/constants"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestGetCategoryAttributes(t *testing.T) {
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
	// PUBLIC ACCESS WITH SELLER ID
	// ============================================================================

	t.Run("Get attributes of global category", func(t *testing.T) {
		// Admin creates global category with attributes
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create global category
		categoryReq := map[string]interface{}{
			"name":        "Global Electronics Category",
			"description": "Global electronics",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(
			t,
			categoryW,
			http.StatusCreated,
			)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create attribute definition with correct structure
		attrReq := map[string]interface{}{
			"key":           "brand",
			"name":          "Brand",
			"description":   "Product brand",
			"allowedValues": []string{"Apple", "Samsung", "Sony"},
		}
		attrW := client.Post(t, "/api/attributes", attrReq)
		attrResponse := helpers.AssertSuccessResponse(
			t,
			attrW,
			http.StatusCreated,
			)
		_ = helpers.GetResponseData(t, attrResponse, "attribute") // Just validate structure

		// Create and link attribute to category using /api/attributes/:categoryId endpoint
		// This endpoint creates a category-specific attribute definition
		catAttrReq := map[string]interface{}{
			"key":           "color",
			"name":          "Color",
			"description":   "Product color for this category",
			"allowedValues": []string{"Red", "Blue", "Green"},
		}
		catAttrW := client.Post(t, fmt.Sprintf("/api/attributes/%d", categoryID), catAttrReq)
		helpers.AssertSuccessResponse(t, catAttrW, http.StatusCreated)

		// Get attributes using public API
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			)

		// Verify response structure matches AttributeDefinitionsResponse
		data := response["data"].(map[string]interface{})

		// Safely get attributes array
		var attributes []interface{}
		if attrData, ok := data["attributes"]; ok && attrData != nil {
			attributes = attrData.([]interface{})
		}

		assert.GreaterOrEqual(t, len(attributes), 1, "Should have at least 1 attribute")

		// Verify attribute structure matches AttributeDefinitionResponse
		if len(attributes) > 0 {
			firstAttr := attributes[0].(map[string]interface{})
			assert.NotNil(t, firstAttr["id"], "Should have id")
			assert.NotNil(t, firstAttr["key"], "Should have key")
			assert.NotNil(t, firstAttr["name"], "Should have name")
			assert.NotNil(t, firstAttr["createdAt"], "Should have createdAt")
		}
	})

	t.Run("Get attributes of seller's own category", func(t *testing.T) {
		// Seller creates category with attributes
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create category
		categoryReq := map[string]interface{}{
			"name":        "Seller's Custom Category",
			"description": "Seller specific category",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(
			t,
			categoryW,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create category-specific attribute using correct endpoint and structure
		attrReq := map[string]interface{}{
			"key":         "size",
			"name":        "Size",
			"description": "Product size",
		}
		attrW := client.Post(t, fmt.Sprintf("/api/attributes/%d", categoryID), attrReq)
		helpers.AssertSuccessResponse(
			t,
			attrW,
			http.StatusCreated,
		)

		// Get attributes using public API with same seller
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
		)

		data := response["data"].(map[string]interface{})

		// Safely get attributes array
		var attributes []interface{}
		if attrData, ok := data["attributes"]; ok && attrData != nil {
			attributes = attrData.([]interface{})
		}

		assert.GreaterOrEqual(
			t,
			len(attributes),
			1,
			"Should return seller's own category attributes",
		)
	})

	t.Run("Cannot get attributes of other seller's category", func(t *testing.T) {
		// Seller1 creates category with attributes
		seller1Token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(seller1Token)

		categoryReq := map[string]interface{}{
			"name":        "Seller1 Private Attributes Category",
			"description": "Private category",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(
			t,
			categoryW,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Try to access as Seller2
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "4") // Seller2

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusNotFound,
		)
	})

	t.Run("Get attributes with inheritance - child inherits from parent", func(t *testing.T) {
		// Admin creates parent category with attributes
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create parent category
		parentReq := map[string]interface{}{
			"name":        "Parent Category Inheritance",
			"description": "Parent with attributes",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
		)
		parent := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parent["id"].(float64))

		// Create parent attributes using category-specific endpoint
		attr1Req := map[string]interface{}{
			"key":         "material_inherit",
			"name":        "Material",
			"description": "Product material",
		}
		attr1W := client.Post(t, fmt.Sprintf("/api/attributes/%d", parentID), attr1Req)
		helpers.AssertSuccessResponse(t, attr1W, http.StatusCreated)

		attr2Req := map[string]interface{}{
			"key":         "color_inherit",
			"name":        "Color",
			"description": "Product color",
		}
		attr2W := client.Post(t, fmt.Sprintf("/api/attributes/%d", parentID), attr2Req)
		helpers.AssertSuccessResponse(t, attr2W, http.StatusCreated)

		// Create child category
		childReq := map[string]interface{}{
			"name":        "Child Category Inheritance",
			"description": "Child with own attribute",
			"parentId":    parentID,
		}
		childW := client.Post(t, "/api/categories", childReq)
		childResponse := helpers.AssertSuccessResponse(
			t,
			childW,
			http.StatusCreated,
		)
		child := helpers.GetResponseData(t, childResponse, "category")
		childID := uint(child["id"].(float64))

		// Add child's own attribute
		attr3Req := map[string]interface{}{
			"key":         "weight_inherit",
			"name":        "Weight",
			"description": "Product weight",
			"unit":        "kg",
		}
		attr3W := client.Post(t, fmt.Sprintf("/api/attributes/%d", childID), attr3Req)
		helpers.AssertSuccessResponse(t, attr3W, http.StatusCreated)

		// Get child's attributes with inheritance
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", childID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
		)

		data := response["data"].(map[string]interface{})

		// Safely get attributes array
		var attributes []interface{}
		if attrData, ok := data["attributes"]; ok && attrData != nil {
			attributes = attrData.([]interface{})
		}

		// Should have 3 attributes (2 inherited + 1 own)
		assert.GreaterOrEqual(t, len(attributes), 3, "Should have inherited + own attributes")

		// Verify attribute names are present
		attrNames := make(map[string]bool)
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			attrNames[attrMap["name"].(string)] = true
		}

		assert.True(t, attrNames["Material"], "Should inherit Material from parent")
		assert.True(t, attrNames["Color"], "Should inherit Color from parent")
		assert.True(t, attrNames["Weight"], "Should have own Weight attribute")
	})

	t.Run("Get attributes with multi-level inheritance", func(t *testing.T) {
		// Create Grandparent → Parent → Child hierarchy with attributes
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create grandparent
		grandparentReq := map[string]interface{}{
			"name":        "Grandparent Multi-Level",
			"description": "Grandparent category",
		}
		grandparentW := client.Post(t, "/api/categories", grandparentReq)
		grandparentResponse := helpers.AssertSuccessResponse(
			t,
			grandparentW,
			http.StatusCreated,
		)
		grandparent := helpers.GetResponseData(t, grandparentResponse, "category")
		grandparentID := uint(grandparent["id"].(float64))

		// Add grandparent attribute
		gAttrReq := map[string]interface{}{
			"key":         "origin_multilevel",
			"name":        "Origin",
			"description": "Country of origin",
		}
		gAttrW := client.Post(t, fmt.Sprintf("/api/attributes/%d", grandparentID), gAttrReq)
		helpers.AssertSuccessResponse(t, gAttrW, http.StatusCreated)

		// Create parent
		parentReq := map[string]interface{}{
			"name":        "Parent Multi-Level",
			"description": "Parent category",
			"parentId":    grandparentID,
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
		)
		parent := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parent["id"].(float64))

		// Add parent attribute
		pAttrReq := map[string]interface{}{
			"key":         "warranty_multilevel",
			"name":        "Warranty",
			"description": "Warranty period",
		}
		pAttrW := client.Post(t, fmt.Sprintf("/api/attributes/%d", parentID), pAttrReq)
		helpers.AssertSuccessResponse(t, pAttrW, http.StatusCreated)

		// Create child
		childReq := map[string]interface{}{
			"name":        "Child Multi-Level",
			"description": "Child category",
			"parentId":    parentID,
		}
		childW := client.Post(t, "/api/categories", childReq)
		childResponse := helpers.AssertSuccessResponse(
			t,
			childW,
			http.StatusCreated,
		)
		child := helpers.GetResponseData(t, childResponse, "category")
		childID := uint(child["id"].(float64))

		// Add child attribute
		cAttrReq := map[string]interface{}{
			"key":         "model_multilevel",
			"name":        "Model",
			"description": "Model number",
		}
		cAttrW := client.Post(t, fmt.Sprintf("/api/attributes/%d", childID), cAttrReq)
		helpers.AssertSuccessResponse(t, cAttrW, http.StatusCreated)

		// Get child's attributes - should include all 3 levels
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", childID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
		)

		data := response["data"].(map[string]interface{})

		// Safely get attributes array
		var attributes []interface{}
		if attrData, ok := data["attributes"]; ok && attrData != nil {
			attributes = attrData.([]interface{})
		}

		// Should have 3 attributes from all levels
		assert.GreaterOrEqual(
			t,
			len(attributes),
			3,
			"Should have attributes from all inheritance levels",
		)

		// Verify all attributes are present
		attrNames := make(map[string]bool)
		for _, attr := range attributes {
			attrMap := attr.(map[string]interface{})
			attrNames[attrMap["name"].(string)] = true
		}

		assert.True(t, attrNames["Origin"], "Should inherit from grandparent")
		assert.True(t, attrNames["Warranty"], "Should inherit from parent")
		assert.True(t, attrNames["Model"], "Should have own attribute")
	})

	// ============================================================================
	// PUBLIC ACCESS VALIDATION
	// ============================================================================

	t.Run("Public access without seller ID fails", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "")

		getW := client.Get(t, "/api/categories/1/attributes")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusBadRequest,
		)
	})

	t.Run("Public access with invalid seller ID fails", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "invalid")

		getW := client.Get(t, "/api/categories/1/attributes")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusBadRequest,
		)
	})

	// ============================================================================
	// AUTHENTICATED ACCESS
	// ============================================================================

	t.Run("Seller authenticated can get own category attributes", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create category with attribute
		categoryReq := map[string]interface{}{
			"name":        "Auth Seller Category",
			"description": "Category for auth test",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(
			t,
			categoryW,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Get attributes with authentication
		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
		)

		data := response["data"].(map[string]interface{})
		_, hasAttributes := data["attributes"]
		assert.True(t, hasAttributes, "Should return attributes field")
	})

	t.Run("Admin authenticated can get any category attributes", func(t *testing.T) {
		// Create seller category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		categoryReq := map[string]interface{}{
			"name":        "Seller Category for Admin",
			"description": "Seller category",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(
			t,
			categoryW,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Admin accesses seller's category attributes
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
		)

		data := response["data"].(map[string]interface{})
		_, hasAttributes := data["attributes"]
		assert.True(t, hasAttributes, "Admin should access any category attributes")
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Get attributes of category with no attributes", func(t *testing.T) {
		// Create category without attributes
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		categoryReq := map[string]interface{}{
			"name":        "Empty Attributes Category",
			"description": "No attributes",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(
			t,
			categoryW,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Get attributes
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
		)

		data := response["data"].(map[string]interface{})

		// Safely get attributes array
		var attributes []interface{}
		if attrData, ok := data["attributes"]; ok && attrData != nil {
			attributes = attrData.([]interface{})
		}

		assert.Equal(t, 0, len(attributes), "Should return empty array when no attributes")
	})

	t.Run("Get attributes of non-existent category", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, "/api/categories/99999/attributes")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusNotFound,
		)
	})

	t.Run("Invalid category ID format", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, "/api/categories/invalid/attributes")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusBadRequest,
			)
	})

	t.Run("Verify attribute response fields", func(t *testing.T) {
		// Create category with attribute to verify response structure
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		categoryReq := map[string]interface{}{
			"name":        "Field Validation Category",
			"description": "For response validation",
		}
		categoryW := client.Post(t, "/api/categories", categoryReq)
		categoryResponse := helpers.AssertSuccessResponse(
			t,
			categoryW,
			http.StatusCreated,
		)
		category := helpers.GetResponseData(t, categoryResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Create detailed attribute
		attrReq := map[string]interface{}{
			"key":           "detailed",
			"name":          "DetailedAttribute",
			"description":   "Detailed attribute",
			"allowedValues": []string{"Option1", "Option2"},
		}
		client.Post(t, fmt.Sprintf("/api/attributes/%d", categoryID), attrReq)

		// Get attributes and verify response structure
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, fmt.Sprintf("/api/categories/%d/attributes", categoryID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
		)

		data := response["data"].(map[string]interface{})

		// Safely get attributes array
		var attributes []interface{}
		if attrData, ok := data["attributes"]; ok && attrData != nil {
			attributes = attrData.([]interface{})
		}

		assert.GreaterOrEqual(t, len(attributes), 1, "Should have at least 1 attribute")

		// Verify response fields match AttributeDefinitionResponse
		if len(attributes) > 0 {
			attr := attributes[0].(map[string]interface{})
			assert.NotNil(t, attr["id"], "Should have id field")
			assert.NotNil(t, attr["key"], "Should have key field")
			assert.NotNil(t, attr["name"], "Should have name field")
			assert.NotNil(t, attr["createdAt"], "Should have createdAt field")
		}
	})
}
