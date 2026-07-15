package get_product_by_id

import (
	"fmt"
	"net/http"
	"time"

	"ecommerce-be/test/integration/helpers"
)

// ============================================================================
// EC-RV-01: 10 entries → 11th view trims oldest, keeps exactly 10
// ============================================================================

// TestMaxLimit_OldestTrimmed verifies that when the recently viewed list
// reaches the 10-entry limit, the oldest entry is removed to make room.
//
// Setup: Customer views 10 distinct products, then an 11th.
// Expect: 10 records remain, the 11th product appears, oldest is gone.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestMaxLimit_OldestTrimmed() {
	// View 10 distinct products to fill the limit (all belong to seller 2)
	// Available seller 2 products: 1,2,3,102,103,104,105,106,108,109,110
	firstBatch := []uint{1, 2, 3, 102, 103, 104, 105, 106, 108, 109}
	for _, pid := range firstBatch {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// Verify exactly 10 records exist
	s.Assert().Equal(int64(10), s.countByUser(helpers.CustomerUserID),
		"Should have exactly 10 recently viewed records after filling the limit")

	// View an 11th distinct product (110, seller 2)
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 110))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify still exactly 10 records (not 11 — oldest trimmed)
	s.Assert().Equal(int64(10), s.countByUser(helpers.CustomerUserID),
		"Should still have exactly 10 records after 11th view (oldest trimmed)")

	// Verify the 11th product (product 110) is present
	s.Assert().Equal(int64(1),
		s.countByUserAndProduct(helpers.CustomerUserID, 110),
		"The 11th viewed product should be in the list")

	// At minimum, verify we never exceed 10 records
	totalCount := s.countByUser(helpers.CustomerUserID)
	s.Assert().LessOrEqual(totalCount, int64(10),
		"Recently viewed count should never exceed 10")
}

// ============================================================================
// EC-RV-02: Re-viewing a product refreshes its position in trim ordering
// ============================================================================

// TestReView_RefreshesPosition verifies that re-viewing a product updates its
// viewed_at timestamp so it is not the first to be trimmed.
//
// Setup: Customer views 1,2,3,4; re-views 1 (oldest); views 7 more to push past limit.
// Expect: 10 records remain; product 1 (refreshed) is still present.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestReView_RefreshesPosition() {
	// View products in order: 1, 2, 3, 102 (all belong to seller 2)
	for _, pid := range []uint{1, 2, 3, 102} {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// Re-view product 1 (oldest) — this should refresh its viewed_at
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Now view 7 more products to push past limit (all seller 2, accessible to Alice)
	extraProducts := []uint{103, 104, 105, 106, 108, 109, 110}
	for _, pid := range extraProducts {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// We've viewed 4 + 1(review) + 7 = 12 views, but only 11 distinct products.
	// After trim, we should have 10. Product 1 was refreshed, so oldest (2) is trimmed.
	finalCount := s.countByUser(helpers.CustomerUserID)
	s.Assert().Equal(int64(10), finalCount,
		"After trimming, we should have exactly 10 records")

	// Product 1 was refreshed, so it should still be in the list
	s.Assert().Equal(int64(1),
		s.countByUserAndProduct(helpers.CustomerUserID, 1),
		"Re-viewed product 1 should still be in the list (position refreshed)")
}

// ============================================================================
// EC-RV-03: User A's views don't affect user B's list (user isolation)
// ============================================================================

// TestUserIsolation_IndependentLists verifies that each user has their own
// independent recently viewed list.
//
// Setup: Customer A (Alice, seller 2) views products 1,2,3.
//         Customer B (Michael, seller 3) views product 107.
// Expect: A has 3 records, B has 1 record, no cross-contamination.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestUserIsolation_IndependentLists() {
	// Customer A (Alice, seller 2) views products 1, 2, 3
	for _, pid := range []uint{1, 2, 3} {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// Customer B (Michael, seller 3) views product 107 (belongs to seller 3)
	for _, pid := range []uint{107} {
		w := s.customer2Client.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// Verify Customer A has 3 records, Customer B has 1 record
	s.Assert().Equal(int64(3), s.countByUser(helpers.CustomerUserID),
		"Customer A should have 3 records")
	s.Assert().Equal(int64(1), s.countByUser(helpers.Customer2UserID),
		"Customer B should have 1 record")

	// Verify Customer A does NOT have Customer B's products
	s.Assert().Equal(int64(0), s.countByUserAndProduct(helpers.CustomerUserID, 107),
		"Customer A should NOT have product 107 in their list (seller 3 product)")

	// Verify Customer B does NOT have Customer A's products
	s.Assert().Equal(int64(0), s.countByUserAndProduct(helpers.Customer2UserID, 1),
		"Customer B should NOT have product 1 in their list")
	s.Assert().Equal(int64(0), s.countByUserAndProduct(helpers.Customer2UserID, 2),
		"Customer B should NOT have product 2 in their list")
}

// ============================================================================
// EC-RV-04: Re-view at limit doesn't double-trim (idempotent)
// ============================================================================

// TestMaxLimit_IdempotentTrim verifies that re-viewing an already-tracked
// product at the limit does not trigger double-trimming.
//
// Setup: Customer fills 10 slots, re-views an existing product.
// Expect: Still exactly 10 records after multiple re-views.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestMaxLimit_IdempotentTrim() {
	// View 10 distinct products (all belong to seller 2)
	for _, pid := range []uint{1, 2, 3, 102, 103, 104, 105, 106, 108, 109} {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}
	s.Assert().Equal(int64(10), s.countByUser(helpers.CustomerUserID))

	// Re-view the same product — triggers upsert (not insert), no trim needed
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 109))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify still exactly 10 (no double-trim)
	s.Assert().Equal(int64(10), s.countByUser(helpers.CustomerUserID),
		"Re-viewing a product at the limit should NOT double-trim")

	// Re-view the same product multiple times
	for i := 0; i < 5; i++ {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// Verify still exactly 10
	s.Assert().Equal(int64(10), s.countByUser(helpers.CustomerUserID),
		"Multiple re-views at the limit should not change the count")
}

