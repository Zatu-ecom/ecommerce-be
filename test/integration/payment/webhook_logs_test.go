package payment_test

import (
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

const WebhookLogsAPIEndpoint = "/api/payment/webhook-logs"

// ─── Webhook Logs: Owner Sees Captured Delivery ──────────────────────────────
// Scenario: A captured webhook was processed for the seller's payment.
// Validates: the seller lists it with event type, id, and status.
func (s *PaymentSuite) TestWebhookLogsForOwner() {
	txID, _ := s.capturePaymentForOrder()

	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w := seller2.Get(s.T(), WebhookLogsAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")
	assert.Equal(s.T(), float64(1), data["total"], "one processed delivery expected")

	items, ok := data["items"].([]any)
	s.Require().True(ok, "items should be a list")
	s.Require().Len(items, 1, "one log row expected")
	item, _ := items[0].(map[string]any)
	assert.Equal(s.T(), "payment.captured", item["eventType"], "raw provider event preserved")
	assert.Equal(s.T(), "processed", item["status"], "delivery processed")
	assert.NotEmpty(s.T(), item["eventId"], "idempotency key required")
	assert.Equal(s.T(), txID, item["transactionId"], "log must link the transaction public id")
}

// ─── Webhook Logs: Other Seller Sees Nothing ─────────────────────────────────
// Scenario: A seller with no payments lists webhook logs.
// Validates: empty result (tenant-scoped).
func (s *PaymentSuite) TestWebhookLogsOtherSellerEmpty() {
	s.capturePaymentForOrder()

	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)
	w := seller3.Get(s.T(), WebhookLogsAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")
	assert.Equal(s.T(), float64(0), data["total"], "other seller must see no logs")
}

// ─── Webhook Logs: Unmatched Deliveries Not Listed ───────────────────────────
// Scenario: A validly-signed delivery matches no transaction (no row persisted).
// Validates: the owner list stays empty.
func (s *PaymentSuite) TestWebhookLogsExcludeUnmatched() {
	rawBody := webhookPayload("payment.captured", "pay_ghost_logs", "order_ghost_logs", 99900)
	w := s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code)

	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w = seller2.Get(s.T(), WebhookLogsAPIEndpoint)
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")
	assert.Equal(s.T(), float64(0), data["total"], "unmatched deliveries must not be listed")
}
