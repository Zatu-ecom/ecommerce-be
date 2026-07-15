package get_product_by_id

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

// ============================================================================
// SP-RV-01: Unauthenticated user → product returns 200, no recording
// ============================================================================

// TestPublicUser_NoRecordCreated verifies that unauthenticated (public) users
// do NOT trigger recently viewed recording (US2 — Role Restriction).
//
// Setup: No auth token, X-Seller-ID set to 2, view product 1.
// Expect: HTTP 200 with product data, zero recently viewed records.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestPublicUser_NoRecordCreated() {
	// Create a public client (no token)
	publicClient := helpers.NewAPIClient(s.server)
	publicClient.SetHeader("X-Seller-ID", "2")

	// View product as public user
	w := publicClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify NO record was created for any user
	totalCount := s.countByUser(helpers.CustomerUserID)
	s.Assert().Equal(int64(0), totalCount,
		"Public user viewing a product should NOT create a recently viewed record")

	// Verify no records exist at all
	s.Assert().Equal(int64(0), s.countAllRecentlyViewed(),
		"No recently viewed records should exist for unauthenticated requests")
}

// ============================================================================
// SP-RV-02: Non-existent product → 404, no recording
// ============================================================================

// TestNonExistentProduct_NoRecordCreated verifies that a 404 product response
// does NOT create a recently viewed record.
//
// Setup: Customer tries to view product 99999.
// Expect: HTTP 404, zero recently viewed records.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestNonExistentProduct_NoRecordCreated() {
	// Try to view non-existent product
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 99999))

	// Verify 404 response
	s.Assert().Equal(http.StatusNotFound, w.Code, "Non-existent product should return 404")

	// Verify NO record was created
	totalCount := s.countByUser(helpers.CustomerUserID)
	s.Assert().Equal(int64(0), totalCount,
		"Non-existent product should NOT create a recently viewed record")
}

// ============================================================================
// SP-RV-03: Non-existent seller via PublicAPIAuth → 403, no recording
// ============================================================================

// TestWrongSellerID_NoRecordCreated verifies that a seller ID that does not
// exist returns 403 Forbidden and does not create a recently viewed record.
//
// This test uses PublicAPIAuth (no JWT token) with a non-existent X-Seller-ID.
// When a JWT token IS present, the seller_id from JWT claims takes precedence
// over X-Seller-ID header, so the only way to get a seller validation error
// is to use the public API path without JWT.
//
// Setup: Public client (no token) with non-existent X-Seller-ID = 99999 views product 1.
// Expect: HTTP 403 due to seller validation failure, zero recently viewed records.
func (s *RecentlyViewedTestSuite) TestWrongSellerID_NoRecordCreated() {
	// Create a public client (no token) with non-existent X-Seller-ID
	wrongSellerClient := helpers.NewAPIClient(s.server)
	wrongSellerClient.SetHeader("X-Seller-ID", "99999") // This seller does not exist

	// View product 1 with non-existent seller — PublicAPIAuth will reject
	w := wrongSellerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))

	// Verify 403 Forbidden due to seller validation failure
	s.Assert().Equal(http.StatusForbidden, w.Code,
		"Non-existent X-Seller-ID should return 403 via PublicAPIAuth")

	// Verify NO record was created
	totalCount := s.countAllRecentlyViewed()
	s.Assert().Equal(int64(0), totalCount,
		"Non-existent seller should NOT create a recently viewed record")
}

// ============================================================================
// SP-RV-04: Non-numeric product ID → 400, no recording
// ============================================================================

// TestInvalidProductID_NoRecordCreated verifies that a 400 validation error
// does NOT create a recently viewed record.
//
// Setup: Customer requests /api/product/abc.
// Expect: HTTP 400, zero recently viewed records.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestInvalidProductID_NoRecordCreated() {
	// Try non-numeric product ID
	w := s.customerClient.Get(s.T(), ProductAPIEndpoint+"/abc")

	// Verify 400 response
	s.Assert().Equal(http.StatusBadRequest, w.Code,
		"Non-numeric product ID should return 400")

	// Verify NO record was created
	totalCount := s.countByUser(helpers.CustomerUserID)
	s.Assert().Equal(int64(0), totalCount,
		"Invalid product ID should NOT create a recently viewed record")
}

// ============================================================================
// SP-RV-05: Multiple customers viewing distinct products → correct per-user counts
// ============================================================================

