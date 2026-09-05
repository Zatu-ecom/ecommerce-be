package service

import (
	"context"
	"fmt"
	"time"

	"ecommerce-be/common/db"
	orderService "ecommerce-be/order/service"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"
	"ecommerce-be/payment/factory"
	paymentModel "ecommerce-be/payment/model"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
	"ecommerce-be/payment/utils/constant"
)

// WebhookService processes verified inbound gateway webhooks.
type WebhookService interface {
	HandleWebhook(
		ctx context.Context,
		gatewayCode string,
		rawBody []byte,
		signature string,
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
	}
}

// HandleWebhook verifies, dedupes, and applies a gateway webhook.
func (s *WebhookServiceImpl) HandleWebhook(
	ctx context.Context,
	gatewayCode string,
	rawBody []byte,
	signature string,
	ipAddress string,
) error {
	// 1. Resolve the gateway record + adapter.
	gatewayEntity, err := s.gatewayRepo.FindByCode(ctx, gatewayCode)
	if err != nil {
		return err
	}
	adapter, err := s.gatewayFactory.GetPaymentGatewayByCode(gatewayCode)
	if err != nil {
		return err
	}

	// 2. Verify the signature using the gateway's webhook secret.
	//    Find the active config for the gateway and decrypt its credentials.
	config, err := s.findGatewayConfigForWebhook(ctx, gatewayEntity.ID)
	if err != nil {
		return err
	}
	creds, err := gateway.DecryptSensitive(config.Credentials)
	if err != nil {
		return err
	}
	valid, err := adapter.VerifyWebhook(rawBody, signature, map[string]any{
		"key_id":         creds.KeyID,
		"key_secret":     creds.KeySecret,
		"webhook_secret": creds.WebhookSecret,
		"account_id":     creds.AccountID,
	})
	if err != nil || !valid {
		return paymenterrors.ErrorInvalidWebhookSignature
	}

	// 3. Parse the event.
	event, err := adapter.ParseWebhook(rawBody)
	if err != nil {
		return err
	}
	event.GatewayCode = gatewayCode

	// 4. Record the webhook log; the unique (gateway_id, event_id) index dedupes replays.
	logEntry := &entity.PaymentWebhookLog{
		GatewayID: &gatewayEntity.ID,
		EventType: event.EventType,
		EventID:   event.EventID,
		Payload:   db.JSONMap(event.Payload),
		Status:    entity.WebhookStatusReceived,
		IPAddress: ipAddress,
	}

	// 5. Apply state transitions atomically.
	return db.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.webhookLogRepo.Create(txCtx, logEntry); err != nil {
			// Unique index violation → duplicate event; mark ignored, treat as success.
			if isUniqueViolation(err) {
				_ = s.webhookLogRepo.MarkIgnored(txCtx, logEntry.ID)
				return nil
			}
			return err
		}

		switch event.EventType {
		case constant.RAZORPAY_EVENT_PAYMENT_CAPTURED:
			if err := s.applyCaptured(txCtx, event, logEntry); err != nil {
				_ = s.webhookLogRepo.MarkFailed(txCtx, logEntry.ID, err.Error())
				return err
			}
		case constant.RAZORPAY_EVENT_PAYMENT_AUTHORIZED:
			if err := s.applyAuthorized(txCtx, event, logEntry); err != nil {
				_ = s.webhookLogRepo.MarkFailed(txCtx, logEntry.ID, err.Error())
				return err
			}
		case constant.RAZORPAY_EVENT_PAYMENT_FAILED:
			if err := s.applyFailed(txCtx, event, logEntry); err != nil {
				_ = s.webhookLogRepo.MarkFailed(txCtx, logEntry.ID, err.Error())
				return err
			}
		case constant.RAZORPAY_EVENT_REFUND_CREATED,
			constant.RAZORPAY_EVENT_REFUND_PROCESSED,
			constant.RAZORPAY_EVENT_REFUND_FAILED:
			if err := s.applyRefundEvent(txCtx, event, logEntry); err != nil {
				_ = s.webhookLogRepo.MarkFailed(txCtx, logEntry.ID, err.Error())
				return err
			}
		default:
			// Unknown event type: record as processed (ignored) without error.
			_ = s.webhookLogRepo.MarkIgnored(txCtx, logEntry.ID)
			return nil
		}

		_ = s.webhookLogRepo.MarkProcessed(txCtx, logEntry.ID,
			logEntry.TransactionID, logEntry.RefundID)
		return nil
	})
}

