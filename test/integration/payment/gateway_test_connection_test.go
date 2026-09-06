package payment_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"
)

// ─── Test Connection: Saved Credentials ──────────────────────────────────────
// Scenario: A seller tests the suite-seeded sandbox config without sending
// credentials.
// Validates: ok + environment echoed, using the saved row.
func (s *PaymentSuite) TestConnectionWithSavedCredentials() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	w := seller2.Post(s.T(), "/api/payment/gateways/razorpay/test", map[string]any{
		"environment": "sandbox",
	})
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")
	s.Assert().True(data["ok"].(bool), "saved credentials should test ok")
	s.Assert().Equal("sandbox", data["environment"], "environment should be echoed")
}

// ─── Test Connection: Invalid Credentials → 400 ──────────────────────────────
// Scenario: A seller tests credentials the provider rejects (HTTP 401).
// Validates: 400 (bad keys), not 500.
func (s *PaymentSuite) TestConnectionRejectsInvalidCredentials() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	w := seller2.Post(s.T(), "/api/payment/gateways/razorpay/test", map[string]any{
		"environment": "sandbox",
		"credentials": map[string]any{
			"key_id":         TestKeyID,
			"key_secret":     "wrong_secret_000000000000000000000000",
			"webhook_secret": TestWebhookSecret,
		},
	})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	s.Assert().Equal("GATEWAY_VALIDATION", resp["code"], "provider 401 must map to 400 validation")
}

// ─── Test Connection: Unsaved Credentials Are Not Persisted ──────────────────
// Scenario: A seller probes new credentials that the provider accepts.
// Validates: ok, but the saved row is untouched (detail hints unchanged).
func (s *PaymentSuite) TestConnectionDoesNotPersistUnsavedCredentials() {
	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)

	// Saved row first: suite test keys so probes authenticate.
	w := seller3.Put(s.T(), "/api/payment/gateways/razorpay/configure", map[string]any{
		"credentials": map[string]any{
			"key_id":         TestKeyID,
			"key_secret":     TestKeySecret,
			"webhook_secret": TestWebhookSecret,
			"account_id":     "acc_saved_probe",
		},
		"environment": "sandbox",
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	before := s.configHints(seller3, "razorpay", "sandbox")

	// Valid provider keys but different metadata: accepted, not saved.
	w = seller3.Post(s.T(), "/api/payment/gateways/razorpay/test", map[string]any{
		"environment": "sandbox",
		"credentials": map[string]any{
			"key_id":         TestKeyID,
			"key_secret":     TestKeySecret,
			"webhook_secret": "unsaved_probe_webhook_secret_000000",
			"account_id":     "acc_unsaved_probe",
		},
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	after := s.configHints(seller3, "razorpay", "sandbox")
	s.Assert().Equal(before["keyId"], after["keyId"], "test must not persist unsaved key_id")
	s.Assert().Equal(before["accountId"], after["accountId"], "test must not persist unsaved account_id")
}

// ─── Test Connection: Unknown Gateway → 404 ──────────────────────────────────
func (s *PaymentSuite) TestConnectionUnknownGateway() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	w := seller2.Post(s.T(), "/api/payment/gateways/stripe/test", map[string]any{
		"environment": "sandbox",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}