// TestMultipleDistinctProducts_NoDuplication verifies that two customers
// viewing different products get correct per-user recently viewed counts.
//
// Setup: Customer A (Alice, seller 2) views products 1,2,3.
//         Customer B (Michael, seller 3) views product 107.
// Expect: Customer A has 3 records, Customer B has 1 record, correct IDs.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestMultipleDistinctProducts_NoDuplication() {
	// Customer A views products 1, 2, 3 (all seller 2)
	for _, pid := range []uint{1, 2, 3} {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// Customer B views product 107 (belongs to seller 3 — Michael's seller)
	w := s.customer2Client.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 107))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify customer A has 3 records
	s.Assert().Equal(int64(3), s.countByUser(helpers.CustomerUserID),
		"Customer should have 3 recently viewed records")

	// Verify customer B has 1 record
	s.Assert().Equal(int64(1), s.countByUser(helpers.Customer2UserID),
		"Customer 2 should have 1 recently viewed record")

	// Verify the correct product IDs exist for customer A
	for _, pid := range []uint{1, 2, 3} {
		s.Assert().Equal(int64(1), s.countByUserAndProduct(helpers.CustomerUserID, pid),
			"Customer record for product %d should exist", pid)
	}

	// Verify customer B has product 107
	s.Assert().Equal(int64(1), s.countByUserAndProduct(helpers.Customer2UserID, 107),
		"Customer 2 record for product 107 should exist")
}

// ============================================================================
// SP-RV-06: Fire-and-Forget — product response is always HTTP 200 with full data
// ============================================================================

// TestRecordingFailure_ProductResponseUnchanged verifies the fire-and-forget
// guarantee (US3): the product response is always returned with full data
// regardless of any recording logic.
//
// Setup: Customer views a valid product.
// Expect: HTTP 200 with complete product response structure.
//
// NOTE: This test will PASS even without Phase 5, since handler currently
//
//	returns full product responses without recording.
//	Once Phase 5 is complete, this verifies the product response is
//	returned even if recording fails (fire-and-forget resilience).
func (s *RecentlyViewedTestSuite) TestRecordingFailure_ProductResponseUnchanged() {
	// View a valid product
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))

	// Product response must be HTTP 200 with full product data regardless
	response := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify complete product data is returned
	product := response["data"].(map[string]any)["product"].(map[string]any)
	s.Assert().Equal(float64(1), product["id"], "Product ID should be 1")
	s.Assert().Equal("iPhone 15 Pro", product["name"], "Product name should match")
	s.Assert().NotNil(product["category"], "Category should be present")
	s.Assert().NotNil(product["variants"], "Variants should be present")
	s.Assert().NotNil(product["options"], "Options should be present")

	// Verify the response structure is complete (all expected fields present)
	s.Assert().NotNil(product["hasVariants"], "hasVariants should be present")
	s.Assert().NotNil(product["allowPurchase"], "allowPurchase should be present")
	s.Assert().NotEmpty(product["createdAt"], "createdAt should be present")
	s.Assert().NotEmpty(product["updatedAt"], "updatedAt should be present")
}

// ============================================================================
// RV-GET-02 (T035): Unauthenticated request to GET recently viewed → 401
// ============================================================================

// TestGetRecentlyViewed_Unauthenticated verifies that an unauthenticated
// request to the recently viewed endpoint returns 401 Unauthorized.
//
// Setup: No auth token, call GET /api/product/recently-viewed.
// Expect: HTTP 401 with error response.
//
// NOTE: This test will FAIL until Phase 6 handler integration adds the GET endpoint.
func (s *RecentlyViewedTestSuite) TestGetRecentlyViewed_Unauthenticated() {
	// Create a public client (no token)
	publicClient := helpers.NewAPIClient(s.server)

	// Call recently viewed endpoint without auth
	w := publicClient.Get(s.T(), RecentlyViewedAPIEndpoint)

	// Verify 401 Unauthorized
	s.Assert().Equal(http.StatusUnauthorized, w.Code,
		"Unauthenticated request should return 401")
}

// ============================================================================
// RV-GET-03 (T036): Customer with no recently viewed → empty array
// ============================================================================

// TestGetRecentlyViewed_EmptyList verifies that a customer with no recently
// viewed products gets an empty array (not null).
//
// Setup: Customer has no recently viewed records, calls the endpoint.
// Expect: HTTP 200 with productIds as empty array [].
//
// NOTE: This test will FAIL until Phase 6 handler integration adds the GET endpoint.
func (s *RecentlyViewedTestSuite) TestGetRecentlyViewed_EmptyList() {
	// Call recently viewed endpoint as authenticated customer with no views
	w := s.customerClient.Get(s.T(), RecentlyViewedAPIEndpoint)
	response := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Extract productIds array — should be empty
	data := response["data"].(map[string]any)
	productIDs := data["productIds"].([]any)

	// Verify it's an empty array (not null)
	s.Assert().NotNil(productIDs, "productIds should not be nil")
	s.Assert().Equal(0, len(productIDs),
		"Customer with no views should get an empty array")
}
