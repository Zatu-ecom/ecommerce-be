package payment_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/payment/entity"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ─── List Filter: Status Enum ────────────────────────────────────────────────
// Scenario: A seller holds payments in every terminal state.
// Validates: ?status= filters server-side with correct filtered totals.
func (s *PaymentSuite) TestListTransactionsFiltersByStatus() {
	pendingTx := s.initiatePaymentForOrder()
	pendingID := s.transactionIDForOrder(pendingTx)

	completedID, _ := s.capturePaymentForOrder()

	failedOrder := s.initiatePaymentForOrder()
	failedID := s.transactionIDForOrder(failedOrder)
	s.failPaymentViaWebhook(failedID)

	partialID, _ := s.capturePaymentForOrder()
	s.refundViaAPI(partialID, 5000)
	s.completeRefundViaWebhook(partialID, 5000)
	s.verifyTransactionStatus(partialID, "partially_refunded")

	refundedID, _ := s.capturePaymentForOrder()
	fullAmount := s.amountCentsForTransaction(refundedID)
	s.refundViaAPI(refundedID, fullAmount)
	s.completeRefundViaWebhook(refundedID, fullAmount)
	s.verifyTransactionStatus(refundedID, "refunded")

	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	cases := map[string]string{
		"pending":            pendingID,
		"completed":          completedID,
		"failed":             failedID,
		"partially_refunded": partialID,
		"refunded":           refundedID,
	}
	for status, wantTx := range cases {
		w := seller2.Get(s.T(), TransactionsAPIEndpoint+"?status="+status)
		resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
		data, ok := resp["data"].(map[string]any)
		s.Require().True(ok, "data should be an object")
		assert.Equal(s.T(), float64(1), data["total"], "status=%s should match exactly one row", status)

		items, ok := data["items"].([]any)
		s.Require().True(ok, "items should be a list")
		s.Require().Len(items, 1, "status=%s should return one item", status)
		item, _ := items[0].(map[string]any)
		assert.Equal(s.T(), wantTx, item["transactionId"], "status=%s wrong row", status)
		assert.Equal(s.T(), status, item["status"], "status=%s wrong status", status)
		_, hasEvents := item["events"]
		assert.False(s.T(), hasEvents, "list payload must not include events")
	}
}

// ─── List Filter: Unknown Status ─────────────────────────────────────────────
// Scenario: A seller filters by a status outside the enum.
// Validates: rejection with 400.
func (s *PaymentSuite) TestListTransactionsRejectsUnknownStatus() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	w := seller2.Get(s.T(), TransactionsAPIEndpoint+"?status=bogus")
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── List: Page Size Cap ─────────────────────────────────────────────────────
// Scenario: A seller requests an unbounded page.
// Validates: at most 100 rows return; total reflects the full filtered count.
func (s *PaymentSuite) TestListTransactionsCapsPageSize() {
	s.seedTransactions(105)

	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w := seller2.Get(s.T(), TransactionsAPIEndpoint+"?pageSize=500")
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")
	items, ok := data["items"].([]any)
	s.Require().True(ok, "items should be a list")
	assert.LessOrEqual(s.T(), len(items), 100, "pageSize must be capped at 100")
	assert.Equal(s.T(), float64(105), data["total"], "total must reflect the full count")
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func (s *PaymentSuite) failPaymentViaWebhook(txID string) {
	sessionID := s.sessionIDForTransaction(txID)
	rawBody := webhookPayload("payment.failed", "pay_fail_"+txID, sessionID, 99900)
	w := s.postWebhook(rawBody, signWebhook(rawBody))
	s.Require().Equal(http.StatusOK, w.Code)
	s.verifyTransactionStatus(txID, "failed")
}

func (s *PaymentSuite) refundViaAPI(txID string, amount int64) {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w := seller2.Post(s.T(), RefundsAPIEndpoint, map[string]any{
		"transactionId": txID,
		"amountCents":   amount,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
}

// seedTransactions inserts bare pending rows for pagination tests (fast path:
// no orders or provider calls needed since listing never joins them).
func (s *PaymentSuite) seedTransactions(n int) {
	var gateway entity.PaymentGateway
	s.Require().NoError(s.container.DB.Where("code = ?", "razorpay").First(&gateway).Error)
	for i := 0; i < n; i++ {
		txn := entity.PaymentTransaction{
			TransactionID: fmt.Sprintf("TXN_BULK_%04d", i),
			UserID:        helpers.CustomerUserID,
			SellerID:      helpers.Seller2UserID,
			GatewayID:     &gateway.ID,
			ReferenceType: entity.ReferenceTypeOrder,
			ReferenceID:   uint(900000 + i),
			Currency:      "INR",
			AmountCents:   1000,
			Status:        entity.TransactionStatusPending,
			Environment:   entity.EnvironmentSandbox,
		}
		s.Require().NoError(s.container.DB.Create(&txn).Error)
	}
}
