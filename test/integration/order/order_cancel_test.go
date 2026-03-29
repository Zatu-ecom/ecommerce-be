package order_test

import (
	"net/http"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
)

// ─── 5.1 Cancel pending order ────────────────────────────────────────────────

func (s *OrderSuite) TestScenario5_1_CancelPendingOrder() {
	orderID := s.createPendingOrderAndGetID()

	w := s.customerClient.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{
		"reason": "changed my mind",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("cancelled", data["status"])
}

// ─── 5.2 GET reflects cancelled status ───────────────────────────────────────

func (s *OrderSuite) TestScenario5_2_GetReflectsCancelledStatus() {
	orderID := s.createPendingOrderAndGetID()
	s.customerClient.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{
		"reason": "no longer needed",
	})

	w := s.customerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("cancelled", data["status"])
}

// ─── 5.3 Cart reverts to active after cancel (pending order) ─────────────────

func (s *OrderSuite) TestScenario5_3_CartRevertsToActiveOnCancelPending() {
	orderID := s.createPendingOrderAndGetID()
	s.customerClient.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{
		"reason": "wallet empty",
	})

	// GET /api/order/cart must return an active cart (the original one reverted).
	w := s.customerClient.Get(s.T(), "/api/order/cart")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("active", data["status"])
}

// ─── 5.4 Order history records the cancellation ──────────────────────────────

func (s *OrderSuite) TestScenario5_4_OrderHistoryRecordedOnCancel() {
	orderID := s.createPendingOrderAndGetID()
	s.customerClient.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{
		"reason": "testing history",
	})

	var entry orderEntity.OrderHistory
	s.Require().NoError(
		s.container.DB.
			Where("order_id = ? AND to_status = ?", orderID, "cancelled").
			First(&entry).Error,
		"order_history must have a cancelled entry",
	)
}

// ─── 5.5 Cancel confirmed order ──────────────────────────────────────────────

func (s *OrderSuite) TestScenario5_5_CancelConfirmedOrder() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_055",
	})

	w := s.customerClient.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{
		"reason": "seller out of stock",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("cancelled", data["status"])
}

// ─── 5.6 Cannot cancel completed order ───────────────────────────────────────

func (s *OrderSuite) TestScenario5_6_CannotCancelCompletedOrder() {
	orderID := s.createPendingOrderAndGetID()
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "confirmed", "transactionId": "pay_txn_056",
	})
	s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "completed",
	})

	w := s.customerClient.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{
		"reason": "too late",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 5.7 Cannot cancel already-cancelled order ───────────────────────────────

func (s *OrderSuite) TestScenario5_7_CannotCancelAlreadyCancelled() {
	orderID := s.createPendingOrderAndGetID()
	s.customerClient.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{"reason": "first cancel"})

	w := s.customerClient.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{"reason": "second cancel"})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 5.8 Customer cannot cancel another user's order ─────────────────────────

func (s *OrderSuite) TestScenario5_8_CustomerCannotCancelOtherUsersOrder() {
	orderID := s.createPendingOrderAndGetID()

	customer2 := helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), customer2, helpers.Customer2Email, helpers.Customer2Password)
	customer2.SetToken(token)

	w := customer2.Post(s.T(), s.getOrderCancelURL(orderID), map[string]any{"reason": "theft attempt"})
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ─── 5.9 Unauthenticated request ─────────────────────────────────────────────

func (s *OrderSuite) TestScenario5_9_CancelOrderUnauthenticated() {
	w := s.client.Post(s.T(), s.getOrderCancelURL(1), map[string]any{
		"reason": "test",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// ─── 5.10 Non-existent order returns 404 ─────────────────────────────────────

func (s *OrderSuite) TestScenario5_10_CancelNonExistentOrder() {
	w := s.customerClient.Post(s.T(), s.getOrderCancelURL(999999), map[string]any{"reason": "ghost"})
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}


