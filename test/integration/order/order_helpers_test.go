package order_test

import (
	"fmt"
	"net/http"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
)

func (s *OrderSuite) addItemToCart(variantID uint, quantity int) {
	w := s.customerClient.Post(s.T(), "/api/order/cart/item", map[string]any{
		"items": []map[string]any{
			{
				"variantId": variantID,
				"quantity":  quantity,
			},
		},
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
}

func (s *OrderSuite) createOrderRequest() map[string]any {
	return map[string]any{
		"shippingAddressId": 1,
		"billingAddressId":  1,
		"fulfillmentType":   "directship",
		"metadata":          map[string]any{"source": "integration-test"},
	}
}

func (s *OrderSuite) createActiveEmptyCartForCustomer() {
	cart := &orderEntity.Cart{
		UserID: helpers.CustomerUserID,
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
