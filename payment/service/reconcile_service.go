package service

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"ecommerce-be/common/config"
	"ecommerce-be/common/db"
	"ecommerce-be/common/log"
	orderService "ecommerce-be/order/service"
	"ecommerce-be/payment/entity"
	"ecommerce-be/payment/factory"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

// ReconcileService heals payments the webhook pipeline missed: abandoned
// checkouts stuck in pending and refunds stuck in pending/processing.
// It polls the provider (FetchRemoteStatus / FetchRefundStatus) and applies
// results through the SAME applyNormalized path as webhooks (source=system),
// so terminal transitions stay idempotent and order hooks can't diverge.
type ReconcileService interface {
	// ReconcilePending runs one bounded reconciliation pass. It never returns
	// an error: per-row failures are logged and retried on the next tick.
	ReconcilePending()
}

// ReconcileServiceImpl implements ReconcileService.
type ReconcileServiceImpl struct {
	transactions repository.PaymentTransactionRepository
	refunds      repository.PaymentRefundRepository
	events       repository.PaymentTransactionEventRepository
	gateways     repository.PaymentGatewayRepository
	configs      repository.PaymentGatewayConfigRepository
	factory      *factory.PaymentGatewayFactory
	orders       orderService.OrderService
	applier      *NormalizedApplier

	running atomic.Bool
}

// NewReconcileService creates the reconciliation worker.
func NewReconcileService(
	transactions repository.PaymentTransactionRepository,
	refunds repository.PaymentRefundRepository,
	events repository.PaymentTransactionEventRepository,
	gateways repository.PaymentGatewayRepository,
	configs repository.PaymentGatewayConfigRepository,
	gatewayFactory *factory.PaymentGatewayFactory,
	orders orderService.OrderService,
) ReconcileService {
	return &ReconcileServiceImpl{
		transactions: transactions,
		refunds:      refunds,
		events:       events,
		gateways:     gateways,
		configs:      configs,
		factory:      gatewayFactory,
		orders:       orders,
		applier:      NewNormalizedApplier(transactions, refunds, events, orders),
	}
}

// reconcileOutcome classifies one row for the run summary.
type reconcileOutcome int

const (
	outcomeSkipped reconcileOutcome = iota
	outcomeApplied
	outcomeExpired
	outcomeError
)

// ReconcilePending runs one bounded pass over stale pending payments and
// stuck refunds. Overlapping runs are skipped (single-flight per process;
// cross-instance overlap is safe anyway: applies are conditional/idempotent).
func (s *ReconcileServiceImpl) ReconcilePending() {
	ctx := context.Background()
	if !s.running.CompareAndSwap(false, true) {
		log.InfoWithContext(ctx, "Cron: payment reconcile skipped (previous run active)")
		return
	}
	defer s.running.Store(false)

	pendingTTL, sessionlessTTL, refundTTL := reconcileTTLs()
	now := time.Now().UTC()

	var polled, applied, expired, errored int

	// Pending pass: select rows older than the SHORT (sessionless) TTL inside
	// one transaction so FOR UPDATE SKIP LOCKED is meaningful, then process
	// outside the transaction — locks must never span provider HTTP calls.
	// Conditional status updates keep webhook-vs-cron races safe.
	var candidates []entity.PaymentTransaction
	if err := db.WithTransaction(ctx, func(txCtx context.Context) error {
		var err error
		candidates, err = s.transactions.ListPendingForReconcile(txCtx, now.Add(-sessionlessTTL), 0)
		return err
	}); err != nil {
		log.ErrorWithContext(ctx, "Cron: payment reconcile list failed", err)
		return
	}
	for i := range candidates {
		polled++
		switch s.reconcileTransaction(ctx, &candidates[i], now, pendingTTL, sessionlessTTL) {
		case outcomeApplied:
			applied++
		case outcomeExpired:
			expired++
		case outcomeError:
			errored++
		}
	}

	// Stuck-refund pass.
	var stuck []entity.PaymentRefund
	if err := db.WithTransaction(ctx, func(txCtx context.Context) error {
		var err error
		stuck, err = s.refunds.ListStuckRefunds(txCtx, now.Add(-refundTTL), 0)
		return err
	}); err != nil {
		log.ErrorWithContext(ctx, "Cron: payment reconcile refund list failed", err)
		return
	}
	for i := range stuck {
		polled++
		switch s.reconcileRefund(ctx, &stuck[i]) {
		case outcomeApplied:
			applied++
		case outcomeError:
			errored++
		}
	}

	log.InfoWithContext(ctx, fmt.Sprintf(
		"Cron: payment reconcile done (polled=%d applied=%d expired=%d errors=%d)",
		polled, applied, expired, errored))
}

