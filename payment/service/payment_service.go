package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	commonConstants "ecommerce-be/common/constants"
	"ecommerce-be/common/db"
	"ecommerce-be/common/log"
	orderService "ecommerce-be/order/service"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"
	"ecommerce-be/payment/factory"
	paymentModel "ecommerce-be/payment/model"
	"ecommerce-be/payment/repository"
	gateway "ecommerce-be/payment/service/payment_gateway"
	"ecommerce-be/payment/utils/constant"
	userService "ecommerce-be/user/service"
)

// PaymentService defines the business workflow for payments.
type PaymentService interface {
	InitiatePayment(
		ctx context.Context,
		userID, sellerID uint,
		req paymentModel.InitiatePaymentRequest,
	) (*paymentModel.InitiatePaymentResponse, error)
	GetPaymentStatus(
		ctx context.Context,
		userID uint,
		role string,
		transactionID string,
	) (*paymentModel.PaymentTransactionResponse, error)
	ListSellerTransactions(
		ctx context.Context,
		sellerID uint,
		page, pageSize int,
		status string,
	) ([]paymentModel.PaymentTransactionResponse, int64, error)
	InitiateRefund(
		ctx context.Context,
		sellerID uint,
		req paymentModel.RefundRequest,
	) (*paymentModel.RefundResponse, error)
	ListWebhookLogs(
		ctx context.Context,
		sellerID uint,
		page, pageSize int,
		status, eventType string,
	) ([]paymentModel.WebhookLogResponse, int64, error)
}

// PaymentServiceImpl implements PaymentService.
type PaymentServiceImpl struct {
	transactionRepo   repository.PaymentTransactionRepository
	eventRepo         repository.PaymentTransactionEventRepository
	refundRepo        repository.PaymentRefundRepository
	webhookLogRepo    repository.PaymentWebhookLogRepository
	gatewayRepo       repository.PaymentGatewayRepository
	gatewayConfigRepo repository.PaymentGatewayConfigRepository
	gatewayFactory    *factory.PaymentGatewayFactory
	orderService      orderService.OrderService
	userService       userService.UserService
}

// NewPaymentService creates the payment service.
func NewPaymentService(
	transactionRepo repository.PaymentTransactionRepository,
	eventRepo repository.PaymentTransactionEventRepository,
	refundRepo repository.PaymentRefundRepository,
	webhookLogRepo repository.PaymentWebhookLogRepository,
	gatewayRepo repository.PaymentGatewayRepository,
	gatewayConfigRepo repository.PaymentGatewayConfigRepository,
	gatewayFactory *factory.PaymentGatewayFactory,
	orderService orderService.OrderService,
	userService userService.UserService,
) PaymentService {
	return &PaymentServiceImpl{
		transactionRepo:   transactionRepo,
		eventRepo:         eventRepo,
		refundRepo:        refundRepo,
		webhookLogRepo:    webhookLogRepo,
		gatewayRepo:       gatewayRepo,
		gatewayConfigRepo: gatewayConfigRepo,
		gatewayFactory:    gatewayFactory,
		orderService:      orderService,
		userService:       userService,
	}
}

