package get_product_by_id

import (
	"fmt"
	"net/http"
	"time"

	"ecommerce-be/test/integration/helpers"
)

// ============================================================================
// HP-RV-01: Customer views product → row inserted with correct user_id,
//           seller_id, product_id
// ============================================================================

// TestCustomerView_ProductRecorded verifies that a customer viewing a product
// creates a recently viewed record with the correct user_id and product_id.
//
// Setup: Customer (Alice) logs in, views product 1.
// Expect: Exactly 1 recently viewed record for Alice + product 1.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestCustomerView_ProductRecorded() {
	// View product 1 (iPhone 15 Pro, seller_id = 2) as customer
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Drain: wait for fire-and-forget recording goroutine to complete
	time.Sleep(300 * time.Millisecond)

	// Verify a record was created
	count := s.countByUserAndProduct(helpers.CustomerUserID, 1)
	s.Assert().Equal(int64(1), count,
		"Customer viewing product 1 should create a recently viewed record")

	// Verify only 1 record exists for this user
	totalCount := s.countByUser(helpers.CustomerUserID)
	s.Assert().Equal(int64(1), totalCount,
		"Customer should have exactly 1 recently viewed record")
}

// ============================================================================
// HP-RV-02: Customer views product → seller_id correctly captured from product
// ============================================================================

// TestCustomerView_SellerIDCaptured verifies that the recently viewed record
// captures the seller_id from the viewed product, not from any request header.
//
// Setup: Customer views two products from different sellers.
// Expect: seller_id matches each product's actual seller.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestCustomerView_SellerIDCaptured() {
	// View product 1 (seller_id = 2) as Alice (seller 2 customer)
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Drain: wait for fire-and-forget recording goroutine to complete
	time.Sleep(300 * time.Millisecond)

	// Query seller_id on the recently viewed record
	var sellerID uint
	err := s.DB().Raw(
		"SELECT seller_id FROM user_recently_viewed WHERE user_id = ? AND product_id = ?",
		helpers.CustomerUserID, 1,
	).Scan(&sellerID).Error
	s.Require().NoError(err, "Should be able to query the recently viewed record")
	s.Assert().Equal(uint(2), sellerID,
		"Seller ID should match the product's seller_id (2)")

	// View product 107 (belongs to seller 3) as Michael (seller 3 customer)
	w = s.customer2Client.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 107))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Drain: wait for fire-and-forget recording goroutine to complete
	time.Sleep(300 * time.Millisecond)

	err = s.DB().Raw(
		"SELECT seller_id FROM user_recently_viewed WHERE user_id = ? AND product_id = ?",
		helpers.Customer2UserID, 107,
	).Scan(&sellerID).Error
	s.Require().NoError(err, "Should be able to query the recently viewed record")
	s.Assert().Equal(uint(3), sellerID,
		"Seller ID should match product 107's seller_id (3)")
}

// ============================================================================
// HP-RV-03: Re-view same product → viewed_at updated, no duplicate row
// ============================================================================

// TestReView_SameProduct_UpdatesTimestamp verifies that viewing an already-viewed
// product updates the timestamp rather than creating a duplicate row.
//
// Setup: Customer views product 1, waits briefly, views product 1 again.
// Expect: Still exactly 1 record, viewed_at is newer.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestReView_SameProduct_UpdatesTimestamp() {
	// First view
	w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Drain: wait for fire-and-forget recording goroutine to complete
	time.Sleep(300 * time.Millisecond)

	// Verify exactly 1 record
	s.Assert().Equal(int64(1), s.countByUser(helpers.CustomerUserID))

	// Get the initial viewed_at timestamp
	var firstViewedAt string
	err := s.DB().Raw(
		"SELECT viewed_at::text FROM user_recently_viewed WHERE user_id = ? AND product_id = ?",
		helpers.CustomerUserID, 1,
	).Scan(&firstViewedAt).Error
	s.Require().NoError(err)

	// Second view
	w = s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, 1))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Drain: wait for fire-and-forget recording goroutine to complete
	time.Sleep(300 * time.Millisecond)

	// Verify still exactly 1 record (no duplicate)
	s.Assert().Equal(int64(1), s.countByUser(helpers.CustomerUserID),
		"Re-viewing the same product should NOT create a duplicate record")

	// Verify viewed_at was updated
	var updatedViewedAt string
	err = s.DB().Raw(
		"SELECT viewed_at::text FROM user_recently_viewed WHERE user_id = ? AND product_id = ?",
		helpers.CustomerUserID, 1,
	).Scan(&updatedViewedAt).Error
	s.Require().NoError(err)
	s.Assert().NotEqual(firstViewedAt, updatedViewedAt,
		"Re-viewing should update the viewed_at timestamp")
}

