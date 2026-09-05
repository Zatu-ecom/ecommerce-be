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

	rawBody := webhookPayload("payment.captured", paymentID, sessionID, 10000)
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

	rawBody := webhookPayload("payment.captured", "pay_no_corr", sessionID, 10000)
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

	rawBody := webhookPayload("payment.captured", paymentID, sessionID, 10000)
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
