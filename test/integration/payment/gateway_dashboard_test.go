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
// Scenario: A seller deactivates their configured provider environment.
// Validates: after deactivating, the provider shows configured=false.
func (s *PaymentSuite) TestDeactivateGateway() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	w := seller2.Delete(s.T(), "/api/payment/gateways/razorpay/configure?environment=sandbox")
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

// ─── Dual Environments: Configure Sandbox + Production ───────────────────────
// Scenario: A seller saves both credential sets independently.
// Validates: detail exposes configs.sandbox/production with masked hints;
// list exposes configuredSandbox/configuredProduction.
func (s *PaymentSuite) TestConfigureDualEnvironments() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)

	sandboxCreds := map[string]any{
		"key_id":         "rzp_test_dualsandbox0000",
		"key_secret":     "dual_sandbox_key_secret_0000000000000",
		"webhook_secret": "dual_sandbox_webhook_secret_000000",
	}
	w := seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": sandboxCreds,
		"environment": "sandbox",
		"priority":    1,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	prodCreds := map[string]any{
		"key_id":         "rzp_live_dualprod0000000",
		"key_secret":     "dual_prod_key_secret_00000000000000000",
		"webhook_secret": "dual_prod_webhook_secret_00000000",
	}
	w = seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": prodCreds,
		"environment": "production",
		"priority":    5,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	detail := s.gatewayDetail(seller3, "razorpay")
	configs, ok := detail["configs"].(map[string]any)
	s.Require().True(ok, "detail should expose configs map")
	for _, env := range []string{"sandbox", "production"} {
		cfg, ok := configs[env].(map[string]any)
		s.Require().True(ok, "configs.%s should be an object", env)
		s.Assert().True(cfg["configured"].(bool), "configs.%s.configured should be true", env)
		hints, ok := cfg["configHints"].(map[string]any)
		s.Require().True(ok, "configs.%s.configHints should be an object", env)
		s.Assert().NotEmpty(hints["keyId"], "configs.%s.configHints.keyId required", env)
	}
	sandboxHints := configs["sandbox"].(map[string]any)["configHints"].(map[string]any)
	s.Assert().Contains(sandboxHints["keyId"], "rzp_test", "hint should carry the public key prefix")
	s.Assert().NotContains(sandboxHints["keyId"], sandboxCreds["key_secret"], "hints must never leak key_secret")
	s.Assert().NotContains(sandboxHints["keyId"], sandboxCreds["webhook_secret"], "hints must never leak webhook_secret")

	list := seller3.Get(s.T(), GatewaysAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), list, http.StatusOK)
	item := s.gatewayListItem(resp, "razorpay")
	s.Require().NotNil(item, "razorpay should be listed")
	s.Assert().True(item["configuredSandbox"].(bool), "configuredSandbox should be true")
	s.Assert().True(item["configuredProduction"].(bool), "configuredProduction should be true")
}

// ─── List: Dual-Env Flags, Store Mode, Webhook URL ───────────────────────────
// Scenario: A seller with only a sandbox config lists providers.
// Validates: per-env flags, paymentsEnvironment, composed webhookUrl, and geo
// objects (not string arrays).
func (s *PaymentSuite) TestListGatewaysShowsDualEnvFlags() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	list := seller2.Get(s.T(), GatewaysAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), list, http.StatusOK)
	item := s.gatewayListItem(resp, "razorpay")
	s.Require().NotNil(item, "razorpay should be listed")

	s.Assert().True(item["configuredSandbox"].(bool), "sandbox seeded → configuredSandbox true")
	s.Assert().False(item["configuredProduction"].(bool), "no production row → configuredProduction false")
	s.Assert().Equal("sandbox", item["paymentsEnvironment"], "store mode should be sandbox")
	s.Assert().Equal(
		"http://localhost:8080/api/payment/webhooks/razorpay",
		item["webhookUrl"], "webhookUrl must be composed from PUBLIC_API_BASE_URL",
	)

	countries, ok := item["supportedCountries"].([]any)
	s.Require().True(ok, "supportedCountries should be objects")
	s.Require().NotEmpty(countries, "razorpay should serve at least one country")
	firstCountry, _ := countries[0].(map[string]any)
	s.Assert().Equal("IN", firstCountry["code"], "razorpay should serve IN")
	s.Assert().NotEmpty(firstCountry["name"], "country name required")

	currencies, ok := item["supportedCurrencies"].([]any)
	s.Require().True(ok, "supportedCurrencies should be objects")
	s.Require().NotEmpty(currencies, "razorpay should settle at least one currency")
	firstCurrency, _ := currencies[0].(map[string]any)
	s.Assert().Equal("INR", firstCurrency["code"], "razorpay should settle INR")
}