// ============================================================================
// HP-RV-04: View when under 10-entry limit → count increments normally
// ============================================================================

// TestUnderLimit_CountIncrements verifies that viewing distinct products under
// the 10-entry limit increases the count normally.
//
// Setup: Customer views 5 distinct products.
// Expect: Exactly 5 recently viewed records.
//
// NOTE: This test will FAIL until Phase 5 handler integration adds the recording call.
func (s *RecentlyViewedTestSuite) TestUnderLimit_CountIncrements() {
	// View 5 distinct products (all belong to seller 2)
	productIDs := []uint{1, 2, 3, 102, 105}
	for _, pid := range productIDs {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// Drain: wait for fire-and-forget recording goroutines to complete
	time.Sleep(300 * time.Millisecond)

	// Verify exactly 5 records exist
	finalCount := s.countByUser(helpers.CustomerUserID)
	s.Assert().Equal(int64(5), finalCount,
		"Viewing 5 distinct products should result in exactly 5 records")
}

// ============================================================================
// RV-GET-01 (T034): GET recently viewed returns full product details
// ============================================================================

// TestGetRecentlyViewed_ReturnsProductIDs verifies that a customer with
// recently viewed products can retrieve full product details via the GET endpoint.
//
// Setup: Customer views 5 distinct products, then calls the recently-viewed endpoint.
// Expect: HTTP 200 with ProductsResponse (products array + pagination) containing
// full product data (name, category, variants, etc.) in newest-first order.
func (s *RecentlyViewedTestSuite) TestGetRecentlyViewed_ReturnsProductIDs() {
	// View 5 distinct products in order: oldest first → newest last
	viewOrder := []uint{1, 2, 3, 102, 105}
	for _, pid := range viewOrder {
		w := s.customerClient.Get(s.T(), fmt.Sprintf(ProductByIDAPIEndpoint, pid))
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}

	// Drain: wait for fire-and-forget recording goroutines to complete
	time.Sleep(300 * time.Millisecond)

	// Call the recently viewed endpoint
	w := s.customerClient.Get(s.T(), RecentlyViewedAPIEndpoint)
	response := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Extract products array and pagination from the standard ProductsResponse
	data := response["data"].(map[string]any)
	products := data["products"].([]any)
	pagination := data["pagination"].(map[string]any)

	// Verify 5 products returned
	s.Assert().Equal(5, len(products),
		"Should return exactly 5 products")

	// Verify newest-first order: 105 was last viewed, 1 was first.
	// Product names come from seed data:
	//   105 = "Free Sample Product" (cat 4, seller 2)
	//   102 = "Minimal Product"     (cat 4, seller 2)
	//   3   = "MacBook Pro 16\""    (cat 5, seller 2)
	//   2   = "Samsung Galaxy S24"  (cat 4, seller 2)
	//   1   = "iPhone 15 Pro"       (cat 4, seller 2)
	s.Assert().Equal("Free Sample Product", products[0].(map[string]any)["name"],
		"Newest viewed product (105) should be first")
	s.Assert().Equal("Minimal Product", products[1].(map[string]any)["name"],
		"Second newest (102) should be second")
	s.Assert().Equal("MacBook Pro 16\"", products[2].(map[string]any)["name"],
		"Third newest (3) should be third")
	s.Assert().Equal("Samsung Galaxy S24", products[3].(map[string]any)["name"],
		"Fourth newest (2) should be fourth")
	s.Assert().Equal("iPhone 15 Pro", products[4].(map[string]any)["name"],
		"Oldest viewed product (1) should be last")

	// Verify full product data is present (not just IDs).
	// Variants may be null for products returned via GetAllProducts with IDs filter,
	// but other product fields should always be populated.
	product := products[0].(map[string]any)
	s.Assert().NotNil(product["id"], "Product ID should be present")
	s.Assert().NotNil(product["category"], "Category should be present")
	s.Assert().NotNil(product["sellerId"], "Seller ID should be present")
	s.Assert().NotNil(product["hasVariants"], "hasVariants should be present")
	s.Assert().NotEmpty(product["name"], "Product name should not be empty")

	// Verify pagination metadata
	s.Assert().Equal(float64(1), pagination["currentPage"],
		"Pagination current page should be 1")
	s.Assert().Equal(5, int(pagination["totalItems"].(float64)),
		"Pagination total items should be 5")
}
