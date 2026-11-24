package payment_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

const WebhookAPIEndpoint = "/api/payment/webhooks/razorpay"

// webhookPayload builds a raw Razorpay webhook body for a payment event.
// orderID is the Razorpay order id (our gateway_session_id) used to locate the txn.
func webhookPayload(event, paymentID, orderID string, amount float64) []byte {
	body, _ := json.Marshal(map[string]any{
		"event": event,
		"payload": map[string]any{
			"payment": map[string]any{
				"entity": map[string]any{
					"id":       paymentID,
					"order_id": orderID,
					"amount":   amount,
					"currency": "INR",
					"fee":      0,
					"status":   "captured",
				},
			},
		},
	})
	return body
}

func (s *PaymentSuite) postWebhook(rawBody []byte, signature string) *httptest.ResponseRecorder {
	// Use a fresh client so the header is per-request.
	client := helpers.NewAPIClient(s.server)
	if signature != "" {
		client.SetHeader("X-Razorpay-Signature", signature)
	}
	return client.PostRaw(s.T(), WebhookAPIEndpoint, rawBody)
}

func (s *PaymentSuite) postWebhookWithoutCorrelationID(rawBody []byte, signature string) *httptest.ResponseRecorder {
	client := helpers.NewAPIClient(s.server)
	client.SetHeader("X-Correlation-ID", "")
	if signature != "" {
		client.SetHeader("X-Razorpay-Signature", signature)
	}
	return client.PostRaw(s.T(), WebhookAPIEndpoint, rawBody)
}

// ─── Webhook: payment.captured ─────────────────────────────────────────────────
// Scenario: A captured payment webhook arrives for an initiated payment.
// Validates: transaction completed, order confirmed, event recorded.
func (s *PaymentSuite) TestWebhookCapturedCompletesPaymentAndConfirmsOrder() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)
	paymentID := "pay_captured_1"

	// The captured amount must equal the transaction amount: the apply path
	// refuses completion on amount/currency mismatch (wrong-order capture).
	rawBody := webhookPayload("payment.captured", paymentID, sessionID, float64(s.amountCentsForTransaction(txID)))
	w := s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code, "webhook should be accepted")

	// Transaction completed.
	s.verifyTransactionStatus(txID, "completed")
	// Order confirmed.
	s.verifyOrderStatus(orderID, "confirmed")
	// A captured event was recorded.
	s.verifyEventExists(txID, "captured")
}

// ─── Webhook: payment.authorized (observation only) ────────────────────────────
// Scenario: An authorized payment webhook arrives (pre-capture).
// Validates: an authorized event is recorded, but the order is NOT confirmed.
func (s *PaymentSuite) TestWebhookAuthorizedRecordsObservationOnly() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)

	rawBody := webhookPayload("payment.authorized", "pay_authorized_1", sessionID, 10000)
	w := s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code)

	// Transaction still pending.
	s.verifyTransactionStatus(txID, "pending")
	// Order still pending.
	s.verifyOrderStatus(orderID, "pending")
	// Authorized observation event recorded.
	s.verifyEventExists(txID, "authorized")
}

// ─── Webhook: payment.failed ───────────────────────────────────────────────────
// Scenario: A failed payment webhook arrives.
// Validates: transaction failed, order failed.
func (s *PaymentSuite) TestWebhookFailedFailsPaymentAndOrder() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)

	rawBody := webhookPayload("payment.failed", "pay_failed_1", sessionID, 10000)
	w := s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code)

	s.verifyTransactionStatus(txID, "failed")
	s.verifyOrderStatus(orderID, "failed")
	s.verifyEventExists(txID, "failed")
}

// ─── Webhook: Invalid Signature ────────────────────────────────────────────────
// Scenario: A webhook arrives with a missing/invalid signature.
// Validates: 401 and no state change.
func (s *PaymentSuite) TestWebhookRejectsInvalidSignature() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)

	rawBody := webhookPayload("payment.captured", "pay_invalid_sig", sessionID, 10000)
	w := s.postWebhook(rawBody, "not-a-valid-signature")
	s.Require().Equal(http.StatusUnauthorized, w.Code)

	s.verifyTransactionStatus(txID, "pending")
	s.verifyOrderStatus(orderID, "pending")
}

