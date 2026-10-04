package fulfillment_test

import (
	"fmt"
	"net/http"
	"sync"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// ─── Fixtures ────────────────────────────────────────────────────────────────
// Seller-2 topology (john.seller@example.com owns the iPhone catalog and
// warehouses 1/2): variant 1 lives in both warehouses, variant 4 splits
// 8 @ loc1 / 4 @ loc2 (1 reserved), variant 2 sits only in loc1.

// johnClient logs in as the seller owning the fixture catalog.
func (s *FulfillmentSuite) johnClient() *helpers.APIClient {
	t := s.T()
	client := helpers.NewAPIClient(s.server)
	token := helpers.Login(t, client, helpers.Seller2Email, helpers.Seller2Password)
	client.SetToken(token)
	return client
}

// clearCart empties the customer's active cart (best-effort).
func (s *FulfillmentSuite) clearCart() {
	t := s.T()
	w := s.customerClient.Get(t, "/api/order/cart")
	if w.Code != http.StatusOK {
		return
	}
	resp := helpers.ParseResponse(t, w.Body)
	data, _ := resp["data"].(map[string]any)
	id, _ := data["id"].(float64)
	if id == 0 {
		return
	}
	s.customerClient.Delete(t, fmt.Sprintf("/api/order/cart/%d", uint(id)))
}

// placeAndConfirmOrder checks out variant qty as the customer and confirms
// as the owning seller. Auto-plan fires on the confirm commit (T032), so a
// covered order already has drafts on return.
func (s *FulfillmentSuite) placeAndConfirmOrder(variantID uint, qty int) uint {
	t := s.T()
	s.clearCart()

	w := s.customerClient.Post(t, "/api/order/cart/item",
		helpers.AddCartItemsPayload(variantID, qty))
	helpers.AssertSuccessResponse(t, w, http.StatusCreated)

	w = s.customerClient.Post(t, "/api/order", helpers.DefaultCreateOrderRequest())
	resp := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	orderID := uint(resp["data"].(map[string]any)["id"].(float64))

	john := s.johnClient()
	w = john.Patch(t, fmt.Sprintf("/api/order/%d/status", orderID), map[string]any{
		"status":        "confirmed",
		"transactionId": fmt.Sprintf("txn-plan-%d-%d", orderID, qty),
	})
	helpers.AssertSuccessResponse(t, w, http.StatusOK)
	return orderID
}

// listShipments returns the seller-visible box list for an order.
func (s *FulfillmentSuite) listShipments(orderID uint) []any {
	t := s.T()
	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments?orderId=%d", orderID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	shipments, ok := resp["data"].(map[string]any)["shipments"].([]any)
	require.True(t, ok, "data.shipments must be a list")
	return shipments
}

func shipmentByID(shipments []any, id float64) map[string]any {
	for _, sh := range shipments {
		if sh.(map[string]any)["id"] == id {
			return sh.(map[string]any)
		}
	}
	return nil
}

// ─── US2: auto-plan on confirm, single warehouse ─────────────────────────────
// Scenario: variant stocked in one warehouse → exactly one draft with the
// warehouse + delivery refs and full quantity. PATCH-confirm records payment
// (PaidAt set), so this prepaid-style flow collects no COD.
func (s *FulfillmentSuite) TestPlan_AutoPlansOnConfirm() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)

	shipments := s.listShipments(orderID)
	require.Len(t, shipments, 1)
	box := shipments[0].(map[string]any)
	require.Equal(t, "draft", box["status"])
	require.Equal(t, float64(1), box["pickupLocationId"])
	require.NotEqual(t, float64(0), box["deliveryAddressId"])
	require.Nil(t, box["providerCode"])
	require.Nil(t, box["awb"])

	items, ok := box["items"].([]any)
	require.True(t, ok && len(items) == 1, "one box line")
	require.Equal(t, float64(1), items[0].(map[string]any)["quantity"])

	cod, ok := box["cod"].(map[string]any)
	require.True(t, ok, "cod object stamped")
	require.Equal(t, float64(0), cod["amountCents"], "PATCH-confirmed means paid")
}

// ─── US6: planner stamps draft weight from catalog specs ─────────────────────
// Scenario: product 1 carries weight_g = 650; confirming an order for
// variant 2 × 3 auto-plans drafts whose stamped weights sum to 1950g.
func (s *FulfillmentSuite) TestPlan_WeightFillFromCatalogSpecs() {
	t := s.T()
	w := s.johnClient().Post(t, "/api/product/1/attribute", map[string]any{
		"attributeDefinitionId": 100,
		"value":                 "650",
	})
	helpers.AssertSuccessResponse(t, w, http.StatusCreated)

	orderID := s.placeAndConfirmOrder(2, 3)
	shipments := s.listShipments(orderID)
	require.NotEmpty(t, shipments)
	var total float64
	for _, sh := range shipments {
		box := sh.(map[string]any)
		weight, ok := box["weightGrams"].(float64)
		require.True(t, ok, "draft carries stamped weight")
		total += weight
	}
	require.Equal(t, float64(3*650), total)
}

