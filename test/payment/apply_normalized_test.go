package payment_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	paymentservice "ecommerce-be/payment/service"

	"ecommerce-be/payment/entity"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

// ─── In-memory fakes (no DB) ─────────────────────────────────────────────────

type fakeTxnRepo struct {
	mu   sync.Mutex
	rows map[uint]*entity.PaymentTransaction
	seq  uint
}

func newFakeTxnRepo() *fakeTxnRepo {
	return &fakeTxnRepo{rows: map[uint]*entity.PaymentTransaction{}}
}

func (f *fakeTxnRepo) seed(txn *entity.PaymentTransaction) *entity.PaymentTransaction {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	txn.ID = f.seq
	cp := *txn
	f.rows[txn.ID] = &cp
	return &cp
}

func (f *fakeTxnRepo) get(id uint) *entity.PaymentTransaction {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *f.rows[id]
	return &cp
}

func (f *fakeTxnRepo) Create(_ context.Context, txn *entity.PaymentTransaction) error {
	f.seed(txn)
	return nil
}

func (f *fakeTxnRepo) FindByID(_ context.Context, id uint) (*entity.PaymentTransaction, error) {
	return f.get(id), nil
}

func (f *fakeTxnRepo) FindByTransactionID(_ context.Context, _ string) (*entity.PaymentTransaction, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTxnRepo) FindByGatewaySessionID(_ context.Context, _ string) (*entity.PaymentTransaction, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTxnRepo) FindByGatewayPaymentID(_ context.Context, _ string) (*entity.PaymentTransaction, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeTxnRepo) FindByReference(_ context.Context, _ entity.ReferenceType, _ uint) ([]entity.PaymentTransaction, error) {
	return nil, nil
}

func (f *fakeTxnRepo) FindBySellerID(_ context.Context, _ uint, _, _ int, _ entity.TransactionStatus) ([]entity.PaymentTransaction, int64, error) {
	return nil, 0, nil
}

func (f *fakeTxnRepo) UpdateStatusIfCurrent(
	_ context.Context, id uint, from, to entity.TransactionStatus, patch map[string]any,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	txn := f.rows[id]
	if txn.Status != from {
		return false, nil
	}
	txn.Status = to
	for k, v := range patch {
		switch k {
		case "gateway_payment_id":
			if s, ok := v.(string); ok {
				txn.GatewayPaymentID = s
			}
		case "completed_at":
			if tm, ok := v.(time.Time); ok {
				txn.CompletedAt = &tm
			}
		case "gateway_fee_cents":
			switch fee := v.(type) {
			case *int64:
				txn.GatewayFeeCents = fee
			case int64:
				txn.GatewayFeeCents = &fee
			}
		case "failure_code":
			if s, ok := v.(string); ok {
				txn.FailureCode = s
			}
		case "failure_message":
			if s, ok := v.(string); ok {
				txn.FailureMessage = s
			}
		case "payment_method_details":
			if m, ok := v.(map[string]any); ok {
				txn.PaymentMethodDetails = m
			}
		}
	}
	return true, nil
}

func (f *fakeTxnRepo) ListPendingForReconcile(_ context.Context, _ time.Time, _ int) ([]entity.PaymentTransaction, error) {
	return nil, nil
}

type fakeRefundRepo struct {
	mu   sync.Mutex
	rows map[uint]*entity.PaymentRefund
	seq  uint
}

func newFakeRefundRepo() *fakeRefundRepo {
	return &fakeRefundRepo{rows: map[uint]*entity.PaymentRefund{}}
}

func (f *fakeRefundRepo) Create(_ context.Context, refund *entity.PaymentRefund) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	refund.ID = f.seq
	cp := *refund
	f.rows[refund.ID] = &cp
	return nil
}

func (f *fakeRefundRepo) FindByRefundID(_ context.Context, refundID string) (*entity.PaymentRefund, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.RefundID == refundID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, errors.New("refund not found")
}

func (f *fakeRefundRepo) FindByGatewayRefundID(_ context.Context, gatewayRefundID string) (*entity.PaymentRefund, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.GatewayRefundID == gatewayRefundID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, errors.New("refund not found")
}