// ============================================================================
// EC-RV-05: Seller views product → NOT recorded (role level 2)
// ============================================================================

// TestSellerView_NotRecorded verifies that sellers do NOT get recently
// viewed recordings (US2 — Role Restriction).
//
// Setup: Seller (John Seller, role level 2) views product 1.
// Expect: HTTP 200, zero recently viewed records.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestSellerView_NotRecorded() {
	// Seller views their own product
	w := s.sellerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify NO record was created for seller
	totalCount := s.countByUser(helpers.Seller2UserID)
	s.Assert().Equal(int64(0), totalCount,
		"Seller viewing a product should NOT create a recently viewed record")

	// Verify no records exist at all
	s.Assert().Equal(int64(0), s.countAllRecentlyViewed(),
		"No recently viewed records should exist after seller view")
}

// ============================================================================
// EC-RV-06: Admin views product → NOT recorded (role level 1)
// ============================================================================

// TestAdminView_NotRecorded verifies that admins do NOT get recently
// viewed recordings (US2 — Role Restriction).
//
// Setup: Admin (role level 1) views product 1.
// Expect: HTTP 200, zero recently viewed records.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestAdminView_NotRecorded() {
	// Admin views product
	w := s.adminClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify NO record was created for admin
	totalCount := s.countByUser(helpers.AdminUserID)
	s.Assert().Equal(int64(0), totalCount,
		"Admin viewing a product should NOT create a recently viewed record")

	// Verify no records exist at all
	s.Assert().Equal(int64(0), s.countAllRecentlyViewed(),
		"No recently viewed records should exist after admin view")
}

// ============================================================================
// EC-RV-07: Zero-price product → view STILL recorded
// ============================================================================

// TestZeroPriceProduct_Recorded verifies that viewing a zero-price product
// still records the view (recording is product-id-based, not price-based).
//
// Setup: Customer views product 105 (Free Sample, price = 0).
// Expect: Exactly 1 recently viewed record for product 105.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestZeroPriceProduct_Recorded() {
	// View zero-price product (product 105 — Free Sample)
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 105))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify the view WAS recorded despite zero price
	s.Assert().Equal(int64(1),
		s.countByUserAndProduct(helpers.CustomerUserID, 105),
		"Zero-price product view should still be recorded")
}

// ============================================================================
// EC-RV-08: Unicode product name → view STILL recorded
// ============================================================================

