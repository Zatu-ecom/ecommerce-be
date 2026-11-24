package payment_test

import (
	"ecommerce-be/test/integration/helpers"
)

// createPendingOrder creates a pending order for a given customer by posting to
// the order API as that customer. It first adds a seeded product to the cart.
func (s *PaymentSuite) createPendingOrder(customerUserID uint) uint {
	s.seedCartForCustomer(customerUserID)

	client := s.clientForCustomer(customerUserID)
	w := client.Post(s.T(), "/api/order", map[string]any{
		"shippingAddressId": s.shippingAddressID(customerUserID),
		"billingAddressId":  s.shippingAddressID(customerUserID),
		"fulfillmentType":   "directship",
	})
	s.Require().Equal(201, w.Code, "order should be created, got %d: %s", w.Code, w.Body.String())

	resp := helpers.ParseResponse(s.T(), w.Body)
	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "order response should have data")
	return uint(data["id"].(float64))
}

// seedCartForCustomer adds a seeded product variant to the customer's cart.
// Customer 5 (Alice, seller 2) uses variant 1 (iPhone); customer 6 (Michael,
// seller 3) uses variant 9 (Nike t-shirt) so the item belongs to their seller.
func (s *PaymentSuite) seedCartForCustomer(customerUserID uint) {
	client := s.clientForCustomer(customerUserID)
	variantID := 1
	if customerUserID == helpers.Customer2UserID {
		variantID = 9
	}
	qty := 1
	w := client.Post(s.T(), "/api/order/cart/item", map[string]any{
		"items": []map[string]any{
			{"variantId": variantID, "quantity": &qty},
		},
	})
	// The API returns 200 (existing cart) or 201 (new cart); both are success.
	s.Require().Contains([]int{200, 201}, w.Code,
		"add to cart failed: %d %s", w.Code, w.Body.String())
}

func (s *PaymentSuite) clientForCustomer(customerUserID uint) *helpers.APIClient {
	if customerUserID == helpers.CustomerUserID {
		return s.customerClient
	}
	return s.customer2Client()
}

func (s *PaymentSuite) customer2Client() *helpers.APIClient {
	client := helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), client, helpers.Customer2Email, helpers.Customer2Password)
	client.SetToken(token)
	return client
}

// shippingAddressID returns the customer's default address id from seed data.
func (s *PaymentSuite) shippingAddressID(customerUserID uint) uint {
	switch customerUserID {
	case helpers.CustomerUserID:
		return 1 // Alice (user 5) HOME address
	case helpers.Customer2UserID:
		return 3 // Michael (user 6) HOME address
	default:
		s.T().Fatalf("no address seeded for customer %d", customerUserID)
		return 0
	}
}