// InitiatePayment starts a payment for a pending order and returns the checkout payload.
func (s *PaymentServiceImpl) InitiatePayment(
	ctx context.Context,
	userID, sellerID uint,
	req paymentModel.InitiatePaymentRequest,
) (*paymentModel.InitiatePaymentResponse, error) {
	// 1. Load the order and verify ownership/payable state.
	order, err := s.orderService.GetOrderByID(ctx, userID, "customer", req.OrderID)
	if err != nil {
		return nil, err
	}
	if order == nil || string(order.Status) != constant.TRANSACTION_STATUS_PENDING {
		return nil, paymenterrors.ErrorPaymentOrderNotFound
	}

	// 1b. Reject duplicate payment for the same order (only one active payment).
	existing, err := s.transactionRepo.FindByReference(ctx, entity.ReferenceTypeOrder, req.OrderID)
	if err != nil {
		return nil, err
	}
	for _, txn := range existing {
		if txn.Status == entity.TransactionStatusPending || txn.Status == entity.TransactionStatusCompleted {
			return nil, paymenterrors.ErrorDuplicatePayment
		}
	}

	// 2. Resolve the seller's base currency.
	ccy, err := s.userService.GetSellerDefaultCurrency(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("resolve seller currency: %w", err)
	}
	currency := ccy.Code
	if currency == "" {
		return nil, paymenterrors.ErrorGatewayUnsupportedCurrency
	}

	// 3. Select the gateway: active configs for the store's payments
	// environment, gateway active, seller geo supported, highest priority
	// wins. Never hardcoded, never cross-environment fallback.
	env, err := s.storePaymentsEnvironment(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	selected, adapter, decrypted, err := s.selectGateway(ctx, sellerID, env)
	if err != nil {
		return nil, err
	}

	// 4. Optional paymentMethodType filter against the SELECTED gateway.
	if method := strings.TrimSpace(string(req.PaymentMethodType)); method != "" &&
		!supportedPaymentMethod(selected.Gateway, method) {
		return nil, paymenterrors.ErrorGatewayValidation.WithMessagef(
			"unsupported payment method type %q for gateway %q", method, selected.Gateway.Code)
	}

	// 5. Create the transaction (pending) + initiated event atomically.
	transactionID := s.generateTransactionID()
	amountCents := order.Total.AmountCents
	txn := &entity.PaymentTransaction{
		TransactionID:     transactionID,
		UserID:            userID,
		SellerID:          sellerID,
		GatewayID:         &selected.Gateway.ID,
		ReferenceType:     entity.ReferenceTypeOrder,
		ReferenceID:       req.OrderID,
		Currency:          currency,
		AmountCents:       amountCents,
		Status:            entity.TransactionStatusPending,
		PaymentMethodType: req.PaymentMethodType,
		Environment:       env,
	}

	err = db.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.transactionRepo.Create(txCtx, txn); err != nil {
			return err
		}
		return s.eventRepo.Create(txCtx, &entity.PaymentTransactionEvent{
			TransactionID: txn.ID,
			EventType:     entity.EventTypeInitiated,
			ToStatus:      entity.TransactionStatusPending,
			Source:        entity.EventSourceAPI,
			ActorID:       &userID,
			ActorType:     entity.ActorTypeCustomer,
		})
	})
	if err != nil {
		return nil, err
	}

	// 6. Call the gateway to create a checkout session.
	output, err := adapter.InitiatePayment(ctx, paymentModel.InitiatePaymentInput{
		TransactionID: transactionID,
		AmountCents:   amountCents,
		Currency:      currency,
		PaymentMethod: string(req.PaymentMethodType),
		CustomerID:    userID,
		SellerID:      sellerID,
		ReferenceType: string(entity.ReferenceTypeOrder),
		ReferenceID:   req.OrderID,
		Credentials:   decrypted,
	})
	if err != nil {
		// Provider refused the session: fail THIS transaction row (with the
		// provider message) so the customer can retry. The order stays pending
		// and failed rows never block a new initiate (see duplicate check).
		log.ErrorWithContext(ctx, "initiatePayment: gateway call failed", err)
		s.markInitiateFailed(ctx, txn, userID, err.Error())
		return nil, err
	}

	// 7. Persist gateway_session_id + event, then attach the transaction id to the order.
	err = db.WithTransaction(ctx, func(txCtx context.Context) error {
		if _, err := s.transactionRepo.UpdateStatusIfCurrent(txCtx, txn.ID,
			entity.TransactionStatusPending, entity.TransactionStatusPending, map[string]any{
				"gateway_session_id": output.GatewaySessionID,
			}); err != nil {
			return err
		}
		return s.eventRepo.Create(txCtx, &entity.PaymentTransactionEvent{
			TransactionID:   txn.ID,
			EventType:       entity.EventTypeGatewaySessionCreated,
			FromStatus:      entity.TransactionStatusPending,
			ToStatus:        entity.TransactionStatusPending,
			GatewayResponse: output.GatewayResponse,
			Source:          entity.EventSourceAPI,
			ActorID:         &userID,
			ActorType:       entity.ActorTypeCustomer,
		})
	})
	if err != nil {
		return nil, err
	}

	if err := s.orderService.AttachTransactionID(ctx, req.OrderID, sellerID, transactionID); err != nil {
		return nil, err
	}

	// 8. Return the adapter's checkout payload (generic contract).
	return &paymentModel.InitiatePaymentResponse{
		TransactionID:    transactionID,
		GatewaySessionID: output.GatewaySessionID,
		AmountCents:      amountCents,
		Currency:         currency,
		Status:           string(entity.TransactionStatusPending),
		Checkout:         output.Checkout,
	}, nil
}

