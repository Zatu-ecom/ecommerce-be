package service

import (
	"context"
	"errors"
	"time"

	orderService "ecommerce-be/order/service"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

// Mismatch errors: the provider event does not describe OUR transaction.
// Callers record the webhook log as failed and change no money state.
var (
	ErrApplyAmountMismatch   = errors.New("webhook amount does not match transaction")
	ErrApplyCurrencyMismatch = errors.New("webhook currency does not match transaction")
)

// PaymentOrderHooks is the narrow order-module surface payments may call.
// orderService.OrderService satisfies it; the narrow type keeps tests DB-free
// and documents that payments never touch order repositories.
type PaymentOrderHooks interface {
	ConfirmPaymentByTransactionID(ctx context.Context, transactionID string) error
	FailPaymentByTransactionID(ctx context.Context, transactionID, reason string) error
}

// Compile-time guard: the real order service satisfies the narrow interface.
var _ PaymentOrderHooks = (orderService.OrderService)(nil)

// ApplyResult links the rows an apply touched for the caller's bookkeeping
// (webhook log linkage, cron metrics).
type ApplyResult struct {
	TransactionID uint
	RefundID      *uint
}

// NormalizedApplier applies verified NormalizedWebhook events to transactions
// and refunds. It is the ONE apply path shared by the webhook pipeline
// (source=webhook) and the reconciliation cron (source=system), so the two
// can never diverge on status machines, idempotency, or order hooks.
//
// The switch is ONLY on gateway.WebhookAction — provider event names never
// reach this file.
type NormalizedApplier struct {
	transactions repository.PaymentTransactionRepository
	refunds      repository.PaymentRefundRepository
	events       repository.PaymentTransactionEventRepository
	orders       PaymentOrderHooks
}

// NewNormalizedApplier creates the shared apply path.
func NewNormalizedApplier(
	transactions repository.PaymentTransactionRepository,
	refunds repository.PaymentRefundRepository,
	events repository.PaymentTransactionEventRepository,
	orders PaymentOrderHooks,
) *NormalizedApplier {
	return &NormalizedApplier{
		transactions: transactions,
		refunds:      refunds,
		events:       events,
		orders:       orders,
	}
}

// Apply applies one normalized event to an already-located transaction.
func (a *NormalizedApplier) Apply(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
	source entity.EventSource,
) (*ApplyResult, error) {
	result := &ApplyResult{TransactionID: txn.ID}

	switch n.Action {
	case gateway.WebhookActionIgnore:
		return result, nil
	case gateway.WebhookActionAuthorized:
		return result, a.applyAuthorized(ctx, txn, n, source)
	case gateway.WebhookActionPaymentCompleted:
		return result, a.applyPaymentCompleted(ctx, txn, n, source, result)
	case gateway.WebhookActionPaymentFailed:
		return result, a.applyPaymentFailed(ctx, txn, n, source)
	case gateway.WebhookActionRefundPending:
		return result, a.applyRefundPending(ctx, txn, n, result)
	case gateway.WebhookActionRefundCompleted:
		return result, a.applyRefundCompleted(ctx, txn, n, source, result)
	case gateway.WebhookActionRefundFailed:
		return result, a.applyRefundFailed(ctx, txn, n, source, result)
	default:
		return result, nil
	}
}

// applyAuthorized records an observation event only — no status or order change.
func (a *NormalizedApplier) applyAuthorized(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
	source entity.EventSource,
) error {
	return a.events.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:  txn.ID,
		EventType:      entity.EventTypeAuthorized,
		FromStatus:     txn.Status,
		ToStatus:       txn.Status,
		GatewayEventID: n.EventID,
		Source:         source,
		ActorType:      entity.ActorTypeSystem,
	})
}

