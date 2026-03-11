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

// TestMoveWishlistItem tests the POST /api/product/wishlist/:id/item/:itemId/move endpoint
//
// Request Body:
//   - targetWishlistId: uint (required) - ID of the destination wishlist
//
// Business Rules:
//   - User can only move items from their own wishlists
//   - User can only move items to their own wishlists
//   - Cannot move item to the same wishlist (returns 400)
//   - Cannot move item if same variant already exists in target wishlist (returns 409)
//   - Item must exist and belong to the source wishlist
func TestMoveWishlistItem(t *testing.T) {
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

	// Get variant IDs from seed data for testing
	variantID1 := uint(1)  // iPhone variant
	variantID2 := uint(5)  // Samsung variant
	variantID3 := uint(9)  // Nike T-Shirt variant
	variantID4 := uint(12) // MacBook variant
	variantID5 := uint(14) // Running shoes variant

	// ============================================================================
	// Setup: Create wishlists and items for testing
	// ============================================================================

	// Login as customer (Alice - user 5)
	aliceToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
	client.SetToken(aliceToken)

	// Create source wishlist for Alice
	createReq := map[string]interface{}{
		"name": "Alice Source Wishlist",
	}
	w := client.Post(t, "/api/product/wishlist", createReq)
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceSourceWishlist := helpers.GetResponseData(t, response, "wishlist")
	aliceSourceWishlistID := uint(aliceSourceWishlist["id"].(float64))

	// Create target wishlist for Alice
	createReq = map[string]interface{}{
		"name": "Alice Target Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceTargetWishlist := helpers.GetResponseData(t, response, "wishlist")
	aliceTargetWishlistID := uint(aliceTargetWishlist["id"].(float64))

	// Create third wishlist for Alice (for additional tests)
	createReq = map[string]interface{}{
		"name": "Alice Third Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceThirdWishlist := helpers.GetResponseData(t, response, "wishlist")
	aliceThirdWishlistID := uint(aliceThirdWishlist["id"].(float64))

	// Add items to Alice's source wishlist
	addReq := map[string]interface{}{"variantId": variantID1}
	w = client.Post(t, fmt.Sprintf("/api/product/wishlist/%d/item", aliceSourceWishlistID), addReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	item1 := helpers.GetResponseData(t, response, "wishlistItem")
	aliceItemID1 := uint(item1["id"].(float64))

	addReq = map[string]interface{}{"variantId": variantID2}
	w = client.Post(t, fmt.Sprintf("/api/product/wishlist/%d/item", aliceSourceWishlistID), addReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	item2 := helpers.GetResponseData(t, response, "wishlistItem")
	aliceItemID2 := uint(item2["id"].(float64))

	addReq = map[string]interface{}{"variantId": variantID3}
	w = client.Post(t, fmt.Sprintf("/api/product/wishlist/%d/item", aliceSourceWishlistID), addReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	item3 := helpers.GetResponseData(t, response, "wishlistItem")
	aliceItemID3 := uint(item3["id"].(float64))

	// Add item to target wishlist (for duplicate test)
	addReq = map[string]interface{}{"variantId": variantID4}
	w = client.Post(t, fmt.Sprintf("/api/product/wishlist/%d/item", aliceTargetWishlistID), addReq)
	helpers.AssertSuccessResponse(t, w, http.StatusCreated)

	// Login as Michael (user 6) and create wishlist for authorization tests
	michaelToken := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
	client.SetToken(michaelToken)

	createReq = map[string]interface{}{
		"name": "Michael Test Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	michaelWishlist := helpers.GetResponseData(t, response, "wishlist")
	michaelWishlistID := uint(michaelWishlist["id"].(float64))

	// Add item to Michael's wishlist
	addReq = map[string]interface{}{"variantId": variantID5}
	w = client.Post(t, fmt.Sprintf("/api/product/wishlist/%d/item", michaelWishlistID), addReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	michaelItem := helpers.GetResponseData(t, response, "wishlistItem")
	michaelItemID := uint(michaelItem["id"].(float64))

	// ============================================================================
	// Happy Path Scenarios
	// ============================================================================

	t.Run("HP-001: Move item to another wishlist successfully", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID1,
			),
			moveReq,
		)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Validate response
		item := helpers.GetResponseData(t, response, "wishlistItem")
		assert.NotNil(t, item["id"], "Item should have ID")
		assert.Equal(t, float64(variantID1), item["variantId"], "Variant ID should match")
	})

	t.Run("HP-002: Verify item moved from source to target wishlist", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Verify item is no longer in source wishlist
		var sourceCount int64
		containers.DB.Model(&entity.WishlistItem{}).
			Where("wishlist_id = ? AND id = ?", aliceSourceWishlistID, aliceItemID1).
			Count(&sourceCount)
		assert.Equal(t, int64(0), sourceCount, "Item should not be in source wishlist")

		// Verify item is now in target wishlist
		var targetCount int64
		containers.DB.Model(&entity.WishlistItem{}).
			Where("wishlist_id = ? AND id = ?", aliceTargetWishlistID, aliceItemID1).
			Count(&targetCount)
		assert.Equal(t, int64(1), targetCount, "Item should be in target wishlist")
	})

	t.Run("HP-003: Move multiple items sequentially", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Move second item to third wishlist
		moveReq := map[string]interface{}{
			"targetWishlistId": aliceThirdWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID2,
			),
			moveReq,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify item count in source wishlist decreased
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceSourceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		// Only item3 should remain (item1 moved in HP-001, item2 moved now)
		assert.Equal(
			t,
			float64(1),
			wishlist["itemCount"].(float64),
			"Source should have 1 item left",
		)

		// Verify item count in third wishlist increased
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceThirdWishlistID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist = helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			float64(1),
			wishlist["itemCount"].(float64),
			"Third wishlist should have 1 item",
		)
	})

	// ============================================================================
	// Negative Scenarios - Authentication
	// ============================================================================

	t.Run("NEG-AUTH-001: Move item without authentication returns 401", func(t *testing.T) {
		client.SetToken("")

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("NEG-AUTH-002: Move item with invalid token returns 401", func(t *testing.T) {
		client.SetToken("invalid-token-here")

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("NEG-AUTH-003: Move item with expired token returns 401", func(t *testing.T) {
		// Using a malformed JWT that looks expired
		client.SetToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDAwMDAwMDB9.invalid")

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	// ============================================================================
	// Negative Scenarios - Authorization
	// ============================================================================

	t.Run("NEG-AUTHZ-001: Seller role cannot move wishlist items", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-AUTHZ-002: Admin role cannot move wishlist items", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-AUTHZ-003: User cannot move item from another user's wishlist", func(t *testing.T) {
		// Login as Michael and try to move Alice's item
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": michaelWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-AUTHZ-004: User cannot move item to another user's wishlist", func(t *testing.T) {
		// Login as Alice and try to move to Michael's wishlist
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": michaelWishlistID, // Michael's wishlist
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run(
		"NEG-AUTHZ-005: User cannot move another user's item even within own wishlists",
		func(t *testing.T) {
			// Login as Alice and try to move Michael's item to Alice's wishlist
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			moveReq := map[string]interface{}{
				"targetWishlistId": aliceTargetWishlistID,
			}
			// Trying to move Michael's item from Michael's wishlist
			w := client.Post(
				t,
				fmt.Sprintf(
					"/api/product/wishlist/%d/item/%d/move",
					michaelWishlistID,
					michaelItemID,
				),
				moveReq,
			)

			helpers.AssertErrorResponse(t, w, http.StatusForbidden)
		},
	)

	// ============================================================================
	// Negative Scenarios - Validation
	// ============================================================================

	t.Run("NEG-VAL-001: Move item with missing targetWishlistId returns 400", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run(
		"NEG-VAL-002: Move item with invalid targetWishlistId type returns 400",
		func(t *testing.T) {
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			moveReq := map[string]interface{}{
				"targetWishlistId": "not-a-number",
			}
			w := client.Post(
				t,
				fmt.Sprintf(
					"/api/product/wishlist/%d/item/%d/move",
					aliceSourceWishlistID,
					aliceItemID3,
				),
				moveReq,
			)

			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		},
	)

	t.Run(
		"NEG-VAL-003: Move item with invalid source wishlist ID format returns 400",
		func(t *testing.T) {
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			moveReq := map[string]interface{}{
				"targetWishlistId": aliceTargetWishlistID,
			}
			w := client.Post(t, "/api/product/wishlist/abc/item/1/move", moveReq)

			helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
		},
	)

	t.Run("NEG-VAL-004: Move item with invalid item ID format returns 400", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/xyz/move", aliceSourceWishlistID),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run(
		"NEG-VAL-005: Move item from non-existent source wishlist returns 404",
		func(t *testing.T) {
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			moveReq := map[string]interface{}{
				"targetWishlistId": aliceTargetWishlistID,
			}
			w := client.Post(t, "/api/product/wishlist/99999/item/1/move", moveReq)

			helpers.AssertErrorResponse(t, w, http.StatusNotFound)
		},
	)

	t.Run("NEG-VAL-006: Move non-existent item returns 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/99999/move", aliceSourceWishlistID),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("NEG-VAL-007: Move to non-existent target wishlist returns 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": uint(99999),
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// Negative Scenarios - Business Logic
	// ============================================================================

	t.Run("NEG-BUS-001: Move item to the same wishlist returns 400", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceSourceWishlistID, // Same as source
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run(
		"NEG-BUS-002: Move item when same variant exists in target returns 409 Conflict",
		func(t *testing.T) {
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			// Add same variant to both wishlists
			// First, add variant to source wishlist
			addReq := map[string]interface{}{"variantId": variantID4}
			w := client.Post(
				t,
				fmt.Sprintf("/api/product/wishlist/%d/item", aliceSourceWishlistID),
				addReq,
			)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			newItem := helpers.GetResponseData(t, response, "wishlistItem")
			newItemID := uint(newItem["id"].(float64))

			// Try to move it
			// variantID4 is already in target wishlist (added in setup)
			moveReq := map[string]interface{}{
				"targetWishlistId": aliceTargetWishlistID,
			}
			w = client.Post(
				t,
				fmt.Sprintf(
					"/api/product/wishlist/%d/item/%d/move",
					aliceSourceWishlistID,
					newItemID,
				),
				moveReq,
			)

			helpers.AssertErrorResponse(t, w, http.StatusConflict)
		},
	)

	t.Run(
		"NEG-BUS-003: Move item that doesn't belong to source wishlist returns 404",
		func(t *testing.T) {
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			// Get item from target wishlist and try to move from source
			var targetItem entity.WishlistItem
			err := containers.DB.Where("wishlist_id = ?", aliceTargetWishlistID).
				First(&targetItem).
				Error
			require.NoError(t, err, "Should find target wishlist item")

			moveReq := map[string]interface{}{
				"targetWishlistId": aliceThirdWishlistID,
			}
			// Try to move target's item as if it was in source wishlist
			w := client.Post(
				t,
				fmt.Sprintf(
					"/api/product/wishlist/%d/item/%d/move",
					aliceSourceWishlistID,
					targetItem.ID,
				),
				moveReq,
			)

			helpers.AssertErrorResponse(t, w, http.StatusNotFound)
		},
	)

	// ============================================================================
	// Edge Cases
	// ============================================================================

	t.Run("EDGE-001: Move item with ID zero returns error", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/0/move", aliceSourceWishlistID),
			moveReq,
		)

		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
			"Should reject item ID 0")
	})

	t.Run("EDGE-002: Move item with very large item ID returns 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/9999999999/move", aliceSourceWishlistID),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("EDGE-003: Move with targetWishlistId zero returns error", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": 0,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		// Should return 400 (validation) or 404 (not found)
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
			"Should reject targetWishlistId 0")
	})

	t.Run("EDGE-004: Move with negative targetWishlistId returns 400", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": -1,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EDGE-005: Move with very large targetWishlistId returns 404", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": 9999999999,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/%d/move",
				aliceSourceWishlistID,
				aliceItemID3,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	// ============================================================================
	// Security Scenarios
	// ============================================================================

	t.Run("SEC-001: SQL injection in source wishlist ID is prevented", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(t, "/api/product/wishlist/1;DROP TABLE wishlist;--/item/1/move", moveReq)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("SEC-002: SQL injection in item ID is prevented", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(
			t,
			fmt.Sprintf(
				"/api/product/wishlist/%d/item/1;DROP TABLE wishlist_item;--/move",
				aliceSourceWishlistID,
			),
			moveReq,
		)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run(
		"SEC-003: User isolation - cross-user move is blocked and data unchanged",
		func(t *testing.T) {
			// Get initial count of Michael's wishlist items
			var initialCount int64
			containers.DB.Model(&entity.WishlistItem{}).
				Where("wishlist_id = ?", michaelWishlistID).
				Count(&initialCount)

			// Login as Alice and try to move Michael's item
			token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
			client.SetToken(token)

			moveReq := map[string]interface{}{
				"targetWishlistId": aliceTargetWishlistID,
			}
			w := client.Post(
				t,
				fmt.Sprintf(
					"/api/product/wishlist/%d/item/%d/move",
					michaelWishlistID,
					michaelItemID,
				),
				moveReq,
			)

			helpers.AssertErrorResponse(t, w, http.StatusForbidden)

			// Verify Michael's wishlist items unchanged
			var finalCount int64
			containers.DB.Model(&entity.WishlistItem{}).
				Where("wishlist_id = ?", michaelWishlistID).
				Count(&finalCount)
			assert.Equal(t, initialCount, finalCount, "Michael's wishlist should be unchanged")

			// Verify item is still in Michael's wishlist
			var item entity.WishlistItem
			err := containers.DB.Where("id = ?", michaelItemID).First(&item).Error
			require.NoError(t, err)
			assert.Equal(
				t,
				michaelWishlistID,
				item.WishlistID,
				"Item should still be in Michael's wishlist",
			)
		},
	)

	t.Run("SEC-004: Path traversal in wishlist ID is prevented", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		moveReq := map[string]interface{}{
			"targetWishlistId": aliceTargetWishlistID,
		}
		w := client.Post(t, "/api/product/wishlist/../../../etc/passwd/item/1/move", moveReq)

		// Should return 400 or 404 (path traversal blocked)
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
			"Path traversal should be blocked")
	})

	// ============================================================================
	// Integration Scenarios
	// ============================================================================

	t.Run("INT-001: Full move lifecycle - verify source and target counts", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create fresh wishlists for this test
		createReq := map[string]interface{}{"name": "Integration Source"}
		w := client.Post(t, "/api/product/wishlist", createReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		sourceData := helpers.GetResponseData(t, response, "wishlist")
		sourceID := uint(sourceData["id"].(float64))

		createReq = map[string]interface{}{"name": "Integration Target"}
		w = client.Post(t, "/api/product/wishlist", createReq)
		response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		targetData := helpers.GetResponseData(t, response, "wishlist")
		targetID := uint(targetData["id"].(float64))

		// Add item to source
		addReq := map[string]interface{}{"variantId": uint(18)}
		w = client.Post(t, fmt.Sprintf("/api/product/wishlist/%d/item", sourceID), addReq)
		response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		itemData := helpers.GetResponseData(t, response, "wishlistItem")
		itemID := uint(itemData["id"].(float64))

		// Verify initial counts
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", sourceID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		sourceWishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			float64(1),
			sourceWishlist["itemCount"].(float64),
			"Source should have 1 item",
		)

		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", targetID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		targetWishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			float64(0),
			targetWishlist["itemCount"].(float64),
			"Target should have 0 items",
		)

		// Move item
		moveReq := map[string]interface{}{"targetWishlistId": targetID}
		w = client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/%d/move", sourceID, itemID),
			moveReq,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify final counts
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", sourceID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		sourceWishlist = helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			float64(0),
			sourceWishlist["itemCount"].(float64),
			"Source should have 0 items after move",
		)

		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", targetID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		targetWishlist = helpers.GetResponseData(t, response, "wishlist")
		assert.Equal(
			t,
			float64(1),
			targetWishlist["itemCount"].(float64),
			"Target should have 1 item after move",
		)
	})

	t.Run("INT-002: Move item then move it back", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create fresh wishlists
		createReq := map[string]interface{}{"name": "Round Trip Source"}
		w := client.Post(t, "/api/product/wishlist", createReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		sourceData := helpers.GetResponseData(t, response, "wishlist")
		sourceID := uint(sourceData["id"].(float64))

		createReq = map[string]interface{}{"name": "Round Trip Target"}
		w = client.Post(t, "/api/product/wishlist", createReq)
		response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		targetData := helpers.GetResponseData(t, response, "wishlist")
		targetID := uint(targetData["id"].(float64))

		// Add item to source
		addReq := map[string]interface{}{"variantId": uint(19)}
		w = client.Post(t, fmt.Sprintf("/api/product/wishlist/%d/item", sourceID), addReq)
		response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		itemData := helpers.GetResponseData(t, response, "wishlistItem")
		itemID := uint(itemData["id"].(float64))

		// Move to target
		moveReq := map[string]interface{}{"targetWishlistId": targetID}
		w = client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/%d/move", sourceID, itemID),
			moveReq,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Move back to source
		moveReq = map[string]interface{}{"targetWishlistId": sourceID}
		w = client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/%d/move", targetID, itemID),
			moveReq,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify item is back in source
		var item entity.WishlistItem
		err := containers.DB.Where("id = ?", itemID).First(&item).Error
		require.NoError(t, err)
		assert.Equal(t, sourceID, item.WishlistID, "Item should be back in source wishlist")
	})

	t.Run("INT-003: Move items between multiple wishlists", func(t *testing.T) {
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create 3 wishlists
		var wishlistIDs []uint
		for i := 1; i <= 3; i++ {
			createReq := map[string]interface{}{"name": fmt.Sprintf("Multi Move WL %d", i)}
			w := client.Post(t, "/api/product/wishlist", createReq)
			response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
			data := helpers.GetResponseData(t, response, "wishlist")
			wishlistIDs = append(wishlistIDs, uint(data["id"].(float64)))
		}

		// Add item to first wishlist
		addReq := map[string]interface{}{"variantId": uint(20)}
		w := client.Post(t, fmt.Sprintf("/api/product/wishlist/%d/item", wishlistIDs[0]), addReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		itemData := helpers.GetResponseData(t, response, "wishlistItem")
		itemID := uint(itemData["id"].(float64))

		// Move: WL1 -> WL2
		moveReq := map[string]interface{}{"targetWishlistId": wishlistIDs[1]}
		w = client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/%d/move", wishlistIDs[0], itemID),
			moveReq,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Move: WL2 -> WL3
		moveReq = map[string]interface{}{"targetWishlistId": wishlistIDs[2]}
		w = client.Post(
			t,
			fmt.Sprintf("/api/product/wishlist/%d/item/%d/move", wishlistIDs[1], itemID),
			moveReq,
		)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify item is in WL3
		var item entity.WishlistItem
		err := containers.DB.Where("id = ?", itemID).First(&item).Error
		require.NoError(t, err)
		assert.Equal(t, wishlistIDs[2], item.WishlistID, "Item should be in third wishlist")
	})
}
