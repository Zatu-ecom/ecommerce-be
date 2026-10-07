package fulfillment_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// orderItemID returns the first order line id for an order (customer view).
func (s *FulfillmentSuite) orderItemID(orderID uint) float64 {
	t := s.T()
	w := s.customerClient.Get(t, fmt.Sprintf("/api/order/%d", orderID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	items := resp["data"].(map[string]any)["items"].([]any)
	require.NotEmpty(t, items)
	return items[0].(map[string]any)["id"].(float64)
}

// ─── US2: manual create over covered lines is refused ────────────────────────
// Scenario: planner covered every line; a manual draft for the same lines
// → 409, no new row.
func (s *FulfillmentSuite) TestDrafts_ManualOverAllocate409() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)
	itemID := s.orderItemID(orderID)

	w := s.johnClient().Post(t, "/api/fulfillment/shipments", map[string]any{
		"orderId":          orderID,
		"pickupLocationId": 1,
		"items":            []map[string]any{{"orderItemId": itemID, "quantity": 1}},
	})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusConflict)
	require.Equal(t, "FULFILLMENT_INVALID_STATE", errResp["code"])
	require.Len(t, s.listShipments(orderID), 1, "no extra draft")
}

// ─── US2: manual create on pending order ─────────────────────────────────────
// Scenario: unconfirmed order → 409 INVALID_STATE.
func (s *FulfillmentSuite) TestDrafts_ManualOnPending409() {
	t := s.T()
	s.clearCart()
	s.customerClient.Post(t, "/api/order/cart/item", helpers.AddCartItemsPayload(2, 1))
	w := s.customerClient.Post(t, "/api/order", helpers.DefaultCreateOrderRequest())
	resp := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	orderID := uint(resp["data"].(map[string]any)["id"].(float64))

	w = s.johnClient().Post(t, "/api/fulfillment/shipments", map[string]any{
		"orderId":          orderID,
		"pickupLocationId": 1,
		"items":            []map[string]any{{"orderItemId": 999999, "quantity": 1}},
	})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusConflict)
	require.Equal(t, "FULFILLMENT_INVALID_STATE", errResp["code"])
}

// ─── US2: manual create validation ──────────────────────────────────────────
// Scenario: missing items and zero quantity are 400s.
func (s *FulfillmentSuite) TestDrafts_ManualValidation400() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)

	w := s.johnClient().Post(t, "/api/fulfillment/shipments", map[string]any{
		"orderId":          orderID,
		"pickupLocationId": 1,
	})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

	w = s.johnClient().Post(t, "/api/fulfillment/shipments", map[string]any{
		"orderId":          orderID,
		"pickupLocationId": 1,
		"items":            []map[string]any{{"orderItemId": s.orderItemID(orderID), "quantity": 0}},
	})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
}

// ─── US2: manual create foreign line ────────────────────────────────────────
// Scenario: an orderItemId from another order → 404 (not in this order's scope).
func (s *FulfillmentSuite) TestDrafts_ForeignLine404() {
	t := s.T()
	orderA := s.placeAndConfirmOrder(2, 1)
	orderB := s.placeAndConfirmOrder(1, 1)

	w := s.johnClient().Post(t, "/api/fulfillment/shipments", map[string]any{
		"orderId":          orderA,
		"pickupLocationId": 1,
		"items":            []map[string]any{{"orderItemId": s.orderItemID(orderB), "quantity": 1}},
	})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	require.Equal(t, "FULFILLMENT_NOT_FOUND", errResp["code"])
}

// ─── US2: patch draft weight and dims ────────────────────────────────────────
// Scenario: PATCH on a draft updates measurements; response reflects them.
func (s *FulfillmentSuite) TestDrafts_PatchWeightDims() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := s.listShipments(orderID)[0].(map[string]any)["id"]

	w := s.johnClient().Patch(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d", uint(shipmentID.(float64))), map[string]any{
			"weightGrams": 700,
			"lengthCm":    20,
		})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	box := resp["data"].(map[string]any)["shipment"].(map[string]any)
	require.Equal(t, float64(700), box["weightGrams"])
	require.Equal(t, float64(20), box["lengthCm"])
}

// ─── US2: patch validation ───────────────────────────────────────────────────
// Scenario: empty body, zero/negative weight, and unknown id are handled.
func (s *FulfillmentSuite) TestDrafts_PatchValidation() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))

	w := s.johnClient().Patch(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID), map[string]any{})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

	w = s.johnClient().Patch(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID), map[string]any{
			"weightGrams": 0,
		})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

	w = s.johnClient().Patch(t, "/api/fulfillment/shipments/999999", map[string]any{
		"weightGrams": 100,
	})
	helpers.AssertErrorResponse(t, w, http.StatusNotFound)
}

// ─── US2: seller isolation on drafts ─────────────────────────────────────────
// Scenario: another seller's reads and writes on these boxes are 404s.
func (s *FulfillmentSuite) TestDrafts_SellerIsolation() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))

	w := s.sellerClient.Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	require.Equal(t, http.StatusNotFound, w.Code)

	w = s.sellerClient.Patch(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID),
		map[string]any{"weightGrams": 100})
	require.Equal(t, http.StatusNotFound, w.Code)

	w = s.sellerClient.Post(t, "/api/fulfillment/shipments", map[string]any{
		"orderId":          orderID,
		"pickupLocationId": 1,
		"items":            []map[string]any{{"orderItemId": s.orderItemID(orderID), "quantity": 1}},
	})
	require.Equal(t, http.StatusNotFound, w.Code)
}

// ─── US2: detail shape vs list shape ─────────────────────────────────────────
// Scenario: detail carries items/events/ndr; list rows omit them.
func (s *FulfillmentSuite) TestDrafts_DetailVsListShape() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)
	shipmentID := uint(s.listShipments(orderID)[0].(map[string]any)["id"].(float64))

	listRow := s.listShipments(orderID)[0].(map[string]any)
	_, hasEvents := listRow["events"]
	require.False(t, hasEvents, "list omits events")

	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	detail := resp["data"].(map[string]any)["shipment"].(map[string]any)
	require.Contains(t, detail, "items")
	require.Contains(t, detail, "events")
}
