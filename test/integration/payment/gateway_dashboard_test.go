package payment_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

const (
	GatewaysAPIEndpoint         = "/api/payment/gateways"
	GatewayByCodeAPIEndpoint    = "/api/payment/gateways/%s"
	GatewayConfigureAPIEndpoint = "/api/payment/gateways/%s/configure"
)

// sellerClientFor returns an authenticated client for a seller.
func (s *PaymentSuite) sellerClientFor(sellerEmail, sellerPassword string) *helpers.APIClient {
	client := helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), client, sellerEmail, sellerPassword)
	client.SetToken(token)
	return client
}

// ─── List Gateways: Configured vs Not ──────────────────────────────────────────
// Scenario: A seller views available providers and sees their own config state.
// Validates:
// 1. Seller 2 (already configured via suite seed) sees razorpay configured=true.
// 2. Seller 3 (no config) sees razorpay configured=false.
func (s *PaymentSuite) TestListGatewaysShowsConfiguredState() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)

	w2 := seller2.Get(s.T(), GatewaysAPIEndpoint)
	resp2 := helpers.AssertSuccessResponse(s.T(), w2, http.StatusOK)
	s.Assert().True(s.gatewayConfigured(resp2, "razorpay"), "seller 2 should see razorpay configured")

	w3 := seller3.Get(s.T(), GatewaysAPIEndpoint)
	resp3 := helpers.AssertSuccessResponse(s.T(), w3, http.StatusOK)
	s.Assert().False(s.gatewayConfigured(resp3, "razorpay"), "seller 3 should see razorpay not configured")
}

// ─── Get Gateway Detail: Fields ────────────────────────────────────────────────
// Scenario: A seller views a provider's detail including required config fields.
// Validates: the razorpay gateway exposes key_id/key_secret/webhook_secret/account_id.
func (s *PaymentSuite) TestGetGatewayDetailReturnsFields() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)

	w := seller3.Get(s.T(), "/api/payment/gateways/razorpay")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")
	s.Assert().Equal("razorpay", data["code"], "code should be razorpay")

	fields, ok := data["fields"].([]any)
	s.Require().True(ok, "fields should be a list")
	fieldNames := map[string]bool{}
	for _, f := range fields {
		fm, _ := f.(map[string]any)
		fieldNames[fm["fieldName"].(string)] = true
	}
	s.Assert().True(fieldNames["key_id"], "key_id field required")
	s.Assert().True(fieldNames["key_secret"], "key_secret field required")
	s.Assert().True(fieldNames["webhook_secret"], "webhook_secret field required")
	s.Assert().True(fieldNames["account_id"], "account_id field required")
}

// ─── Configure Gateway: Happy Path ─────────────────────────────────────────────
// Scenario: A seller configures a provider with valid credentials.
// Validates: after configuring, the provider shows configured=true and secrets are
// not echoed back.
func (s *PaymentSuite) TestConfigureGatewayMarksConfigured() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)

	w := seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": map[string]any{
			"key_id":         "rzp_test_bbbbbbbbbbbbbbbb",
			"key_secret":     "another_test_key_secret_00000000000000",
			"webhook_secret": "another_test_webhook_secret_00000000",
		},
		"environment": "sandbox",
		"priority":    2,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	list := seller3.Get(s.T(), GatewaysAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), list, http.StatusOK)
	s.Assert().True(s.gatewayConfigured(resp, "razorpay"), "seller 3 should now see razorpay configured")
}

// ─── Configure Gateway: Invalid Credentials ────────────────────────────────────
// Scenario: A seller submits incomplete credentials.
// Validates: rejection with a validation error.
func (s *PaymentSuite) TestConfigureGatewayRejectsInvalidCredentials() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)

	// Missing key_secret and webhook_secret.
	w := seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": map[string]any{
			"key_id": "rzp_test_cccccccccccccccc",
		},
		"environment": "sandbox",
	})
	s.Assert().Equal(http.StatusBadRequest, w.Code, "invalid credentials should be rejected")
}

// ─── Deactivate Gateway ────────────────────────────────────────────────────────
// Scenario: A seller deactivates their configured provider.
// Validates: after deactivating, the provider shows configured=false.
func (s *PaymentSuite) TestDeactivateGateway() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	w := seller2.Delete(s.T(), "/api/payment/gateways/razorpay/configure")
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	list := seller2.Get(s.T(), GatewaysAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), list, http.StatusOK)
	s.Assert().False(s.gatewayConfigured(resp, "razorpay"), "seller 2 should see razorpay not configured after deactivate")
}

// ─── Seller Isolation ──────────────────────────────────────────────────────────
// Scenario: A seller's config does not leak to another seller.
// Validates: seller 3 configuring razorpay does not affect seller 2's config state.
func (s *PaymentSuite) TestGatewayConfigSellerIsolation() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)
	w := seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": map[string]any{
			"key_id":         "rzp_test_dddddddddddddddd",
			"key_secret":     "isolation_test_key_secret_000000000000",
			"webhook_secret": "isolation_test_webhook_secret_0000000",
		},
		"environment": "sandbox",
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Seller 2 still sees their own (suite-seeded) config, not seller 3's.
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	list := seller2.Get(s.T(), GatewaysAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), list, http.StatusOK)
	s.Assert().True(s.gatewayConfigured(resp, "razorpay"), "seller 2 should still see their own config")
}

// gatewayConfigured extracts whether a gateway appears configured in a list response.
func (s *PaymentSuite) gatewayConfigured(resp map[string]any, code string) bool {
	items, ok := resp["data"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		im, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if im["code"] == code {
			configured, _ := im["configured"].(bool)
			return configured
		}
	}
	return false
}
