package order_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

// E2E.1 Full happy path: Add to cart -> Create order -> Confirm -> Complete.
func (s *OrderSuite) TestScenarioE2E_1_FullHappyPath() {
	orderID := s.createPendingOrderAndGetID()

	confirmResp := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status":        "confirmed",
		"transactionId": "pay_e2e_txn_001",
	})
	helpers.AssertSuccessResponse(s.T(), confirmResp, http.StatusOK)

	completeResp := s.sellerClient.Patch(s.T(), s.getOrderStatusURL(orderID), map[string]any{
		"status": "completed",
	})
	helpers.AssertSuccessResponse(s.T(), completeResp, http.StatusOK)

	getResp := s.customerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), getResp, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal("completed", data["status"])
}