func (f *fakeRefundRepo) SumByTransactionAndStatuses(
	_ context.Context, transactionID uint, statuses []entity.RefundStatus, sum *int64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	want := map[entity.RefundStatus]bool{}
	for _, s := range statuses {
		want[s] = true
	}
	var total int64
	for _, r := range f.rows {
		if r.TransactionID == transactionID && want[r.Status] {
			total += r.AmountCents
		}
	}
	*sum = total
	return nil
}

func (f *fakeRefundRepo) UpdateStatusIfCurrent(
	_ context.Context, id uint, from, to entity.RefundStatus, patch map[string]any,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.rows[id]
	if r.Status != from {
		return false, nil
	}
	r.Status = to
	for k, v := range patch {
		switch k {
		case "completed_at":
			if tm, ok := v.(time.Time); ok {
				r.CompletedAt = &tm
			}
		case "failure_reason":
			if s, ok := v.(string); ok {
				r.FailureReason = s
			}
		}
	}
	return true, nil
}

func (f *fakeRefundRepo) ListStuckRefunds(_ context.Context, _ time.Time, _ int) ([]entity.PaymentRefund, error) {
	return nil, nil
}

type fakeEventRepo struct {
	mu     sync.Mutex
	events []entity.PaymentTransactionEvent
}

func (f *fakeEventRepo) Create(_ context.Context, event *entity.PaymentTransactionEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, *event)
	return nil
}

func (f *fakeEventRepo) ListByTransactionID(_ context.Context, transactionID uint) ([]entity.PaymentTransactionEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []entity.PaymentTransactionEvent
	for _, e := range f.events {
		if e.TransactionID == transactionID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeEventRepo) count(txnID uint, eventType entity.EventType) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, e := range f.events {
		if e.TransactionID == txnID && e.EventType == eventType {
			n++
		}
	}
	return n
}

type fakeOrderHooks struct {
	mu        sync.Mutex
	confirmed []string
	failed    []string
}

func (f *fakeOrderHooks) ConfirmPaymentByTransactionID(_ context.Context, transactionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.confirmed = append(f.confirmed, transactionID)
	return nil
}

func (f *fakeOrderHooks) FailPaymentByTransactionID(_ context.Context, transactionID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = append(f.failed, transactionID)
	return nil
}

// Compile-time guards: fakes satisfy the production interfaces.
var (
	_ repository.PaymentTransactionRepository      = (*fakeTxnRepo)(nil)
	_ repository.PaymentRefundRepository           = (*fakeRefundRepo)(nil)
	_ repository.PaymentTransactionEventRepository = (*fakeEventRepo)(nil)
)

func newApplier() (*paymentservice.NormalizedApplier, *fakeTxnRepo, *fakeRefundRepo, *fakeEventRepo, *fakeOrderHooks) {
	txns := newFakeTxnRepo()
	refunds := newFakeRefundRepo()
	events := &fakeEventRepo{}
	orders := &fakeOrderHooks{}
	return paymentservice.NewNormalizedApplier(txns, refunds, events, orders), txns, refunds, events, orders
}

