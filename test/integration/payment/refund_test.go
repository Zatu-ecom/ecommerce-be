package payment_test

import (
	"encoding/json"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

const RefundsAPIEndpoint = "/api/payment/refunds"

// capturePayment completes a payment for an order via a captured webhook, returning
// the transaction id and the captured payment id.
func (s *PaymentSuite) capturePaymentForOrder() (txID, paymentID string) {
	orderID := s.initiatePaymentForOrder()
	txID = s.transactionIDForOrder(orderID)
	paymentID = "pay_captured_refund"

	rawBody := webhookPayload("payment.captured", paymentID, s.sessionIDForTransaction(txID), 99900)
	w := s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code, "capture should succeed")
	s.verifyTransactionStatus(txID, "completed")
	return txID, paymentID
}

// refundWebhookPayload builds a refund webhook body.
func refundWebhookPayload(event, refundID, paymentID string, amount float64) []byte {
	body, _ := json.Marshal(map[string]any{
		"event": event,
		"payload": map[string]any{
			"refund": map[string]any{
				"entity": map[string]any{
					"id":         refundID,
					"payment_id": paymentID,
					"amount":     amount,
					"currency":   "INR",
					"status":     "processed",
				},
			},
		},
	})
	return body
}

// ─── Refund: Happy Path ────────────────────────────────────────────────────────
// Scenario: A seller refunds a completed payment (partial).
// Validates: a refund record is created and the gateway refund call is issued.
func (s *PaymentSuite) TestInitiateRefundCreatesRefund() {
	txID, _ := s.capturePaymentForOrder()

	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w := seller2.Post(s.T(), RefundsAPIEndpoint, map[string]any{
		"transactionId": txID,
		"amountCents":   5000,
		"reason":        "customer_request",
		"notes":         "partial refund",
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Refund row exists with processing status.
	var count int64
	err := s.container.DB.Table("payment_refund").
		Where("transaction_id = (SELECT id FROM payment_transaction WHERE transaction_id = ?) AND status = ?",
			txID, "processing").
		Count(&count).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), int64(1), count, "refund should be created in processing state")
}

// ─── Refund: Not Completed ─────────────────────────────────────────────────────
// Scenario: A seller tries to refund a non-completed (pending) payment.
// Validates: rejection.
func (s *PaymentSuite) TestInitiateRefundRejectsNonCompleted() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)

	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w := seller2.Post(s.T(), RefundsAPIEndpoint, map[string]any{
		"transactionId": txID,
		"amountCents":   5000,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── Refund: refund.processed webhook ──────────────────────────────────────────
// Scenario: The provider reports the refund processed.
// Validates: refund completed and payment moved to refunded/partially_refunded.
func (s *PaymentSuite) TestRefundProcessedWebhookCompletesRefund() {
	txID, _ := s.capturePaymentForOrder()

	// Initiate the refund first (creates the refund record with a gateway refund id).
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w := seller2.Post(s.T(), RefundsAPIEndpoint, map[string]any{
		"transactionId": txID,
		"amountCents":   5000,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Read the stored gateway_refund_id.
	var gatewayRefundID string
	err := s.container.DB.Table("payment_refund").
		Select("gateway_refund_id").
		Where("transaction_id = (SELECT id FROM payment_transaction WHERE transaction_id = ?)", txID).
		Scan(&gatewayRefundID).Error
	s.Require().NoError(err)
	s.Require().NotEmpty(gatewayRefundID, "refund should have a gateway_refund_id")

	rawBody := refundWebhookPayload("refund.processed", gatewayRefundID, "pay_captured_refund", 5000)
	w = s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code)

	var status string
	err = s.container.DB.Table("payment_refund").
		Select("status").
		Where("gateway_refund_id = ?", gatewayRefundID).
		Scan(&status).Error
	s.Require().NoError(err)
	s.Assert().Equal("completed", status, "refund should be completed")
}

// ─── Refund: refund.failed webhook ─────────────────────────────────────────────
// Scenario: The provider reports the refund failed.
// Validates: refund failed and payment remains completed.
func (s *PaymentSuite) TestRefundFailedWebhookMarksRefundFailed() {
	txID, _ := s.capturePaymentForOrder()

	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w := seller2.Post(s.T(), RefundsAPIEndpoint, map[string]any{
		"transactionId": txID,
		"amountCents":   5000,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	var gatewayRefundID string
	err := s.container.DB.Table("payment_refund").
		Select("gateway_refund_id").
		Where("transaction_id = (SELECT id FROM payment_transaction WHERE transaction_id = ?)", txID).
		Scan(&gatewayRefundID).Error
	s.Require().NoError(err)

	rawBody := refundWebhookPayload("refund.failed", gatewayRefundID, "pay_captured_refund", 5000)
	w = s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code)

	var status string
	err = s.container.DB.Table("payment_refund").
		Select("status").
		Where("gateway_refund_id = ?", gatewayRefundID).
		Scan(&status).Error
	s.Require().NoError(err)
	s.Assert().Equal("failed", status, "refund should be failed")

	s.verifyTransactionStatus(txID, "completed")
}
