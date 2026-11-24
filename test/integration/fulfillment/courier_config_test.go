package fulfillment_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/require"
)

// ─── US1: courier catalog ─────────────────────────────────────────────────────
// Scenario: seller lists couriers. Shiprocket appears with its capability
// flags, webhook URL, and unconfigured state on a fresh database.
func (s *FulfillmentSuite) TestCouriers_ListShowsShiprocketUnconfigured() {
	t := s.T()
	w := s.sellerClient.Get(t, "/api/fulfillment/couriers")
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)

	couriers, ok := resp["data"].(map[string]any)["couriers"].([]any)
	require.True(t, ok, "data.couriers must be a list")
	require.NotEmpty(t, couriers)

	var shiprocket map[string]any
	for _, c := range couriers {
		if c.(map[string]any)["code"] == "shiprocket" {
			shiprocket = c.(map[string]any)
		}
	}
	require.NotNil(t, shiprocket, "shiprocket in catalog")
	require.Equal(t, "aggregator", shiprocket["kind"])
	require.Equal(t, true, shiprocket["supportsNdr"])
	require.Contains(t, shiprocket["webhookUrl"], "/api/fulfillment/webhooks/shiprocket")
}

// freshSeller returns a logged-in client for a seller no other test
// configures (Seller4), giving order-independent "clean state" assertions.
func (s *FulfillmentSuite) freshSeller() *helpers.APIClient {
	t := s.T()
	client := helpers.NewAPIClient(s.server)
	token := helpers.Login(t, client, helpers.Seller4Email, helpers.Seller4Password)
	client.SetToken(token)
	return client
}

// ─── US1: courier detail ─────────────────────────────────────────────────────
// Scenario: known code returns the credential form + null config when
// nothing is saved; unknown code 404s without leaking provider existence.
func (s *FulfillmentSuite) TestCouriers_DetailKnownAndUnknown() {
	t := s.T()
	w := s.sellerClient.Get(t, "/api/fulfillment/couriers/shiprocket")
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	data := resp["data"].(map[string]any)
	require.Equal(t, "shiprocket", data["code"])

	fields, ok := data["fields"].([]any)
	require.True(t, ok && len(fields) > 0, "credential fields present")
	names := map[string]bool{}
	for _, f := range fields {
		names[f.(map[string]any)["fieldName"].(string)] = true
	}
	require.True(t, names["api_email"] && names["api_password"])

	w = s.sellerClient.Get(t, "/api/fulfillment/couriers/no-such-courier")
	errResp := helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	require.Equal(t, "FULFILLMENT_PROVIDER_NOT_SUPPORTED", errResp["code"])
}

// ─── US1: fresh seller sees null config ──────────────────────────────────────
// Scenario: a seller that never configured anything sees config null.
// Uses the dedicated untouched seller so execution order cannot pollute it.
func (s *FulfillmentSuite) TestCouriers_FreshSellerSeesNullConfig() {
	t := s.T()
	w := s.freshSeller().Get(t, "/api/fulfillment/couriers/shiprocket")
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Nil(t, resp["data"].(map[string]any)["config"])
}

// ─── US1: configure happy path ───────────────────────────────────────────────
// Scenario: valid credentials + flags save; response shows masked hints
// (email masked, no password or secret anywhere).
func (s *FulfillmentSuite) TestCouriers_ConfigureShowsMaskedHints() {
	t := s.T()
	w := s.sellerClient.Put(t, "/api/fulfillment/couriers/shiprocket/configure", map[string]any{
		"environment": "production",
		"credentials": map[string]any{
			"api_email":      "seller@shop.com",
			"api_password":   "secret-1",
			"webhook_secret": "whsec-1",
		},
		"autoBook":       false,
		"ratePreference": "cheapest",
		"isActive":       true,
	})
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	config := resp["data"].(map[string]any)
	require.Equal(t, true, config["configured"])
	require.Equal(t, false, config["autoBook"])
	require.Equal(t, "production", config["environment"])

	hints, ok := config["configHints"].(map[string]any)
	require.True(t, ok, "masked hints present")
	require.Contains(t, hints["api_email"], "s***@shop.com")
	require.NotContains(t, hints, "api_password")
	require.NotContains(t, hints, "webhook_secret")

	// Detail now shows the saved config (still masked).
	w = s.sellerClient.Get(t, "/api/fulfillment/couriers/shiprocket")
	resp = helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.NotNil(t, resp["data"].(map[string]any)["config"])
}

// ─── US1: configure validation ───────────────────────────────────────────────
// Scenario: missing environment and bad rate preference are 400s.
func (s *FulfillmentSuite) TestCouriers_ConfigureValidation() {
	t := s.T()
	w := s.sellerClient.Put(t, "/api/fulfillment/couriers/shiprocket/configure", map[string]any{
		"credentials": map[string]any{"api_email": "seller@shop.com"},
	})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)

	w = s.sellerClient.Put(t, "/api/fulfillment/couriers/shiprocket/configure", map[string]any{
		"environment":    "production",
		"credentials":    map[string]any{"api_email": "seller@shop.com", "api_password": "x"},
		"ratePreference": "teleport",
	})
	helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
}

// ─── US1: auth matrix ────────────────────────────────────────────────────────
// Scenario: no token → 401; missing correlation id → 400.
func (s *FulfillmentSuite) TestCouriers_AuthMatrix() {
	t := s.T()
	anon := helpers.NewAPIClient(s.server)
	w := anon.Get(t, "/api/fulfillment/couriers")
	require.Equal(t, http.StatusUnauthorized, w.Code)

	noCorrelation := helpers.NewAPIClient(s.server)
	noCorrelation.SetToken(s.sellerToken())
	noCorrelation.SetHeader("X-Correlation-ID", "")
	w = noCorrelation.Get(t, "/api/fulfillment/couriers")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── US1: seller isolation ───────────────────────────────────────────────────
// Scenario: seller A's saved config is invisible to seller B. B is Seller4
// (bob), whom no configuring test touches — booking fixtures use john.
func (s *FulfillmentSuite) TestCouriers_SellerIsolation() {
	t := s.T()
	s.sellerClient.Put(t, "/api/fulfillment/couriers/shiprocket/configure", map[string]any{
		"environment": "production",
		"credentials": map[string]any{"api_email": "a@shop.com", "api_password": "secret-a"},
	})

	sellerB := helpers.NewAPIClient(s.server)
	tokenB := helpers.Login(t, sellerB, helpers.Seller4Email, helpers.Seller4Password)
	sellerB.SetToken(tokenB)

	w := sellerB.Get(t, "/api/fulfillment/couriers/shiprocket")
	resp := helpers.AssertSuccessResponse(t, w, http.StatusOK)
	require.Nil(t, resp["data"].(map[string]any)["config"], "seller B sees no config")
}

// sellerToken logs in as the primary seller for clients built mid-test.
func (s *FulfillmentSuite) sellerToken() string {
	t := s.T()
	probe := helpers.NewAPIClient(s.server)
	return helpers.Login(t, probe, helpers.SellerEmail, helpers.SellerPassword)
}