// reconcileTransaction heals one pending transaction.
func (s *ReconcileServiceImpl) reconcileTransaction(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	now time.Time,
	pendingTTL, sessionlessTTL time.Duration,
) reconcileOutcome {
	if txn.GatewayID == nil {
		return outcomeSkipped // non-gateway payment (e.g. COD): nothing to poll
	}
	age := now.Sub(txn.CreatedAt)
	sessionID := strings.TrimSpace(txn.GatewaySessionID)
	paymentID := strings.TrimSpace(txn.GatewayPaymentID)

	// No provider session (and no payment): initiate never reached the
	// provider. Fail fast after the short TTL; never call the provider.
	if sessionID == "" && paymentID == "" {
		if age < sessionlessTTL {
			return outcomeSkipped
		}
		return s.expireUnpaid(ctx, txn, "no gateway session within sessionless TTL")
	}

	// Young sessions stay pending until the TTL lapses.
	if sessionID != "" && age < pendingTTL {
		return outcomeSkipped
	}

	adapter, creds, ok := s.adapterForTransaction(ctx, txn)
	if !ok {
		return outcomeError
	}
	remote, err := adapter.FetchRemoteStatus(ctx, sessionID, paymentID, creds)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: payment fetch remote status failed", err)
		return outcomeError
	}

	// Still open past the TTL → abandoned checkout: expire it.
	if remote.Action == gateway.WebhookActionIgnore {
		if sessionID != "" && age < pendingTTL {
			return outcomeSkipped
		}
		return s.expireUnpaid(ctx, txn, "session still open past pending TTL")
	}

	if err := s.applyNormalizedSystem(ctx, txn, remote); err != nil {
		log.ErrorWithContext(ctx, "Cron: payment apply remote status failed", err)
		return outcomeError
	}
	return outcomeApplied
}

// reconcileRefund re-polls one stuck refund through the provider.
func (s *ReconcileServiceImpl) reconcileRefund(
	ctx context.Context,
	refund *entity.PaymentRefund,
) reconcileOutcome {
	txn, err := s.transactions.FindByID(ctx, refund.TransactionID)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: payment reconcile refund txn missing", err)
		return outcomeError
	}
	if txn.GatewayID == nil {
		return outcomeSkipped
	}
	adapter, creds, ok := s.adapterForTransaction(ctx, txn)
	if !ok {
		return outcomeError
	}
	fetcher, ok := adapter.(gateway.RefundStatusFetcher)
	if !ok {
		log.WarnWithContext(ctx,
			"Cron: payment adapter cannot poll refunds, skipping stuck refund")
		return outcomeSkipped
	}
	refundID := refund.GatewayRefundID
	if refundID == "" {
		refundID = refund.RefundID
	}
	remote, err := fetcher.FetchRefundStatus(ctx, refundID, creds)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: payment fetch refund status failed", err)
		return outcomeError
	}
	if err := s.applyNormalizedSystem(ctx, txn, remote); err != nil {
		log.ErrorWithContext(ctx, "Cron: payment apply refund status failed", err)
		return outcomeError
	}
	return outcomeApplied
}

