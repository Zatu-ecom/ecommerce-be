package wishlist

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/product/entity"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteWishlist tests the Delete Wishlist (DELETE /api/product/wishlist/:id) API
//
// This endpoint deletes a wishlist and all its items.
//
// Endpoint: DELETE /api/product/wishlist/:id
// Authentication: Required (Customer Auth only)
//
// Business Rules:
//   - User can only delete their own wishlists
//   - Cannot delete the default wishlist
//   - Deleting a wishlist removes all its items
//   - Returns 200 OK on successful deletion
func TestDeleteWishlist(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// Setup: Create wishlists for testing
	// ============================================================================

	// Login as customer (Alice - user 5)
	aliceToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
	client.SetToken(aliceToken)

	// Create first wishlist (will be default)
	createReq := map[string]interface{}{
		"name": "Alice Default Wishlist",
	}
	w := client.Post(t, "/api/product/wishlist", createReq)
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceDefaultWishlist := helpers.GetResponseData(t, response, "wishlist")
	aliceDefaultWishlistID := uint(aliceDefaultWishlist["id"].(float64))

	// Create second wishlist (non-default) for deletion tests
	createReq = map[string]interface{}{
		"name": "Alice Secondary Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceWishlist2 := helpers.GetResponseData(t, response, "wishlist")
	aliceWishlistID2 := uint(aliceWishlist2["id"].(float64))

	// Create third wishlist for additional tests
	createReq = map[string]interface{}{
		"name": "Alice Third Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceWishlist3 := helpers.GetResponseData(t, response, "wishlist")
	aliceWishlistID3 := uint(aliceWishlist3["id"].(float64))

	// Create fourth wishlist with items for testing cascade delete
	createReq = map[string]interface{}{
		"name": "Alice Wishlist With Items",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceWishlist4 := helpers.GetResponseData(t, response, "wishlist")
	aliceWishlistID4 := uint(aliceWishlist4["id"].(float64))

	// Add items to fourth wishlist
	client.SetHeader("X-Seller-ID", "2")
	w = client.Get(t, "/api/product?page=1&pageSize=3")
	response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data := response["data"].(map[string]interface{})

	if resultsRaw, ok := data["results"]; ok && resultsRaw != nil {
		products := resultsRaw.([]interface{})
		for i := 0; i < 2 && i < len(products); i++ {
			product := products[i].(map[string]interface{})
			if variantsRaw, vOk := product["variants"]; vOk && variantsRaw != nil {
				variants := variantsRaw.([]interface{})
				if len(variants) > 0 {
					variant := variants[0].(map[string]interface{})
					variantID := uint(variant["id"].(float64))

					addReq := map[string]interface{}{
						"variantId": variantID,
					}
					w = client.Post(
						t,
						fmt.Sprintf("/api/product/wishlist/%d/item", aliceWishlistID4),
						addReq,
					)
					helpers.AssertSuccessResponse(t, w, http.StatusCreated)
				}
			}
		}
	}

	// Login as Michael (user 6) and create wishlist for authorization tests
	michaelToken := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
	client.SetToken(michaelToken)

	// Create default wishlist for Michael
	createReq = map[string]interface{}{
		"name": "Michael Default Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	michaelWishlist := helpers.GetResponseData(t, response, "wishlist")
	michaelWishlistID := uint(michaelWishlist["id"].(float64))

	// Create non-default wishlist for Michael
	createReq = map[string]interface{}{
		"name": "Michael Secondary Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	michaelWishlist2 := helpers.GetResponseData(t, response, "wishlist")
	michaelWishlistID2 := uint(michaelWishlist2["id"].(float64))

	// ============================================================================
	// Happy Path Scenarios
	// ============================================================================

	t.Run("HP-001: Delete non-default wishlist successfully", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Verify wishlist exists before deletion
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID2))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Delete the wishlist
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID2))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify wishlist no longer exists
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID2))
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("HP-002: Delete wishlist with items (cascade delete)", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Verify wishlist exists and check if it has items
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID4))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Check itemCount safely (may be 0 if products weren't available in seed data)
		var itemCount int
		if itemCountRaw, ok := wishlist["itemCount"]; ok && itemCountRaw != nil {
			itemCount = int(itemCountRaw.(float64))
		}
		t.Logf("Wishlist has %d items before deletion", itemCount)

		// Delete the wishlist (should work regardless of item count)
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID4))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify wishlist no longer exists
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID4))
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("HP-003: Delete wishlist then verify removed from GetAllWishlists", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get all wishlists count before
		w := client.Get(t, "/api/product/wishlist")
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlistsBefore := response["data"].(map[string]interface{})["wishlists"].([]interface{})
		countBefore := len(wishlistsBefore)

		// Delete third wishlist
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID3))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Get all wishlists count after
		w = client.Get(t, "/api/product/wishlist")
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlistsAfter := response["data"].(map[string]interface{})["wishlists"].([]interface{})
		countAfter := len(wishlistsAfter)

		// Verify count decreased by 1
		assert.Equal(t, countBefore-1, countAfter, "Wishlist count should decrease by 1")

		// Verify deleted wishlist is not in the list
		for _, wl := range wishlistsAfter {
			wishlist := wl.(map[string]interface{})
			assert.NotEqual(
				t,
				float64(aliceWishlistID3),
				wishlist["id"].(float64),
				"Deleted wishlist should not appear in list",
			)
		}
	})

	// ============================================================================
	// Negative Scenarios - Authentication
	// ============================================================================

	t.Run("NEG-001: Delete wishlist without authentication returns 401", func(t *testing.T) {
		client.SetToken("")
		w := client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("NEG-002: Delete wishlist with invalid token returns 401", func(t *testing.T) {
		client.SetToken("invalid-token-here")
		w := client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	// ============================================================================
	// Negative Scenarios - Authorization
	// ============================================================================

	t.Run("NEG-003: Seller role cannot delete wishlist", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(token)
		w := client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-004: Admin role cannot delete wishlist", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(token)
		w := client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-005: User cannot delete another user's wishlist", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)
		w := client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-006: User A deleting User B's wishlist gets 403 not 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		w := client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	// ============================================================================
	// Negative Scenarios - Business Rules
	// ============================================================================

	t.Run("NEG-007: Cannot delete default wishlist returns 422", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Verify it's the default wishlist
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.True(t, wishlist["isDefault"].(bool), "Should be default wishlist")

		// Try to delete default wishlist
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusUnprocessableEntity)

		// Verify wishlist still exists
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	// ============================================================================
	// Negative Scenarios - Validation
	// ============================================================================

	t.Run("NEG-008: Delete wishlist with non-existent ID returns 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		w := client.Delete(t, "/api/product/wishlist/99999")
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("NEG-009: Delete wishlist with invalid ID format (string)", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		w := client.Delete(t, "/api/product/wishlist/abc")
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-010: Delete wishlist with invalid ID format (negative)", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		w := client.Delete(t, "/api/product/wishlist/-1")
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-011: Delete already deleted wishlist returns 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// Delete Michael's second wishlist first time
		w := client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID2))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Try to delete same wishlist again
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID2))
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// Edge Case Scenarios
	// ============================================================================

	t.Run("EDGE-001: Delete wishlist with very large ID returns 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		w := client.Delete(t, "/api/product/wishlist/9999999999")
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("EDGE-002: Delete wishlist with ID zero returns 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		w := client.Delete(t, "/api/product/wishlist/0")
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("EDGE-003: Delete request is idempotent (second delete returns 404)", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create a new wishlist specifically for this test
		createReq := map[string]interface{}{
			"name": "Alice Idempotent Test Wishlist",
		}
		w := client.Post(t, "/api/product/wishlist", createReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		newWishlist := helpers.GetResponseData(t, response, "wishlist")
		newWishlistID := uint(newWishlist["id"].(float64))

		// First delete - should succeed
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", newWishlistID))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Second delete - should return 404
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", newWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// Security Scenarios
	// ============================================================================

	t.Run("SEC-001: SQL injection in wishlist ID is prevented", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		w := client.Delete(t, "/api/product/wishlist/1;DROP TABLE wishlist;--")
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("SEC-002: Path traversal in ID is prevented", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)
		// Path traversal with ".." is resolved at router level before reaching our handler
		// The router resolves the path and either routes to a different endpoint (404)
		// or the ID parser rejects it (400). Either response is acceptable as it prevents access.
		w := client.Delete(t, "/api/product/wishlist/../../../etc/passwd")
		assert.True(
			t,
			w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
			"Path traversal should be rejected with 400 or 404, got %d",
			w.Code,
		)
	})

	t.Run("SEC-003: User isolation - verify no cross-user deletion", func(t *testing.T) {
		// Create a fresh wishlist for Michael
		michaelToken := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(michaelToken)

		createReq := map[string]interface{}{
			"name": "Michael Protected Wishlist",
		}
		w := client.Post(t, "/api/product/wishlist", createReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		michaelProtectedWishlist := helpers.GetResponseData(t, response, "wishlist")
		michaelProtectedID := uint(michaelProtectedWishlist["id"].(float64))

		// Login as Alice and try to delete Michael's wishlist
		aliceToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(aliceToken)

		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", michaelProtectedID))
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)

		// Verify Michael's wishlist still exists
		client.SetToken(michaelToken)
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelProtectedID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			"Michael Protected Wishlist",
			wishlist["name"],
			"Michael's wishlist should still exist after Alice's failed delete",
		)
	})

	// ============================================================================
	// Business Logic Scenarios
	// ============================================================================

	t.Run("BL-001: After deletion default wishlist still exists", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create a non-default wishlist
		createReq := map[string]interface{}{
			"name": "Alice Temp Wishlist",
		}
		w := client.Post(t, "/api/product/wishlist", createReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		tempWishlist := helpers.GetResponseData(t, response, "wishlist")
		tempWishlistID := uint(tempWishlist["id"].(float64))

		// Delete the temp wishlist
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", tempWishlistID))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify default wishlist still exists
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceDefaultWishlistID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.True(t, wishlist["isDefault"].(bool), "Default wishlist should still exist")
	})

	t.Run("BL-002: User can have only default wishlist left", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// Get all wishlists
		w := client.Get(t, "/api/product/wishlist")
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlists := response["data"].(map[string]interface{})["wishlists"].([]interface{})

		// Delete all non-default wishlists
		for _, wl := range wishlists {
			wishlist := wl.(map[string]interface{})
			if !wishlist["isDefault"].(bool) {
				wishlistID := uint(wishlist["id"].(float64))
				w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", wishlistID))
				// Might be already deleted in previous tests, so accept both 200 and 404
				assert.True(
					t,
					w.Code == http.StatusOK || w.Code == http.StatusNotFound,
					"Should either delete or already be deleted",
				)
			}
		}

		// Verify only default wishlist remains
		w = client.Get(t, "/api/product/wishlist")
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlistsAfter := response["data"].(map[string]interface{})["wishlists"].([]interface{})

		// Should have at least the default wishlist
		assert.GreaterOrEqual(
			t,
			len(wishlistsAfter),
			1,
			"Should have at least the default wishlist",
		)

		// Verify the remaining wishlist is default
		hasDefault := false
		for _, wl := range wishlistsAfter {
			wishlist := wl.(map[string]interface{})
			if wishlist["isDefault"].(bool) {
				hasDefault = true
				break
			}
		}
		assert.True(t, hasDefault, "Default wishlist should remain")
	})

	t.Run(
		"BL-003: Make non-default wishlist default then try delete returns 422",
		func(t *testing.T) {
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			// Create a new wishlist
			createReq := map[string]interface{}{
				"name": "Alice New Default Candidate",
			}
			w := client.Post(t, "/api/product/wishlist", createReq)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			newWishlist := helpers.GetResponseData(t, response, "wishlist")
			newWishlistID := uint(newWishlist["id"].(float64))

			// Make it default
			updateReq := map[string]interface{}{
				"isDefault": true,
			}
			w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", newWishlistID), updateReq)
			helpers.AssertSuccessResponse(t, w, http.StatusOK)

			// Try to delete it (now it's default)
			w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", newWishlistID))
			helpers.AssertErrorResponse(t, w, http.StatusUnprocessableEntity)
		},
	)

	// ============================================================================
	// Integration Scenarios
	// ============================================================================

	t.Run("INT-001: Create, add items, then delete - verify cascade delete", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create a new wishlist
		createReq := map[string]interface{}{
			"name": "Alice Cascade Delete Test Wishlist",
		}
		w := client.Post(t, "/api/product/wishlist", createReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		newWishlist := helpers.GetResponseData(t, response, "wishlist")
		newWishlistID := uint(newWishlist["id"].(float64))

		// Insert wishlist items directly into database (bypass API since no products seeded)
		// Using fake variant IDs - cascade delete will work regardless of FK constraints
		// because we're testing the cascade on wishlist_item -> wishlist relationship
		item1 := entity.WishlistItem{
			WishlistID: newWishlistID,
			VariantID:  99901, // Fake variant ID for testing
		}
		item2 := entity.WishlistItem{
			WishlistID: newWishlistID,
			VariantID:  99902, // Fake variant ID for testing
		}
		item3 := entity.WishlistItem{
			WishlistID: newWishlistID,
			VariantID:  99903, // Fake variant ID for testing
		}

		// Insert items directly - note: we need to disable FK check or use real variant IDs
		// For this test, we'll query existing variants from DB if available
		var existingVariantIDs []uint
		containers.DB.Raw("SELECT id FROM product_variant LIMIT 3").Scan(&existingVariantIDs)

		var insertedItemIDs []uint
		if len(existingVariantIDs) >= 1 {
			// Use real variant IDs if available
			for _, variantID := range existingVariantIDs {
				item := entity.WishlistItem{
					WishlistID: newWishlistID,
					VariantID:  variantID,
				}
				err := containers.DB.Create(&item).Error
				require.NoError(t, err, "Should insert wishlist item")
				insertedItemIDs = append(insertedItemIDs, item.ID)
			}
		} else {
			// No variants exist, create items without FK constraint (raw SQL)
			containers.DB.Exec(
				"INSERT INTO wishlist_item (wishlist_id, variant_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW())",
				item1.WishlistID, 1,
			)
			containers.DB.Exec(
				"INSERT INTO wishlist_item (wishlist_id, variant_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW())",
				item2.WishlistID, 2,
			)
			containers.DB.Exec(
				"INSERT INTO wishlist_item (wishlist_id, variant_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW())",
				item3.WishlistID, 3,
			)
		}

		// Verify items exist in database before deletion
		var itemCountBefore int64
		containers.DB.Model(&entity.WishlistItem{}).
			Where("wishlist_id = ?", newWishlistID).
			Count(&itemCountBefore)
		require.Greater(
			t,
			itemCountBefore,
			int64(0),
			"Should have items in database before deletion",
		)
		t.Logf("Wishlist %d has %d items before deletion", newWishlistID, itemCountBefore)

		// Delete the wishlist via API
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", newWishlistID))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify wishlist is gone via API
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", newWishlistID))
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)

		// CRITICAL: Verify cascade delete - items should be automatically deleted
		var itemCountAfter int64
		containers.DB.Model(&entity.WishlistItem{}).
			Where("wishlist_id = ?", newWishlistID).
			Count(&itemCountAfter)
		assert.Equal(
			t,
			int64(0),
			itemCountAfter,
			"All wishlist items should be cascade deleted when wishlist is deleted",
		)
		t.Logf("Wishlist items after deletion: %d (expected 0)", itemCountAfter)
	})

	t.Run("INT-002: Delete affects only target wishlist not others", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create two wishlists
		createReq := map[string]interface{}{
			"name": "Alice Keep Wishlist",
		}
		w := client.Post(t, "/api/product/wishlist", createReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		keepWishlist := helpers.GetResponseData(t, response, "wishlist")
		keepWishlistID := uint(keepWishlist["id"].(float64))

		createReq = map[string]interface{}{
			"name": "Alice Delete Wishlist",
		}
		w = client.Post(t, "/api/product/wishlist", createReq)
		response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		deleteWishlist := helpers.GetResponseData(t, response, "wishlist")
		deleteWishlistID := uint(deleteWishlist["id"].(float64))

		// Delete one wishlist
		w = client.Delete(t, fmt.Sprintf("/api/product/wishlist/%d", deleteWishlistID))
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify the other wishlist still exists
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", keepWishlistID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			"Alice Keep Wishlist",
			wishlist["name"],
			"Kept wishlist should be unchanged",
		)
	})
}