// applyPaymentCompleted completes a pending transaction and confirms the order.
// A mismatched amount/currency refuses completion: the provider event does not
// describe our transaction (wrong-order capture), so nothing moves.
func (a *NormalizedApplier) applyPaymentCompleted(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
	source entity.EventSource,
	result *ApplyResult,
) error {
	if n.AmountCents > 0 && n.AmountCents != txn.AmountCents {
		return ErrApplyAmountMismatch
	}
	if n.Currency != "" && n.Currency != txn.Currency {
		return ErrApplyCurrencyMismatch
	}

	patch := map[string]any{"completed_at": time.Now().UTC()}
	if n.PaymentID != "" {
		patch["gateway_payment_id"] = n.PaymentID
	}
	if n.FeeCents != nil {
		patch["gateway_fee_cents"] = n.FeeCents
	}
	if details := methodDetailsSnapshot(n); details != nil {
		patch["payment_method_details"] = details
	}

	changed, err := a.transactions.UpdateStatusIfCurrent(
		ctx, txn.ID,
		entity.TransactionStatusPending,
		entity.TransactionStatusCompleted,
		patch,
	)
	if err != nil {
		return err
	}
	if !changed {
		// Already terminal (completed/failed/…) — idempotent no-op, no event spam.
		return nil
	}

	if err := a.events.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:   txn.ID,
		EventType:       entity.EventTypeCaptured,
		FromStatus:      entity.TransactionStatusPending,
		ToStatus:        entity.TransactionStatusCompleted,
		GatewayEventID:  n.EventID,
		GatewayResponse: n.Payload,
		Source:          source,
		ActorType:       entity.ActorTypeSystem,
	}); err != nil {
		return err
	}

	return a.orders.ConfirmPaymentByTransactionID(ctx, txn.TransactionID)
}

// applyPaymentFailed fails a pending transaction and its order.
func (a *NormalizedApplier) applyPaymentFailed(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
	source entity.EventSource,
) error {
	changed, err := a.transactions.UpdateStatusIfCurrent(
		ctx, txn.ID,
		entity.TransactionStatusPending,
		entity.TransactionStatusFailed,
		map[string]any{
			"failure_code":    n.FailureCode,
			"failure_message": n.FailureMessage,
		},
	)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	if err := a.events.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:  txn.ID,
		EventType:      entity.EventTypeFailed,
		FromStatus:     entity.TransactionStatusPending,
		ToStatus:       entity.TransactionStatusFailed,
		GatewayEventID: n.EventID,
		FailureCode:    n.FailureCode,
		FailureMessage: n.FailureMessage,
		Source:         source,
		ActorType:      entity.ActorTypeSystem,
	}); err != nil {
		return err
	}

	return a.orders.FailPaymentByTransactionID(ctx, txn.TransactionID, n.FailureMessage)
}

// applyRefundPending records a provider-observed refund without moving money.
func (a *NormalizedApplier) applyRefundPending(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
	result *ApplyResult,
) error {
	if n.RefundID == "" {
		return paymenterrors.ErrorRefundNotFound
	}
	refund, err := a.findOrCreateRefund(ctx, txn, n)
	if err != nil {
		return err
	}
	result.RefundID = &refund.ID
	return nil
}

// applyRefundCompleted completes a refund row and recomputes the transaction's
// refunded/partially_refunded state from completed totals.
func (a *NormalizedApplier) applyRefundCompleted(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
	source entity.EventSource,
	result *ApplyResult,
) error {
	if n.RefundID == "" {
		return paymenterrors.ErrorRefundNotFound
	}
	refund, err := a.findOrCreateRefund(ctx, txn, n)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	// Transition from either pending (webhook-observed) or processing (API-initiated).
	if _, err = a.refunds.UpdateStatusIfCurrent(ctx, refund.ID,
		entity.RefundStatusPending, entity.RefundStatusCompleted,
		map[string]any{"completed_at": now}); err != nil {
		return err
	}
	if _, err = a.refunds.UpdateStatusIfCurrent(ctx, refund.ID,
		entity.RefundStatusProcessing, entity.RefundStatusCompleted,
		map[string]any{"completed_at": now}); err != nil {
		return err
	}
	result.RefundID = &refund.ID

	if err := a.events.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:  txn.ID,
		EventType:      entity.EventTypeRefundCompleted,
		FromStatus:     txn.Status,
		ToStatus:       txn.Status,
		GatewayEventID: n.EventID,
		Source:         source,
		ActorType:      entity.ActorTypeSystem,
	}); err != nil {
		return err
	}

	return a.recomputeTransactionRefundStatus(ctx, txn.ID)
}