// GetPaymentStatus returns a single payment's current state (owner or seller scoped).
func (s *PaymentServiceImpl) GetPaymentStatus(
	ctx context.Context,
	userID uint,
	role string,
	transactionID string,
) (*paymentModel.PaymentTransactionResponse, error) {
	txn, err := s.transactionRepo.FindByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	// Role names are canonical uppercase (common/constants ROLE_NAME).
	// Unknown roles are denied: fail closed, no cross-tenant leak.
	switch role {
	case commonConstants.CUSTOMER_ROLE_NAME:
		if txn.UserID != userID {
			return nil, paymenterrors.ErrorPaymentTransactionNotFound
		}
	case commonConstants.SELLER_ROLE_NAME:
		if txn.SellerID != userID {
			return nil, paymenterrors.ErrorPaymentTransactionNotFound
		}
	default:
		return nil, paymenterrors.ErrorPaymentTransactionNotFound
	}
	return s.toDetailResponse(ctx, txn)
}

// ListSellerTransactions lists a seller's payments (paginated, optional
// server-side status filter). List rows never carry events.
func (s *PaymentServiceImpl) ListSellerTransactions(
	ctx context.Context,
	sellerID uint,
	page, pageSize int,
	status string,
) ([]paymentModel.PaymentTransactionResponse, int64, error) {
	txnStatus, err := parseTransactionStatusFilter(status)
	if err != nil {
		return nil, 0, err
	}
	txns, total, err := s.transactionRepo.FindBySellerID(ctx, sellerID, page, pageSize, txnStatus)
	if err != nil {
		return nil, 0, err
	}
	responses := make([]paymentModel.PaymentTransactionResponse, 0, len(txns))
	for i := range txns {
		responses = append(responses, *s.toResponse(&txns[i]))
	}
	return responses, total, nil
}

// ListWebhookLogs lists a seller's webhook deliveries (paginated, newest
// first). Only deliveries linked to the seller's transactions are visible.
func (s *PaymentServiceImpl) ListWebhookLogs(
	ctx context.Context,
	sellerID uint,
	page, pageSize int,
	status, eventType string,
) ([]paymentModel.WebhookLogResponse, int64, error) {
	if status != "" {
		switch entity.WebhookStatus(status) {
		case entity.WebhookStatusReceived,
			entity.WebhookStatusProcessed,
			entity.WebhookStatusFailed,
			entity.WebhookStatusIgnored:
		default:
			return nil, 0, paymenterrors.ErrorGatewayValidation.WithMessagef("invalid log status filter %q", status)
		}
	}
	logs, total, err := s.webhookLogRepo.FindBySellerID(ctx, sellerID, page, pageSize, status, eventType)
	if err != nil {
		return nil, 0, err
	}
	responses := make([]paymentModel.WebhookLogResponse, 0, len(logs))
	for i := range logs {
		l := &logs[i]
		responses = append(responses, paymentModel.WebhookLogResponse{
			ID:            l.ID,
			EventType:     l.EventType,
			EventID:       l.EventID,
			Status:        l.Status,
			ErrorMessage:  l.ErrorMessage,
			TransactionID: l.PublicTransactionID,
			RefundID:      l.RefundID,
			CreatedAt:     l.CreatedAt,
		})
	}
	return responses, total, nil
}

// parseTransactionStatusFilter validates the optional ?status= enum.
// Empty means unfiltered; anything else must be a transaction status.
func parseTransactionStatusFilter(status string) (entity.TransactionStatus, error) {
	switch entity.TransactionStatus(status) {
	case "":
		return "", nil
	case entity.TransactionStatusPending,
		entity.TransactionStatusCompleted,
		entity.TransactionStatusFailed,
		entity.TransactionStatusRefunded,
		entity.TransactionStatusPartiallyRefunded:
		return entity.TransactionStatus(status), nil
	default:
		return "", paymenterrors.ErrorGatewayValidation.WithMessagef("invalid status filter %q", status)
	}
}

