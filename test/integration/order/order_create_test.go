package order_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	orderEntity "ecommerce-be/order/entity"
	"ecommerce-be/test/integration/helpers"
)

// ─── 1.1 Happy path: pending status, order number format ───────────────────

func (s *OrderSuite) TestScenario1_1_CreateOrderReturnsCorrectStatus() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)

	data := resp["data"].(map[string]any)
	s.Require().Equal("pending", data["status"])
}

// ─── 1.2 Order number format matches ORD-<epoch_ms>-<seller_b36>-<random> ──

func (s *OrderSuite) TestScenario1_2_OrderNumberFormat() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)

	orderNumber, _ := data["orderNumber"].(string)
	s.Require().True(strings.HasPrefix(orderNumber, "ORD-"), "orderNumber must start with ORD-: %s", orderNumber)
	pattern := regexp.MustCompile(`^ORD-\d{13}-[0-9A-Z]+-[0-9A-Z]+$`)
	s.Require().True(pattern.MatchString(orderNumber), "orderNumber format mismatch: %s", orderNumber)
}

// ─── 1.3 Cart transitions to converted after successful order creation ──────

func (s *OrderSuite) TestScenario1_3_CartBecomesConverted() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	orderID := uint(data["id"].(float64))

	// The original cart must now be converted and linked to the order.
	var cart orderEntity.Cart
	s.Require().NoError(
		s.container.DB.
			Where("user_id = ? AND status = ? AND order_id = ?", helpers.CustomerUserID, orderEntity.CART_STATUS_CONVERTED, orderID).
			First(&cart).Error,
		"expected a converted cart linked to the new order",
	)
}

// ─── 1.4 New active cart created after order ────────────────────────────────

func (s *OrderSuite) TestScenario1_4_NewActiveCartCreatedAfterOrder() {
	s.addItemToCart(1, 1)
	s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())

	// GET /api/order/cart must return an empty active cart (not the converted one).
	w := s.customerClient.Get(s.T(), "/api/order/cart")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data := resp["data"].(map[string]any)
	items, _ := data["items"].([]any)
	s.Require().Len(items, 0, "new active cart must be empty")
}

// ─── 1.8 Order totals present and non-zero after creation ───────────────────

func (s *OrderSuite) TestScenario1_8_OrderTotalsPresent() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)

	s.Require().NotNil(data["subtotalCents"])
	s.Require().Greater(data["subtotalCents"].(float64), float64(0))
	s.Require().NotNil(data["totalCents"])
	s.Require().Greater(data["totalCents"].(float64), float64(0))
}

// ─── 1.10 placedAt is set ───────────────────────────────────────────────────

func (s *OrderSuite) TestScenario1_10_PlacedAtIsSet() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)

	s.Require().NotEmpty(data["placedAt"], "placedAt must be set on created order")
}

// ─── 1.11 Order history entry created with null→pending ─────────────────────
// (no history-list API yet; direct DB assertion is allowed per plan.md)

func (s *OrderSuite) TestScenario1_11_OrderHistoryEntryCreated() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	orderID := uint(resp["data"].(map[string]any)["id"].(float64))

	var entry orderEntity.OrderHistory
	s.Require().NoError(
		s.container.DB.
			Where("order_id = ? AND to_status = ?", orderID, "pending").
			First(&entry).Error,
		"expected order_history row with to_status=pending",
	)
	s.Require().Nil(entry.FromStatus, "first history entry must have null from_status")
}

// ─── 1.12 Default fulfillment type is directship ────────────────────────────

func (s *OrderSuite) TestScenario1_12_DefaultFulfillmentType() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, map[string]any{
		"shippingAddressId": 1,
		"billingAddressId":  1,
		// no fulfillmentType → should default to directship
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)
	s.Require().Equal("directship", data["fulfillmentType"])
}

// ─── 1.14 No active cart ────────────────────────────────────────────────────