// ─── US2: direct-confirmed (COD) order plans with collect amount ─────────────
// Scenario: an order born confirmed (COD creation path, no payment) is
// auto-planned by the CreateOrder trigger with cod == order total.
func (s *FulfillmentSuite) TestPlan_DirectConfirmedCollectsCOD() {
	t := s.T()
	s.clearCart()
	s.customerClient.Post(t, "/api/order/cart/item", helpers.AddCartItemsPayload(2, 1))
	body := helpers.DefaultCreateOrderRequest()
	body["status"] = "confirmed"
	w := s.customerClient.Post(t, "/api/order", body)
	resp := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	orderID := uint(resp["data"].(map[string]any)["id"].(float64))

	shipments := s.listShipments(orderID)
	require.Len(t, shipments, 1)
	cod := shipments[0].(map[string]any)["cod"].(map[string]any)
	require.Greater(t, cod["amountCents"], float64(0), "unpaid order collects COD")
}

// ─── US2: replan covered order returns existing ─────────────────────────────
// Scenario: explicit plan on a fully covered order → 200 with the live set,
// no new rows.
func (s *FulfillmentSuite) TestPlan_ReplanCoveredReturnsExisting() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)
	before := s.listShipments(orderID)
	beforeID := before[0].(map[string]any)["id"]

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/orders/%d/plan", orderID), map[string]any{})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	shipments := resp["data"].(map[string]any)["shipments"].([]any)
	require.Len(t, shipments, 1)
	require.Equal(t, beforeID, shipments[0].(map[string]any)["id"])

	after := s.listShipments(orderID)
	require.Len(t, after, 1, "no duplicate drafts")
}

// ─── US2: split across warehouses ────────────────────────────────────────────
// Scenario: a quantity no single warehouse covers follows the checkout
// holds across both warehouses — one draft per holding warehouse summing
// to the ordered qty. Assertions stay allocator-agnostic (sum, box count,
// distinct locations) since checkout owns placement.
func (s *FulfillmentSuite) TestPlan_SplitsAcrossWarehouses() {
	t := s.T()
	// Cart availability subtracts thresholds, so zero them for variant 4:
	// otherwise no orderable quantity can exceed a single warehouse.
	require.NoError(t, s.container.DB.Exec(
		"UPDATE inventory SET quantity = 50, threshold = 0 WHERE variant_id = 4 AND location_id = 2").Error)
	require.NoError(t, s.container.DB.Exec(
		"UPDATE inventory SET threshold = 0 WHERE variant_id = 4 AND location_id = 1").Error)
	orderID := s.placeAndConfirmOrder(4, 50)

	shipments := s.listShipments(orderID)
	require.Len(t, shipments, 2, "neither warehouse covers 50 alone")

	total := 0
	locations := map[float64]bool{}
	for _, sh := range shipments {
		box := sh.(map[string]any)
		require.Equal(t, "draft", box["status"])
		locations[box["pickupLocationId"].(float64)] = true
		for _, it := range box["items"].([]any) {
			total += int(it.(map[string]any)["quantity"].(float64))
		}
	}
	require.Equal(t, 50, total)
	require.True(t, locations[1] && locations[2], "one box per holding warehouse")
}

// ─── US2: plan on pending order ──────────────────────────────────────────────
// Scenario: unconfirmed order → 409 INVALID_STATE, nothing created.
func (s *FulfillmentSuite) TestPlan_PendingOrder409() {
	t := s.T()
	s.clearCart()
	s.customerClient.Post(t, "/api/order/cart/item", helpers.AddCartItemsPayload(2, 1))
	w := s.customerClient.Post(t, "/api/order", helpers.DefaultCreateOrderRequest())
	resp := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	orderID := uint(resp["data"].(map[string]any)["id"].(float64))

	w = s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/orders/%d/plan", orderID), map[string]any{})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusConflict)
	require.Equal(t, "FULFILLMENT_INVALID_STATE", errResp["code"])
	require.Empty(t, s.listShipments(orderID))
}

// ─── US2: concurrent double plan, no dupes ───────────────────────────────────
// Scenario: racing plan calls converge on the covered set — the order-row
// lock plus coverage re-check inside the plan tx make duplicates impossible.
func (s *FulfillmentSuite) TestPlan_ConcurrentDoublePlanNoDupes() {
	t := s.T()
	orderID := s.placeAndConfirmOrder(2, 1)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.johnClient().Post(t,
				fmt.Sprintf("/api/fulfillment/orders/%d/plan", orderID), map[string]any{})
		}()
	}
	wg.Wait()

	require.Len(t, s.listShipments(orderID), 1)
}