// ─── Webhook: No X-Correlation-ID ──────────────────────────────────────────────
// Scenario: Razorpay delivers a webhook without X-Correlation-ID.
// Validates: the skip rule auto-generates a correlation ID and the webhook is processed.
func (s *PaymentSuite) TestWebhookWorksWithoutCorrelationID() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)

	rawBody := webhookPayload("payment.captured", "pay_no_corr", sessionID, float64(s.amountCentsForTransaction(txID)))
	w := s.postWebhookWithoutCorrelationID(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code, "webhook should be accepted without X-Correlation-ID")
	s.Require().NotEmpty(w.Header().Get("X-Correlation-ID"), "server should generate a correlation ID")

	s.verifyTransactionStatus(txID, "completed")
	s.verifyOrderStatus(orderID, "confirmed")
	s.verifyEventExists(txID, "captured")
}

// ─── Webhook: Duplicate Idempotency ────────────────────────────────────────────
// Scenario: The same event is delivered twice.
// Validates: state changes only once; second delivery is ignored.
func (s *PaymentSuite) TestWebhookDuplicateIsIdempotent() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)
	paymentID := "pay_dup_1"

	rawBody := webhookPayload("payment.captured", paymentID, sessionID, float64(s.amountCentsForTransaction(txID)))
	sig := signWebhook(rawBody)

	first := s.postWebhook(rawBody, sig)
	s.Require().Equal(http.StatusOK, first.Code)
	s.verifyTransactionStatus(txID, "completed")

	// Second delivery of the same event id → ignored, still completed.
	second := s.postWebhook(rawBody, sig)
	s.Require().Equal(http.StatusOK, second.Code)
	s.verifyTransactionStatus(txID, "completed")

	// Only one captured event recorded.
	var count int64
	err := s.container.DB.Table("payment_transaction_event").
		Where("transaction_id = (SELECT id FROM payment_transaction WHERE transaction_id = ?) AND event_type = ?",
			txID, "captured").
		Count(&count).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), int64(1), count, "captured event should be recorded exactly once")
}

// ─── Webhook: Amount Mismatch Does Not Complete ───────────────────────────────
// Scenario: A captured webhook arrives whose amount differs from our
// transaction (wrong-order capture).
// Validates: transaction stays pending, order stays pending, webhook log is
// recorded as failed. HTTP is still 200 (provider stops retrying).
func (s *PaymentSuite) TestWebhookAmountMismatchDoesNotComplete() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)

	wrongAmount := float64(s.amountCentsForTransaction(txID)) - 1
	rawBody := webhookPayload("payment.captured", "pay_mismatch_1", sessionID, wrongAmount)
	w := s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code)

	s.verifyTransactionStatus(txID, "pending")
	s.verifyOrderStatus(orderID, "pending")

	// No captured event recorded.
	var count int64
	err := s.container.DB.Table("payment_transaction_event").
		Where("transaction_id = (SELECT id FROM payment_transaction WHERE transaction_id = ?) AND event_type = ?",
			txID, "captured").
		Count(&count).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), int64(0), count, "mismatched capture must not record a captured event")

	// Webhook log recorded as failed.
	var logStatus string
	err = s.container.DB.Table("payment_webhook_log").
		Select("status").
		Where("transaction_id = (SELECT id FROM payment_transaction WHERE transaction_id = ?)", txID).
		Order("id DESC").
		Limit(1).
		Scan(&logStatus).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), "failed", logStatus, "mismatched capture must fail the webhook log")
}

// ─── Webhook: Wrong-Seller Signature ─────────────────────────────────────────
// Scenario: A webhook for seller A's payment arrives signed with seller B's
// webhook secret (well-formed HMAC under the wrong key).
// Validates: 401, no state change, and nothing persisted (verification
// happens before any write).
func (s *PaymentSuite) TestWebhookRejectsWrongSellerSignature() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)

	s.seedSecondSellerGatewayConfig()

	rawBody := webhookPayload("payment.captured", "pay_wrong_seller", sessionID, float64(s.amountCentsForTransaction(txID)))
	w := s.postWebhook(rawBody, signWebhookWithSecret(rawBody, SecondSellerWebhookSecret))
	s.Require().Equal(http.StatusUnauthorized, w.Code)

	s.verifyTransactionStatus(txID, "pending")
	s.verifyOrderStatus(orderID, "pending")
	s.verifyWebhookLogCount(txID, 0)
}

// ─── Webhook: Two-Seller Isolation ───────────────────────────────────────────
// Scenario: Two sellers hold different webhook secrets for the same provider.
// Validates: B's signature cannot complete A's payment, while A's own
// signature completes it (tenant-safe verification).
func (s *PaymentSuite) TestWebhookTwoSellerIsolation() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)

	s.seedSecondSellerGatewayConfig()

	rawBody := webhookPayload("payment.captured", "pay_isolation_1", sessionID, float64(s.amountCentsForTransaction(txID)))

	// B's signature on A's payment → 401, still pending.
	w := s.postWebhook(rawBody, signWebhookWithSecret(rawBody, SecondSellerWebhookSecret))
	s.Require().Equal(http.StatusUnauthorized, w.Code)
	s.verifyTransactionStatus(txID, "pending")

	// A's own signature on the same body → 200, completed + confirmed.
	w = s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code)
	s.verifyTransactionStatus(txID, "completed")
	s.verifyOrderStatus(orderID, "confirmed")
}

