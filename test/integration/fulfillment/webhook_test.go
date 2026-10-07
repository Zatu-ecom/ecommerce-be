package fulfillment_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// ─── Shared webhook helpers ──────────────────────────────────────────────────
// Webhook pushes carry no JWT and no correlation id (skip rules generate
// one); the X-Api-Key signature is the auth.

// bookBoxWithAWB places, weights, and books one box (no pickupAt), returning
// order, shipment, and the fake AWB.
func (s *FulfillmentSuite) bookBoxWithAWB(variantID uint, qty int) (orderID, shipmentID uint, awb string) {
	t := s.T()
	s.stubShiprocketBooking()
	s.configureJohnShiprocket()
	orderID = s.placeAndConfirmOrder(variantID, qty)
	shipmentID = uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/book", shipmentID), map[string]any{
			"providerCode": "shiprocket",
		})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	box := resp["data"].(map[string]any)["shipment"].(map[string]any)
	return orderID, shipmentID, box["awb"].(string)
}

// pushWebhook POSTs a raw courier push. key is the X-Api-Key signature.
// Set per call: the suite client is shared, so headers must never leak.
func (s *FulfillmentSuite) pushWebhook(awb, status, date, key string) *httptest.ResponseRecorder {
	t := s.T()
	s.webhookClient.SetHeader("X-Api-Key", key)
	body := fmt.Sprintf(
		`{"awb": "%s", "courier_name": "Delhivery Surface", "current_status": "%s", "order_id": "77", "date": "%s"}`,
		awb, status, date)
	return s.webhookClient.PostRaw(t, "/api/fulfillment/webhooks/shiprocket", []byte(body))
}

// webhookLogCount counts seller-visible webhook rows, optionally filtered.
// Counts are per-AWB: the suite database is shared, so absolute totals
// would couple tests together.
func (s *FulfillmentSuite) webhookLogCount(status, awb string) int {
	t := s.T()
	path := "/api/fulfillment/webhook-logs?awb=" + awb
	if status != "" {
		path += "&status=" + status
	}
	w := s.johnClient().Get(t, path)
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	logs, ok := resp["data"].(map[string]any)["logs"].([]any)
	require.True(t, ok, "data.logs must be a list")
	return len(logs)
}

// ─── US4: verified push applies ──────────────────────────────────────────────
// Scenario: IN TRANSIT then DELIVERED advance the box; delivery completes
// the single-box order (customer-visible).
func (s *FulfillmentSuite) TestWebhook_ValidApply() {
	t := s.T()
	orderID, _, awb := s.bookBoxWithAWB(2, 1)

	w := s.pushWebhook(awb, "IN TRANSIT", "2026-10-02 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)

	w = s.pushWebhook(awb, "DELIVERED", "2026-10-03 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)

	w = s.customerClient.Get(t, fmt.Sprintf("/api/order/%d", orderID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "completed", resp["data"].(map[string]any)["status"])
	require.Equal(t, 2, s.webhookLogCount("applied", awb))
}

// ─── US4: bad signature stores nothing ───────────────────────────────────────
// Scenario: wrong key → 401, no webhook rows, box untouched.
func (s *FulfillmentSuite) TestWebhook_BadSignature401() {
	t := s.T()
	_, _, awb := s.bookBoxWithAWB(2, 1)

	w := s.pushWebhook(awb, "DELIVERED", "2026-10-03 10:00:00", "wrong-key")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Equal(t, 0, s.webhookLogCount("", awb))
}

// ─── US4: unknown AWB is 200 with no rows ─────────────────────────────────────
// Scenario: couriers retry 4xx until disabled, so unknown tracking numbers
// must never be 4xx — and nothing is stored.
func (s *FulfillmentSuite) TestWebhook_UnknownAWB200() {
	t := s.T()
	s.bookBoxWithAWB(2, 1)

	w := s.pushWebhook("AWB-NEVER-EXISTED", "DELIVERED", "2026-10-03 10:00:00", "whsec-test")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, s.webhookLogCount("", "AWB-NEVER-EXISTED"))
}

// ─── US4: replay applies once ────────────────────────────────────────────────
// Scenario: identical redelivery → 200, one log row, one delivered ledger
// event (no double order effects).
func (s *FulfillmentSuite) TestWebhook_ReplaySingleApply() {
	t := s.T()
	_, shipmentID, awb := s.bookBoxWithAWB(2, 1)

	for i := 0; i < 2; i++ {
		w := s.pushWebhook(awb, "DELIVERED", "2026-10-03 10:00:00", "whsec-test")
		require.Equal(t, http.StatusOK, w.Code)
	}
	require.Equal(t, 1, s.webhookLogCount("", awb))

	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	delivered := 0
	for _, e := range resp["data"].(map[string]any)["shipment"].(map[string]any)["events"].([]any) {
		if e.(map[string]any)["toStatus"] == "delivered" {
			delivered++
		}
	}
	require.Equal(t, 1, delivered, "delivered exactly once")
}

// ─── US4: out-of-order movement allowed ──────────────────────────────────────
// Scenario: OUT FOR DELIVERY then IN TRANSIT (deferred back to hub) lands
// on in_transit — the allow-list is not a ladder.
func (s *FulfillmentSuite) TestWebhook_OutOfOrderAllowed() {
	t := s.T()
	_, shipmentID, awb := s.bookBoxWithAWB(2, 1)

	require.Equal(t, http.StatusOK,
		s.pushWebhook(awb, "OUT FOR DELIVERY", "2026-10-02 10:00:00", "whsec-test").Code)
	require.Equal(t, http.StatusOK,
		s.pushWebhook(awb, "IN TRANSIT", "2026-10-02 12:00:00", "whsec-test").Code)

	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "in_transit",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])
}

// ─── US4: terminal states never regress ──────────────────────────────────────
// Scenario: DELIVERED then IN TRANSIT stays delivered.
func (s *FulfillmentSuite) TestWebhook_TerminalNoRegress() {
	t := s.T()
	_, shipmentID, awb := s.bookBoxWithAWB(2, 1)

	require.Equal(t, http.StatusOK,
		s.pushWebhook(awb, "DELIVERED", "2026-10-03 10:00:00", "whsec-test").Code)
	require.Equal(t, http.StatusOK,
		s.pushWebhook(awb, "IN TRANSIT", "2026-10-03 12:00:00", "whsec-test").Code)

	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "delivered",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])
}

// ─── US4: NDR push opens a round ─────────────────────────────────────────────
// Scenario: failed-attempt push moves the box to ndr_pending with exactly
// one open round for the seller to act on.
func (s *FulfillmentSuite) TestWebhook_NDROpensRound() {
	t := s.T()
	_, shipmentID, awb := s.bookBoxWithAWB(2, 1)

	require.Equal(t, http.StatusOK,
		s.pushWebhook(awb, "UNDELIVERED - CONSIGNEE NOT AVAILABLE", "2026-10-03 10:00:00", "whsec-test").Code)

	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	detail := resp["data"].(map[string]any)["shipment"].(map[string]any)
	require.Equal(t, "ndr_pending", detail["status"])
	ndr, ok := detail["ndr"].(map[string]any)
	require.True(t, ok && ndr != nil, "open NDR round present")
	require.Equal(t, float64(1), ndr["attemptNo"])
}
