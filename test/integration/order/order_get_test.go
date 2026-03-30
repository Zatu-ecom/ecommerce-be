package order_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

// ─── 2.1 Customer gets their own order ──────────────────────────────────────

func (s *OrderSuite) TestScenario2_1_CustomerGetsOwnOrder() {
	orderID := s.createPendingOrderAndGetID()

	w := s.customerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)

	s.Require().Equal(float64(orderID), data["id"])
	s.Require().Equal("pending", data["status"])
	s.Require().NotEmpty(data["orderNumber"])
}

// ─── 2.2 Seller gets order from their store ──────────────────────────────────

func (s *OrderSuite) TestScenario2_2_SellerGetsOrderFromOwnStore() {
	orderID := s.createPendingOrderAndGetID()

	w := s.sellerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal(float64(orderID), data["id"])
}

// ─── 2.3 Seller sees customer info (name, email) ─────────────────────────────

func (s *OrderSuite) TestScenario2_3_SellerSeesCustomerInfo() {
	orderID := s.createPendingOrderAndGetID()

	w := s.sellerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)

	customer, ok := data["customer"].(map[string]any)
	s.Require().True(ok, "seller must see 'customer' field in order response")
	s.Require().NotEmpty(customer["firstName"])
	s.Require().NotEmpty(customer["email"])
}

// ─── 2.4 Customer does NOT see other customer's order (forbidden) ────────────

func (s *OrderSuite) TestScenario2_4_CustomerCannotGetOtherUsersOrder() {
	orderID := s.createPendingOrderAndGetID()

	// customer2Client uses a different user account
	customer2Client := helpers.NewAPIClient(s.server)
	token := helpers.Login(
		s.T(),
		customer2Client,
		helpers.Customer2Email,
		helpers.Customer2Password,
	)
	customer2Client.SetToken(token)

	w := customer2Client.Get(s.T(), s.getOrderByIDURL(orderID))
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ─── 2.5 Admin gets any order ────────────────────────────────────────────────

func (s *OrderSuite) TestScenario2_5_AdminGetsAnyOrder() {
	orderID := s.createPendingOrderAndGetID()

	w := s.adminClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	s.Require().Equal(float64(orderID), data["id"])
}

// ─── 2.6 Non-existent order returns 404 ─────────────────────────────────────

func (s *OrderSuite) TestScenario2_6_NonExistentOrderReturns404() {
	w := s.customerClient.Get(s.T(), s.getOrderByIDURL(999999))
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ─── 2.7 Unauthenticated request returns 401 ─────────────────────────────────

func (s *OrderSuite) TestScenario2_7_GetOrderUnauthenticated() {
	w := s.client.Get(s.T(), s.getOrderByIDURL(1))
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// ─── 2.8 Get order response includes items, addresses ────────────────────────

func (s *OrderSuite) TestScenario2_8_GetOrderIncludesItemsAndAddresses() {
	orderID := s.createPendingOrderAndGetID()

	w := s.customerClient.Get(s.T(), s.getOrderByIDURL(orderID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)

	items, ok := data["items"].([]any)
	s.Require().True(ok, "items must be present")
	s.Require().NotEmpty(items)

	addresses, ok := data["addresses"].([]any)
	s.Require().True(ok, "addresses must be present")
	s.Require().NotEmpty(addresses)
}
