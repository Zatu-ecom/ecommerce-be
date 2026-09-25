package payment_test

import (
	"fmt"

	"ecommerce-be/payment/entity"
	paymentSingleton "ecommerce-be/payment/factory/singleton"
	"ecommerce-be/test/integration/helpers"

	"github.com/stretchr/testify/assert"
)

// ─── Reconcile: Stale Pending With Open Session Expires ──────────────────────
// Scenario: A pending payment older than PAYMENT_PENDING_TTL_MINUTES whose
// provider session is still open (no capture).
// Validates: transaction failed with EXPIRED_UNPAID, order failed.
func (s *PaymentSuite) TestReconcileExpiresStalePendingWithOpenSession() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)

	// Provider session still open (fake defaults to created).
	s.backdateTransaction(txID, 60)

	s.reconcileNow()

	s.verifyTransactionStatus(txID, "failed")
	s.verifyFailureCode(txID, "EXPIRED_UNPAID")
	s.verifyOrderStatus(orderID, "failed")
	s.verifyEventExists(txID, "failed")
}

// ─── Reconcile: Stale Pending With Paid Remote Completes ─────────────────────
// Scenario: A pending payment older than the TTL whose provider session shows
// paid (webhook missed).
// Validates: transaction completed, order confirmed.
func (s *PaymentSuite) TestReconcileCompletesStalePendingWithPaidRemote() {
	orderID := s.initiatePaymentForOrder()
	txID := s.transactionIDForOrder(orderID)
	sessionID := s.sessionIDForTransaction(txID)

	amount := s.amountCentsForTransaction(txID)
	s.fakeRazorpay.setRemoteOrderStatus(sessionID, "paid", amount, "INR")
	s.backdateTransaction(txID, 60)

	s.reconcileNow()

	s.verifyTransactionStatus(txID, "completed")
	s.verifyOrderStatus(orderID, "confirmed")
	s.verifyEventExists(txID, "captured")
}

// ─── Reconcile: Sessionless Stale Pending Fails Fast ─────────────────────────
// Scenario: A pending payment with no gateway session (initiate crashed before
// the provider call) older than PAYMENT_SESSIONLESS_TTL_MINUTES.
// Validates: transaction failed, order failed — without provider interaction.
func (s *PaymentSuite) TestReconcileFailsStaleSessionlessPending() {
	orderID := s.createPendingOrder(helpers.CustomerUserID)

	var gateway entity.PaymentGateway
	s.Require().NoError(s.container.DB.Where("code = ?", "razorpay").First(&gateway).Error)

	txID := fmt.Sprintf("TXN_SESSLESS_%d", orderID)
	txn := entity.PaymentTransaction{
		TransactionID: txID,
		UserID:        helpers.CustomerUserID,
		SellerID:      helpers.Seller2UserID,
		GatewayID:     &gateway.ID,
		ReferenceType: entity.ReferenceTypeOrder,
		ReferenceID:   orderID,
		Currency:      "INR",
		AmountCents:   99900,
		Status:        entity.TransactionStatusPending,
		Environment:   entity.EnvironmentSandbox,
		// No GatewaySessionID: initiate never reached the provider.
	}
	s.Require().NoError(s.container.DB.Create(&txn).Error)
	s.Require().NoError(s.container.DB.Table(`"order"`).
		Where("id = ?", orderID).
		Update("transaction_id", txID).Error)
	s.backdateTransaction(txID, 6)

	s.reconcileNow()

	s.verifyTransactionStatus(txID, "failed")
	s.verifyOrderStatus(orderID, "failed")
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

// reconcileNow triggers one reconciliation pass through the production wiring.
func (s *PaymentSuite) reconcileNow() {
	paymentSingleton.GetInstance().GetServiceFactory().GetReconcileService().ReconcilePending()
}

// backdateTransaction moves a transaction's created_at back by minutes.
func (s *PaymentSuite) backdateTransaction(transactionID string, minutesAgo int) {
	s.Require().NoError(s.container.DB.Exec(
		`UPDATE payment_transaction SET created_at = NOW() - (INTERVAL '1 minute' * ?) WHERE transaction_id = ?`,
		minutesAgo, transactionID).Error)
}

func (s *PaymentSuite) verifyFailureCode(transactionID, code string) {
	var stored string
	err := s.container.DB.Table("payment_transaction").
		Select("failure_code").
		Where("transaction_id = ?", transactionID).
		Scan(&stored).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), code, stored, "failure code mismatch")
}
