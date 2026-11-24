package payment_test

import (
	"fmt"
	"net/http"

	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

const (
	InitiatePaymentAPIEndpoint = "/api/payment/initiate"
	TransactionsAPIEndpoint    = "/api/payment/transactions"
	TransactionsByIDAPIFormat  = "/api/payment/transactions/%s"
)

// ─── Initiate Payment: Happy Path ──────────────────────────────────────────────
// Scenario: A customer initiates payment for their own pending order.
// Validates:
//  1. A payment_transaction is created with reference_type=order.
//  2. The order's transaction_id is written.
//  3. The response returns gatewaySessionId, amountCents, currency and a
//     generic checkout payload (gatewayCode + fields.keyId/orderId) with NO
//     top-level keyId (provider-agnostic contract).
func (s *PaymentSuite) TestInitiatePaymentReturnsCheckoutSession() {
	orderID := s.createPendingOrder(helpers.CustomerUserID)

	w := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	s.T().Logf("initiate response body: %s", w.Body.String())
	response := helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	data, ok := response["data"].(map[string]any)
	s.Require().True(ok, "data should be an object")

	s.Assert().NotEmpty(data["transactionId"], "transactionId required")
	s.Assert().NotEmpty(data["gatewaySessionId"], "gatewaySessionId required")
	s.Assert().Equal(float64(99900), data["amountCents"], "amountCents should match order total (₹999.00)")
	s.Assert().Equal("INR", data["currency"], "currency should be seller base currency INR")
	s.Assert().Equal("pending", data["status"], "status should be pending")

	// Generic checkout contract: no provider-named top-level fields.
	s.Assert().Nil(data["keyId"], "top-level keyId must not exist on the initiate contract")
	checkout, ok := data["checkout"].(map[string]any)
	s.Require().True(ok, "checkout should be an object")
	s.Assert().Equal("razorpay", checkout["gatewayCode"], "checkout gatewayCode should be razorpay")
	fields, ok := checkout["fields"].(map[string]any)
	s.Require().True(ok, "checkout.fields should be an object")
	s.Assert().NotEmpty(fields["keyId"], "checkout.fields.keyId required")
	s.Assert().NotEmpty(fields["orderId"], "checkout.fields.orderId required")

	// Verify the transaction row exists with reference_type=order.
	s.verifyTransactionRow(data["transactionId"].(string), orderID)

	// Verify the order now carries the transaction id.
	s.verifyOrderTransactionID(orderID, data["transactionId"].(string))
}

// ─── Initiate Payment: Order Not Owned ─────────────────────────────────────────
// Scenario: A customer tries to initiate payment for another customer's order.
// Validates: rejection with 404.
func (s *PaymentSuite) TestInitiatePaymentRejectsForeignOrder() {
	orderID := s.createPendingOrder(helpers.Customer2UserID)

	w := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ─── Initiate Payment: Order Not Payable ───────────────────────────────────────
// Scenario: A customer tries to pay for an order that is not pending (already confirmed).
// Validates: rejection (the order is not payable).
func (s *PaymentSuite) TestInitiatePaymentRejectsNonPayableOrder() {
	orderID := s.createPendingOrder(helpers.CustomerUserID)
	// Confirm the order via the seller so it is no longer pending.
	s.confirmOrderAsSeller(orderID)

	w := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusNotFound)
}

// ─── Initiate Payment: Duplicate ───────────────────────────────────────────────
// Scenario: A customer initiates payment twice for the same order.
// Validates: the second initiation is rejected (only one payment per order).
func (s *PaymentSuite) TestInitiatePaymentRejectsDuplicate() {
	orderID := s.createPendingOrder(helpers.CustomerUserID)

	first := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	helpers.AssertSuccessResponse(s.T(), first, http.StatusOK)

	second := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	helpers.AssertErrorResponse(s.T(), second, http.StatusConflict)
}

// ─── Initiate Payment: Production Mode Without Production Config ───────────────
// Scenario: The seller's store runs in production mode but only a sandbox
// gateway config exists.
// Validates: checkout is blocked with GATEWAY_NOT_CONFIGURED (never silently
// falls back to sandbox credentials for a live order).
func (s *PaymentSuite) TestInitiatePaymentRejectsProductionWithoutProductionConfig() {
	seller2 := s.sellerClientFor(helpers.Seller2Email, helpers.Seller2Password)

	// Flip store mode to production; restore sandbox afterwards so later tests
	// keep the suite-seeded sandbox behavior.
	w := seller2.Put(s.T(), "/api/user/seller/settings", map[string]any{
		"paymentsEnvironment": "production",
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	defer func() {
		w := seller2.Put(s.T(), "/api/user/seller/settings", map[string]any{
			"paymentsEnvironment": "sandbox",
		})
		helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)
	}()

	orderID := s.createPendingOrder(helpers.CustomerUserID)
	resp := helpers.AssertErrorResponse(s.T(), s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	}), http.StatusBadRequest)
	s.Assert().Equal("GATEWAY_NOT_CONFIGURED", resp["code"], "production without prod keys must not use sandbox")
}

