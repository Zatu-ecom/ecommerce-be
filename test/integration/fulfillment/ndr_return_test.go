package fulfillment_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// stubNDRAction serves the NDR action endpoint for one AWB (paths embed
// the AWB, so tests register after booking when the AWB is known).
func (s *FulfillmentSuite) stubNDRAction(awb string) {
	s.stubShiprocketAuth()
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/ndr/"+awb+"/action",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"success": true}`)
		})
}

// ─── Shared return stubs ─────────────────────────────────────────────────────
// stubShiprocketReturn serves the return-order create endpoint. AWB
// assignment reuses the counter stub from book_test.go.
func (s *FulfillmentSuite) stubShiprocketReturn() {
	s.stubShiprocketAuth()
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/orders/create/return",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"order_id": 7001, "shipment_id": 6001, "status": "NEW"}`)
		})
}

// deliverBox books a box and drives it to delivered via webhook.
func (s *FulfillmentSuite) deliverBox(variantID uint, qty int) (orderID, shipmentID uint, awb string) {
	t := s.T()
	orderID, shipmentID, awb = s.bookBoxWithAWB(variantID, qty)
	w := s.pushWebhook(awb, "DELIVERED", "2026-10-03 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)
	return orderID, shipmentID, awb
}

// openNDRBox books a box and raises one failed-attempt round via webhook.
func (s *FulfillmentSuite) openNDRBox() (orderID, shipmentID uint, awb string) {
	t := s.T()
	orderID, shipmentID, awb = s.bookBoxWithAWB(2, 1)
	w := s.pushWebhook(awb, "UNDELIVERED - CONSIGNEE NOT AVAILABLE", "2026-10-03 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)
	return orderID, shipmentID, awb
}

func (s *FulfillmentSuite) ndrAttempt(shipmentID uint) int {
	t := s.T()
	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	detail := resp["data"].(map[string]any)["shipment"].(map[string]any)
	ndr, ok := detail["ndr"].(map[string]any)
	if !ok || ndr == nil {
		return 0
	}
	return int(ndr["attemptNo"].(float64))
}

// ─── US5: NDR reattempt closes, next push opens round 2 ─────────────────────
// Scenario: act reattempt on the open round → closed; a new failed push
// opens attempt 2 (same reason is a new round, not a block).
func (s *FulfillmentSuite) TestNDR_ActReattemptThenRound2() {
	t := s.T()
	_, shipmentID, awb := s.openNDRBox()
	require.Equal(t, 1, s.ndrAttempt(shipmentID))
	s.stubNDRAction(awb)

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/ndr", shipmentID), map[string]any{
			"action":      "reattempt",
			"addressNote": "Call before delivery",
		})
	helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, 0, s.ndrAttempt(shipmentID), "round closed")

	w = s.pushWebhook(awb, "UNDELIVERED - CONSIGNEE NOT AVAILABLE", "2026-10-04 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 2, s.ndrAttempt(shipmentID), "same reason, new round")
}

// ─── US5: NDR rto action ─────────────────────────────────────────────────────
// Scenario: act rto closes the round with the rto marker.
func (s *FulfillmentSuite) TestNDR_ActRTO() {
	t := s.T()
	_, shipmentID, awb := s.openNDRBox()
	s.stubNDRAction(awb)

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/ndr", shipmentID), map[string]any{
			"action": "rto",
		})
	helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, 0, s.ndrAttempt(shipmentID))
}

// ─── US5: NDR validation ─────────────────────────────────────────────────────
// Scenario: unknown action → 400; acting with no open round → 400.
func (s *FulfillmentSuite) TestNDR_Validation() {
	t := s.T()
	_, shipmentID, _ := s.openNDRBox()

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/ndr", shipmentID), map[string]any{
			"action": "teleport",
		})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

	_, freshID, _ := s.bookBoxWithAWB(2, 1)
	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/ndr", freshID), map[string]any{
			"action": "reattempt",
		})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	require.Equal(t, "FULFILLMENT_CAPABILITY_UNSUPPORTED", errResp["code"])
}