// InitiateRefund issues a full or partial refund for a completed payment.
func (s *PaymentServiceImpl) InitiateRefund(
	ctx context.Context,
	sellerID uint,
	req paymentModel.RefundRequest,
) (*paymentModel.RefundResponse, error) {
	// 1. Load the transaction and verify seller ownership + refundable status.
	// Refunds are allowed on completed and partially_refunded rows so sellers
	// can issue successive partial refunds until nothing remains.
	txn, err := s.transactionRepo.FindByTransactionID(ctx, req.TransactionID)
	if err != nil {
		return nil, err
	}
	if txn.SellerID != sellerID {
		return nil, paymenterrors.ErrorPaymentTransactionNotFound
	}
	if txn.Status != entity.TransactionStatusCompleted &&
		txn.Status != entity.TransactionStatusPartiallyRefunded {
		return nil, paymenterrors.ErrorRefundNotAllowed
	}

	// 2. Verify the refund amount does not exceed the remaining refundable amount.
	remaining, err := s.remainingRefundableCents(ctx, txn)
	if err != nil {
		return nil, err
	}
	if req.AmountCents <= 0 || req.AmountCents > remaining {
		return nil, paymenterrors.ErrorRefundNotAllowed
	}

	// 3. Resolve the transaction's gateway (never a hardcoded provider code)
	// + its config for the frozen environment + adapter.
	if txn.GatewayID == nil {
		return nil, paymenterrors.ErrorGatewayNotConfigured
	}
	adapter, err := s.gatewayFactory.GetPaymentGateway(ctx, *txn.GatewayID)
	if err != nil {
		return nil, err
	}
	env := txn.Environment
	if env == "" {
		if env, err = s.storePaymentsEnvironment(ctx, sellerID); err != nil {
			return nil, err
		}
	}
	config, err := s.gatewayConfigRepo.FindBySellerGatewayAndEnvironment(ctx, sellerID, *txn.GatewayID, env)
	if err != nil {
		return nil, err
	}
	if config == nil || !config.IsActive {
		return nil, paymenterrors.ErrorGatewayNotConfigured
	}
	decrypted, err := adapter.Decrypt(config.Credentials)
	if err != nil {
		return nil, err
	}

	// 4. Create the refund record (processing) and call the gateway.
	refundID := s.generateRefundID()
	refund := &entity.PaymentRefund{
		RefundID:        refundID,
		TransactionID:   txn.ID,
		Currency:        txn.Currency,
		AmountCents:     req.AmountCents,
		Status:          entity.RefundStatusProcessing,
		Reason:          req.Reason,
		Notes:           req.Notes,
		InitiatedBy:     &sellerID,
		InitiatedByType: entity.InitiatedBySeller,
	}

	output, err := adapter.Refund(ctx, paymentModel.RefundInput{
		GatewayPaymentID: txn.GatewayPaymentID,
		RefundID:         refundID,
		AmountCents:      req.AmountCents,
		Currency:         txn.Currency,
		Notes:            req.Notes,
		Credentials:      decrypted,
	})
	if err != nil {
		return nil, err
	}
	refund.GatewayRefundID = output.GatewayRefundID

	// 5. Persist the refund + a refund_initiated event atomically.
	err = db.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.refundRepo.Create(txCtx, refund); err != nil {
			return err
		}
		return s.eventRepo.Create(txCtx, &entity.PaymentTransactionEvent{
			TransactionID:   txn.ID,
			EventType:       entity.EventTypeRefundInitiated,
			FromStatus:      txn.Status,
			ToStatus:        txn.Status,
			GatewayResponse: output.GatewayResponse,
			Source:          entity.EventSourceAPI,
			ActorID:         &sellerID,
			ActorType:       entity.ActorTypeSeller,
		})
	})
	if err != nil {
		return nil, err
	}

	return &paymentModel.RefundResponse{
		RefundID:        refund.RefundID,
		TransactionID:   refund.TransactionID,
		GatewayRefundID: refund.GatewayRefundID,
		AmountCents:     refund.AmountCents,
		Currency:        refund.Currency,
		Status:          refund.Status,
		Reason:          refund.Reason,
		Notes:           refund.Notes,
		CreatedAt:       refund.CreatedAt,
	}, nil
}