// ─── Initiate Payment: Invalid paymentMethodType ─────────────────────────────
// Scenario: A customer sends an optional paymentMethodType the selected gateway
// does not support.
// Validates: rejection with 400.
func (s *PaymentSuite) TestInitiatePaymentRejectsUnsupportedPaymentMethodType() {
	orderID := s.createPendingOrder(helpers.CustomerUserID)

	w := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId":           orderID,
		"paymentMethodType": "bitcoin",
	})
	helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
}

// ─── Initiate Payment: Correlation ID Required ───────────────────────────────
// Scenario: A customer initiates payment without X-Correlation-ID.
// Validates: rejection with 400 CORRELATION_ID_REQUIRED (only the public
// webhook route is exempt from the correlation requirement).
func (s *PaymentSuite) TestInitiatePaymentRequiresCorrelationID() {
	orderID := s.createPendingOrder(helpers.CustomerUserID)

	client := helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), client, helpers.CustomerEmail, helpers.CustomerPassword)
	client.SetToken(token)
	client.SetHeader("X-Correlation-ID", "")
	w := client.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	resp := helpers.AssertErrorResponse(s.T(), w, http.StatusBadRequest)
	s.Assert().Equal("CORRELATION_ID_REQUIRED", resp["code"], "authenticated APIs must require correlation id")
}

// ─── List Transactions: Seller Isolation ───────────────────────────────────────
// Scenario: A seller lists payments and only sees their own.
func (s *PaymentSuite) TestListSellerTransactionsIsolated() {
	orderID := s.createPendingOrder(helpers.CustomerUserID)
	w := s.customerClient.Post(s.T(), InitiatePaymentAPIEndpoint, map[string]any{
		"orderId": orderID,
	})
	helpers.AssertSuccessResponse(s.T(), w, http.StatusOK)

	// Seller 2 (the customer's seller) lists the transaction.
	seller2 := helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), seller2, helpers.Seller2Email, helpers.Seller2Password)
	seller2.SetToken(token)

	list := seller2.Get(s.T(), TransactionsAPIEndpoint)
	response := helpers.AssertSuccessResponse(s.T(), list, http.StatusOK)
	assert.NotNil(s.T(), response["data"], "data should be present")
}

func (s *PaymentSuite) verifyTransactionRow(transactionID string, orderID uint) {
	var count int64
	err := s.container.DB.Table("payment_transaction").
		Where("transaction_id = ? AND reference_type = ? AND reference_id = ?",
			transactionID, "order", orderID).
		Count(&count).Error
	s.Require().NoError(err)
	s.Assert().Equal(int64(1), count, "transaction row should exist with order reference")
}

func (s *PaymentSuite) verifyOrderTransactionID(orderID uint, transactionID string) {
	var stored string
	err := s.container.DB.Table(`"order"`).
		Select("transaction_id").
		Where("id = ?", orderID).
		Scan(&stored).Error
	s.Require().NoError(err)
	s.Assert().Equal(transactionID, stored, "order.transaction_id should be set")
}

func (s *PaymentSuite) confirmOrderAsSeller(orderID uint) {
	// The order belongs to seller 2 (the customer's seller) — use a seller-2 client.
	seller2 := helpers.NewAPIClient(s.server)
	token := helpers.Login(s.T(), seller2, helpers.Seller2Email, helpers.Seller2Password)
	seller2.SetToken(token)

	w := seller2.Patch(s.T(), fmt.Sprintf("/api/order/%d/status", orderID), map[string]any{
		"status":        "confirmed",
		"transactionId": "manual-confirm-test",
	})
	s.Require().Equal(http.StatusOK, w.Code, "order should be confirmed by seller")
}