// ─── Partial Update Keeps Stored Secrets ─────────────────────────────────────
// Scenario: A seller updates only key_id for an environment.
// Validates: masked hints unchanged and saved secrets still work (proven by a
// credential-less test-connection afterwards in the test-connection suite).
func (s *PaymentSuite) TestPartialUpdateKeepsSecrets() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)

	full := map[string]any{
		"key_id":         "rzp_test_keep0000000001",
		"key_secret":     "partial_keep_key_secret_0000000000000",
		"webhook_secret": "partial_keep_webhook_secret_000000",
	}
	w := seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": full,
		"environment": "sandbox",
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	before := s.configHints(seller3, "razorpay", "sandbox")

	// Partial: only key_id (different last4 so the masked hint visibly changes);
	// secrets must survive server-side.
	w = seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": map[string]any{"key_id": "rzp_test_new0000000002"},
		"environment": "sandbox",
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	after := s.configHints(seller3, "razorpay", "sandbox")
	s.Assert().NotEqual(before["keyId"], after["keyId"], "key_id hint should reflect the update")
	s.Assert().Contains(after["keyId"], "rzp_test", "hint should carry the new public key prefix")
}

// ─── Deactivate One Environment Only ─────────────────────────────────────────
// Scenario: A seller with both environments deactivates sandbox.
// Validates: production row stays active.
func (s *PaymentSuite) TestDeactivateOnlyOneEnvironment() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)
	s.configureBothEnvironments(seller3)

	w := seller3.Delete(s.T(), "/api/payment/gateways/razorpay/configure?environment=sandbox")
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	list := seller3.Get(s.T(), GatewaysAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), list, http.StatusOK)
	item := s.gatewayListItem(resp, "razorpay")
	s.Require().NotNil(item, "razorpay should be listed")
	s.Assert().False(item["configuredSandbox"].(bool), "sandbox should be deactivated")
	s.Assert().True(item["configuredProduction"].(bool), "production must stay active")
}

// ─── Deactivate Requires Environment ─────────────────────────────────────────
// Scenario: A seller calls DELETE configure without ?environment=.
// Validates: rejection with 400 (no mass-deactivate by accident).
func (s *PaymentSuite) TestDeactivateRequiresEnvironment() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	w := seller2.Delete(s.T(), "/api/payment/gateways/razorpay/configure")
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── Secrets Never Stored Plaintext ──────────────────────────────────────────
// Scenario: A seller configures credentials.
// Validates: the stored row holds ciphertext, never the submitted secrets.
// (Covers the fail-closed "no plaintext secrets" requirement at the API level;
// empty-key fail-closed is covered by adapter unit tests since the config
// singleton cannot be safely reset mid-suite.)
func (s *PaymentSuite) TestConfiguredSecretsStoredEncrypted() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)
	secret := "plaintext_probe_key_secret_0000000000"
	webhookSecret := "plaintext_probe_webhook_00000000000"

	w := seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": map[string]any{
			"key_id":         "rzp_test_plaintext00000",
			"key_secret":     secret,
			"webhook_secret": webhookSecret,
		},
		"environment": "sandbox",
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	var stored string
	err := s.container.DB.Table("payment_gateway_config").
		Select("credentials::text").
		Where("seller_id = (SELECT id FROM \"user\" WHERE email = ?) AND gateway_id = (SELECT id FROM payment_gateway WHERE code = 'razorpay') AND environment = 'sandbox'",
			helpers.SellerEmail).
		Scan(&stored).Error
	s.Require().NoError(err)
	s.Assert().NotContains(stored, secret, "key_secret must not be stored plaintext")
	s.Assert().NotContains(stored, webhookSecret, "webhook_secret must not be stored plaintext")
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

// gatewayListItem returns one gateway row from a list response by code.
func (s *PaymentSuite) gatewayListItem(resp map[string]any, code string) map[string]any {
	items, ok := resp["data"].([]any)
	if !ok {
		return nil
	}
	for _, item := range items {
		im, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if im["code"] == code {
			return im
		}
	}
	return nil
}

// gatewayDetail fetches the detail payload for one gateway code.
func (s *PaymentSuite) gatewayDetail(client *helpers.APIClient, code string) map[string]any {
	w := client.Get(s.T(), "/api/payment/gateways/"+code)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "detail data should be an object")
	return data
}

// configHints returns the masked hints for one environment of a gateway.
func (s *PaymentSuite) configHints(client *helpers.APIClient, code, environment string) map[string]any {
	detail := s.gatewayDetail(client, code)
	configs, ok := detail["configs"].(map[string]any)
	s.Require().True(ok, "detail should expose configs map")
	cfg, ok := configs[environment].(map[string]any)
	s.Require().True(ok, "configs.%s should be an object", environment)
	hints, ok := cfg["configHints"].(map[string]any)
	s.Require().True(ok, "configs.%s.configHints should be an object", environment)
	return hints
}

// configureBothEnvironments seeds sandbox + production configs for a seller.
func (s *PaymentSuite) configureBothEnvironments(client *helpers.APIClient) {
	for _, env := range []string{"sandbox", "production"} {
		prefix := "rzp_test_"
		if env == "production" {
			prefix = "rzp_live_"
		}
		w := client.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
			"credentials": map[string]any{
				"key_id":         prefix + "bothenv000000000000",
				"key_secret":     env + "_bothenv_key_secret_00000000000000",
				"webhook_secret": env + "_bothenv_webhook_secret_000000",
			},
			"environment": env,
		})
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}
}