// remainingRefundableCents computes the amount still refundable for a transaction.
func (s *PaymentServiceImpl) remainingRefundableCents(
	ctx context.Context,
	txn *entity.PaymentTransaction,
) (int64, error) {
	// Query refunds for the transaction and sum non-failed refunds.
	// For simplicity we rely on the transaction's amount minus refunded totals.
	var refunded int64
	err := s.refundRepo.SumByTransactionAndStatuses(ctx, txn.ID, []entity.RefundStatus{
		entity.RefundStatusPending,
		entity.RefundStatusProcessing,
		entity.RefundStatusCompleted,
	}, &refunded)
	if err != nil {
		return 0, err
	}
	return txn.AmountCents - refunded, nil
}

// generateRefundID builds a unique internal refund id.
func (s *PaymentServiceImpl) generateRefundID() string {
	return fmt.Sprintf("RFD_%d", time.Now().UnixNano())
}

// ─── Internal helpers ──────────────────────────────────────────────────────────

// storePaymentsEnvironment resolves the seller's checkout mode, defaulting to
// sandbox. Production credentials are never touched unless the store
// explicitly runs in production mode.
func (s *PaymentServiceImpl) storePaymentsEnvironment(
	ctx context.Context,
	sellerID uint,
) (entity.GatewayEnvironment, error) {
	mode, err := s.userService.GetSellerPaymentsEnvironment(ctx, sellerID)
	if err != nil {
		return "", err
	}
	env := entity.GatewayEnvironment(mode)
	if env != entity.EnvironmentSandbox && env != entity.EnvironmentProduction {
		env = entity.EnvironmentSandbox
	}
	return env, nil
}

// selectedGateway carries the winning initiate-time config with its catalog row.
type selectedGateway struct {
	Config  *entity.PaymentGatewayConfig
	Gateway *entity.PaymentGateway
}

