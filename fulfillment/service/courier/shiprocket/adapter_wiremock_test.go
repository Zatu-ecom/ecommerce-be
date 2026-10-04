package shiprocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	commonError "ecommerce-be/common/error"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	courier "ecommerce-be/fulfillment/service/courier"

	"github.com/stretchr/testify/require"
)

func ctx() context.Context { return context.Background() }

func intPtr(v int) *int { return &v }

// wiremock spins an httptest server with per-path handlers and an adapter
// pointed at it. Unmapped paths 404 so missing stubs fail loudly.
func wiremock(t *testing.T, routes map[string]http.HandlerFunc) (*Adapter, *int64) {
	t.Helper()
	var calls int64
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		if h, ok := routes[r.Method+" "+r.URL.Path]; ok {
			h(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"unstubbed"}`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return New(server.URL), &calls
}

func testCreds() map[string]any {
	return map[string]any{"api_email": "api@shop.com", "api_password": "pw"}
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// ─── Book happy path ─────────────────────────────────────────────────────────
// create adhoc → assign AWB → pickup (PickupAt set). The provider order id
// sent MUST be our shipment id (webhook locate-back depends on it).
func TestBookShipment_HappyPath(t *testing.T) {
	var createBody map[string]any
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"POST /v1/external/orders/create/adhoc": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&createBody)
			writeJSON(w, 200, `{"order_id": 9001, "shipment_id": 8001, "status": "NEW"}`)
		},
		"POST /v1/external/courier/assign/awb": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"awb_assign_status": 1, "response": {"data": {"awb_code": "AWB1", "courier_name": "Delhivery Surface"}}}`)
		},
		"POST /v1/external/courier/generate/pickup": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"pickup_status": 1}`)
		},
	})

	out, err := a.BookShipment(ctx(), fulfillmentmodel.BookShipmentInput{
		ShipmentID: 77, OrderID: 15, SellerID: 2,
		PickupLocationID: 8, WeightGrams: intPtr(650),
		Items:         []fulfillmentmodel.ShipmentItemInput{{OrderItemID: 1, Quantity: 2}},
		RecipientName: "Ramesh", RecipientPhone: "9876543210",
		Street: "1 Main", City: "Pune", State: "MH", Pincode: "411001",
		Lines:    []fulfillmentmodel.BookLine{{Name: "Phone", SKU: "PH-1", Quantity: 2, UnitPriceCents: 49900}},
		CodCents: 99800, SubTotalCents: 99800,
	}, testCreds())
	require.NoError(t, err)
	require.Equal(t, "AWB1", out.AWB)
	require.Equal(t, "Delhivery Surface", out.CourierName)
	require.Equal(t, "77", fmt.Sprintf("%v", createBody["order_id"]), "provider order id is our shipment id")
}

// ─── Book skips pickup when unset ────────────────────────────────────────────
func TestBookShipment_NoPickupWhenUnset(t *testing.T) {
	var pickupCalls int64
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"POST /v1/external/orders/create/adhoc": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"order_id": 9001, "shipment_id": 8001}`)
		},
		"POST /v1/external/courier/assign/awb": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"response": {"data": {"awb_code": "AWB2"}}}`)
		},
		"POST /v1/external/courier/generate/pickup": func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt64(&pickupCalls, 1)
			writeJSON(w, 200, `{}`)
		},
	})

	_, err := a.BookShipment(ctx(), fulfillmentmodel.BookShipmentInput{
		ShipmentID: 78, PickupLocationID: 8, WeightGrams: intPtr(650),
		Items: []fulfillmentmodel.ShipmentItemInput{{OrderItemID: 1, Quantity: 1}},
	}, testCreds())
	require.NoError(t, err)
	require.Zero(t, atomic.LoadInt64(&pickupCalls), "no pickup call without PickupAt")
}

// ─── Cancel ──────────────────────────────────────────────────────────────────
func TestCancel_OkAndNotFound(t *testing.T) {
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"POST /v1/external/orders/cancel": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"success": true}`)
		},
	})
	require.NoError(t, a.Cancel(ctx(),
		fulfillmentmodel.CancelInput{ShipmentID: 1, AWB: "A", ProviderOrderID: "1"}, testCreds()))

	b, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"POST /v1/external/orders/cancel": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 404, `{"message":"not found"}`)
		},
	})
	require.Error(t, b.Cancel(ctx(),
		fulfillmentmodel.CancelInput{ShipmentID: 1, AWB: "A", ProviderOrderID: "1"}, testCreds()))
}

// ─── Label streams bytes ─────────────────────────────────────────────────────
// Scenario: generate returns a label_url; the adapter downloads it and
// returns raw PDF bytes (never stored, only streamed by callers).
func TestGetLabel_StreamsBytes(t *testing.T) {
	pdf := []byte("%PDF-1.4 fake-label")
	var labelURL string
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"POST /v1/external/courier/generate/label": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, `{"label_url": "`+labelURL+`"}`)
		},
		"GET /labels/AWB1.pdf": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(200)
			_, _ = w.Write(pdf)
		},
	})
	labelURL = "http://" + strings.TrimPrefix(a.BaseURL, "http://") + "/labels/AWB1.pdf"

	got, err := a.GetLabel(ctx(),
		fulfillmentmodel.LabelInput{ShipmentID: 1, AWB: "AWB1", ProviderShipmentID: "8001"}, testCreds())
	require.NoError(t, err)
	require.Equal(t, pdf, got)
}