// ─── Webhook: Unknown Gateway Code ───────────────────────────────────────────
// Scenario: A webhook arrives for a provider code that does not exist.
// Validates: 404.
func (s *PaymentSuite) TestWebhookUnknownCodeReturns404() {
	rawBody := webhookPayload("payment.captured", "pay_unknown_code", "order_unknown_code", 10000)
	client := helpers.NewAPIClient(s.server)
	client.SetHeader("X-Razorpay-Signature", signWebhook(rawBody))
	w := client.PostRaw(s.T(), "/api/payment/webhooks/stripe", rawBody)
	s.Require().Equal(http.StatusNotFound, w.Code)
}

// ─── Webhook: Unmatched Delivery Persists Nothing ────────────────────────────
// Scenario: A validly-signed webhook arrives that matches no transaction
// (initiate still in flight or unknown ids).
// Validates: 200 with zero webhook-log rows written (no unverified persist).
func (s *PaymentSuite) TestWebhookWithoutTransactionPersistsNothing() {
	before := s.webhookLogCount()

	rawBody := webhookPayload("payment.captured", "pay_ghost_1", "order_ghost_1", 99900)
	w := s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code, "unmatched delivery must be a 200 no-op")

	s.Assert().Equal(before, s.webhookLogCount(), "unmatched delivery must persist no rows")
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func (s *PaymentSuite) initiatePaymentForOrder() uint {
	orderID := s.createPendingOrder(helpers.CustomerUserID)
	w := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	s.Require().Equal(http.StatusOK, w.Code, "initiate should succeed: %s", w.Body.String())
	return orderID
}

func (s *PaymentSuite) transactionIDForOrder(orderID uint) string {
	var txID string
	err := s.container.DB.Table(`"order"`).
		Select("transaction_id").
		Where("id = ?", orderID).
		Scan(&txID).Error
	s.Require().NoError(err)
	s.Require().NotEmpty(txID, "order should have a transaction_id after initiate")
	return txID
}

func (s *PaymentSuite) sessionIDForTransaction(transactionID string) string {
	var sessionID string
	err := s.container.DB.Table("payment_transaction").
		Select("gateway_session_id").
		Where("transaction_id = ?", transactionID).
		Scan(&sessionID).Error
	s.Require().NoError(err)
	s.Require().NotEmpty(sessionID, "transaction should have a gateway_session_id after initiate")
	return sessionID
}

func (s *PaymentSuite) amountCentsForTransaction(transactionID string) int64 {
	var amountCents int64
	err := s.container.DB.Table("payment_transaction").
		Select("amount_cents").
		Where("transaction_id = ?", transactionID).
		Scan(&amountCents).Error
	s.Require().NoError(err)
	s.Require().Positive(amountCents, "transaction should have a positive amount")
	return amountCents
}

func (s *PaymentSuite) verifyTransactionStatus(transactionID, status string) {
	var stored string
	err := s.container.DB.Table("payment_transaction").
		Select("status").
		Where("transaction_id = ?", transactionID).
		Scan(&stored).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), status, stored, "transaction status mismatch")
}

func (s *PaymentSuite) verifyOrderStatus(orderID uint, status string) {
	var stored string
	err := s.container.DB.Table(`"order"`).
		Select("status").
		Where("id = ?", orderID).
		Scan(&stored).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), status, stored, "order status mismatch")
}

func (s *PaymentSuite) verifyEventExists(transactionID, eventType string) {
	var count int64
	err := s.container.DB.Table("payment_transaction_event").
		Where("transaction_id = (SELECT id FROM payment_transaction WHERE transaction_id = ?) AND event_type = ?",
			transactionID, eventType).
		Count(&count).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), int64(1), count, "event %s should be recorded", eventType)
}

func (s *PaymentSuite) webhookLogCount() int64 {
	var count int64
	err := s.container.DB.Table("payment_webhook_log").Count(&count).Error
	s.Require().NoError(err)
	return count
}

func (s *PaymentSuite) verifyWebhookLogCount(transactionID string, want int64) {
	var count int64
	err := s.container.DB.Table("payment_webhook_log").
		Where("transaction_id = (SELECT id FROM payment_transaction WHERE transaction_id = ?)", transactionID).
		Count(&count).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), want, count, "webhook log count mismatch")
}