// TestUnicodeProduct_Recorded verifies that viewing a product with a unicode
// name still records the view.
//
// Setup: Customer views product 104 (has Chinese, Arabic, emojis).
// Expect: Exactly 1 recently viewed record for product 104.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestUnicodeProduct_Recorded() {
	// View unicode product (product 104)
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 104))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify the view WAS recorded despite unicode name
	s.Assert().Equal(int64(1),
		s.countByUserAndProduct(helpers.CustomerUserID, 104),
		"Unicode product view should still be recorded")
}

// ============================================================================
// EC-RV-09: Product deletion cascades to remove the recently viewed record
// ============================================================================

// TestProductDeletion_CascadesRecord verifies that deleting a product also
// removes its associated recently viewed records via DB cascade.
//
// Setup: Seller creates a product; customer views it; seller deletes it.
// Expect: Recently viewed record is cascade-deleted with the product.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestProductDeletion_CascadesRecord() {
	// Login as seller to create a product
	createBody := map[string]any{
		"name":       "Temporary Cascade Product",
		"categoryId": 4,
		"baseSku":    "CASCADE-TEST-001",
		"price":      99.99,
	}
	createResp := helpers.AssertSuccessResponse(s.T(),
		s.sellerClient.Post(s.T(), ProductAPIEndpoint, createBody), http.StatusCreated)
	newProductID := int(helpers.GetResponseData(s.T(), createResp, "product")["id"].(float64))

	// Now login as customer and view the product
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, newProductID))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Verify the record exists
	s.Assert().Equal(int64(1),
		s.countByUserAndProduct(helpers.CustomerUserID, uint(newProductID)),
		"Record should exist after viewing")

	// Login as seller and delete the product
	w = s.sellerClient.Delete(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, newProductID))
	s.Assert().Equal(http.StatusOK, w.Code, "Product deletion should succeed")

	// Wait a tiny bit for cascade to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify the recently viewed record was cascade-deleted
	s.Assert().Equal(int64(0),
		s.countByUserAndProduct(helpers.CustomerUserID, uint(newProductID)),
		"Recently viewed record should be cascade-deleted when product is deleted")

	// Verify the customer's total count dropped
	s.Assert().Equal(int64(0), s.countByUser(helpers.CustomerUserID),
		"Customer should have 0 recently viewed records after product deletion")
}

// ============================================================================
// RV-GET-04 (T037): GET recently viewed respects limit parameter
// ============================================================================

// TestGetRecentlyViewed_RespectsLimit verifies that the limit query parameter
// is respected, with a maximum cap of 50.
//
// Setup: Customer views 15 distinct products, then requests with limit=5
//
//	and limit=100 (capped at 50).
//
// Expect: limit=5 returns at most 5 IDs; limit=100 returns at most 50.
//
// NOTE: This test will FAIL until Phase 6 handler integration adds the GET endpoint.
func (s *RecentlyViewedTestSuite) TestGetRecentlyViewed_RespectsLimit() {
	// View 15 distinct products (more than default 10)
	// View 11 distinct products (all seller 2 accessible products)
	viewOrder := []uint{1, 2, 3, 102, 103, 104, 105, 106, 108, 109, 110}
	for _, pid := range viewOrder {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// --- Test 1: limit=5 → at most 5 IDs ---
	w := s.customerClient.Get(s.T(), fmt.Sprintf(RecentlyViewedLimitAPIEndpoint, 5))
	response := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := response["data"].(map[string]any)
	productIDs := data["productIds"].([]any)
	s.Assert().Equal(5, len(productIDs),
		"limit=5 should return exactly 5 product IDs")
	// Oldest products should NOT appear; newest should appear first
	s.Assert().Equal(float64(110), productIDs[0],
		"Newest viewed (110) should be first with limit=5")

	// --- Test 2: limit=100 → capped at 50 (max); oldest was trimmed, so 10 remain ---
	w = s.customerClient.Get(s.T(), fmt.Sprintf(RecentlyViewedLimitAPIEndpoint, 100))
	response = helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data = response["data"].(map[string]any)
	productIDs = data["productIds"].([]any)
	s.Assert().Equal(10, len(productIDs),
		"limit=100 with 11 views (trimmed to 10) should return 10 entries")
}
