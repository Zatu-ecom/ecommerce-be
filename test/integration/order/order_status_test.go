package order_test

import (
	"net/http"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
)

// ─── 4.1 pending → confirmed with transactionId ──────────────────────────────

func (s *OrderSuite) TestScenario4_1_PendingToConfirmedWithTransactionID() {
	orderID := s.createPendingOrderAndGetID()

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status":        "confirmed",
		"transactionId": "pay_test_txn_001",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("confirmed", data["status"])
	s.Require().Equal("pending", data["previousStatus"])
}

// ─── 4.2 confirmed → completed (shipment leaves) ────────────────────────────

func (s *OrderSuite) TestScenario4_2_ConfirmedToCompleted() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_002",
	})

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "completed",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("completed", data["status"])
}

// ─── 4.3 pending → cancelled (reservation released) ─────────────────────────

func (s *OrderSuite) TestScenario4_3_PendingToCancelled() {
	orderID := s.createPendingOrderAndGetID()

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "cancelled",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("cancelled", data["status"])
}

// ─── 4.4 pending → failed with failureReason ─────────────────────────────────

func (s *OrderSuite) TestScenario4_4_PendingToFailedWithFailureReason() {
	orderID := s.createPendingOrderAndGetID()

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status":        "failed",
		"failureReason": "payment_declined",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("failed", data["status"])
}

// ─── 4.5 confirmed → cancelled ───────────────────────────────────────────────

func (s *OrderSuite) TestScenario4_5_ConfirmedToCancelled() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_005",
	})

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "cancelled",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("cancelled", data["status"])
}

// ─── 4.6 completed → returned ────────────────────────────────────────────────

func (s *OrderSuite) TestScenario4_6_CompletedToReturned() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_006",
	})
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "completed",
	})

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "returned",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("returned", data["status"])
}

// ─── 4.7 GET reflects new status after update ────────────────────────────────

func (s *OrderSuite) TestScenario4_7_GetReflectsNewStatus() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_007",
	})

	w := s.customerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("confirmed", data["status"])
}

// ─── 4.8 Order history row created on status transition ──────────────────────
// (direct DB assertion — no history-list API yet)

func (s *OrderSuite) TestScenario4_8_OrderHistoryRecordedOnConfirm() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_008",
	})

	var entry orderEntity.OrderHistory
	s.Require().NoError(
		s.container.DB.
			Where("order_id = ? AND from_status = ? AND to_status = ?", orderID, "pending", "confirmed").
			First(&entry).Error,
		"order_history must have pending→confirmed entry",
	)
}

// ─── 4.9 paidAt is set on confirmation ───────────────────────────────────────

func (s *OrderSuite) TestScenario4_9_PaidAtSetOnConfirm() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_009",
	})

	// Verify via GET /api/order/:id that paidAt is set.
	w := s.sellerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().NotEmpty(data["paidAt"], "paidAt must be set after confirming")
}

// ─── 4.10 Cart reverts to active after pending → failed ──────────────────────

func (s *OrderSuite) TestScenario4_10_CartRevertsToActiveOnFailed() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status":        "failed",
		"failureReason": "payment_timeout",
	})

	// GET /api/order/cart must return an active cart (the reverted one).
	w := s.customerClient.Get(s.T(), "/api/order/cart")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("active", data["status"])
}

// ─── 4.17 Seller cannot update another seller's order ────────────────────────

func (s *OrderSuite) TestScenario4_17_SellerCannotUpdateOtherSellersOrder() {
	orderID := s.createPendingOrderAndGetID()

	// Seller from a different store (Seller2 created the cart, but let's use a different seller token)
	otherSeller := helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), otherSeller, helpers.SellerEmail, helpers.SellerPassword)
	otherSeller.SetToken(token)

	w := otherSeller.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_017",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ─── 4.18 Customer cannot call update status (seller-only) ───────────────────

func (s *OrderSuite) TestScenario4_18_CustomerCannotUpdateStatus() {
	orderID := s.createPendingOrderAndGetID()

	w := s.customerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_018",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusForbidden)
}

// ─── 4.19 Terminal state: cancelled → confirmed is invalid ───────────────────

func (s *OrderSuite) TestScenario4_19_InvalidTransitionFromTerminalState() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "cancelled",
	})

	// Attempt to confirm a cancelled order.
	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_019",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 4.20 pending → confirmed without transactionId ──────────────────────────

func (s *OrderSuite) TestScenario4_20_MissingTransactionIDForConfirm() {
	orderID := s.createPendingOrderAndGetID()

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 4.21 pending → failed without failureReason ─────────────────────────────

func (s *OrderSuite) TestScenario4_21_MissingFailureReasonForFailed() {
	orderID := s.createPendingOrderAndGetID()

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "failed",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 4.22 Invalid status value ───────────────────────────────────────────────

func (s *OrderSuite) TestScenario4_22_InvalidStatusValue() {
	orderID := s.createPendingOrderAndGetID()

	w := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "shipped", // not a valid order status
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 4.23 Unauthenticated update status ──────────────────────────────────────

func (s *OrderSuite) TestScenario4_23_UnauthenticatedUpdateStatus() {
	w := s.client.Patch(s.T(), s.getOrderStatusURL(1), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_023",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}


