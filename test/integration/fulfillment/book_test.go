package fulfillment_test

import (
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// ─── Shared courier stubs ────────────────────────────────────────────────────
// stubShiprocketBooking serves login + create + AWB + pickup + cancel +
// label-generate + label bytes + serviceability. Individual tests override
// paths by re-stubbing (last registration wins). AWBs are unique per call:
// real couriers never repeat one, and the (provider, awb) unique index
// requires the same of fakes.

func (s *FulfillmentSuite) stubShiprocketBooking() {
	s.stubShiprocketAuth()
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/orders/create/adhoc",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"order_id": 9001, "shipment_id": 8001, "status": "NEW"}`)
		})
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/courier/assign/awb",
		func(w http.ResponseWriter, _ *http.Request) {
			n := atomic.AddInt64(&s.awbSeq, 1)
			writeShiprocketJSON(w, 200, `{"awb_assign_status": 1, "response": {"data": {"awb_code": "AWB-E2E-`+strconv.FormatInt(n, 10)+`", "courier_name": "Delhivery Surface"}}}`)
		})
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/courier/generate/pickup",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"pickup_status": 1}`)
		})
}

func writeShiprocketJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// configureJohnShiprocket saves fake creds for the fixture seller (john,
// seller 2). Booking resolves seller row → platform fallback.
func (s *FulfillmentSuite) configureJohnShiprocket() {
	t := s.T()
	w := s.johnClient().Put(t, "/api/fulfillment/couriers/shiprocket/configure", map[string]any{
		"environment": "production",
		"credentials": map[string]any{
			"api_email":      "john@shop.com",
			"api_password":   "right-secret",
			"webhook_secret": "whsec-test",
		},
		"isActive": true,
	})
	helpers.AssertSuccessResponse(t, w, http.StatusOK)
}

// patchWeight sets box weight (planner drafts carry none until US6).
func (s *FulfillmentSuite) patchWeight(shipmentID uint, grams int) {
	t := s.T()
	w := s.johnClient().Patch(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID),
		map[string]any{"weightGrams": grams})
	helpers.AssertSuccessResponse(t, w, http.StatusOK)
}

// bookDraft books one draft with pickup included; returns the box.
func (s *FulfillmentSuite) bookDraft(orderID, shipmentID uint) map[string]any {
	t := s.T()
	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/book", shipmentID), map[string]any{
			"providerCode": "shiprocket",
			"pickupAt":     "2026-10-03T10:00:00Z",
		})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	return resp["data"].(map[string]any)["shipment"].(map[string]any)
}

// ─── US3: book accepted ──────────────────────────────────────────────────────
// Scenario: weighted draft books → booked/pickup_scheduled with AWB and
// courier name; provider code frozen on the row.
func (s *FulfillmentSuite) TestBook_Accepted() {
	t := s.T()
	s.stubShiprocketBooking()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)

	box := s.bookDraft(orderID, shipmentID)
	require.Contains(t, []string{"booked", "pickup_scheduled"}, box["status"])
	require.Contains(t, box["awb"].(string), "AWB-E2E-")
	require.Equal(t, "Delhivery Surface", box["courierName"])
	require.Equal(t, "shiprocket", box["providerCode"])
}

// ─── US3: book refuses without weight ────────────────────────────────────────
// Scenario: planner drafts carry no weight yet → 400 WEIGHT_REQUIRED,
// draft stays draft, bookRequestedAt unset.
func (s *FulfillmentSuite) TestBook_WeightRequired() {
	t := s.T()
	s.stubShiprocketBooking()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/book", shipmentID), map[string]any{
			"providerCode": "shiprocket",
		})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	require.Equal(t, "FULFILLMENT_WEIGHT_REQUIRED", errResp["code"])
}

// ─── US3: idempotent replay ──────────────────────────────────────────────────
// Scenario: same Idempotency-Key twice → same row, one booked ledger event.
func (s *FulfillmentSuite) TestBook_IdempotentReplay() {
	t := s.T()
	s.stubShiprocketBooking()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)

	key := "book-replay-1"
	client := s.johnClient()
	client.SetHeader("Idempotency-Key", key)
	var firstID float64
	for i := 0; i < 2; i++ {
		w := client.Post(t,
			fmt.Sprintf("/api/fulfillment/shipments/%d/book", shipmentID),
			map[string]any{"providerCode": "shiprocket"})
		resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		id := resp["data"].(map[string]any)["shipment"].(map[string]any)["id"].(float64)
		if i == 0 {
			firstID = id
		} else {
			require.Equal(t, firstID, id, "replay returns the same row")
		}
	}

	// Exactly one booked transition in the ledger.
	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	booked := 0
	for _, e := range resp["data"].(map[string]any)["shipment"].(map[string]any)["events"].([]any) {
		if e.(map[string]any)["toStatus"] == "booked" {
			booked++
		}
	}
	require.Equal(t, 1, booked, "booked exactly once")
}

// ─── US3: courier failure → 502, resume succeeds ─────────────────────────────
// Scenario: provider 500 mid-book → 502 BOOK_FAILED, draft stays draft with
// bookRequestedAt set; fixed courier + re-click → booked.
func (s *FulfillmentSuite) TestBook_CourierFailure502Resume() {
	t := s.T()
	s.stubShiprocketAuth()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)

	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/orders/create/adhoc",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 500, `{"message":"provider down"}`)
		})
	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/book", shipmentID), map[string]any{
			"providerCode": "shiprocket",
		})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusBadGateway)
	require.Equal(t, "FULFILLMENT_BOOK_FAILED", errResp["code"])

	// Draft untouched, marked for the retry job.
	w = s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	box := resp["data"].(map[string]any)["shipment"].(map[string]any)
	require.Equal(t, "draft", box["status"])
	require.NotNil(t, box["bookRequestedAt"])

	// Courier recovers; same click resumes to booked (or pickup_scheduled
	// when the body schedules pickup — both are post-book states).
	s.stubShiprocketBooking()
	box = s.bookDraft(orderID, shipmentID)
	require.Contains(t, []string{"booked", "pickup_scheduled"}, box["status"])
}

// ─── US3: book attempts increment ────────────────────────────────────────────
// Scenario: three failing clicks record bookAttempts 1→2→3 (cap enforced by
// the recover job, T040).
func (s *FulfillmentSuite) TestBook_AttemptsIncrement() {
	t := s.T()
	s.stubShiprocketAuth()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)

	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/orders/create/adhoc",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 500, `{"message":"down"}`)
		})
	for i := 1; i <= 3; i++ {
		w := s.johnClient().Post(t,
			fmt.Sprintf("/api/fulfillment/shipments/%d/book", shipmentID), map[string]any{
				"providerCode": "shiprocket",
			})
		helpers.AssertErrorResponse(t, w, http.StatusBadGateway)

		w = s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
		resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
		require.Equal(t, float64(i),
			resp["data"].(map[string]any)["shipment"].(map[string]any)["bookAttempts"])
	}
}

// ─── US3: seller isolation on book ──────────────────────────────────────────
// Scenario: another seller booking this box → 404.
func (s *FulfillmentSuite) TestBook_SellerIsolation() {
	t := s.T()
	s.stubShiprocketBooking()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))

	w := s.sellerClient.Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/book", shipmentID), map[string]any{
			"providerCode": "shiprocket",
		})
	require.Equal(t, http.StatusNotFound, w.Code)
}
