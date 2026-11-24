package payment_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ─── Get Transaction: Seller Detail ──────────────────────────────────────────
// Scenario: A seller opens a completed payment.
// Validates: environment, remaining refundable amount, and full event history
// are present on the detail payload.
func (s *PaymentSuite) TestGetTransactionByIdAsSeller() {
	txID, _ := s.capturePaymentForOrder()

	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)
	w := seller2.Get(s.T(), fmt.Sprintf(TransactionsByIDAPIFormat, txID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")
	assert.Equal(s.T(), txID, data["transactionId"], "wrong transaction")
	assert.Equal(s.T(), "sandbox", data["environment"], "frozen environment required")
	assert.Equal(s.T(), float64(99900), data["refundableAmountCents"], "nothing refunded yet")

	events, ok := data["events"].([]any)
	s.Require().True(ok, "detail must include events")
	s.Assert().GreaterOrEqual(len(events), 3, "expected initiated/session/captured history")
	types := map[string]bool{}
	for _, e := range events {
		em, _ := e.(map[string]any)
		types[em["eventType"].(string)] = true
	}
	for _, want := range []string{"initiated", "gateway_session_created", "captured"} {
		assert.True(s.T(), types[want], "event %s should be present", want)
	}
}

// ─── Get Transaction: Customer Detail ────────────────────────────────────────
// Scenario: The owning customer opens their payment.
// Validates: same detail shape (no regression on the existing customer path).
func (s *PaymentSuite) TestGetTransactionByIdAsCustomer() {
	txID, _ := s.capturePaymentForOrder()

	w := s.customerClient.Get(s.T(), fmt.Sprintf(TransactionsByIDAPIFormat, txID))
	resp := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := resp["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")
	assert.Equal(s.T(), txID, data["transactionId"], "wrong transaction")
	assert.Equal(s.T(), "sandbox", data["environment"], "frozen environment required")
	_, hasEvents := data["events"]
	assert.True(s.T(), hasEvents, "customer detail must include events")
}

// ─── Get Transaction: Other Seller ───────────────────────────────────────────
// Scenario: A seller opens another seller's payment.
// Validates: 404 (no cross-tenant leak).
func (s *PaymentSuite) TestGetTransactionByIdOtherSeller404() {
	txID, _ := s.capturePaymentForOrder()

	seller3 := s.sellerClientFor(helpers.SellerEmail, helpers.SellerPassword)
	w := seller3.Get(s.T(), fmt.Sprintf(TransactionsByIDAPIFormat, txID))
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}