// applyCaptured completes the payment and confirms the order.
func (s *WebhookServiceImpl) applyCaptured(
	ctx context.Context,
	event *paymentModel.WebhookEvent,
	logEntry *entity.PaymentWebhookLog,
) error {
	txn, err := s.locateTransaction(ctx, event)
	if err != nil {
		return err
	}
	if txn == nil {
		return fmt.Errorf("razorpay: no transaction matched captured event")
	}

	now := time.Now().UTC()
	changed, err := s.transactionRepo.UpdateStatusIfCurrent(
		ctx,
		txn.ID,
		entity.TransactionStatusPending,
		entity.TransactionStatusCompleted,
		map[string]any{
			"gateway_payment_id": event.PaymentID,
			"completed_at":       now,
			"gateway_fee_cents":  event.FeeCents,
		},
	)
	if err != nil {
		return err
	}
	if !changed {
		// Already terminal (completed/failed) — idempotent no-op.
		logEntry.TransactionID = &txn.ID
		return nil
	}

	logEntry.TransactionID = &txn.ID
	if err := s.eventRepo.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:   txn.ID,
		EventType:       entity.EventTypeCaptured,
		FromStatus:      entity.TransactionStatusPending,
		ToStatus:        entity.TransactionStatusCompleted,
		GatewayEventID:  event.EventID,
		GatewayResponse: event.Payload,
		Source:          entity.EventSourceWebhook,
		ActorType:       entity.ActorTypeSystem,
	}); err != nil {
		return err
	}

	// Auto-confirm the order through the order service (cross-module interface).
	return s.orderService.ConfirmPaymentByTransactionID(ctx, txn.TransactionID)
}

// applyAuthorized records an observation event only — no order confirmation.
func (s *WebhookServiceImpl) applyAuthorized(
	ctx context.Context,
	event *paymentModel.WebhookEvent,
	logEntry *entity.PaymentWebhookLog,
) error {
	txn, err := s.locateTransaction(ctx, event)
	if err != nil {
		return err
	}
	if txn == nil {
		return nil
	}
	logEntry.TransactionID = &txn.ID
	return s.eventRepo.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:   txn.ID,
		EventType:       entity.EventTypeAuthorized,
		FromStatus:      txn.Status,
		ToStatus:        txn.Status,
		GatewayEventID:  event.EventID,
		GatewayResponse: event.Payload,
		Source:          entity.EventSourceWebhook,
		ActorType:       entity.ActorTypeSystem,
	})
}

// applyFailed fails the payment and the order.
func (s *WebhookServiceImpl) applyFailed(
	ctx context.Context,
	event *paymentModel.WebhookEvent,
	logEntry *entity.PaymentWebhookLog,
) error {
	txn, err := s.locateTransaction(ctx, event)
	if err != nil {
		return err
	}
	if txn == nil {
		return nil
	}

	changed, err := s.transactionRepo.UpdateStatusIfCurrent(
		ctx,
		txn.ID,
		entity.TransactionStatusPending,
		entity.TransactionStatusFailed,
		map[string]any{
			"failure_code":    event.FailureCode,
			"failure_message": event.FailureMessage,
		},
	)
	if err != nil {
		return err
	}
	if !changed {
		logEntry.TransactionID = &txn.ID
		return nil
	}

	logEntry.TransactionID = &txn.ID
	if err := s.eventRepo.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:   txn.ID,
		EventType:       entity.EventTypeFailed,
		FromStatus:      entity.TransactionStatusPending,
		ToStatus:        entity.TransactionStatusFailed,
		GatewayEventID:  event.EventID,
		GatewayResponse: event.Payload,
		FailureCode:     event.FailureCode,
		FailureMessage:  event.FailureMessage,
		Source:          entity.EventSourceWebhook,
		ActorType:       entity.ActorTypeSystem,
	}); err != nil {
		return err
	}

	return s.orderService.FailPaymentByTransactionID(ctx, txn.TransactionID, event.FailureMessage)
}

// findRefund locates a refund by the event's refund id (which is the gateway
// refund id in Razorpay webhooks), falling back to our internal refund id.
func (s *WebhookServiceImpl) findRefund(
	ctx context.Context,
	event *paymentModel.WebhookEvent,
) (*entity.PaymentRefund, error) {
	if event.RefundID == "" {
		return nil, paymenterrors.ErrorRefundNotFound
	}
	if refund, err := s.refundRepo.FindByGatewayRefundID(ctx, event.RefundID); err == nil {
		return refund, nil
	}
	return s.refundRepo.FindByRefundID(ctx, event.RefundID)
}