func pendingTxn() *entity.PaymentTransaction {
	return &entity.PaymentTransaction{
		TransactionID: "TXN_APPLY_1",
		UserID:        5,
		SellerID:      2,
		Currency:      "INR",
		AmountCents:   10000,
		Status:        entity.TransactionStatusPending,
		Environment:   entity.EnvironmentSandbox,
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestApplyPaymentCompletedConfirmsOrder(t *testing.T) {
	apply, txns, _, events, orders := newApplier()
	txn := txns.seed(pendingTxn())
	fee := int64(200)

	res, err := apply.Apply(t.Context(), txn, &gateway.NormalizedWebhook{
		Action:      gateway.WebhookActionPaymentCompleted,
		EventID:     "evt_completed_1",
		SessionID:   "order_sess_1",
		PaymentID:   "pay_completed_1",
		AmountCents: 10000,
		Currency:    "INR",
		FeeCents:    &fee,
		Payload:     map[string]any{},
	}, entity.EventSourceWebhook)
	if err != nil {
		t.Fatalf("apply completed: %v", err)
	}
	if res == nil || res.TransactionID != txn.ID {
		t.Fatalf("result must link the transaction: %+v", res)
	}

	stored := txns.get(txn.ID)
	if stored.Status != entity.TransactionStatusCompleted {
		t.Fatalf("status = %q, want completed", stored.Status)
	}
	if stored.GatewayPaymentID != "pay_completed_1" {
		t.Fatalf("payment id not persisted: %q", stored.GatewayPaymentID)
	}
	if stored.GatewayFeeCents == nil || *stored.GatewayFeeCents != 200 {
		t.Fatalf("fee not persisted: %v", stored.GatewayFeeCents)
	}
	if len(orders.confirmed) != 1 || orders.confirmed[0] != "TXN_APPLY_1" {
		t.Fatalf("order must be confirmed once: %v", orders.confirmed)
	}
	if n := events.count(txn.ID, entity.EventTypeCaptured); n != 1 {
		t.Fatalf("captured events = %d, want 1", n)
	}
}

func TestApplyRefusesAmountMismatch(t *testing.T) {
	apply, txns, _, events, orders := newApplier()
	txn := txns.seed(pendingTxn())

	_, err := apply.Apply(t.Context(), txn, &gateway.NormalizedWebhook{
		Action:      gateway.WebhookActionPaymentCompleted,
		EventID:     "evt_mismatch_1",
		PaymentID:   "pay_mismatch_1",
		AmountCents: 9999, // wrong amount
		Currency:    "INR",
	}, entity.EventSourceWebhook)
	if !errors.Is(err, paymentservice.ErrApplyAmountMismatch) {
		t.Fatalf("err = %v, want amount mismatch", err)
	}
	if stored := txns.get(txn.ID); stored.Status != entity.TransactionStatusPending {
		t.Fatalf("mismatched webhook must not complete, status = %q", stored.Status)
	}
	if len(orders.confirmed) != 0 {
		t.Fatalf("order must not be confirmed on mismatch: %v", orders.confirmed)
	}
	if n := events.count(txn.ID, entity.EventTypeCaptured); n != 0 {
		t.Fatalf("no captured event on mismatch, got %d", n)
	}
}

func TestApplyRefusesCurrencyMismatch(t *testing.T) {
	apply, txns, _, _, orders := newApplier()
	txn := txns.seed(pendingTxn())

	_, err := apply.Apply(t.Context(), txn, &gateway.NormalizedWebhook{
		Action:      gateway.WebhookActionPaymentCompleted,
		EventID:     "evt_fx_1",
		PaymentID:   "pay_fx_1",
		AmountCents: 10000,
		Currency:    "USD", // wrong currency
	}, entity.EventSourceWebhook)
	if !errors.Is(err, paymentservice.ErrApplyCurrencyMismatch) {
		t.Fatalf("err = %v, want currency mismatch", err)
	}
	if stored := txns.get(txn.ID); stored.Status != entity.TransactionStatusPending {
		t.Fatalf("status = %q, want pending", stored.Status)
	}
	if len(orders.confirmed) != 0 {
		t.Fatalf("order must not be confirmed: %v", orders.confirmed)
	}
}

func TestApplyAuthorizedKeepsPending(t *testing.T) {
	apply, txns, _, events, orders := newApplier()
	txn := txns.seed(pendingTxn())

	if _, err := apply.Apply(t.Context(), txn, &gateway.NormalizedWebhook{
		Action:    gateway.WebhookActionAuthorized,
		EventID:   "evt_auth_1",
		PaymentID: "pay_auth_1",
	}, entity.EventSourceWebhook); err != nil {
		t.Fatalf("authorized must not error: %v", err)
	}
	if stored := txns.get(txn.ID); stored.Status != entity.TransactionStatusPending {
		t.Fatalf("authorized must stay pending, got %q", stored.Status)
	}
	if len(orders.confirmed) != 0 || len(orders.failed) != 0 {
		t.Fatalf("authorized must not touch the order: %+v %+v", orders.confirmed, orders.failed)
	}
	if n := events.count(txn.ID, entity.EventTypeAuthorized); n != 1 {
		t.Fatalf("authorized events = %d, want 1", n)
	}
}

func TestApplyPaymentFailedFailsOrder(t *testing.T) {
	apply, txns, _, events, orders := newApplier()
	txn := txns.seed(pendingTxn())

	if _, err := apply.Apply(t.Context(), txn, &gateway.NormalizedWebhook{
		Action:         gateway.WebhookActionPaymentFailed,
		EventID:        "evt_fail_1",
		FailureCode:    "BAD_REQUEST_ERROR",
		FailureMessage: "payment failed",
	}, entity.EventSourceWebhook); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if stored := txns.get(txn.ID); stored.Status != entity.TransactionStatusFailed {
		t.Fatalf("status = %q, want failed", stored.Status)
	}
	if len(orders.failed) != 1 {
		t.Fatalf("order must be failed once: %v", orders.failed)
	}
	if n := events.count(txn.ID, entity.EventTypeFailed); n != 1 {
		t.Fatalf("failed events = %d, want 1", n)
	}
}

func TestApplyCompletedIsIdempotentOnTerminalTxn(t *testing.T) {
	apply, txns, _, events, orders := newApplier()
	txn := pendingTxn()
	txn.Status = entity.TransactionStatusCompleted
	txn = txns.seed(txn)

	if _, err := apply.Apply(t.Context(), txn, &gateway.NormalizedWebhook{
		Action:      gateway.WebhookActionPaymentCompleted,
		EventID:     "evt_replay_2",
		PaymentID:   "pay_completed_1",
		AmountCents: 10000,
		Currency:    "INR",
	}, entity.EventSourceWebhook); err != nil {
		t.Fatalf("replay must be a no-op, not an error: %v", err)
	}
	if n := events.count(txn.ID, entity.EventTypeCaptured); n != 0 {
		t.Fatalf("no duplicate captured event on replay, got %d", n)
	}
	if len(orders.confirmed) != 0 {
		t.Fatalf("order must not be confirmed twice: %v", orders.confirmed)
	}
}

func TestApplyRefundCompletedPartiallyRefunds(t *testing.T) {
	apply, txns, _, events, _ := newApplier()
	txn := pendingTxn()
	txn.Status = entity.TransactionStatusCompleted
	txn = txns.seed(txn)

	res, err := apply.Apply(t.Context(), txn, &gateway.NormalizedWebhook{
		Action:      gateway.WebhookActionRefundCompleted,
		EventID:     "evt_rf_1",
		RefundID:    "rf_partial_1",
		AmountCents: 4000,
		Currency:    "INR",
	}, entity.EventSourceWebhook)
	if err != nil {
		t.Fatalf("apply refund completed: %v", err)
	}
	if res == nil || res.RefundID == nil {
		t.Fatalf("result must link the refund: %+v", res)
	}
	if stored := txns.get(txn.ID); stored.Status != entity.TransactionStatusPartiallyRefunded {
		t.Fatalf("status = %q, want partially_refunded", stored.Status)
	}

	// Second partial for the remainder flips to fully refunded.
	if _, err := apply.Apply(t.Context(), txns.get(txn.ID), &gateway.NormalizedWebhook{
		Action:      gateway.WebhookActionRefundCompleted,
		EventID:     "evt_rf_2",
		RefundID:    "rf_partial_2",
		AmountCents: 6000,
		Currency:    "INR",
	}, entity.EventSourceWebhook); err != nil {
		t.Fatalf("second partial: %v", err)
	}
	if stored := txns.get(txn.ID); stored.Status != entity.TransactionStatusRefunded {
		t.Fatalf("status = %q, want refunded", stored.Status)
	}
	if n := events.count(txn.ID, entity.EventTypeRefundCompleted); n != 2 {
		t.Fatalf("refund_completed events = %d, want 2", n)
	}
}

func TestApplyIgnoreIsNoOp(t *testing.T) {
	apply, txns, _, events, orders := newApplier()
	txn := txns.seed(pendingTxn())

	if _, err := apply.Apply(t.Context(), txn, &gateway.NormalizedWebhook{
		Action:  gateway.WebhookActionIgnore,
		EventID: "evt_unknown_1",
	}, entity.EventSourceWebhook); err != nil {
		t.Fatalf("ignore must not error: %v", err)
	}
	if stored := txns.get(txn.ID); stored.Status != entity.TransactionStatusPending {
		t.Fatalf("status = %q, want pending", stored.Status)
	}
	if len(events.events) != 0 || len(orders.confirmed) != 0 || len(orders.failed) != 0 {
		t.Fatal("ignore must change nothing")
	}
}