// selectGateway picks the checkout provider for a seller + environment:
// active configs in priority order, catalog-active gateway, seller geo
// (business country + base currency) supported via FK join tables. The first
// full match wins with its decrypted credentials. No environment fallback.
//
// Errors distinguish: no usable config → GATEWAY_NOT_CONFIGURED; configs
// exist but geo fails → GATEWAY_UNSUPPORTED_COUNTRY / CURRENCY.
func (s *PaymentServiceImpl) selectGateway(
	ctx context.Context,
	sellerID uint,
	env entity.GatewayEnvironment,
) (*selectedGateway, gateway.PaymentGateway, map[string]any, error) {
	configs, err := s.gatewayConfigRepo.FindActiveBySellerAndEnvironment(ctx, sellerID, env)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(configs) == 0 {
		return nil, nil, nil, paymenterrors.ErrorGatewayNotConfigured
	}

	countryID, err := s.userService.GetSellerBusinessCountryID(ctx, sellerID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("resolve seller country: %w", err)
	}
	ccy, err := s.userService.GetSellerDefaultCurrency(ctx, sellerID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("resolve seller currency: %w", err)
	}

	var countryFailed, currencyFailed bool
	for i := range configs {
		cfg := &configs[i]
		g := cfg.Gateway
		if g == nil || !g.IsActive {
			continue
		}
		ok, err := s.gatewayRepo.SupportsCountry(ctx, g.ID, countryID)
		if err != nil {
			return nil, nil, nil, err
		}
		if !ok {
			countryFailed = true
			continue
		}
		ok, err = s.gatewayRepo.SupportsCurrency(ctx, g.ID, ccy.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		if !ok {
			currencyFailed = true
			continue
		}
		adapter, err := s.gatewayFactory.GetPaymentGatewayByCode(g.Code)
		if err != nil {
			return nil, nil, nil, err
		}
		decrypted, err := adapter.Decrypt(cfg.Credentials)
		if err != nil {
			return nil, nil, nil, err
		}
		return &selectedGateway{Config: cfg, Gateway: g}, adapter, decrypted, nil
	}

	if countryFailed {
		return nil, nil, nil, paymenterrors.ErrorGatewayUnsupportedCountry
	}
	if currencyFailed {
		return nil, nil, nil, paymenterrors.ErrorGatewayUnsupportedCurrency
	}
	return nil, nil, nil, paymenterrors.ErrorGatewayNotConfigured
}

// supportedPaymentMethod reports whether an optional client-requested method
// type is in the gateway's declared list. Empty request skips the check.
func supportedPaymentMethod(g *entity.PaymentGateway, method string) bool {
	for _, m := range g.SupportedPaymentMethods {
		if m == method {
			return true
		}
	}
	return false
}

// markInitiateFailed transitions a provider-refused transaction to failed with
// the provider message (best effort: the initiate already returns the error).
func (s *PaymentServiceImpl) markInitiateFailed(
	ctx context.Context,
	txn *entity.PaymentTransaction,
	userID uint,
	reason string,
) {
	_, err := s.transactionRepo.UpdateStatusIfCurrent(ctx, txn.ID,
		entity.TransactionStatusPending, entity.TransactionStatusFailed,
		map[string]any{"failure_message": reason})
	if err != nil {
		log.ErrorWithContext(ctx, "initiatePayment: mark failed failed", err)
		return
	}
	if err := s.eventRepo.Create(ctx, &entity.PaymentTransactionEvent{
		TransactionID:  txn.ID,
		EventType:      entity.EventTypeFailed,
		FromStatus:     entity.TransactionStatusPending,
		ToStatus:       entity.TransactionStatusFailed,
		FailureMessage: reason,
		Source:         entity.EventSourceAPI,
		ActorID:        &userID,
		ActorType:      entity.ActorTypeCustomer,
	}); err != nil {
		log.ErrorWithContext(ctx, "initiatePayment: mark failed event failed", err)
	}
}

func (s *PaymentServiceImpl) generateTransactionID() string {
	return fmt.Sprintf("TXN_%d", time.Now().UnixNano())
}

func (s *PaymentServiceImpl) toResponse(txn *entity.PaymentTransaction) *paymentModel.PaymentTransactionResponse {
	return &paymentModel.PaymentTransactionResponse{
		ID:                txn.ID,
		TransactionID:     txn.TransactionID,
		ReferenceType:     txn.ReferenceType,
		ReferenceID:       txn.ReferenceID,
		GatewayID:         txn.GatewayID,
		GatewaySessionID:  txn.GatewaySessionID,
		GatewayPaymentID:  txn.GatewayPaymentID,
		Currency:          txn.Currency,
		AmountCents:       txn.AmountCents,
		GatewayFeeCents:   txn.GatewayFeeCents,
		Status:            txn.Status,
		FailureCode:       txn.FailureCode,
		FailureMessage:    txn.FailureMessage,
		PaymentMethodType: txn.PaymentMethodType,
		CompletedAt:       txn.CompletedAt,
		CreatedAt:         txn.CreatedAt,
	}
}

// toDetailResponse enriches the base view with the frozen environment, the
// remaining refundable amount, and the full event history. Detail-only: the
// list mapper (toResponse) never loads events.
func (s *PaymentServiceImpl) toDetailResponse(
	ctx context.Context,
	txn *entity.PaymentTransaction,
) (*paymentModel.PaymentTransactionResponse, error) {
	resp := s.toResponse(txn)
	resp.Environment = string(txn.Environment)

	remaining, err := s.remainingRefundableCents(ctx, txn)
	if err != nil {
		return nil, err
	}
	resp.RefundableAmountCents = &remaining

	events, err := s.eventRepo.ListByTransactionID(ctx, txn.ID)
	if err != nil {
		return nil, err
	}
	resp.Events = make([]paymentModel.PaymentTransactionEventResponse, 0, len(events))
	for i := range events {
		e := &events[i]
		resp.Events = append(resp.Events, paymentModel.PaymentTransactionEventResponse{
			EventType:      string(e.EventType),
			FromStatus:     string(e.FromStatus),
			ToStatus:       string(e.ToStatus),
			GatewayEventID: e.GatewayEventID,
			FailureCode:    e.FailureCode,
			FailureMessage: e.FailureMessage,
			Source:         string(e.Source),
			CreatedAt:      e.CreatedAt,
		})
	}
	return resp, nil
}