// applyRefundEvent creates/updates a refund and adjusts transaction refund state.
func (s *WebhookServiceImpl) applyRefundEvent(
	ctx context.Context,
	event *paymentModel.WebhookEvent,
	logEntry *entity.PaymentWebhookLog,
) error {
	txn, err := s.locateTransaction(ctx, event)
	if err != nil {
		return err
	}
	if txn == nil {
		return nil
	}
	logEntry.TransactionID = &txn.ID

	switch event.EventType {
	case constant.RAZORPAY_EVENT_REFUND_CREATED:
		// Refund request observed; record a pending refund if not already present.
		refund, err := s.findRefund(ctx, event)
		if err != nil {
			// NotFound → create.
			refund = &entity.PaymentRefund{
				RefundID:        event.RefundID,
				GatewayRefundID: event.RefundID,
				TransactionID:   txn.ID,
				Currency:        event.Currency,
				AmountCents:     event.AmountCents,
				Status:          entity.RefundStatusPending,
			}
			if err := s.refundRepo.Create(ctx, refund); err != nil {
				return err
			}
		}
		logEntry.RefundID = &refund.ID
	case constant.RAZORPAY_EVENT_REFUND_PROCESSED:
		refund, err := s.findRefund(ctx, event)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		// Transition from either pending (created-only) or processing (initiated via API).
		_, err = s.refundRepo.UpdateStatusIfCurrent(ctx, refund.ID,
			entity.RefundStatusPending, entity.RefundStatusCompleted,
			map[string]any{"completed_at": now})
		if err != nil {
			return err
		}
		if _, err = s.refundRepo.UpdateStatusIfCurrent(ctx, refund.ID,
			entity.RefundStatusProcessing, entity.RefundStatusCompleted,
			map[string]any{"completed_at": now}); err != nil {
			return err
		}
		logEntry.RefundID = &refund.ID
		return s.recomputeTransactionRefundStatus(ctx, txn.ID)
	case constant.RAZORPAY_EVENT_REFUND_FAILED:
		refund, err := s.findRefund(ctx, event)
		if err != nil {
			return err
		}
		_, err = s.refundRepo.UpdateStatusIfCurrent(ctx, refund.ID,
			entity.RefundStatusPending, entity.RefundStatusFailed,
			map[string]any{"failure_reason": event.FailureMessage})
		if err != nil {
			return err
		}
		_, err = s.refundRepo.UpdateStatusIfCurrent(ctx, refund.ID,
			entity.RefundStatusProcessing, entity.RefundStatusFailed,
			map[string]any{"failure_reason": event.FailureMessage})
		if err != nil {
			return err
		}
		logEntry.RefundID = &refund.ID
	}
	return nil
}

// recomputeTransactionRefundStatus sets refunded/partially_refunded based on refund totals.
func (s *WebhookServiceImpl) recomputeTransactionRefundStatus(
	ctx context.Context,
	transactionID uint,
) error {
	txn, err := s.transactionRepo.FindByID(ctx, transactionID)
	if err != nil {
		return err
	}

	var refunded int64
	if err := s.refundRepo.SumByTransactionAndStatuses(ctx, transactionID, []entity.RefundStatus{
		entity.RefundStatusCompleted,
	}, &refunded); err != nil {
		return err
	}

	target := entity.TransactionStatusPartiallyRefunded
	if refunded >= txn.AmountCents {
		target = entity.TransactionStatusRefunded
	}

	_, err = s.transactionRepo.UpdateStatusIfCurrent(ctx, transactionID,
		entity.TransactionStatusCompleted, target, map[string]any{})
	return err
}

// locateTransaction finds the transaction matching a webhook event.
// Priority: our internal transaction id (from notes) → gateway session id (order id)
// → gateway payment id (charge id).
func (s *WebhookServiceImpl) locateTransaction(
	ctx context.Context,
	event *paymentModel.WebhookEvent,
) (*entity.PaymentTransaction, error) {
	if event.TransactionID != "" {
		return s.transactionRepo.FindByTransactionID(ctx, event.TransactionID)
	}
	if event.SessionID != "" {
		return s.transactionRepo.FindByGatewaySessionID(ctx, event.SessionID)
	}
	if event.PaymentID != "" {
		return s.transactionRepo.FindByGatewayPaymentID(ctx, event.PaymentID)
	}
	return nil, nil
}

// findGatewayConfigForWebhook returns a config carrying the webhook secret.
// Configs are per-seller; we return the first active config for the gateway
// (deterministic order). A production implementation would resolve by the seller
// hinted in the payload or an inbound URL token.
func (s *WebhookServiceImpl) findGatewayConfigForWebhook(
	ctx context.Context,
	gatewayID uint,
) (*entity.PaymentGatewayConfig, error) {
	configs, err := s.configRepo.FindAllForGateway(ctx, gatewayID)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, paymenterrors.ErrorGatewayNotConfigured
	}
	return &configs[0], nil
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
