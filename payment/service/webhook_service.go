package service

import (
	"context"
	"net/http"

	"ecommerce-be/common/db"
	orderService "ecommerce-be/order/service"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"
	"ecommerce-be/payment/factory"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
)

// WebhookService processes verified inbound gateway webhooks.
type WebhookService interface {
	HandleWebhook(
		ctx context.Context,
		gatewayCode string,
		rawBody []byte,
		headers http.Header,
		ipAddress string,
	) error
}

// WebhookServiceImpl implements WebhookService.
type WebhookServiceImpl struct {
	gatewayRepo     repository.PaymentGatewayRepository
	configRepo      repository.PaymentGatewayConfigRepository
	transactionRepo repository.PaymentTransactionRepository
	refundRepo      repository.PaymentRefundRepository
	webhookLogRepo  repository.PaymentWebhookLogRepository
	eventRepo       repository.PaymentTransactionEventRepository
	gatewayFactory  *factory.PaymentGatewayFactory
	orderService    orderService.OrderService
	applier         *NormalizedApplier
}

// NewWebhookService creates the webhook service.
func NewWebhookService(
	gatewayRepo repository.PaymentGatewayRepository,
	configRepo repository.PaymentGatewayConfigRepository,
	transactionRepo repository.PaymentTransactionRepository,
	refundRepo repository.PaymentRefundRepository,
	webhookLogRepo repository.PaymentWebhookLogRepository,
	eventRepo repository.PaymentTransactionEventRepository,
	gatewayFactory *factory.PaymentGatewayFactory,
	orderService orderService.OrderService,
) WebhookService {
	return &WebhookServiceImpl{
		gatewayRepo:     gatewayRepo,
		configRepo:      configRepo,
		transactionRepo: transactionRepo,
		refundRepo:      refundRepo,
		webhookLogRepo:  webhookLogRepo,
		eventRepo:       eventRepo,
		gatewayFactory:  gatewayFactory,
		orderService:    orderService,
		applier:         NewNormalizedApplier(transactionRepo, refundRepo, eventRepo, orderService),
	}
}

// HandleWebhook verifies, dedupes, and applies a gateway webhook.
//
// Locate-then-verify: untrusted locator hints select the transaction first;
// the HMAC is then checked with THAT transaction's seller secret (never a
// first-config guess, never an N-seller brute force). Unverifiable or
// unlocatable deliveries change no state. After successful verification the
// result is always nil (HTTP 200) so the provider stops retrying; apply
// failures are recorded on the webhook log and healed by the reconcile cron.
func (s *WebhookServiceImpl) HandleWebhook(
	ctx context.Context,
	gatewayCode string,
	rawBody []byte,
	headers http.Header,
	ipAddress string,
) error {
	// 1. Resolve the gateway record + adapter. Unknown code → 404.
	gatewayEntity, err := s.gatewayRepo.FindByCode(ctx, gatewayCode)
	if err != nil {
		return err
	}
	adapter, err := s.gatewayFactory.GetPaymentGatewayByCode(gatewayCode)
	if err != nil {
		return err
	}

	// 2. Peek untrusted locators and find our transaction. No match → 200
	// with NO payload persistence (initiate may still be in flight).
	txn, err := s.locateTransaction(ctx, adapter.PeekLocators(rawBody))
	if err != nil {
		return err
	}
	if txn == nil {
		return nil
	}

	// 3. Load the config for the transaction's seller + frozen environment.
	config, err := s.configForTransaction(ctx, txn)
	if err != nil {
		return err
	}
	decrypted, err := adapter.Decrypt(config.Credentials)
	if err != nil {
		return err
	}

	// 4. Verify (HMAC on raw bytes) and normalize. Bad signature → 401.
	normalized, err := adapter.NormalizeWebhook(rawBody, headers, decrypted)
	if err != nil {
		return err
	}

	// 5. Record the webhook log; the unique (gateway_id, event_id) index
	// dedupes replays into a 200 no-op. The transaction link is set up front
	// (known once located) so every outcome — processed, ignored, failed —
	// stays attributable to the seller's ledger.
	logEntry := &entity.PaymentWebhookLog{
		GatewayID:     &gatewayEntity.ID,
		EventType:     normalized.ProviderEvent,
		EventID:       normalized.EventID,
		Payload:       normalized.Payload,
		Status:        entity.WebhookStatusReceived,
		IPAddress:     ipAddress,
		TransactionID: &txn.ID,
	}

	// applyErr carries the apply failure past the transaction boundary: the
	// failed-status row commits inside, the error propagates outside.
	var applyErr error

	if err := db.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.webhookLogRepo.Create(txCtx, logEntry); err != nil {
			if isUniqueViolation(err) {
				return nil
			}
			return err
		}

		// 6. Apply through the single shared path (webhook + cron).
		if normalized.Action == gateway.WebhookActionIgnore {
			return s.webhookLogRepo.MarkIgnored(txCtx, logEntry.ID)
		}
		result, err := s.applier.Apply(txCtx, txn, normalized, entity.EventSourceWebhook)
		if err != nil {
			// Persist the failure and SWALLOW the error inside the
			// transaction: returning err here would roll back the failed-status
			// row we just recorded. The error is returned below, outside the
			// transaction, so the handler still maps it (401 vs log + 200).
			if markErr := s.webhookLogRepo.MarkFailed(
				txCtx, logEntry.ID, &txn.ID, nil, err.Error()); markErr != nil {
				return markErr
			}
			applyErr = err
			return nil
		}
		return s.webhookLogRepo.MarkProcessed(txCtx, logEntry.ID,
			&result.TransactionID, result.RefundID)
	}); err != nil {
		return err
	}
	return applyErr
}

