package fulfillment_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// stubServiceability serves two priced couriers (or none when empty).
func (s *FulfillmentSuite) stubServiceability(companies string) {
	s.stubShiprocketAuth()
	s.fakeShiprocket.stub(http.MethodGet, "/v1/external/courier/serviceability/",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"data": {"available_courier_companies": [`+companies+`]}}`)
		})
}

const twoCouriers = `
{"courier_name": "Delhivery Surface", "courier_company_id": "7", "rate": 82.5, "etd": "2026-10-05"},
{"courier_name": "XpressBees Air", "courier_company_id": "9", "rate": 120.0, "etd": "2026-10-04"}`

// ─── US3: rates return options ───────────────────────────────────────────────
// Scenario: live rate call maps to priced options (paise, not rupees).
func (s *FulfillmentSuite) TestRates_ReturnOptions() {
	t := s.T()
	s.stubServiceability(twoCouriers)
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)

	w := s.johnClient().Post(t, "/api/fulfillment/rates", map[string]any{
		"providerCode":     "shiprocket",
		"orderId":          orderID,
		"pickupLocationId": 1,
		"weightGrams":      650,
	})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	options := resp["data"].(map[string]any)["options"].([]any)
	require.Len(t, options, 2)

	first := options[0].(map[string]any)
	require.Equal(t, "Delhivery Surface", first["courierName"])
	rate := first["rate"].(map[string]any)
	require.Equal(t, float64(8250), rate["amountCents"], "rupees → paise")
	require.NotNil(t, first["etd"])
}

// ─── US3: empty rate list is 200 ─────────────────────────────────────────────
// Scenario: unserved route → 200 with empty options, never an error.
func (s *FulfillmentSuite) TestRates_EmptyOptions200() {
	t := s.T()
	s.stubServiceability("")
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)

	w := s.johnClient().Post(t, "/api/fulfillment/rates", map[string]any{
		"providerCode":     "shiprocket",
		"orderId":          orderID,
		"pickupLocationId": 1,
		// Different weight from the populated test: rate cache keys include
		// weight, and shared test Redis would otherwise serve its rows here.
		"weightGrams": 651,
	})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	options := resp["data"].(map[string]any)["options"].([]any)
	require.Empty(t, options)
}

// ─── US3: label streams PDF ──────────────────────────────────────────────────
// Scenario: booked box label downloads as PDF bytes with the right content
// type; the file is never stored server-side.
func (s *FulfillmentSuite) TestLabel_StreamsPDF() {
	t := s.T()
	pdf := []byte("%PDF-1.4 e2e-label")
	s.stubShiprocketAuth()
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/courier/generate/label",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"label_url": "`+s.labelURL()+`"}`)
		})
	s.fakeShiprocket.stub(http.MethodGet, "/labels/e2e.pdf",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(200)
			_, _ = w.Write(pdf)
		})
	s.stubShiprocketBooking()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)
	s.bookDraft(orderID, shipmentID)

	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d/label", shipmentID))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "application/pdf")
	require.Equal(t, pdf, w.Body.Bytes())
}

// ─── US3: label on draft without AWB ─────────────────────────────────────────
// Scenario: nothing to fetch yet → 400, not a courier call.
func (s *FulfillmentSuite) TestLabel_DraftWithoutAWB400() {
	t := s.T()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))

	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d/label", shipmentID))
	errResp := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	require.Equal(t, "FULFILLMENT_LABEL_FAILED", errResp["code"])
}

// labelURL points the fake label-generate response at the fake file.
func (s *FulfillmentSuite) labelURL() string {
	return "http://" + s.fakeShiprocketBase() + "/labels/e2e.pdf"
}

func (s *FulfillmentSuite) fakeShiprocketBase() string {
	base := s.fakeShiprocket.server.URL
	if len(base) > 7 && base[:7] == "http://" {
		return base[7:]
	}
	return base
}

// ─── US3: pickup schedules separately ────────────────────────────────────────
// Scenario: booked-without-pickup box → explicit pickup schedules it.
func (s *FulfillmentSuite) TestPickup_Schedules() {
	t := s.T()
	s.stubShiprocketBooking()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)

	// Book WITHOUT pickupAt.
	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/book", shipmentID), map[string]any{
			"providerCode": "shiprocket",
		})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "booked",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])

	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/pickup", shipmentID), map[string]any{
			"pickupAt": "2026-10-03T10:00:00Z",
		})
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "pickup_scheduled",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])
}

// ─── US3: draft cancel is local ──────────────────────────────────────────────
// Scenario: draft cancel needs no courier call and keeps stock semantics
// local (no AWB exists to cancel remotely).
func (s *FulfillmentSuite) TestCancel_DraftLocal() {
	t := s.T()
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)

	// Book WITHOUT pickupAt.
	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/cancel", shipmentID), map[string]any{})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "cancelled",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])
}

// ─── US3: booked cancel calls courier ────────────────────────────────────────
// Scenario: booked box cancel calls the provider then marks cancelled.
func (s *FulfillmentSuite) TestCancel_BookedCallsCourier() {
	t := s.T()
	cancelCalls := 0
	s.stubShiprocketBooking()
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/orders/cancel",
		func(w http.ResponseWriter, _ *http.Request) {
			cancelCalls++
			writeShiprocketJSON(w, 200, `{"success": true}`)
		})
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)
	s.bookDraft(orderID, shipmentID)

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/cancel", shipmentID), map[string]any{})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "cancelled",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])
	require.Equal(t, 1, cancelCalls)
}

// ─── US3: courier refusal keeps state ────────────────────────────────────────
// Scenario: provider 500 on cancel → 409, status and stock unchanged.
func (s *FulfillmentSuite) TestCancel_CourierRefusal409() {
	t := s.T()
	s.stubShiprocketBooking()
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/orders/cancel",
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 500, `{"message":"already picked"}`)
		})
	s.configureJohnShiprocket()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))
	s.patchWeight(shipmentID, 700)
	s.bookDraft(orderID, shipmentID)

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/cancel", shipmentID), map[string]any{})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusConflict)
	require.Equal(t, "FULFILLMENT_INVALID_STATE", errResp["code"])

	w = s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	status := resp["data"].(map[string]any)["shipment"].(map[string]any)["status"]
	require.Contains(t, []string{"booked", "pickup_scheduled"}, status)
}