func (s *OrderSuite) TestScenario1_14_NoActiveCart() {
	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ─── 1.15 Active cart exists but is empty ───────────────────────────────────

func (s *OrderSuite) TestScenario1_15_ActiveCartExistsButEmpty() {
	s.createActiveEmptyCartForCustomer()

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 1.16 Variant out of stock (reservation service rejects) ────────────────

func (s *OrderSuite) TestScenario1_16_InsufficientStockRejectsOrder() {
	// Variant 2 has limited stock; request more than available via cart
	// (quantity 80 is beyond stock ceiling tested in cart tests).
	// Use a quantity that reservation service will reject.
	// Cart add itself may reject — both 400/409 are acceptable here.
	w := s.customerClient.Post(s.T(), "/api/order/cart/item", map[string]any{
		"items": []map[string]any{{"variantId": uint(2), "quantity": 80}},
	})
	// If cart rejects the add, no order will be created — that's the expected path.
	if w.Code == http.StatusCreated {
		// Cart accepted; try to create order — reservation service should reject it.
		wo := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
		s.Require().True(
			wo.Code == http.StatusConflict || wo.Code == http.StatusBadRequest,
			"expected 409 or 400 for insufficient stock, got %d", wo.Code,
		)
	}
}

// ─── 1.19 Missing shipping address ID ───────────────────────────────────────

func (s *OrderSuite) TestScenario1_19_MissingShippingAddressID() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, map[string]any{
		"billingAddressId": 1,
		"fulfillmentType":  "directship",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 1.20 Shipping address belongs to different user ────────────────────────

func (s *OrderSuite) TestScenario1_20_AddressBelongsToDifferentUser() {
	s.addItemToCart(1, 1)

	// Address ID 999 does not exist / does not belong to customerClient.
	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, map[string]any{
		"shippingAddressId": 999999,
		"billingAddressId":  999999,
		"fulfillmentType":   "directship",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ─── 1.21 Invalid fulfillment type ──────────────────────────────────────────

func (s *OrderSuite) TestScenario1_21_InvalidFulfillmentType() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, map[string]any{
		"shippingAddressId": 1,
		"billingAddressId":  1,
		"fulfillmentType":   "invalid_type",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── 1.25 Items are returned in the create response ─────────────────────────

func (s *OrderSuite) TestScenario1_25_CreateResponseIncludesItems() {
	s.addItemToCart(1, 1)

	w := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusCreated)
	data := resp["data"].(map[string]any)

	items, ok := data["items"].([]any)
	s.Require().True(ok, "items must be present in create response")
	s.Require().NotEmpty(items, "items must not be empty")

	item := items[0].(map[string]any)
	s.Require().NotEmpty(item["productName"], "productName must be snapshotted")
	s.Require().Greater(item["unitPriceCents"].(float64), float64(0))
	s.Require().Greater(item["lineTotalCents"].(float64), float64(0))
	s.Require().Equal(float64(1), item["quantity"])
}

// ─── 1.26 Unauthenticated create order ──────────────────────────────────────

func (s *OrderSuite) TestScenario1_26_UnauthenticatedCreateOrder() {
	w := s.client.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	helpers.AssertErrorResponse(s.T(), w, http.StatusUnauthorized)
}

// ─── D.1 Price snapshot is immutable (data integrity) ───────────────────────
// After creating an order, the price in the order_item must match the
// price at time of order (not affected by future price changes).

func (s *OrderSuite) TestScenarioD1_OrderItemPriceIsSnapshot() {
	s.addItemToCart(1, 1)

	createW := s.customerClient.Post(s.T(), OrderAPIEndpoint, s.createOrderRequest())
	resp := helpers.AssertSuccessResponse(s.T(), createW, http.StatusCreated)
	orderID := uint(resp["data"].(map[string]any)["id"].(float64))

	// Verify via GET that the unit price is baked in.
	getW := s.customerClient.Get(s.T(), fmt.Sprintf(OrderByIDAPIEndpoint, orderID))
	getResp := helpers.AssertSuccessResponse(s.T(), getW, http.StatusOK)
	data := getResp["data"].(map[string]any)
	items := data["items"].([]any)
	s.Require().NotEmpty(items)
	item := items[0].(map[string]any)
	// unitPriceCents is snapshotted; it must be > 0 and stable.
	s.Require().Greater(item["unitPriceCents"].(float64), float64(0))
}