// locateTransaction finds the transaction matching untrusted webhook hints.
// Priority: our internal transaction id (from notes) → gateway session id
// (order id) → gateway payment id (charge id). Absent rows are normal (initiate
// still in flight, unknown ids) and yield (nil, nil) — never a 404, so the
// caller answers 200 with no persistence. Genuine DB failures propagate.
func (s *WebhookServiceImpl) locateTransaction(
	ctx context.Context,
	loc gateway.Locators,
) (*entity.PaymentTransaction, error) {
	if loc.TransactionID != "" {
		return s.findTransaction(ctx, func() (*entity.PaymentTransaction, error) {
			return s.transactionRepo.FindByTransactionID(ctx, loc.TransactionID)
		})
	}
	if loc.SessionID != "" {
		return s.findTransaction(ctx, func() (*entity.PaymentTransaction, error) {
			return s.transactionRepo.FindByGatewaySessionID(ctx, loc.SessionID)
		})
	}
	if loc.PaymentID != "" {
		return s.findTransaction(ctx, func() (*entity.PaymentTransaction, error) {
			return s.transactionRepo.FindByGatewayPaymentID(ctx, loc.PaymentID)
		})
	}
	return nil, nil
}

// findTransaction maps a miss to (nil, nil); only real failures propagate.
func (s *WebhookServiceImpl) findTransaction(
	_ context.Context,
	find func() (*entity.PaymentTransaction, error),
) (*entity.PaymentTransaction, error) {
	txn, err := find()
	if err != nil {
		if err == paymenterrors.ErrorPaymentTransactionNotFound {
			return nil, nil
		}
		return nil, err
	}
	return txn, nil
}

// configForTransaction loads the seller config for the transaction's frozen
// environment. Rows created before environments existed fall back to the
// seller's first active config for the gateway.
func (s *WebhookServiceImpl) configForTransaction(
	ctx context.Context,
	txn *entity.PaymentTransaction,
) (*entity.PaymentGatewayConfig, error) {
	if txn.GatewayID == nil {
		return nil, paymenterrors.ErrorGatewayNotConfigured
	}
	if txn.Environment != "" {
		config, err := s.configRepo.FindBySellerGatewayAndEnvironment(
			ctx, txn.SellerID, *txn.GatewayID, txn.Environment)
		if err != nil {
			return nil, err
		}
		if config == nil {
			return nil, paymenterrors.ErrorGatewayNotConfigured
		}
		return config, nil
	}
	configs, err := s.configRepo.FindAllBySellerAndGateway(ctx, txn.SellerID, *txn.GatewayID)
	if err != nil {
		return nil, err
	}
	for i := range configs {
		if configs[i].IsActive {
			return &configs[i], nil
		}
	}
	return nil, paymenterrors.ErrorGatewayNotConfigured
}

// isUniqueViolation reports whether a DB error is a unique-constraint violation.
func isUniqueViolation(err error) bool {
	// GORM/Postgres unique violation: code 23505.
	const sqlStateUniqueViolation = "23505"
	type sqlStateError interface{ SQLState() string }
	if se, ok := err.(sqlStateError); ok {
		return se.SQLState() == sqlStateUniqueViolation
	}
	return false
}