// ─── US5: second push without action stays one round ─────────────────────────
// Scenario: two failed pushes, no seller action → still one open round
// (redelivered scans don't fork rounds).
func (s *FulfillmentSuite) TestNDR_SecondPushStaysOneRound() {
	t := s.T()
	_, shipmentID, awb := s.openNDRBox()

	w := s.pushWebhook(awb, "UNDELIVERED - CONSIGNEE NOT AVAILABLE", "2026-10-03 12:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 1, s.ndrAttempt(shipmentID))
}

// ─── US5: proactive RTO mid-transit ──────────────────────────────────────────
// Scenario: picked box without any NDR → proactive RTO accepted, status
// unchanged until the courier confirms the RTO leg.
func (s *FulfillmentSuite) TestRTO_Proactive() {
	t := s.T()
	s.stubShiprocketBooking()
	s.configureJohnShiprocket()
	_, shipmentID, awb := s.bookBoxWithAWB(2, 1)
	s.stubNDRAction(awb)

	w := s.pushWebhook(awb, "PICKED UP", "2026-10-02 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)

	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/rto", shipmentID), map[string]any{})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "picked",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])
}

// ─── US5: return creates a booked return box ─────────────────────────────────
// Scenario: delivered box + full-line return → 201 return draft, booked
// with its own AWB, linked to the original.
func (s *FulfillmentSuite) TestReturn_CreateAndBook() {
	t := s.T()
	s.stubShiprocketReturn()
	orderID, origID, _ := s.deliverBox(2, 2)

	var itemID float64
	w := s.customerClient.Get(t, fmt.Sprintf("/api/order/%d", orderID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	for _, it := range resp["data"].(map[string]any)["items"].([]any) {
		itemID = it.(map[string]any)["id"].(float64)
	}

	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/returns", origID), map[string]any{
			"reason": "wrong_size",
			"items":  []map[string]any{{"orderItemId": itemID, "quantity": 2}},
		})
	resp = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	ret := resp["data"].(map[string]any)["shipment"].(map[string]any)
	require.Contains(t, []string{"booked", "pickup_scheduled"}, ret["status"])
	require.NotNil(t, ret["awb"])

	w = s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d",
		uint(ret["id"].(float64))))
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, float64(origID),
		resp["data"].(map[string]any)["shipment"].(map[string]any)["returnOfShipmentId"])
}

// ─── US5: return of a return rejected ────────────────────────────────────────
// Scenario: requesting a return on a return box → 400.
func (s *FulfillmentSuite) TestReturn_OfReturnRejected() {
	t := s.T()
	s.stubShiprocketReturn()
	orderID, origID, _ := s.deliverBox(2, 2)

	var itemID float64
	w := s.customerClient.Get(t, fmt.Sprintf("/api/order/%d", orderID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	for _, it := range resp["data"].(map[string]any)["items"].([]any) {
		itemID = it.(map[string]any)["id"].(float64)
	}

	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/returns", origID), map[string]any{
			"reason": "wrong_size",
			"items":  []map[string]any{{"orderItemId": itemID, "quantity": 2}},
		})
	resp = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	retID := uint(resp["data"].(map[string]any)["shipment"].(map[string]any)["id"].(float64))

	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/returns", retID), map[string]any{
			"reason": "again",
			"items":  []map[string]any{{"orderItemId": itemID, "quantity": 1}},
		})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
}

// ─── US5: over-quantity return rejected ──────────────────────────────────────
// Scenario: return qty above the original box → 400, nothing created.
func (s *FulfillmentSuite) TestReturn_OverQuantityRejected() {
	t := s.T()
	s.stubShiprocketReturn()
	orderID, origID, _ := s.deliverBox(2, 2)

	var itemID float64
	w := s.customerClient.Get(t, fmt.Sprintf("/api/order/%d", orderID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	for _, it := range resp["data"].(map[string]any)["items"].([]any) {
		itemID = it.(map[string]any)["id"].(float64)
	}

	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/returns", origID), map[string]any{
			"reason": "wrong_size",
			"items":  []map[string]any{{"orderItemId": itemID, "quantity": 3}},
		})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
}

// ─── US5: return delivery restocks exactly once ─────────────────────────────
// Scenario: return box delivered → order returned, stock +2; a redelivered
// push changes nothing more.
func (s *FulfillmentSuite) TestReturn_DeliveredRestocksOnce() {
	t := s.T()
	s.stubShiprocketReturn()
	orderID, origID, _ := s.deliverBox(2, 2)

	var itemID float64
	w := s.customerClient.Get(t, fmt.Sprintf("/api/order/%d", orderID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	for _, it := range resp["data"].(map[string]any)["items"].([]any) {
		itemID = it.(map[string]any)["id"].(float64)
	}

	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/returns", origID), map[string]any{
			"reason": "wrong_size",
			"items":  []map[string]any{{"orderItemId": itemID, "quantity": 2}},
		})
	resp = helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	retBox := resp["data"].(map[string]any)["shipment"].(map[string]any)
	retAWB := retBox["awb"].(string)

	before := s.inventoryQuantity(2, 1)

	w = s.pushWebhook(retAWB, "DELIVERED", "2026-10-05 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)

	w = s.customerClient.Get(t, fmt.Sprintf("/api/order/%d", orderID))
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "returned", resp["data"].(map[string]any)["status"])
	require.Equal(t, before+2, s.inventoryQuantity(2, 1))

	w = s.pushWebhook(retAWB, "DELIVERED", "2026-10-05 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, before+2, s.inventoryQuantity(2, 1), "redelivery restocks nothing")
}

// inventoryQuantity reads live on-hand stock for assertions.
func (s *FulfillmentSuite) inventoryQuantity(variantID, locationID uint) int {
	t := s.T()
	var qty int
	require.NoError(t, s.container.DB.Table("inventory").
		Select("quantity").Where("variant_id = ? AND location_id = ?", variantID, locationID).
		Scan(&qty).Error)
	return qty
}
