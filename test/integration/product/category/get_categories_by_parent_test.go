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

func TestGetCategoriesByParent(t *testing.T) {
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

	// Helper function to safely get categories from response
	getCategoriesFromResponse := func(response map[string]interface{}) []interface{} {
		data := response["data"].(map[string]interface{})
		if categories, ok := data["categories"]; ok && categories != nil {
			return categories.([]interface{})
		}
		return []interface{}{}
	}

	// ============================================================================
	// PUBLIC ACCESS WITH SELLER ID
	// ============================================================================

	t.Run(
		"Get root categories (no parentId) returns global + seller's categories",
		func(t *testing.T) {
			// Login as seller and admin to create test data
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
			client.SetToken(adminToken)

			// Create global root category
			globalReq := map[string]interface{}{
				"name":        "Global Root Category",
				"description": "Global root category",
			}
			client.Post(t, "/api/categories", globalReq)

			// Create seller-specific root category
			client.SetToken(sellerToken)
			sellerReq := map[string]interface{}{
				"name":        "Seller Root Category",
				"description": "Seller's root category",
			}
			client.Post(t, "/api/categories", sellerReq)

			// Get root categories using public API with seller ID
			client.SetToken("")
			client.SetHeader(constants.SELLER_ID_HEADER, "3") // Seller ID = 3

			getW := client.Get(t, "/api/categories/by-parent")
			response := helpers.AssertSuccessResponse(
				t,
				getW,
				http.StatusOK,
				"",
			)

			// Verify response contains categories
			categories := getCategoriesFromResponse(response)

			assert.GreaterOrEqual(t, len(categories), 2, "Should have at least 2 root categories")

			// Verify both global and seller categories are present
			categoryNames := make(map[string]bool)
			for _, cat := range categories {
				catMap := cat.(map[string]interface{})
				categoryNames[catMap["name"].(string)] = true
			}

			assert.True(t, categoryNames["Global Root Category"], "Should include global category")
			assert.True(
				t,
				categoryNames["Seller Root Category"],
				"Should include seller's category",
			)
		},
	)

	t.Run("Get children of specific parent category", func(t *testing.T) {
		// Login as seller to create test data
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create parent category
		parentReq := map[string]interface{}{
			"name":        "Parent Category for Children Test",
			"description": "Parent category",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			"",
		)
		parent := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parent["id"].(float64))

		// Create child categories
		child1Req := map[string]interface{}{
			"name":        "Child Category 1",
			"description": "First child",
			"parentId":    parentID,
		}
		client.Post(t, "/api/categories", child1Req)

		child2Req := map[string]interface{}{
			"name":        "Child Category 2",
			"description": "Second child",
			"parentId":    parentID,
		}
		client.Post(t, "/api/categories", child2Req)

		// Get children using public API
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, fmt.Sprintf("/api/categories/by-parent?parentId=%d", parentID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		// Verify response contains both children
		categories := getCategoriesFromResponse(response)

		assert.Equal(t, 2, len(categories), "Should have exactly 2 children")

		// Verify child names
		childNames := make(map[string]bool)
		for _, cat := range categories {
			catMap := cat.(map[string]interface{})
			childNames[catMap["name"].(string)] = true
		}

		assert.True(t, childNames["Child Category 1"], "Should include first child")
		assert.True(t, childNames["Child Category 2"], "Should include second child")
	})

	t.Run(
		"Multi-tenant filtering - cannot get other seller's category children",
		func(t *testing.T) {
			// Seller1 creates a parent category
			seller1Token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(seller1Token)

			parentReq := map[string]interface{}{
				"name":        "Seller1 Private Parent",
				"description": "Seller1's private parent",
			}
			parentW := client.Post(t, "/api/categories", parentReq)
			parentResponse := helpers.AssertSuccessResponse(
				t,
				parentW,
				http.StatusCreated,
				"",
			)
			parent := helpers.GetResponseData(t, parentResponse, "category")
			parentID := uint(parent["id"].(float64))

			// Create child
			childReq := map[string]interface{}{
				"name":        "Seller1 Private Child",
				"description": "Seller1's private child",
				"parentId":    parentID,
			}
			client.Post(t, "/api/categories", childReq)

			// Try to access as Seller2 (different seller)
			client.SetToken("")
			client.SetHeader(constants.SELLER_ID_HEADER, "4") // Seller2's ID

			getW := client.Get(t, fmt.Sprintf("/api/categories/by-parent?parentId=%d", parentID))
			response := helpers.AssertSuccessResponse(
				t,
				getW,
				http.StatusOK,
				"",
			)

			// Should return empty array (no access to other seller's categories)
			categories := getCategoriesFromResponse(response)

			assert.Equal(t, 0, len(categories), "Should not return other seller's children")
		},
	)

	t.Run("Get children of global category - accessible to all sellers", func(t *testing.T) {
		// Admin creates a global parent category
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		parentReq := map[string]interface{}{
			"name":        "Global Parent Category",
			"description": "Global parent accessible to all",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			"",
		)
		parent := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parent["id"].(float64))

		// Create child categories
		child1Req := map[string]interface{}{
			"name":        "Global Child 1",
			"description": "First global child",
			"parentId":    parentID,
		}
		client.Post(t, "/api/categories", child1Req)

		child2Req := map[string]interface{}{
			"name":        "Global Child 2",
			"description": "Second global child",
			"parentId":    parentID,
		}
		client.Post(t, "/api/categories", child2Req)

		// Access as any seller
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3") // Any seller

		getW := client.Get(t, fmt.Sprintf("/api/categories/by-parent?parentId=%d", parentID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		// Should return global children
		categories := getCategoriesFromResponse(response)

		assert.Equal(t, 2, len(categories), "Should return global children")
	})

	// ============================================================================
	// PUBLIC ACCESS VALIDATION
	// ============================================================================

	t.Run("Public access without seller ID fails", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "")

		getW := client.Get(t, "/api/categories/by-parent")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusBadRequest,
			"",
		)
	})

	t.Run("Public access with invalid seller ID fails", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "invalid")

		getW := client.Get(t, "/api/categories/by-parent")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusBadRequest,
			"",
		)
	})

	// ============================================================================
	// AUTHENTICATED ACCESS
	// ============================================================================

	t.Run("Seller authenticated gets global + own children", func(t *testing.T) {
		// Create test data
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		parentReq := map[string]interface{}{
			"name":        "Auth Test Parent",
			"description": "Parent for auth test",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			"",
		)
		parent := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parent["id"].(float64))

		childReq := map[string]interface{}{
			"name":        "Auth Test Child",
			"description": "Child for auth test",
			"parentId":    parentID,
		}
		client.Post(t, "/api/categories", childReq)

		// Get children with authentication
		getW := client.Get(t, fmt.Sprintf("/api/categories/by-parent?parentId=%d", parentID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		categories := getCategoriesFromResponse(response)

		assert.GreaterOrEqual(t, len(categories), 1, "Should return at least 1 child")
	})

	t.Run("Admin authenticated gets all children", func(t *testing.T) {
		// Admin can see all categories including seller-specific
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create a seller-specific parent
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		parentReq := map[string]interface{}{
			"name":        "Seller Specific for Admin Test",
			"description": "Seller category",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			"",
		)
		parent := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parent["id"].(float64))

		childReq := map[string]interface{}{
			"name":        "Seller Child for Admin Test",
			"description": "Seller child",
			"parentId":    parentID,
		}
		client.Post(t, "/api/categories", childReq)

		// Admin accesses seller's category children
		client.SetToken(adminToken)
		getW := client.Get(t, fmt.Sprintf("/api/categories/by-parent?parentId=%d", parentID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		categories := getCategoriesFromResponse(response)

		assert.GreaterOrEqual(t, len(categories), 1, "Admin should see seller's children")
	})

	// ============================================================================
	// EDGE CASES
	// ============================================================================

	t.Run("Get children of non-existent parent returns empty array", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, "/api/categories/by-parent?parentId=99999")
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		categories := getCategoriesFromResponse(response)

		assert.Equal(t, 0, len(categories), "Should return empty array for non-existent parent")
	})

	t.Run("Get children when parent has no children", func(t *testing.T) {
		// Create parent with no children
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		parentReq := map[string]interface{}{
			"name":        "Childless Parent",
			"description": "Parent with no children",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			"",
		)
		parent := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parent["id"].(float64))

		// Get children
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, fmt.Sprintf("/api/categories/by-parent?parentId=%d", parentID))
		response := helpers.AssertSuccessResponse(
			t,
			getW,
			http.StatusOK,
			"",
		)

		categories := getCategoriesFromResponse(response)

		assert.Equal(t, 0, len(categories), "Should return empty array when no children")
	})

	t.Run("Invalid parentId format", func(t *testing.T) {
		client.SetToken("")
		client.SetHeader(constants.SELLER_ID_HEADER, "3")

		getW := client.Get(t, "/api/categories/by-parent?parentId=invalid")
		helpers.AssertErrorResponse(
			t,
			getW,
			http.StatusBadRequest,
			"",
		)
	})
}
