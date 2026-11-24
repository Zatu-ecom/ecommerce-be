package fulfillment_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	fulfillmentfactory "ecommerce-be/fulfillment/factory"
	fulfillmentmetrics "ecommerce-be/fulfillment/metrics"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	fulfillmentrepository "ecommerce-be/fulfillment/repository"
	fulfillmentservice "ecommerce-be/fulfillment/service"
	"ecommerce-be/fulfillment/service/courier/shiprocket"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// stubOrderHooks no-ops every order callback: reconcile tests assert box
// movement, not order transitions (covered in the order suite).
type stubOrderHooks struct{}

func (stubOrderHooks) WithOrderLock(
	context.Context, uint, func(context.Context) error,
) error {
	return fmt.Errorf("stub: WithOrderLock unused in reconcile tests")
}

func (stubOrderHooks) GetOrderForFulfillment(
	context.Context, uint,
) (*fulfillmentmodel.FulfillmentOrderView, error) {
	return nil, fmt.Errorf("stub: no order view")
}

func (stubOrderHooks) OnShipmentsPlanned(context.Context, uint, []uint) error {
	return nil
}

func (stubOrderHooks) OnShipmentBooked(context.Context, fulfillmentmodel.FulfillmentProgress) error {
	return nil
}

func (stubOrderHooks) OnShipmentDelivered(context.Context, fulfillmentmodel.FulfillmentProgress) error {
	return nil
}

func (stubOrderHooks) OnShipmentFailed(context.Context, fulfillmentmodel.FulfillmentProgress) error {
	return nil
}

func (stubOrderHooks) OnShipmentReturned(context.Context, fulfillmentmodel.FulfillmentProgress) error {
	return nil
}

// reconcileNow runs the real reconciler (repos + applier + fake-backed
// adapter) the way the cron would. Crons have no HTTP surface, so this is
// the honest integration seam.
func (s *FulfillmentSuite) reconcileNow() (int, error) {
	t := s.T()
	ctx := context.Background()
	shipmentRepo := fulfillmentrepository.NewShipmentRepository()
	applier := fulfillmentservice.NewNormalizedApplier(
		shipmentRepo,
		fulfillmentrepository.NewShipmentItemRepository(),
		fulfillmentrepository.NewShipmentEventRepository(),
		fulfillmentrepository.NewNDRRepository(),
		stubOrderHooks{},
	)
	factory := fulfillmentfactory.NewCourierPartnerFactory(
		fulfillmentrepository.NewCourierProviderRepository(),
		fulfillmentrepository.NewCourierProviderConfigRepository(),
		shiprocket.New(os.Getenv("SHIPROCKET_BASE_URL")),
	)
	reconciler := fulfillmentservice.NewReconcileService(
		shipmentRepo,
		fulfillmentrepository.NewNDRRepository(),
		applier,
		factory,
	)
	// T059: exercise the wired metrics path (log sink) like the factory does.
	reconciler.(*fulfillmentservice.ReconcileServiceImpl).SetMetricsRecorder(
		fulfillmentmetrics.LogRecorder(),
	)
	_ = t
	return reconciler.ReconcilePending(ctx)
}

// stubBulkTrack serves the reconciler bulk endpoint, echoing each
// requested AWB with the given status (bulk items match by their own awb).
func (s *FulfillmentSuite) stubBulkTrack(status string) {
	s.stubShiprocketAuth()
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/courier/track/awbs",
		func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				AWBs []string `json:"awbs"`
			}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &req)
			items := make([]string, 0, len(req.AWBs))
			for _, awb := range req.AWBs {
				item, _ := json.Marshal(map[string]any{
					"awb": awb, "shipment_status": status, "courier_name": "Delhivery Surface",
				})
				items = append(items, string(item))
			}
			writeShiprocketJSON(w, 200, `{"data": [`+strings.Join(items, ",")+`]}`)
		})
}

// ─── US4: customer track is PG-only ──────────────────────────────────────────
// Scenario: box at in_transit in our records; the courier would now say
// DELIVERED. Customer GET still shows in_transit — no courier call, no
// status change.
func (s *FulfillmentSuite) TestTrack_CustomerPGOnly() {
	t := s.T()
	orderID, shipmentID, awb := s.bookBoxWithAWB(2, 1)

	s.pushWebhook(awb, "IN TRANSIT", "2026-10-02 10:00:00", "whsec-test")

	s.fakeShiprocket.stub(http.MethodGet, "/v1/external/courier/track/awb/"+awb,
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"tracking_data": {"shipment_status": "DELIVERED"}}`)
		})

	w := s.customerClient.Get(t, fmt.Sprintf("/api/fulfillment/my/orders/%d/shipments", orderID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	shipments := resp["data"].(map[string]any)["shipments"].([]any)
	require.Len(t, shipments, 1)
	box := shipments[0].(map[string]any)
	require.Equal(t, "in_transit", box["status"])
	require.Equal(t, shipmentID, uint(box["id"].(float64)))
	for _, e := range box["events"].([]any) {
		_, hasCode := e.(map[string]any)["failureCode"]
		require.False(t, hasCode, "no failure codes for customers")
	}
}

// ─── US4: seller refresh writes through ─────────────────────────────────────
// Scenario: seller refresh pulls DELIVERED and applies it — box delivered,
// order completed.
func (s *FulfillmentSuite) TestTrack_SellerRefreshWrites() {
	t := s.T()
	orderID, shipmentID, awb := s.bookBoxWithAWB(2, 1)

	s.fakeShiprocket.stub(http.MethodGet, "/v1/external/courier/track/awb/"+awb,
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"tracking_data": {"shipment_status": "DELIVERED"}}`)
		})

	w := s.johnClient().Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/refresh", shipmentID), map[string]any{})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "delivered",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])

	w = s.customerClient.Get(t, fmt.Sprintf("/api/order/%d", orderID))
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "completed", resp["data"].(map[string]any)["status"])
}