// ─── Tracking maps statuses ──────────────────────────────────────────────────
func TestFetchTracking_MapsStatus(t *testing.T) {
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"GET /v1/external/courier/track/awb/AWB1": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"tracking_data": {"shipment_status": "IN TRANSIT", "courier_name": "Delhivery"}}`)
		},
	})
	event, err := a.FetchTracking(ctx(), "AWB1", testCreds())
	require.NoError(t, err)
	require.Equal(t, courier.ShipmentActionInTransit, event.Action)
	require.Equal(t, "AWB1", event.AWB)
}

// ─── Unknown webhook status → ignore ─────────────────────────────────────────
func TestNormalizeWebhook_UnknownStatusIgnored(t *testing.T) {
	a, _ := wiremock(t, nil)
	body := `{"awb": "AWB9", "current_status": "WEIRD CUSTOM", "order_id": "77", "date": "2026-10-01 10:00:00"}`
	event, err := a.NormalizeWebhook([]byte(body),
		http.Header{"X-Api-Key": {"s3cret"}}, map[string]any{"api_email": "api@shop.com", "api_password": "pw", "webhook_secret": "s3cret"})
	require.NoError(t, err)
	require.Equal(t, courier.ShipmentActionIgnore, event.Action)
	require.Equal(t, "AWB9", event.AWB)
}

// ─── Webhook: bad signature + missing AWB ────────────────────────────────────
func TestNormalizeWebhook_BadSignature(t *testing.T) {
	a, _ := wiremock(t, nil)
	body := `{"awb": "AWB9", "current_status": "DELIVERED"}`
	_, err := a.NormalizeWebhook([]byte(body),
		http.Header{"X-Api-Key": {"wrong"}}, map[string]any{"api_email": "api@shop.com", "api_password": "pw", "webhook_secret": "s3cret"})
	require.Error(t, err)
}

func TestNormalizeWebhook_MissingAWB(t *testing.T) {
	a, _ := wiremock(t, nil)
	body := `{"current_status": "DELIVERED"}`
	_, err := a.NormalizeWebhook([]byte(body),
		http.Header{"X-Api-Key": {"s3cret"}}, map[string]any{"api_email": "api@shop.com", "api_password": "pw", "webhook_secret": "s3cret"})
	require.Error(t, err)
}

// ─── TestConnection maps 401 to credentials error ───────────────────────────
func TestTestConnection_Unauthorized(t *testing.T) {
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 401, `{"message":"bad keys"}`)
		},
	})
	err := a.TestConnection(ctx(), testCreds())
	require.Error(t, err)
	appErr, ok := commonError.AsAppError(err)
	require.True(t, ok, "typed AppError, not generic")
	require.Equal(t, "FULFILLMENT_CREDENTIALS_INVALID", appErr.Code)
}

// ─── GET retries 429s ────────────────────────────────────────────────────────
func TestGet_RetriesOn429(t *testing.T) {
	var trackCalls int64
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"GET /v1/external/courier/track/awb/AWB1": func(w http.ResponseWriter, _ *http.Request) {
			if atomic.AddInt64(&trackCalls, 1) < 3 {
				writeJSON(w, 429, `{"message":"slow down"}`)
				return
			}
			writeJSON(w, 200, `{"tracking_data": {"shipment_status": "DELIVERED"}}`)
		},
	})
	event, err := a.FetchTracking(ctx(), "AWB1", testCreds())
	require.NoError(t, err)
	require.Equal(t, courier.ShipmentActionDelivered, event.Action)
	require.Equal(t, int64(3), atomic.LoadInt64(&trackCalls))
}

// ─── POST never retried ──────────────────────────────────────────────────────
func TestPost_NeverRetried(t *testing.T) {
	var createCalls int64
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"POST /v1/external/orders/create/adhoc": func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt64(&createCalls, 1)
			writeJSON(w, 500, `{"message":"boom"}`)
		},
	})
	_, err := a.BookShipment(ctx(), fulfillmentmodel.BookShipmentInput{
		ShipmentID: 79, PickupLocationID: 8, WeightGrams: intPtr(650),
		Items: []fulfillmentmodel.ShipmentItemInput{{OrderItemID: 1, Quantity: 1}},
	}, testCreds())
	require.Error(t, err)
	require.Equal(t, int64(1), atomic.LoadInt64(&createCalls))
}

// ─── Status mapping edge cases ───────────────────────────────────────────────
// Scenario: substring traps (UNDELIVERED contains DELIVERED; PICKUP
// SCHEDULED contains PICKUP) map to the longer, more-specific shape.
func TestMapShipmentStatus_EdgeCases(t *testing.T) {
	cases := map[string]courier.ShipmentAction{
		"UNDELIVERED - CONSIGNEE NOT AVAILABLE": courier.ShipmentActionNDRPending,
		"DELIVERED":                             courier.ShipmentActionDelivered,
		"PICKUP SCHEDULED":                      courier.ShipmentActionPickupScheduled,
		"PICKED UP":                             courier.ShipmentActionPicked,
		"In Transit - Bag Added To Trip":        courier.ShipmentActionInTransit,
		"RTO INITIATED":                         courier.ShipmentActionRTOInTransit,
		"RTO DELIVERED":                         courier.ShipmentActionReturned,
		"SOMETHING ENTIRELY NEW":                courier.ShipmentActionIgnore,
		"":                                      courier.ShipmentActionIgnore,
	}
	for raw, want := range cases {
		require.Equal(t, want, mapShipmentStatus(raw), "status %q", raw)
	}
}

// ─── Bad JSON ────────────────────────────────────────────────────────────────
func TestBadJSON_Errors(t *testing.T) {
	a, _ := wiremock(t, map[string]http.HandlerFunc{
		"POST /v1/external/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, 200, `{"token":"tok"}`)
		},
		"GET /v1/external/courier/track/awb/AWB1": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{not json`))
		},
	})
	_, err := a.FetchTracking(ctx(), "AWB1", testCreds())
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "parse response"))
}