// applyRefundFailed marks a refund attempt failed; transaction money state is
// unchanged for that attempt.
func (a *NormalizedApplier) applyRefundFailed(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
	source entity.EventSource,
	result *ApplyResult,
) error {
	if n.RefundID == "" {
		return paymenterrors.ErrorRefundNotFound
	}
	refund, err := a.findRefund(ctx, n.RefundID)
	if err != nil {
		return err
	}
	if _, err = a.refunds.UpdateStatusIfCurrent(ctx, refund.ID,
		entity.RefundStatusPending, entity.RefundStatusFailed,
		map[string]any{"failure_reason": n.FailureMessage}); err != nil {
		return err
	}
	if _, err = a.refunds.UpdateStatusIfCurrent(ctx, refund.ID,
		entity.RefundStatusProcessing, entity.RefundStatusFailed,
		map[string]any{"failure_reason": n.FailureMessage}); err != nil {
		return err
	}
	result.RefundID = &refund.ID

	return a.events.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:  txn.ID,
		EventType:      entity.EventTypeRefundFailed,
		FromStatus:     txn.Status,
		ToStatus:       txn.Status,
		GatewayEventID: n.EventID,
		FailureMessage: n.FailureMessage,
		Source:         source,
		ActorType:      entity.ActorTypeSystem,
	})
}

// findRefund locates a refund by gateway id, falling back to our internal id.
func (a *NormalizedApplier) findRefund(
	ctx context.Context,
	refundID string,
) (*entity.PaymentRefund, error) {
	if refund, err := a.refunds.FindByGatewayRefundID(ctx, refundID); err == nil {
		return refund, nil
	}
	return a.refunds.FindByRefundID(ctx, refundID)
}

// findOrCreateRefund returns the refund row for a provider refund id, creating
// a pending row when the provider acted first (dashboard-initiated refunds).
func (a *NormalizedApplier) findOrCreateRefund(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
) (*entity.PaymentRefund, error) {
	if refund, err := a.findRefund(ctx, n.RefundID); err == nil {
		return refund, nil
	}
	currency := n.Currency
	if currency == "" {
		currency = txn.Currency
	}
	refund := &entity.PaymentRefund{
		RefundID:        n.RefundID,
		GatewayRefundID: n.RefundID,
		TransactionID:   txn.ID,
		Currency:        currency,
		AmountCents:     n.AmountCents,
		Status:          entity.RefundStatusPending,
	}
	if err := a.refunds.Create(ctx, refund); err != nil {
		return nil, err
	}
	return refund, nil
}

// recomputeTransactionRefundStatus sets refunded/partially_refunded from
// completed refund totals. Handles both first (completed→…) and subsequent
// (partially_refunded→…) partial refunds.
func (a *NormalizedApplier) recomputeTransactionRefundStatus(
	ctx context.Context,
	transactionID uint,
) error {
	txn, err := a.transactions.FindByID(ctx, transactionID)
	if err != nil {
		return err
	}

	var refunded int64
	if err := a.refunds.SumByTransactionAndStatuses(ctx, transactionID, []entity.RefundStatus{
		entity.RefundStatusCompleted,
	}, &refunded); err != nil {
		return err
	}

	target := entity.TransactionStatusPartiallyRefunded
	if refunded >= txn.AmountCents {
		target = entity.TransactionStatusRefunded
	}

	changed, err := a.transactions.UpdateStatusIfCurrent(ctx, transactionID,
		entity.TransactionStatusCompleted, target, map[string]any{})
	if err != nil {
		return err
	}
	if changed {
		return nil
	}
	_, err = a.transactions.UpdateStatusIfCurrent(ctx, transactionID,
		entity.TransactionStatusPartiallyRefunded, target, map[string]any{})
	return err
}

// methodDetailsSnapshot extracts a display-safe payment-method snapshot from a
// normalized payload when the adapter captured one. Nil when absent, so the
// apply leaves any existing snapshot untouched.
func methodDetailsSnapshot(n *gateway.NormalizedWebhook) map[string]any {
	if n.Payload == nil {
		return nil
	}
	if method, ok := n.Payload["method"].(string); ok && method != "" {
		return map[string]any{"method": method}
	}
	return nil
}
