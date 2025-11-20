package category

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestUpdateCategory(t *testing.T) {
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
	// BASIC UPDATE TESTS - Combined (P0)
	// ============================================================================

	t.Run("Admin updates global category (all fields)", func(t *testing.T) {
		// Login as admin
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create initial global category
		createReq := map[string]interface{}{
			"name":        "Electronics",
			"description": "Electronic devices",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))
		originalUpdatedAt := category["updatedAt"].(string)

		// Verify initial state
		assert.True(t, category["isGlobal"].(bool), "Should be global")
		assert.Nil(t, category["sellerId"], "Should have no sellerId")

		// Wait 1 second to ensure timestamp difference (RFC3339 has second precision)
		time.Sleep(1 * time.Second)

		// Update 1: Change name and description
		updateReq1 := map[string]interface{}{
			"name":        "Electronics & Gadgets",
			"description": "Electronic devices and gadgets",
		}
		updateW1 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq1)
		updateResponse1 := helpers.AssertSuccessResponse(
			t,
			updateW1,
			http.StatusOK,
			)
		updatedCategory1 := helpers.GetResponseData(t, updateResponse1, "category")

		assert.Equal(t, "Electronics & Gadgets", updatedCategory1["name"])
		assert.Equal(t, "Electronic devices and gadgets", updatedCategory1["description"])
		assert.True(t, updatedCategory1["isGlobal"].(bool), "Should remain global")
		assert.Nil(t, updatedCategory1["sellerId"], "sellerId should remain nil")
		assert.Equal(
			t,
			float64(categoryID),
			updatedCategory1["id"].(float64),
			"ID should not change",
		)
		// createdAt should not change (both should be in UTC Z format)
		assert.Contains(
			t,
			updatedCategory1["createdAt"].(string),
			"T",
			"createdAt should be in ISO8601 format",
		)
		assert.Contains(
			t,
			updatedCategory1["createdAt"].(string),
			"Z",
			"createdAt should be in UTC format",
		)
		// updatedAt should change
		assert.Contains(
			t,
			updatedCategory1["updatedAt"].(string),
			"T",
			"updatedAt should be in ISO8601 format",
		)
		assert.NotEqual(
			t,
			originalUpdatedAt,
			updatedCategory1["updatedAt"].(string),
			"updatedAt should change",
		)

		// Create a parent category for hierarchy tests
		parentReq := map[string]interface{}{
			"name":        "Technology",
			"description": "Technology parent category",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			)
		parentCategory := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parentCategory["id"].(float64))

		// Update 2: Set parent (convert root to subcategory)
		updateReq2 := map[string]interface{}{
			"name":        "Electronics & Gadgets",
			"description": "Electronic devices and gadgets",
			"parentId":    parentID,
		}
		updateW2 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq2)
		updateResponse2 := helpers.AssertSuccessResponse(
			t,
			updateW2,
			http.StatusOK,
			)
		updatedCategory2 := helpers.GetResponseData(t, updateResponse2, "category")

		assert.Equal(
			t,
			float64(parentID),
			updatedCategory2["parentId"].(float64),
			"Should have parent now",
		)
		assert.True(t, updatedCategory2["isGlobal"].(bool), "Should remain global")

		// Update 3: Remove parent (make it root again)
		updateReq3 := map[string]interface{}{
			"name":        "Electronics & Gadgets",
			"description": "Electronic devices and gadgets",
			"parentId":    nil,
		}
		updateW3 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq3)
		updateResponse3 := helpers.AssertSuccessResponse(
			t,
			updateW3,
			http.StatusOK,
			)
		updatedCategory3 := helpers.GetResponseData(t, updateResponse3, "category")

		assert.Nil(t, updatedCategory3["parentId"], "Should be root category again")
		assert.True(t, updatedCategory3["isGlobal"].(bool), "Should remain global")

		// Create another parent for move test
		parent2Req := map[string]interface{}{
			"name":        "Consumer Electronics",
			"description": "Consumer electronics parent",
		}
		parent2W := client.Post(t, "/api/categories", parent2Req)
		parent2Response := helpers.AssertSuccessResponse(
			t,
			parent2W,
			http.StatusCreated,
			)
		parent2Category := helpers.GetResponseData(t, parent2Response, "category")
		parent2ID := uint(parent2Category["id"].(float64))

		// Update 4: Change parent (move to different parent)
		updateReq4 := map[string]interface{}{
			"name":        "Electronics & Gadgets",
			"description": "Electronic devices and gadgets",
			"parentId":    parent2ID,
		}
		updateW4 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq4)
		updateResponse4 := helpers.AssertSuccessResponse(
			t,
			updateW4,
			http.StatusOK,
			)
		updatedCategory4 := helpers.GetResponseData(t, updateResponse4, "category")

		assert.Equal(
			t,
			float64(parent2ID),
			updatedCategory4["parentId"].(float64),
			"Should have new parent",
		)
		assert.True(t, updatedCategory4["isGlobal"].(bool), "Should remain global")
		assert.Nil(t, updatedCategory4["sellerId"], "sellerId should remain nil")
	})

	t.Run("Seller updates seller-specific category (all fields)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create initial seller-specific category
		createReq := map[string]interface{}{
			"name":        "My Products",
			"description": "My seller products",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))
		originalSellerID := category["sellerId"]

		// Verify initial state
		assert.False(t, category["isGlobal"].(bool), "Should not be global")
		assert.NotNil(t, category["sellerId"], "Should have sellerId")

		time.Sleep(10 * time.Millisecond)

		// Update 1: Change name and description
		updateReq1 := map[string]interface{}{
			"name":        "My Premium Products",
			"description": "My premium seller products",
		}
		updateW1 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq1)
		updateResponse1 := helpers.AssertSuccessResponse(
			t,
			updateW1,
			http.StatusOK,
			)
		updatedCategory1 := helpers.GetResponseData(t, updateResponse1, "category")

		assert.Equal(t, "My Premium Products", updatedCategory1["name"])
		assert.Equal(t, "My premium seller products", updatedCategory1["description"])
		assert.False(t, updatedCategory1["isGlobal"].(bool), "Should remain seller-specific")
		assert.Equal(
			t,
			originalSellerID,
			updatedCategory1["sellerId"],
			"sellerId should not change",
		)
		assert.Equal(
			t,
			float64(categoryID),
			updatedCategory1["id"].(float64),
			"ID should not change",
		)
		// createdAt should not change (both should be in UTC Z format)
		assert.Contains(
			t,
			updatedCategory1["createdAt"].(string),
			"T",
			"createdAt should be in ISO8601 format",
		)
		assert.Contains(
			t,
			updatedCategory1["createdAt"].(string),
			"Z",
			"createdAt should be in UTC format",
		)

		// Create parent for seller
		parentReq := map[string]interface{}{
			"name":        "Seller Categories",
			"description": "Parent for seller categories",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			)
		parentCategory := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parentCategory["id"].(float64))

		// Update 2: Change parent
		updateReq2 := map[string]interface{}{
			"name":        "My Premium Products",
			"description": "My premium seller products",
			"parentId":    parentID,
		}
		updateW2 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq2)
		updateResponse2 := helpers.AssertSuccessResponse(
			t,
			updateW2,
			http.StatusOK,
			)
		updatedCategory2 := helpers.GetResponseData(t, updateResponse2, "category")

		assert.Equal(
			t,
			float64(parentID),
			updatedCategory2["parentId"].(float64),
			"Should have parent now",
		)
		assert.False(t, updatedCategory2["isGlobal"].(bool), "Should remain seller-specific")
		assert.Equal(
			t,
			originalSellerID,
			updatedCategory2["sellerId"],
			"sellerId should not change",
		)
	})

	t.Run("Partial update (single field)", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		createReq := map[string]interface{}{
			"name":        "Books",
			"description": "Books and magazines",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Update only name
		updateReq := map[string]interface{}{
			"name":        "Books & Magazines",
			"description": "Books and magazines", // Same as before
		}
		updateW := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq)
		updateResponse := helpers.AssertSuccessResponse(
			t,
			updateW,
			http.StatusOK,
			)
		updatedCategory := helpers.GetResponseData(t, updateResponse, "category")

		assert.Equal(t, "Books & Magazines", updatedCategory["name"], "Name should be updated")
		assert.Equal(
			t,
			"Books and magazines",
			updatedCategory["description"],
			"Description should remain same",
		)
		assert.Nil(t, updatedCategory["parentId"], "Parent should remain nil")
		// Note: updatedAt timestamp check is thoroughly tested in "Admin updates global category" test
	})

	// ============================================================================
	// VALIDATION TESTS - Combined (P0 & P1)
	// ============================================================================

	t.Run("Field validation on update", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create a category to update
		createReq := map[string]interface{}{
			"name":        "Test Category",
			"description": "For validation tests",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// Empty name
		updateReq1 := map[string]interface{}{
			"name":        "",
			"description": "Updated description",
		}
		w1 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq1)
		helpers.AssertErrorResponse(t, w1, http.StatusBadRequest)

		// Name too long (>100 chars)
		longName := ""
		for i := 0; i < 101; i++ {
			longName += "a"
		}
		updateReq2 := map[string]interface{}{
			"name":        longName,
			"description": "Updated description",
		}
		w2 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq2)
		helpers.AssertErrorResponse(t, w2, http.StatusBadRequest)

		// Name too short (<3 chars)
		updateReq3 := map[string]interface{}{
			"name":        "ab",
			"description": "Updated description",
		}
		w3 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq3)
		helpers.AssertErrorResponse(t, w3, http.StatusBadRequest)

		// Description too long (>500 chars)
		longDesc := ""
		for i := 0; i < 501; i++ {
			longDesc += "a"
		}
		updateReq4 := map[string]interface{}{
			"name":        "Valid Name",
			"description": longDesc,
		}
		w4 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq4)
		helpers.AssertErrorResponse(t, w4, http.StatusBadRequest)

		// Invalid category ID
		updateReq5 := map[string]interface{}{
			"name":        "Valid Name",
			"description": "Valid description",
		}
		w5 := client.Put(t, "/api/categories/99999", updateReq5)
		helpers.AssertErrorResponse(t, w5, http.StatusNotFound)

		// Invalid parent ID
		updateReq6 := map[string]interface{}{
			"name":        "Valid Name",
			"description": "Valid description",
			"parentId":    99999,
		}
		w6 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq6)
		helpers.AssertErrorResponse(t, w6, http.StatusBadRequest)
	})

	// ============================================================================
	// DUPLICATE & NAME CONFLICT TESTS (P0)
	// ============================================================================

	t.Run("Duplicate name handling", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create parent category
		parentReq := map[string]interface{}{
			"name":        "Sports",
			"description": "Sports parent",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			)
		parentCategory := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parentCategory["id"].(float64))

		// Create first child category
		child1Req := map[string]interface{}{
			"name":        "Football",
			"description": "Football equipment",
			"parentId":    parentID,
		}
		child1W := client.Post(t, "/api/categories", child1Req)
		helpers.AssertSuccessResponse(
			t,
			child1W,
			http.StatusCreated,
			)

		// Create second child category
		child2Req := map[string]interface{}{
			"name":        "Basketball",
			"description": "Basketball equipment",
			"parentId":    parentID,
		}
		child2W := client.Post(t, "/api/categories", child2Req)
		child2Response := helpers.AssertSuccessResponse(
			t,
			child2W,
			http.StatusCreated,
			)
		child2Category := helpers.GetResponseData(t, child2Response, "category")
		child2ID := uint(child2Category["id"].(float64))

		// Try to update child2 to have same name as child1 (same parent) - should fail
		updateReq1 := map[string]interface{}{
			"name":        "Football",
			"description": "Basketball equipment",
			"parentId":    parentID,
		}
		w1 := client.Put(t, fmt.Sprintf("/api/categories/%d", child2ID), updateReq1)
		helpers.AssertErrorResponse(t, w1, http.StatusConflict)

		// Create another parent
		parent2Req := map[string]interface{}{
			"name":        "Outdoor Sports",
			"description": "Outdoor sports parent",
		}
		parent2W := client.Post(t, "/api/categories", parent2Req)
		parent2Response := helpers.AssertSuccessResponse(
			t,
			parent2W,
			http.StatusCreated,
			)
		parent2Category := helpers.GetResponseData(t, parent2Response, "category")
		parent2ID := uint(parent2Category["id"].(float64))

		// Update child2 to have same name as child1 but different parent - should succeed
		updateReq2 := map[string]interface{}{
			"name":        "Football",
			"description": "Basketball equipment",
			"parentId":    parent2ID,
		}
		w2 := client.Put(t, fmt.Sprintf("/api/categories/%d", child2ID), updateReq2)
		updateResponse2 := helpers.AssertSuccessResponse(
			t,
			w2,
			http.StatusOK,
			)
		updatedChild2 := helpers.GetResponseData(t, updateResponse2, "category")
		assert.Equal(t, "Football", updatedChild2["name"])
		assert.Equal(t, float64(parent2ID), updatedChild2["parentId"].(float64))

		// Update with same name (no change) - should succeed
		updateReq3 := map[string]interface{}{
			"name":        "Football",
			"description": "Updated description",
			"parentId":    parent2ID,
		}
		w3 := client.Put(t, fmt.Sprintf("/api/categories/%d", child2ID), updateReq3)
		updateResponse3 := helpers.AssertSuccessResponse(
			t,
			w3,
			http.StatusOK,
			)
		updatedChild3 := helpers.GetResponseData(t, updateResponse3, "category")
		assert.Equal(t, "Football", updatedChild3["name"])
		assert.Equal(t, "Updated description", updatedChild3["description"])
	})

	// ============================================================================
	// GLOBAL vs SELLER-SPECIFIC IMMUTABILITY (P0)
	// ============================================================================

	t.Run("Category type immutability", func(t *testing.T) {
		// Test admin updating global category
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		createReq1 := map[string]interface{}{
			"name":        "Immutability Test Global",
			"description": "This is global",
		}
		createW1 := client.Post(t, "/api/categories", createReq1)
		createResponse1 := helpers.AssertSuccessResponse(
			t,
			createW1,
			http.StatusCreated,
			)
		globalCategory := helpers.GetResponseData(t, createResponse1, "category")
		globalCategoryID := uint(globalCategory["id"].(float64))

		// Update global category - verify it remains global
		updateReq1 := map[string]interface{}{
			"name":        "Updated Immutability Global",
			"description": "Still global",
		}
		updateW1 := client.Put(t, fmt.Sprintf("/api/categories/%d", globalCategoryID), updateReq1)
		updateResponse1 := helpers.AssertSuccessResponse(
			t,
			updateW1,
			http.StatusOK,
			)
		updatedGlobal := helpers.GetResponseData(t, updateResponse1, "category")

		assert.True(t, updatedGlobal["isGlobal"].(bool), "Should remain global")
		assert.Nil(t, updatedGlobal["sellerId"], "sellerId should remain nil")

		// Test seller updating seller-specific category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		createReq2 := map[string]interface{}{
			"name":        "Seller Category",
			"description": "This is seller-specific",
		}
		createW2 := client.Post(t, "/api/categories", createReq2)
		createResponse2 := helpers.AssertSuccessResponse(
			t,
			createW2,
			http.StatusCreated,
			)
		sellerCategory := helpers.GetResponseData(t, createResponse2, "category")
		sellerCategoryID := uint(sellerCategory["id"].(float64))
		originalSellerID := sellerCategory["sellerId"]

		// Update seller category - verify it remains seller-specific
		updateReq2 := map[string]interface{}{
			"name":        "Updated Seller Category",
			"description": "Still seller-specific",
		}
		updateW2 := client.Put(t, fmt.Sprintf("/api/categories/%d", sellerCategoryID), updateReq2)
		updateResponse2 := helpers.AssertSuccessResponse(
			t,
			updateW2,
			http.StatusOK,
			)
		updatedSeller := helpers.GetResponseData(t, updateResponse2, "category")

		assert.False(t, updatedSeller["isGlobal"].(bool), "Should remain seller-specific")
		assert.Equal(t, originalSellerID, updatedSeller["sellerId"], "sellerId should not change")
	})

	// ============================================================================
	// HIERARCHY & CIRCULAR REFERENCE TESTS (P0)
	// ============================================================================

	t.Run("Prevent circular references", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category A
		createReqA := map[string]interface{}{
			"name":        "Category A",
			"description": "First category",
		}
		createWA := client.Post(t, "/api/categories", createReqA)
		createResponseA := helpers.AssertSuccessResponse(
			t,
			createWA,
			http.StatusCreated,
			)
		categoryA := helpers.GetResponseData(t, createResponseA, "category")
		categoryAID := uint(categoryA["id"].(float64))

		// Try to set A as its own parent - should fail
		updateReqSelf := map[string]interface{}{
			"name":        "Category A",
			"description": "First category",
			"parentId":    categoryAID,
		}
		wSelf := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryAID), updateReqSelf)
		helpers.AssertErrorResponse(t, wSelf, http.StatusBadRequest)

		// Create category B with A as parent
		createReqB := map[string]interface{}{
			"name":        "Category B",
			"description": "Second category",
			"parentId":    categoryAID,
		}
		createWB := client.Post(t, "/api/categories", createReqB)
		createResponseB := helpers.AssertSuccessResponse(
			t,
			createWB,
			http.StatusCreated,
			)
		categoryB := helpers.GetResponseData(t, createResponseB, "category")
		categoryBID := uint(categoryB["id"].(float64))

		// Create category C with B as parent (A->B->C)
		createReqC := map[string]interface{}{
			"name":        "Category C",
			"description": "Third category",
			"parentId":    categoryBID,
		}
		createWC := client.Post(t, "/api/categories", createReqC)
		createResponseC := helpers.AssertSuccessResponse(
			t,
			createWC,
			http.StatusCreated,
			)
		categoryC := helpers.GetResponseData(t, createResponseC, "category")
		categoryCID := uint(categoryC["id"].(float64))

		// Try to set A's parent as C (creating circular: A->B->C->A) - should fail
		updateReqCircular := map[string]interface{}{
			"name":        "Category A",
			"description": "First category",
			"parentId":    categoryCID,
		}
		wCircular := client.Put(
			t,
			fmt.Sprintf("/api/categories/%d", categoryAID),
			updateReqCircular,
		)
		helpers.AssertErrorResponse(t, wCircular, http.StatusBadRequest)
	})

	t.Run("Cross-boundary parent updates", func(t *testing.T) {
		// Admin creates global parent
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		globalParentReq := map[string]interface{}{
			"name":        "Global Parent",
			"description": "Global parent category",
		}
		globalParentW := client.Post(t, "/api/categories", globalParentReq)
		globalParentResponse := helpers.AssertSuccessResponse(
			t,
			globalParentW,
			http.StatusCreated,
			)
		globalParent := helpers.GetResponseData(t, globalParentResponse, "category")
		globalParentID := uint(globalParent["id"].(float64))

		// Admin creates another global category
		global2Req := map[string]interface{}{
			"name":        "Global Category 2",
			"description": "Another global category",
		}
		global2W := client.Post(t, "/api/categories", global2Req)
		global2Response := helpers.AssertSuccessResponse(
			t,
			global2W,
			http.StatusCreated,
			)
		global2 := helpers.GetResponseData(t, global2Response, "category")
		global2ID := uint(global2["id"].(float64))

		// Admin moves global category under another global category - should succeed
		updateGlobalReq := map[string]interface{}{
			"name":        "Global Category 2",
			"description": "Another global category",
			"parentId":    globalParentID,
		}
		updateGlobalW := client.Put(
			t,
			fmt.Sprintf("/api/categories/%d", global2ID),
			updateGlobalReq,
		)
		updateGlobalResponse := helpers.AssertSuccessResponse(
			t,
			updateGlobalW,
			http.StatusOK,
			)
		updatedGlobal2 := helpers.GetResponseData(t, updateGlobalResponse, "category")
		assert.Equal(t, float64(globalParentID), updatedGlobal2["parentId"].(float64))
		assert.True(t, updatedGlobal2["isGlobal"].(bool))

		// Seller creates seller-specific category
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		sellerCatReq := map[string]interface{}{
			"name":        "Seller Category",
			"description": "Seller's category",
		}
		sellerCatW := client.Post(t, "/api/categories", sellerCatReq)
		sellerCatResponse := helpers.AssertSuccessResponse(
			t,
			sellerCatW,
			http.StatusCreated,
			)
		sellerCat := helpers.GetResponseData(t, sellerCatResponse, "category")
		sellerCatID := uint(sellerCat["id"].(float64))

		// Seller tries to move seller-specific category under global category - should succeed
		updateSellerReq := map[string]interface{}{
			"name":        "Seller Category",
			"description": "Seller's category under global parent",
			"parentId":    globalParentID,
		}
		updateSellerW := client.Put(
			t,
			fmt.Sprintf("/api/categories/%d", sellerCatID),
			updateSellerReq,
		)
		updateSellerResponse := helpers.AssertSuccessResponse(
			t,
			updateSellerW,
			http.StatusOK,
			)
		updatedSellerCat := helpers.GetResponseData(t, updateSellerResponse, "category")
		assert.Equal(
			t,
			globalParentID,
			uint(updatedSellerCat["parentId"].(float64)),
			"Should have global parent",
		)

		// Seller creates another seller-specific category as parent
		sellerParentReq := map[string]interface{}{
			"name":        "Seller Parent",
			"description": "Seller's parent category",
		}
		sellerParentW := client.Post(t, "/api/categories", sellerParentReq)
		sellerParentResponse := helpers.AssertSuccessResponse(
			t,
			sellerParentW,
			http.StatusCreated,
			)
		sellerParent := helpers.GetResponseData(t, sellerParentResponse, "category")
		sellerParentID := uint(sellerParent["id"].(float64))

		// Seller moves seller-specific category under own seller-specific category - should succeed
		updateSellerReq2 := map[string]interface{}{
			"name":        "Seller Category",
			"description": "Seller's category",
			"parentId":    sellerParentID,
		}
		updateSellerW2 := client.Put(
			t,
			fmt.Sprintf("/api/categories/%d", sellerCatID),
			updateSellerReq2,
		)
		updateSellerResponse2 := helpers.AssertSuccessResponse(
			t,
			updateSellerW2,
			http.StatusOK,
			)
		updatedSellerCat2 := helpers.GetResponseData(t, updateSellerResponse2, "category")
		assert.Equal(t, float64(sellerParentID), updatedSellerCat2["parentId"].(float64))
		assert.False(t, updatedSellerCat2["isGlobal"].(bool))
	})

	// ============================================================================
	// AUTHORIZATION & PERMISSIONS TESTS (P0 & P1)
	// ============================================================================

	t.Run("Unauthorized access", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create a category
		createReq := map[string]interface{}{
			"name":        "Category For Auth Tests",
			"description": "For authorization tests",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		// No token - should return 401
		client.SetToken("")
		updateReq := map[string]interface{}{
			"name":        "Updated Name",
			"description": "Updated description",
		}
		w1 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq)
		helpers.AssertErrorResponse(t, w1, http.StatusUnauthorized)

		// Customer role - should return 403 (authenticated but insufficient permissions)
		customerToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(customerToken)
		w2 := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq)
		helpers.AssertErrorResponse(t, w2, http.StatusForbidden)
	})

	t.Run("Cross-user/seller restrictions", func(t *testing.T) {
		// Admin creates global category
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		globalReq := map[string]interface{}{
			"name":        "Cross User Restrictions Global",
			"description": "Global for restrictions test",
		}
		globalW := client.Post(t, "/api/categories", globalReq)
		globalResponse := helpers.AssertSuccessResponse(
			t,
			globalW,
			http.StatusCreated,
			)
		globalCat := helpers.GetResponseData(t, globalResponse, "category")
		globalCatID := uint(globalCat["id"].(float64))

		// Seller tries to update global category - should fail
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		updateGlobalReq := map[string]interface{}{
			"name":        "Hacked Global Category",
			"description": "Trying to hack",
		}
		w1 := client.Put(t, fmt.Sprintf("/api/categories/%d", globalCatID), updateGlobalReq)
		helpers.AssertErrorResponse(t, w1, http.StatusForbidden)

		// Seller creates own category
		sellerReq := map[string]interface{}{
			"name":        "Seller Category",
			"description": "Seller's own category",
		}
		sellerW := client.Post(t, "/api/categories", sellerReq)
		helpers.AssertSuccessResponse(
			t,
			sellerW,
			http.StatusCreated,
			)

		// Note: Testing "another seller" would require a second seller account in seeds
		// For now, we verify admin can update any global category

		// Admin updates global category - should succeed
		client.SetToken(adminToken)
		updateReq := map[string]interface{}{
			"name":        "Updated Restrictions Global",
			"description": "Admin can update",
		}
		w2 := client.Put(t, fmt.Sprintf("/api/categories/%d", globalCatID), updateReq)
		updateResponse := helpers.AssertSuccessResponse(
			t,
			w2,
			http.StatusOK,
			)
		updatedCat := helpers.GetResponseData(t, updateResponse, "category")
		assert.Equal(t, "Updated Restrictions Global", updatedCat["name"])
		assert.True(t, updatedCat["isGlobal"].(bool))
	})

	// ============================================================================
	// EDGE CASES & DATA INTEGRITY (P1)
	// ============================================================================

	t.Run("Update category with relationships", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create parent category
		parentReq := map[string]interface{}{
			"name":        "Parent Category",
			"description": "Has children",
		}
		parentW := client.Post(t, "/api/categories", parentReq)
		parentResponse := helpers.AssertSuccessResponse(
			t,
			parentW,
			http.StatusCreated,
			)
		parent := helpers.GetResponseData(t, parentResponse, "category")
		parentID := uint(parent["id"].(float64))

		// Create child categories
		child1Req := map[string]interface{}{
			"name":        "Child 1",
			"description": "First child",
			"parentId":    parentID,
		}
		child1W := client.Post(t, "/api/categories", child1Req)
		helpers.AssertSuccessResponse(
			t,
			child1W,
			http.StatusCreated,
			)

		child2Req := map[string]interface{}{
			"name":        "Child 2",
			"description": "Second child",
			"parentId":    parentID,
		}
		child2W := client.Post(t, "/api/categories", child2Req)
		helpers.AssertSuccessResponse(
			t,
			child2W,
			http.StatusCreated,
			)

		// Update parent category that has children - should succeed
		updateParentReq := map[string]interface{}{
			"name":        "Updated Parent Category",
			"description": "Still has children",
		}
		updateParentW := client.Put(t, fmt.Sprintf("/api/categories/%d", parentID), updateParentReq)
		updateParentResponse := helpers.AssertSuccessResponse(
			t,
			updateParentW,
			http.StatusOK,
			)
		updatedParent := helpers.GetResponseData(t, updateParentResponse, "category")

		assert.Equal(t, "Updated Parent Category", updatedParent["name"])
		assert.Equal(t, float64(parentID), updatedParent["id"].(float64), "ID should not change")

		// Verify children still exist by getting all categories (children should still be intact)
		getChildW := client.Get(t, "/api/categories")
		helpers.AssertSuccessResponse(t, getChildW, http.StatusOK)
		// Children should still be intact (detailed verification would require GET by ID or checking hierarchy)
	})

	t.Run("Update with no changes", func(t *testing.T) {
		adminToken := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(adminToken)

		// Create category
		createReq := map[string]interface{}{
			"name":        "No Change Category",
			"description": "Will not change",
		}
		createW := client.Post(t, "/api/categories", createReq)
		createResponse := helpers.AssertSuccessResponse(
			t,
			createW,
			http.StatusCreated,
			)
		category := helpers.GetResponseData(t, createResponse, "category")
		categoryID := uint(category["id"].(float64))

		time.Sleep(10 * time.Millisecond)

		// Update with exact same data
		updateReq := map[string]interface{}{
			"name":        "No Change Category",
			"description": "Will not change",
		}
		updateW := client.Put(t, fmt.Sprintf("/api/categories/%d", categoryID), updateReq)
		updateResponse := helpers.AssertSuccessResponse(
			t,
			updateW,
			http.StatusOK,
			)
		updatedCategory := helpers.GetResponseData(t, updateResponse, "category")

		assert.Equal(t, category["name"], updatedCategory["name"])
		assert.Equal(t, category["description"], updatedCategory["description"])
		assert.Equal(t, category["id"], updatedCategory["id"])
		assert.Equal(t, category["createdAt"], updatedCategory["createdAt"])
		// Note: updatedAt may or may not change depending on implementation
	})
}
