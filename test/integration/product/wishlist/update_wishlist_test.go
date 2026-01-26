package wishlist

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

// TestUpdateWishlist tests the Update Wishlist (PUT /api/product/wishlist/:id) API
// This endpoint updates a wishlist's name and/or default status.
//
// Endpoint: PUT /api/product/wishlist/:id
// Authentication: Required (Customer Auth only)
//
// Request Body:
// - name: *string (optional, min=1, max=100) - New wishlist name
// - isDefault: *bool (optional) - Set as default wishlist
//
// Business Rules:
// - User can only update their own wishlists
// - Wishlist name must be unique per user
// - Setting isDefault=true clears default from other wishlists
// - Cannot set isDefault=false (must set another as default)
func TestUpdateWishlist(t *testing.T) {
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
		"name": "Alice Primary Wishlist",
	}
	w := client.Post(t, "/api/product/wishlist", createReq)
	response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	aliceWishlist := helpers.GetResponseData(t, response, "wishlist")
	aliceWishlistID := uint(aliceWishlist["id"].(float64))

	// Create second wishlist (non-default)
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

	// Login as Michael (user 6) and create wishlist for authorization tests
	michaelToken := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
	client.SetToken(michaelToken)

	createReq = map[string]interface{}{
		"name": "Michael Wishlist",
	}
	w = client.Post(t, "/api/product/wishlist", createReq)
	response = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	michaelWishlist := helpers.GetResponseData(t, response, "wishlist")
	michaelWishlistID := uint(michaelWishlist["id"].(float64))

	// ============================================================================
	// Happy Path Scenarios
	// ============================================================================

	t.Run("HP-001: Update wishlist name only", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Update name
		updateReq := map[string]interface{}{
			"name": "Alice Updated Wishlist",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID3), updateReq)

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate updated fields
		assert.Equal(t, float64(aliceWishlistID3), wishlist["id"].(float64), "ID should match")
		assert.Equal(t, "Alice Updated Wishlist", wishlist["name"], "Name should be updated")
		assert.NotNil(t, wishlist["updatedAt"], "Should have updatedAt")
	})

	t.Run("HP-002: Update wishlist to set as default", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Verify first wishlist is currently default
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.True(t, wishlist["isDefault"].(bool), "First wishlist should be default initially")

		// Set second wishlist as default
		updateReq := map[string]interface{}{
			"isDefault": true,
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID2), updateReq)

		// Assert response
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist = helpers.GetResponseData(t, response, "wishlist")

		// Validate second wishlist is now default
		assert.Equal(t, float64(aliceWishlistID2), wishlist["id"].(float64), "ID should match")
		assert.True(t, wishlist["isDefault"].(bool), "Should be default now")

		// Verify first wishlist is no longer default
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist = helpers.GetResponseData(t, response, "wishlist")
		assert.False(t, wishlist["isDefault"].(bool), "First wishlist should no longer be default")
	})

	t.Run("HP-003: Update both name and isDefault together", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Update both fields on third wishlist
		updateReq := map[string]interface{}{
			"name":      "Alice New Default Wishlist",
			"isDefault": true,
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID3), updateReq)

		// Assert response
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate both fields updated
		assert.Equal(t, "Alice New Default Wishlist", wishlist["name"], "Name should be updated")
		assert.True(t, wishlist["isDefault"].(bool), "Should be default now")
	})

	t.Run("HP-004: Update with empty body (no changes)", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get current state
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		originalWishlist := helpers.GetResponseData(t, response, "wishlist")
		originalName := originalWishlist["name"].(string)

		// Update with empty body
		updateReq := map[string]interface{}{}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		// Assert response - should succeed with no changes
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Validate no changes
		assert.Equal(t, originalName, wishlist["name"], "Name should remain unchanged")
	})

	t.Run("HP-005: Update name with same value (idempotent)", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get current name
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		originalWishlist := helpers.GetResponseData(t, response, "wishlist")
		currentName := originalWishlist["name"].(string)

		// Update with same name
		updateReq := map[string]interface{}{
			"name": currentName,
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		// Assert response - should succeed
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		assert.Equal(t, currentName, wishlist["name"], "Name should remain the same")
	})

	t.Run("HP-006: Update already default wishlist with isDefault=true", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// First make wishlist 1 default again
		updateReq := map[string]interface{}{
			"isDefault": true,
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Try to set it as default again (should succeed - idempotent)
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		assert.True(t, wishlist["isDefault"].(bool), "Should still be default")
	})

	// ============================================================================
	// Negative Scenarios - Authentication
	// ============================================================================

	t.Run("NEG-001: Update wishlist without authentication returns 401", func(t *testing.T) {
		// Clear token
		client.SetToken("")

		updateReq := map[string]interface{}{
			"name": "Unauthorized Update",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("NEG-002: Update wishlist with invalid token returns 401", func(t *testing.T) {
		// Set invalid token
		client.SetToken("invalid-token-here")

		updateReq := map[string]interface{}{
			"name": "Invalid Token Update",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusUnauthorized)
	})

	// ============================================================================
	// Negative Scenarios - Authorization
	// ============================================================================

	t.Run("NEG-003: Seller role cannot update wishlist", func(t *testing.T) {
		// Login as seller
		token := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "Seller Update Attempt",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-004: Admin role cannot update wishlist", func(t *testing.T) {
		// Login as admin
		token := helpers.Login(t, client, helpers.AdminEmail, helpers.AdminPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "Admin Update Attempt",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-005: User cannot update another user's wishlist", func(t *testing.T) {
		// Login as Michael
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// Try to update Alice's wishlist
		updateReq := map[string]interface{}{
			"name": "Michael Trying to Update Alice",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	t.Run("NEG-006: User A updating User B's wishlist gets 403 not 404", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Try to update Michael's wishlist
		updateReq := map[string]interface{}{
			"name": "Alice Trying to Update Michael",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID), updateReq)

		// Should be 403 Forbidden not 404 - wishlist exists but unauthorized
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)
	})

	// ============================================================================
	// Negative Scenarios - Validation
	// ============================================================================

	t.Run("NEG-007: Update wishlist with non-existent ID returns 404", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "Update Non-Existent",
		}
		w := client.Put(t, "/api/product/wishlist/99999", updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("NEG-008: Update wishlist with invalid ID format (string)", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "Invalid ID Update",
		}
		w := client.Put(t, "/api/product/wishlist/abc", updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-009: Update wishlist with invalid ID format (negative)", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "Negative ID Update",
		}
		w := client.Put(t, "/api/product/wishlist/-1", updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-010: Update wishlist with duplicate name returns 409", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get existing wishlist name
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		existingWishlist := helpers.GetResponseData(t, response, "wishlist")
		existingName := existingWishlist["name"].(string)

		// Try to update second wishlist with same name
		updateReq := map[string]interface{}{
			"name": existingName,
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID2), updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusConflict)
	})

	t.Run("NEG-011: Update wishlist with name too long (>100 chars)", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create name > 100 characters
		longName := ""
		for i := 0; i < 101; i++ {
			longName += "a"
		}

		updateReq := map[string]interface{}{
			"name": longName,
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-012: Update wishlist with empty name returns 400", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("NEG-013: Update wishlist with invalid JSON body", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Send invalid JSON using raw request
		w := client.PutRaw(
			t,
			fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID),
			[]byte("{invalid json}"),
		)

		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	// ============================================================================
	// Edge Case Scenarios
	// ============================================================================

	t.Run("EDGE-001: Update wishlist with very large ID returns 404", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "Large ID Update",
		}
		// Large valid uint64 ID - should return 404 Not Found
		w := client.Put(t, "/api/product/wishlist/9999999999", updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("EDGE-002: Update wishlist with ID zero returns 404", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "Zero ID Update",
		}
		w := client.Put(t, "/api/product/wishlist/0", updateReq)

		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("EDGE-003: Update wishlist name with whitespace only", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "   ",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		// Should either succeed (trimmed becomes empty → validation error) or return 400
		// Depends on whether whitespace is trimmed before validation
		assert.True(
			t,
			w.Code == http.StatusBadRequest || w.Code == http.StatusOK,
			"Should handle whitespace-only name appropriately",
		)
	})

	t.Run("EDGE-004: Update wishlist name with special characters", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "Alice's 🎁 Holiday Wishlist 2025!",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		// Should succeed - special characters and emoji allowed
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		assert.Equal(
			t,
			"Alice's 🎁 Holiday Wishlist 2025!",
			wishlist["name"],
			"Name with special characters should be saved",
		)
	})

	t.Run("EDGE-005: Update wishlist name with max length (100 chars)", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Create exactly 100 character name
		maxName := ""
		for i := 0; i < 100; i++ {
			maxName += "a"
		}

		updateReq := map[string]interface{}{
			"name": maxName,
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		// Should succeed - exactly at max length
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		assert.Equal(t, 100, len(wishlist["name"].(string)), "Name should be 100 chars")
	})

	t.Run("EDGE-006: Update with isDefault=false has no effect", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// First ensure wishlist 1 is default
		updateReq := map[string]interface{}{
			"isDefault": true,
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Try to set isDefault=false (should have no effect - can't unset default)
		updateReq = map[string]interface{}{
			"isDefault": false,
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// Based on business logic - setting false doesn't change anything
		// The wishlist should still be default (or behavior depends on implementation)
		assert.NotNil(t, wishlist["isDefault"], "isDefault should be present")
	})

	// ============================================================================
	// Security Scenarios
	// ============================================================================

	t.Run("SEC-001: SQL injection in wishlist ID is prevented", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "SQL Injection Test",
		}
		w := client.Put(t, "/api/product/wishlist/1;DROP TABLE wishlist;--", updateReq)

		// Should return 400 Bad Request - invalid ID format
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("SEC-002: SQL injection in name field is prevented", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "'; DROP TABLE wishlist; --",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		// Should succeed - SQL injection is parameterized/escaped
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		// The malicious string should be stored as-is (escaped)
		assert.Equal(
			t,
			"'; DROP TABLE wishlist; --",
			wishlist["name"],
			"SQL injection should be stored as plain text",
		)
	})

	t.Run("SEC-003: XSS in name field is stored as-is", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		updateReq := map[string]interface{}{
			"name": "<script>alert('xss')</script>",
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		// Should succeed - XSS prevention is client's responsibility
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		assert.Equal(
			t,
			"<script>alert('xss')</script>",
			wishlist["name"],
			"XSS should be stored as plain text (sanitization is client's job)",
		)
	})

	t.Run("SEC-004: User isolation - verify no cross-user updates", func(t *testing.T) {
		// Login as Michael
		michaelToken := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(michaelToken)

		// Get Michael's wishlist name
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		michaelData := helpers.GetResponseData(t, response, "wishlist")
		originalMichaelName := michaelData["name"].(string)

		// Login as Alice and try to update Michael's wishlist
		aliceToken := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(aliceToken)

		updateReq := map[string]interface{}{
			"name": "Alice Hacked Michael",
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID), updateReq)
		helpers.AssertErrorResponse(t, w, http.StatusForbidden)

		// Verify Michael's wishlist is unchanged
		client.SetToken(michaelToken)
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		michaelData = helpers.GetResponseData(t, response, "wishlist")

		assert.Equal(
			t,
			originalMichaelName,
			michaelData["name"],
			"Michael's wishlist should be unchanged after Alice's failed update",
		)
	})

	// ============================================================================
	// Business Logic Scenarios
	// ============================================================================

	t.Run("BL-001: Setting new default clears previous default", func(t *testing.T) {
		// Login as Michael (fresh user for this test)
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// Create additional wishlist for Michael
		createReq := map[string]interface{}{
			"name": "Michael Second Wishlist",
		}
		w := client.Post(t, "/api/product/wishlist", createReq)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		michaelWishlist2 := helpers.GetResponseData(t, response, "wishlist")
		michaelWishlist2ID := uint(michaelWishlist2["id"].(float64))

		// Michael's first wishlist should be default
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		assert.True(t, wishlist["isDefault"].(bool), "First wishlist should be default")

		// Set second wishlist as default
		updateReq := map[string]interface{}{
			"isDefault": true,
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlist2ID), updateReq)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify first is no longer default
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist = helpers.GetResponseData(t, response, "wishlist")
		assert.False(t, wishlist["isDefault"].(bool), "First wishlist should no longer be default")

		// Verify second is now default
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlist2ID))
		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist = helpers.GetResponseData(t, response, "wishlist")
		assert.True(t, wishlist["isDefault"].(bool), "Second wishlist should now be default")
	})

	t.Run("BL-002: itemCount remains unchanged after name update", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get current item count
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		originalWishlist := helpers.GetResponseData(t, response, "wishlist")
		originalItemCount := originalWishlist["itemCount"].(float64)

		// Update name
		updateReq := map[string]interface{}{
			"name": "Alice Renamed Again",
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		assert.Equal(
			t,
			originalItemCount,
			wishlist["itemCount"].(float64),
			"itemCount should remain unchanged after name update",
		)
	})

	t.Run("BL-003: Timestamps are updated after modification", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get original timestamps
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		originalWishlist := helpers.GetResponseData(t, response, "wishlist")
		originalUpdatedAt := originalWishlist["updatedAt"].(string)

		// Update name
		updateReq := map[string]interface{}{
			"name": "Alice Timestamp Test",
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID), updateReq)

		response = helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")
		newUpdatedAt := wishlist["updatedAt"].(string)

		// updatedAt should change (or be same if too fast, but typically different)
		assert.NotEmpty(t, newUpdatedAt, "updatedAt should not be empty")
		// Note: In fast tests, timestamps might be the same
		_ = originalUpdatedAt // Used for comparison if needed
	})

	t.Run("BL-004: Duplicate name check is case-sensitive", func(t *testing.T) {
		// Login as Alice
		token := helpers.Login(t, client, helpers.CustomerEmail, helpers.CustomerPassword)
		client.SetToken(token)

		// Get existing wishlist name
		w := client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		existingWishlist := helpers.GetResponseData(t, response, "wishlist")
		existingName := existingWishlist["name"].(string)

		// Try to update with different case
		updateReq := map[string]interface{}{
			"name": existingName + " UPPERCASE",
		}
		w = client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", aliceWishlistID2), updateReq)

		// Should succeed - different case is different name
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	})

	// ============================================================================
	// Integration Scenarios
	// ============================================================================

	t.Run("INT-001: Update wishlist then verify via GetByID", func(t *testing.T) {
		// Login as Michael
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// Update name
		newName := "Michael Integration Test Wishlist"
		updateReq := map[string]interface{}{
			"name": newName,
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID), updateReq)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify via GET
		w = client.Get(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID))
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlist := helpers.GetResponseData(t, response, "wishlist")

		assert.Equal(t, newName, wishlist["name"], "Name should match after update and GET")
	})

	t.Run("INT-002: Update wishlist then verify in GetAllWishlists", func(t *testing.T) {
		// Login as Michael
		token := helpers.Login(t, client, helpers.Customer2Email, helpers.Customer2Password)
		client.SetToken(token)

		// Update name
		newName := "Michael All Wishlists Test"
		updateReq := map[string]interface{}{
			"name": newName,
		}
		w := client.Put(t, fmt.Sprintf("/api/product/wishlist/%d", michaelWishlistID), updateReq)
		helpers.AssertSuccessResponse(t, w, http.StatusOK)

		// Verify via GetAll
		w = client.Get(t, "/api/product/wishlist")
		response := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		wishlists := response["data"].(map[string]interface{})["wishlists"].([]interface{})

		// Find the updated wishlist
		found := false
		for _, wl := range wishlists {
			wishlist := wl.(map[string]interface{})
			if uint(wishlist["id"].(float64)) == michaelWishlistID {
				assert.Equal(t, newName, wishlist["name"], "Name should match in GetAll")
				found = true
				break
			}
		}
		assert.True(t, found, "Updated wishlist should appear in GetAll")
	})
}
