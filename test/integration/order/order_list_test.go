package order_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

// 3.5 Default pagination.
func (s *OrderSuite) TestScenario3_5_ListOrdersDefaultPagination() {
	_ = s.createPendingOrderAndGetID()

	w := s.customerClient.Get(s.T(), OrderAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)

	_, hasOrders := data["orders"]
	_, hasPagination := data["pagination"]
	s.Require().True(hasOrders)
	s.Require().True(hasPagination)
}

// 3.14 No orders exist.
func (s *OrderSuite) TestScenario3_14_NoOrdersExist() {
	w := s.customerClient.Get(s.T(), OrderAPIEndpoint+"?page=1&pageSize=20")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	orders := data["orders"].([]any)
	s.Require().Len(orders, 0)
}
