package order_test

import (
	"fmt"
	"net/http"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
)

func (s *OrderSuite) addItemToCart(variantID uint, quantity int) {
	w := s.customerClient.Post(s.T(), "/api/order/cart/item", helpers.AddCartItemsPayload(variantID, quantity))
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
}

func (s *OrderSuite) createOrderRequest() map[string]any {
	return helpers.DefaultCreateOrderRequest()
}

func (s *OrderSuite) createActiveEmptyCartForCustomer() {
	userID := uint(helpers.CustomerUserID)
	cart := &orderEntity.Cart{
		UserID: &userID,
		Status: orderEntity.CART_STATUS_ACTIVE,
	}
	s.Require().NoError(s.container.DB.Create(cart).Error)
}

func (s *OrderSuite) createPendingOrderAndGetID() uint {
	// Precondition: add one valid cart item for customer seller scope.
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	return uint(data["id"].(float64))
}

func (s *OrderSuite) getOrderByIDURL(orderID uint) string {
	return fmt.Sprintf(OrderByIDAPIEndpoint, orderID)
}

func (s *OrderSuite) getOrderStatusURL(orderID uint) string {
	return fmt.Sprintf(OrderStatusAPIEndpoint, orderID)
}

func (s *OrderSuite) getOrderCancelURL(orderID uint) string {
	return fmt.Sprintf(OrderCancelAPIEndpoint, orderID)
}

func (s *OrderSuite) createSellerDiscountCode(payload map[string]any) uint {
	res := s.sellerClient.Post(s.T(), "/api/promotion/discount-code", payload)
	s.Require().Equal(http.StatusCreated, res.Code, res.Body.String())
	response := helpers.ParseResponse(s.T(), res.Body)
	dc := response["data"].(map[string]any)["discountCode"].(map[string]any)
	return uint(dc["id"].(float64))
}

func (s *OrderSuite) percentageCouponPayload(code string, value int64) map[string]any {
	return helpers.PercentageDiscountCode(code, value)
}

func (s *OrderSuite) applyCoupon(code string) {
	res := s.customerClient.Post(s.T(), "/api/order/cart/coupon", helpers.ApplyCouponPayload(code))
	s.Require().Equal(http.StatusOK, res.Code, res.Body.String())
}

func (s *OrderSuite) createOrderOK() map[string]any {
	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	return resp["data"].(map[string]any)
}
