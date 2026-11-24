package fulfillment_test

import (
	"encoding/json"
	"io"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// stubShiprocketAuth answers POST /v1/external/auth/login: any password
// except the sentinel bad one yields a fake JWT; the sentinel yields 401.
// Registered per test that needs the courier reachable.
func (s *FulfillmentSuite) stubShiprocketAuth() {
	s.fakeShiprocket.stub(http.MethodPost, "/v1/external/auth/login",
		func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			if req["password"] == "wrong-secret" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"invalid credentials"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"fake-jwt-for-tests"}`))
		})
}

// ─── US1: test-connection with bad credentials ───────────────────────────────
// Scenario: wrong password → 400 FULFILLMENT_CREDENTIALS_INVALID (not 500),
// nothing persisted.
func (s *FulfillmentSuite) TestCouriers_TestBadCredentials() {
	t := s.T()
	s.stubShiprocketAuth()
	fresh := s.freshSeller()

	w := fresh.Post(t, "/api/fulfillment/couriers/shiprocket/test", map[string]any{
		"environment": "production",
		"credentials": map[string]any{
			"api_email":    "seller@shop.com",
			"api_password": "wrong-secret",
		},
	})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	require.Equal(t, "FULFILLMENT_CREDENTIALS_INVALID", errResp["code"])

	// Nothing was saved by the probe.
	w = fresh.Get(t, "/api/fulfillment/couriers/shiprocket")
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Nil(t, resp["data"].(map[string]any)["config"])
}

// ─── US1: test-connection happy path ─────────────────────────────────────────
// Scenario: good credentials → 200 ok:true without saving.
func (s *FulfillmentSuite) TestCouriers_TestGoodCredentials() {
	t := s.T()
	s.stubShiprocketAuth()
	fresh := s.freshSeller()

	w := fresh.Post(t, "/api/fulfillment/couriers/shiprocket/test", map[string]any{
		"environment": "production",
		"credentials": map[string]any{
			"api_email":    "seller@shop.com",
			"api_password": "right-secret",
		},
	})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Equal(t, true, resp["data"].(map[string]any)["ok"])

	// Explicit-credential probes save nothing either.
	w = fresh.Get(t, "/api/fulfillment/couriers/shiprocket")
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Nil(t, resp["data"].(map[string]any)["config"])
}

// ─── US1: test-connection with no saved row ──────────────────────────────────
// Scenario: omitted credentials and nothing saved → 409 NOT_CONFIGURED
// (there is nothing to test against).
func (s *FulfillmentSuite) TestCouriers_TestWithoutSavedRow() {
	t := s.T()
	w := s.freshSeller().Post(t, "/api/fulfillment/couriers/shiprocket/test", map[string]any{
		"environment": "sandbox",
	})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusConflict)
	require.Equal(t, "FULFILLMENT_PROVIDER_NOT_CONFIGURED", errResp["code"])
}

// ─── US1: unknown courier code ───────────────────────────────────────────────
// Scenario: test + detail on an unknown code both 404 the same code,
// without distinguishing reasons.
func (s *FulfillmentSuite) TestCouriers_UnknownCode404() {
	t := s.T()
	w := s.sellerClient.Post(t, "/api/fulfillment/couriers/no-such-courier/test", map[string]any{
		"environment": "production",
	})
	errResp := helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	require.Equal(t, "FULFILLMENT_PROVIDER_NOT_SUPPORTED", errResp["code"])
}