// expireUnpaid fails an abandoned pending payment (EXPIRED_UNPAID) through the
// shared apply path so the order fails and the ledger stays consistent.
func (s *ReconcileServiceImpl) expireUnpaid(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	reason string,
) reconcileOutcome {
	n := &gateway.NormalizedWebhook{
		Action:         gateway.WebhookActionPaymentFailed,
		EventID:        "system:" + txn.TransactionID + ":expired_unpaid",
		ProviderEvent:  "reconcile.expired_unpaid",
		TransactionID:  txn.TransactionID,
		SessionID:      txn.GatewaySessionID,
		PaymentID:      txn.GatewayPaymentID,
		FailureCode:    "EXPIRED_UNPAID",
		FailureMessage: "payment expired unpaid: " + reason,
		Payload:        map[string]any{"reason": reason},
	}
	if err := s.applyNormalizedSystem(ctx, txn, n); err != nil {
		log.ErrorWithContext(ctx, "Cron: payment expire unpaid failed", err)
		return outcomeError
	}
	return outcomeExpired
}

// applyNormalizedSystem applies one normalized event with source=system.
func (s *ReconcileServiceImpl) applyNormalizedSystem(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	n *gateway.NormalizedWebhook,
) error {
	_, err := s.applier.Apply(ctx, txn, n, entity.EventSourceSystem)
	return err
}

// adapterForTransaction resolves the adapter + decrypted credentials for the
// transaction's FROZEN environment (never the seller's current store toggle).
func (s *ReconcileServiceImpl) adapterForTransaction(
	ctx context.Context,
	txn *entity.PaymentTransaction,
) (gateway.PaymentGateway, map[string]any, bool) {
	adapter, err := s.factory.GetPaymentGateway(ctx, *txn.GatewayID)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: payment gateway unavailable", err)
		return nil, nil, false
	}
	var config *entity.PaymentGatewayConfig
	if txn.Environment != "" {
		config, err = s.configs.FindBySellerGatewayAndEnvironment(
			ctx, txn.SellerID, *txn.GatewayID, txn.Environment)
		if err != nil {
			log.ErrorWithContext(ctx, "Cron: payment config lookup failed", err)
			return nil, nil, false
		}
	} else {
		var configs []entity.PaymentGatewayConfig
		configs, err = s.configs.FindAllBySellerAndGateway(ctx, txn.SellerID, *txn.GatewayID)
		if err != nil {
			log.ErrorWithContext(ctx, "Cron: payment config lookup failed", err)
			return nil, nil, false
		}
		for i := range configs {
			if configs[i].IsActive {
				config = &configs[i]
				break
			}
		}
	}
	if config == nil {
		log.ErrorWithContext(ctx, "Cron: payment config missing for transaction", nil)
		return nil, nil, false
	}
	creds, err := adapter.Decrypt(config.Credentials)
	if err != nil {
		log.ErrorWithContext(ctx, "Cron: payment credential decrypt failed", err)
		return nil, nil, false
	}
	return adapter, creds, true
}

// reconcileTTLs resolves the cron thresholds from app config with production
// defaults. Non-positive values fall back to defaults (fail operational, not
// destructive: a zero TTL would expire every pending payment immediately).
func reconcileTTLs() (pending, sessionless, refund time.Duration) {
	pending, sessionless, refund =
		45*time.Minute, 5*time.Minute, 30*time.Minute
	if cfg := config.Get(); cfg != nil {
		if v := cfg.App.PaymentPendingTTLMinutes; v > 0 {
			pending = time.Duration(v) * time.Minute
		}
		if v := cfg.App.PaymentSessionlessTTLMinutes; v > 0 {
			sessionless = time.Duration(v) * time.Minute
		}
		if v := cfg.App.RefundStuckTTLMinutes; v > 0 {
			refund = time.Duration(v) * time.Minute
		}
	}
	return pending, sessionless, refund
}
