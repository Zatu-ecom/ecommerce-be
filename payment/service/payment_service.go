package service

import (
	"context"
	"fmt"
	"time"

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
	) ([]paymentModel.PaymentTransactionResponse, int64, error)
	InitiateRefund(
		ctx context.Context,
		sellerID uint,
		req paymentModel.RefundRequest,
	) (*paymentModel.RefundResponse, error)
}

// PaymentServiceImpl implements PaymentService.
type PaymentServiceImpl struct {
	transactionRepo   repository.PaymentTransactionRepository
	eventRepo         repository.PaymentTransactionEventRepository
	refundRepo        repository.PaymentRefundRepository
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

	// 3. Resolve the razorpay gateway record + the seller's active config.
	gatewayEntity, err := s.gatewayRepo.FindByCode(ctx, constant.GATEWAY_CODE_RAZORPAY)
	if err != nil {
		return nil, err
	}
	config, err := s.gatewayConfigRepo.FindBySellerAndGateway(ctx, sellerID, gatewayEntity.ID)
	if err != nil {
		return nil, err
	}
	if config == nil || !config.IsActive {
		return nil, paymenterrors.ErrorGatewayNotConfigured
	}
	creds, err := gateway.DecryptSensitive(config.Credentials)
	if err != nil {
		return nil, err
	}

	// 4. Validate currency support and resolve the adapter.
	if !s.gatewaySupportsCurrency(gatewayEntity, currency) {
		return nil, paymenterrors.ErrorGatewayUnsupportedCurrency
	}
	adapter, err := s.gatewayFactory.GetPaymentGatewayByCode(gatewayEntity.Code)
	if err != nil {
		return nil, err
	}

	// 5. Create the transaction (pending) + initiated event atomically.
	transactionID := s.generateTransactionID()
	amountCents := order.Total.AmountCents
	txn := &entity.PaymentTransaction{
		TransactionID:     transactionID,
		UserID:            userID,
		SellerID:          sellerID,
		GatewayID:         &gatewayEntity.ID,
		ReferenceType:     entity.ReferenceTypeOrder,
		ReferenceID:       req.OrderID,
		Currency:          currency,
		AmountCents:       amountCents,
		Status:            entity.TransactionStatusPending,
		PaymentMethodType: req.PaymentMethodType,
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
		Credentials: map[string]any{
			"key_id":         creds.KeyID,
			"key_secret":     creds.KeySecret,
			"webhook_secret": creds.WebhookSecret,
			"account_id":     creds.AccountID,
		},
	})
	if err != nil {
		log.ErrorWithContext(ctx, "initiatePayment: gateway call failed", err)
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

	// 8. Return the checkout payload.
	return &paymentModel.InitiatePaymentResponse{
		TransactionID:    transactionID,
		GatewaySessionID: output.GatewaySessionID,
		KeyID:            creds.KeyID,
		AmountCents:      amountCents,
		Currency:         currency,
		Status:           string(entity.TransactionStatusPending),
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
	if role == "customer" && txn.UserID != userID {
		return nil, paymenterrors.ErrorPaymentTransactionNotFound
	}
	if role == "seller" && txn.SellerID != userID {
		return nil, paymenterrors.ErrorPaymentTransactionNotFound
	}
	return s.toResponse(txn), nil
}

// ListSellerTransactions lists a seller's payments (paginated).
func (s *PaymentServiceImpl) ListSellerTransactions(
	ctx context.Context,
	sellerID uint,
	page, pageSize int,
) ([]paymentModel.PaymentTransactionResponse, int64, error) {
	txns, total, err := s.transactionRepo.FindBySellerID(ctx, sellerID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	responses := make([]paymentModel.PaymentTransactionResponse, 0, len(txns))
	for i := range txns {
		responses = append(responses, *s.toResponse(&txns[i]))
	}
	return responses, total, nil
}

// InitiateRefund issues a full or partial refund for a completed payment.
func (s *PaymentServiceImpl) InitiateRefund(
	ctx context.Context,
	sellerID uint,
	req paymentModel.RefundRequest,
) (*paymentModel.RefundResponse, error) {
	// 1. Load the transaction and verify seller ownership + completed status.
	txn, err := s.transactionRepo.FindByTransactionID(ctx, req.TransactionID)
	if err != nil {
		return nil, err
	}
	if txn.SellerID != sellerID {
		return nil, paymenterrors.ErrorPaymentTransactionNotFound
	}
	if txn.Status != entity.TransactionStatusCompleted {
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

	// 3. Resolve the gateway config + adapter.
	gatewayEntity, err := s.gatewayRepo.FindByCode(ctx, constant.GATEWAY_CODE_RAZORPAY)
	if err != nil {
		return nil, err
	}
	config, err := s.gatewayConfigRepo.FindBySellerAndGateway(ctx, sellerID, gatewayEntity.ID)
	if err != nil {
		return nil, err
	}
	if config == nil || !config.IsActive {
		return nil, paymenterrors.ErrorGatewayNotConfigured
	}
	creds, err := gateway.DecryptSensitive(config.Credentials)
	if err != nil {
		return nil, err
	}
	adapter, err := s.gatewayFactory.GetPaymentGatewayByCode(gatewayEntity.Code)
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

	output, err := adapter.Refund(ctx, gateway.REFUND_TYPE_PARTIAL, paymentModel.RefundInput{
		GatewayPaymentID: txn.GatewayPaymentID,
		RefundID:         refundID,
		AmountCents:      req.AmountCents,
		Currency:         txn.Currency,
		Notes:            req.Notes,
		Credentials: map[string]any{
			"key_id":         creds.KeyID,
			"key_secret":     creds.KeySecret,
			"webhook_secret": creds.WebhookSecret,
			"account_id":     creds.AccountID,
		},
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
			FromStatus:      entity.TransactionStatusCompleted,
			ToStatus:        entity.TransactionStatusCompleted,
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

func (s *PaymentServiceImpl) gatewaySupportsCurrency(g *entity.PaymentGateway, currency string) bool {
	for _, c := range g.SupportedCurrencies {
		if c == currency {
			return true
		}
	}
	return false
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