// ─── US4: refresh is idempotent ──────────────────────────────────────────────
// Scenario: two identical refreshes yield one delivered ledger row.
func (s *FulfillmentSuite) TestTrack_RefreshIdempotent() {
	t := s.T()
	_, shipmentID, awb := s.bookBoxWithAWB(2, 1)

	s.fakeShiprocket.stub(http.MethodGet, "/v1/external/courier/track/awb/"+awb,
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"tracking_data": {"shipment_status": "DELIVERED"}}`)
		})

	for i := 0; i < 2; i++ {
		w := s.johnClient().Post(t,
			fmt.Sprintf("/api/fulfillment/shipments/%d/refresh", shipmentID), map[string]any{})
		helpers.AssertSuccessResponse(t, w, http.StatusOK)
	}

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

// ─── US4: webhook-log filters ────────────────────────────────────────────────
// Scenario: applied pushes list with status + awb filters.
func (s *FulfillmentSuite) TestTrack_WebhookLogFilters() {
	t := s.T()
	_, _, awb := s.bookBoxWithAWB(2, 1)
	s.pushWebhook(awb, "IN TRANSIT", "2026-10-02 10:00:00", "whsec-test")

	require.Equal(t, 1, s.webhookLogCount("applied", awb))
	require.Equal(t, 0, s.webhookLogCount("failed", awb))

	w := s.johnClient().Get(t, "/api/fulfillment/webhook-logs?awb="+awb)
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Len(t, resp["data"].(map[string]any)["logs"].([]any), 1)

	w = s.johnClient().Get(t, "/api/fulfillment/webhook-logs?awb=AWB-NOPE")
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Empty(t, resp["data"].(map[string]any)["logs"])
}

// ─── US4: track isolation ────────────────────────────────────────────────────
// Scenario: another seller's refresh and another customer's track are 404s.
func (s *FulfillmentSuite) TestTrack_Isolation() {
	t := s.T()
	orderID, shipmentID, _ := s.bookBoxWithAWB(2, 1)

	w := s.sellerClient.Post(t,
		fmt.Sprintf("/api/fulfillment/shipments/%d/refresh", shipmentID), map[string]any{})
	require.Equal(t, http.StatusNotFound, w.Code)

	otherCustomer := helpers.NewAPIClient(s.server)
	otherCustomer.SetToken(helpers.Login(t, otherCustomer,
		helpers.Customer2Email, helpers.Customer2Password))
	w = otherCustomer.Get(t, fmt.Sprintf("/api/fulfillment/my/orders/%d/shipments", orderID))
	require.Equal(t, http.StatusNotFound, w.Code)
}

// ─── US4: reconciler heals stale boxes ───────────────────────────────────────
// Scenario: a box stuck in_transit with an old sync is picked up by the
// reconciler (direct service call — crons have no HTTP surface) and moved
// to delivered once the courier reports it.
func (s *FulfillmentSuite) TestReconcile_HealsStale() {
	t := s.T()
	_, shipmentID, awb := s.bookBoxWithAWB(2, 1)
	s.pushWebhook(awb, "IN TRANSIT", "2026-10-02 10:00:00", "whsec-test")

	// Age the sync far past any threshold without touching status.
	require.NoError(t, s.container.DB.Exec(
		"UPDATE fulfillment_shipment SET last_synced_at = NOW() - INTERVAL '30 hours' WHERE id = ?",
		shipmentID).Error)

	s.fakeShiprocket.stub(http.MethodGet, "/v1/external/courier/track/awb/"+awb,
		func(w http.ResponseWriter, _ *http.Request) {
			writeShiprocketJSON(w, 200, `{"tracking_data": {"shipment_status": "DELIVERED"}}`)
		})
	s.stubBulkTrack("DELIVERED")

	recovered, err := s.reconcileNow()
	require.NoError(t, err)
	require.GreaterOrEqual(t, recovered, 1)

	w := s.johnClient().Get(t, fmt.Sprintf("/api/fulfillment/shipments/%d", shipmentID))
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, "delivered",
		resp["data"].(map[string]any)["shipment"].(map[string]any)["status"])
}
