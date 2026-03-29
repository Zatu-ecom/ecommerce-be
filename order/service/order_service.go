package service

import (
	"context"
	"strings"
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/db"
	inventoryEntity "ecommerce-be/inventory/entity"
	inventoryModel "ecommerce-be/inventory/model"
	inventoryService "ecommerce-be/inventory/service"
	"ecommerce-be/order/entity"
	orderError "ecommerce-be/order/error"
	"ecommerce-be/order/factory"
	"ecommerce-be/order/mapper"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"
	orderUtils "ecommerce-be/order/utils"
	userModel "ecommerce-be/user/model"
	userRepository "ecommerce-be/user/repository"
	userService "ecommerce-be/user/service"
	userConstant "ecommerce-be/user/utils/constant"
)

const reservationExpiresInMinutes = 5

// OrderService defines the business workflow for order lifecycle.
type OrderService interface {
	CreateOrder(
		ctx context.Context,
		userID, sellerID uint,
		req model.CreateOrderRequest,
	) (*model.OrderResponse, error)
	GetOrderByID(
		ctx context.Context,
		userID uint,
		role string,
		orderID uint,
	) (*model.OrderResponse, error)
	ListOrders(
		ctx context.Context,
		userID uint,
		role string,
		filters model.ListOrdersRequest,
	) (*model.PaginatedOrdersResponse, error)
	UpdateOrderStatus(
		ctx context.Context,
		sellerID uint,
		orderID uint,
		req model.UpdateOrderStatusRequest,
	) (*model.UpdateStatusResponse, error)
	CancelOrder(
		ctx context.Context,
		userID uint,
		orderID uint,
		req model.CancelOrderRequest,
	) (*model.UpdateStatusResponse, error)
}

type OrderServiceImpl struct {
	cartSvc             CartService
	orderRepo           repository.OrderRepository
	orderHistoryRepo    repository.OrderHistoryRepository
	inventoryReserveSvc inventoryService.InventoryReservationService
	addressSvc          userService.AddressService
	userRepo            userRepository.UserRepository
}

// createOrderContext carries validated inputs and locked resources required to create an order.
type createOrderContext struct {
	fulfillmentType entity.FulfillmentType
	orderStatus     entity.OrderStatus
	cartSnapshot    *model.CartResponse
	lockedCart      *entity.Cart
	shippingAddress *userModel.AddressResponse
	billingAddress  *userModel.AddressResponse
}

func NewOrderService(
	cartSvc CartService,
	orderRepo repository.OrderRepository,
	orderHistoryRepo repository.OrderHistoryRepository,
	inventoryReserveSvc inventoryService.InventoryReservationService,
	addressSvc userService.AddressService,
	userRepo userRepository.UserRepository,
) OrderService {
	return &OrderServiceImpl{
		cartSvc:             cartSvc,
		orderRepo:           orderRepo,
		orderHistoryRepo:    orderHistoryRepo,
		inventoryReserveSvc: inventoryReserveSvc,
		addressSvc:          addressSvc,
		userRepo:            userRepo,
	}
}

// CreateOrder executes checkout->order conversion.
// Steps:
// 1. Validate fulfillment and status preconditions.
// 2. Read enriched cart snapshot from cart service (items + promo breakdown).
// 3. Lock cart (active -> checkout).
// 4. In one transaction: create order graph, reserve inventory, convert cart, create fresh active cart.
// 5. If order status is confirmed, update reservation status to confirmed.
// 6. On failure, unlock cart back to active.
func (s *OrderServiceImpl) CreateOrder(
	ctx context.Context,
	userID, sellerID uint,
	req model.CreateOrderRequest,
) (*model.OrderResponse, error) {
	// Step 1-4: validate request + cart context and acquire checkout lock.
	createCtx, err := s.prepareCreateOrder(ctx, userID, sellerID, req)
	if err != nil {
		return nil, err
	}

	converted := false
	defer func() {
		if !converted {
			_ = s.cartSvc.UnlockCheckoutCart(context.Background(), createCtx.lockedCart.ID)
		}
	}()

	// Step 5: run full order conversion transaction (all-or-nothing).
	resp, err := s.executeCreateOrderTransaction(ctx, userID, sellerID, req, createCtx)
	if err != nil {
		return nil, err
	}

	converted = true
	return resp, nil
}

// prepareCreateOrder validates request inputs and acquires a cart checkout lock.
func (s *OrderServiceImpl) prepareCreateOrder(
	ctx context.Context,
	userID, sellerID uint,
	req model.CreateOrderRequest,
) (*createOrderContext, error) {
	fulfillmentType, err := normalizeFulfillmentType(req.FulfillmentType)
	if err != nil {
		return nil, err
	}

	orderStatus, err := normalizeCreateOrderStatus(req.Status)
	if err != nil {
		return nil, err
	}

	cartSnapshot, err := s.cartSvc.GetUserCart(ctx, userID, sellerID)
	if err != nil {
		return nil, err
	}
	if len(cartSnapshot.Items) == 0 {
		return nil, orderError.ErrCartEmpty
	}

	lockedCart, err := s.cartSvc.LockActiveCartForCheckout(ctx, userID)
	if err != nil {
		return nil, err
	}

	shippingAddress, billingAddress, err := s.loadAndValidateAddresses(
		ctx,
		userID,
		req.ShippingAddressID,
		req.BillingAddressID,
	)
	if err != nil {
		_ = s.cartSvc.UnlockCheckoutCart(context.Background(), lockedCart.ID)
		return nil, err
	}

	return &createOrderContext{
		fulfillmentType: fulfillmentType,
		orderStatus:     orderStatus,
		cartSnapshot:    cartSnapshot,
		lockedCart:      lockedCart,
		shippingAddress: shippingAddress,
		billingAddress:  billingAddress,
	}, nil
}

// executeCreateOrderTransaction persists order snapshots, handles reservation state,
// converts the cart, and returns a hydrated order response.
func (s *OrderServiceImpl) executeCreateOrderTransaction(
	ctx context.Context,
	userID, sellerID uint,
	req model.CreateOrderRequest,
	createCtx *createOrderContext,
) (*model.OrderResponse, error) {
	return db.WithTransactionResult(
		ctx,
		func(txCtx context.Context) (*model.OrderResponse, error) {
			order, err := s.persistOrderSnapshotGraph(txCtx, userID, sellerID, req, createCtx)
			if err != nil {
				return nil, err
			}

			if err := s.handleCreateOrderReservation(txCtx, sellerID, order.ID, createCtx); err != nil {
				return nil, err
			}

			if err := s.cartSvc.MarkCartConverted(txCtx, createCtx.lockedCart.ID, order.ID, userID); err != nil {
				return nil, err
			}

			return s.loadCreateOrderResponse(txCtx, order.ID)
		},
	)
}

// persistOrderSnapshotGraph writes root order row plus all immutable snapshot tables.
func (s *OrderServiceImpl) persistOrderSnapshotGraph(
	txCtx context.Context,
	userID, sellerID uint,
	req model.CreateOrderRequest,
	createCtx *createOrderContext,
) (*entity.Order, error) {
	now := time.Now().UTC()
	order := s.buildCreateOrderEntity(userID, sellerID, req, createCtx, now)
	if err := s.orderRepo.CreateOrder(txCtx, order); err != nil {
		return nil, err
	}

	orderItems := factory.BuildOrderItemsFromCartSnapshot(order.ID, createCtx.cartSnapshot)
	if err := s.orderRepo.CreateOrderItems(txCtx, orderItems); err != nil {
		return nil, err
	}

	orderAddresses := factory.BuildOrderAddressesFromUserAddresses(
		order.ID,
		createCtx.shippingAddress,
		createCtx.billingAddress,
	)
	if err := s.orderRepo.CreateOrderAddresses(txCtx, orderAddresses); err != nil {
		return nil, err
	}

	orderPromotions := factory.BuildOrderAppliedPromotionsFromCartSnapshot(
		order.ID,
		createCtx.cartSnapshot,
	)
	if err := s.orderRepo.CreateOrderAppliedPromotions(txCtx, orderPromotions); err != nil {
		return nil, err
	}

	itemPromotions := factory.BuildOrderItemAppliedPromotionsFromCartSnapshot(
		order.ID,
		createCtx.cartSnapshot,
		orderItems,
	)
	if err := s.orderRepo.CreateOrderItemAppliedPromotions(txCtx, itemPromotions); err != nil {
		return nil, err
	}

	if err := s.orderHistoryRepo.CreateHistoryEntry(
		txCtx,
		mapper.BuildOrderCreatedHistory(
			order.ID,
			userID,
			constants.CUSTOMER_ROLE_NAME,
			createCtx.orderStatus.String(),
		),
	); err != nil {
		return nil, err
	}

	return order, nil
}

// buildCreateOrderEntity maps request/cart snapshot totals into the persisted order row.
func (s *OrderServiceImpl) buildCreateOrderEntity(
	userID, sellerID uint,
	req model.CreateOrderRequest,
	createCtx *createOrderContext,
	now time.Time,
) *entity.Order {
	shippingCents := int64(0)
	if createCtx.cartSnapshot.Summary.Shipping != nil {
		shippingCents = *createCtx.cartSnapshot.Summary.Shipping
	}
	return mapper.BuildOrderEntity(
		userID,
		sellerID,
		createCtx.fulfillmentType,
		createCtx.orderStatus,
		req.Metadata,
		createCtx.cartSnapshot.Summary.Subtotal,
		createCtx.cartSnapshot.Summary.TotalDiscount,
		shippingCents,
		createCtx.cartSnapshot.Summary.Tax,
		createCtx.cartSnapshot.Summary.Total,
		now,
	)
}

// handleCreateOrderReservation creates reservation for reservable statuses and confirms it immediately
// when order is directly created as confirmed.
func (s *OrderServiceImpl) handleCreateOrderReservation(
	txCtx context.Context,
	sellerID, orderID uint,
	createCtx *createOrderContext,
) error {
	if !shouldReserveInventoryOnCreate(createCtx.orderStatus) {
		return nil
	}

	if _, err := s.inventoryReserveSvc.CreateReservation(txCtx, sellerID, inventoryModel.ReservationRequest{
		ReferenceId:      orderID,
		ExpiresInMinutes: reservationExpiresInMinutes,
		Items:            buildReservationItems(createCtx.cartSnapshot.Items),
	}); err != nil {
		return err
	}

	if createCtx.orderStatus == entity.ORDER_STATUS_CONFIRMED {
		return s.inventoryReserveSvc.UpdateReservationStatus(
			txCtx,
			sellerID,
			inventoryModel.UpdateReservationStatusRequest{
				ReferenceId: orderID,
				Status:      inventoryEntity.ResConfirmed,
			},
		)
	}
	return nil
}

// loadCreateOrderResponse reads order with associations and maps it to API response.
func (s *OrderServiceImpl) loadCreateOrderResponse(
	txCtx context.Context,
	orderID uint,
) (*model.OrderResponse, error) {
	freshOrder, err := s.orderRepo.FindOrderByID(txCtx, orderID)
	if err != nil {
		return nil, err
	}
	if freshOrder == nil {
		return nil, orderError.ErrOrderNotFound
	}
	return factory.BuildOrderResponseFromEntity(freshOrder, nil), nil
}

// GetOrderByID applies role-aware access and returns hydrated order details.
func (s *OrderServiceImpl) GetOrderByID(
	ctx context.Context,
	userID uint,
	role string,
	orderID uint,
) (*model.OrderResponse, error) {
	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil || !canAccessOrder(order, userID, role) {
		return nil, orderError.ErrOrderNotFound
	}

	var customer *model.OrderCustomerResponse
	if shouldIncludeCustomer(role) {
		customer, _ = s.buildOrderCustomer(ctx, order.UserID)
	}
	return factory.BuildOrderResponseFromEntity(order, customer), nil
}

// ListOrders fetches role-scoped order summaries with common pagination.
func (s *OrderServiceImpl) ListOrders(
	ctx context.Context,
	userID uint,
	role string,
	filters model.ListOrdersRequest,
) (*model.PaginatedOrdersResponse, error) {
	filter := filters.ToFilter()

	var (
		orders []entity.Order
		total  int64
		err    error
	)

	switch strings.ToUpper(strings.TrimSpace(role)) {
	case constants.CUSTOMER_ROLE_NAME:
		orders, total, err = s.orderRepo.FindOrdersByUserID(ctx, userID, filter)
	case constants.SELLER_ROLE_NAME:
		orders, total, err = s.orderRepo.FindOrdersBySellerID(ctx, userID, filter)
	default:
		orders, total, err = s.orderRepo.FindAllOrders(ctx, filter)
	}
	if err != nil {
		return nil, err
	}

	out := make([]model.OrderListResponse, 0, len(orders))
	includeCustomer := shouldIncludeCustomer(role)
	for _, order := range orders {
		row := model.OrderListResponse{
			ID:              order.ID,
			OrderNumber:     order.OrderNumber,
			Status:          order.Status,
			TotalCents:      order.TotalCents,
			SubtotalCents:   order.SubtotalCents,
			DiscountCents:   order.DiscountCents,
			FulfillmentType: order.FulfillmentType,
			PlacedAt:        order.PlacedAt,
			PaidAt:          order.PaidAt,
			CreatedAt:       order.CreatedAt,
		}
		if includeCustomer {
			customer, _ := s.buildOrderCustomer(ctx, order.UserID)
			row.Customer = customer
		}
		out = append(out, row)
	}

	return &model.PaginatedOrdersResponse{
		Orders:     out,
		Pagination: common.NewPaginationResponse(filter.Page, filter.PageSize, total),
	}, nil
}

// UpdateOrderStatus validates transition and applies inventory/cart side effects atomically.
func (s *OrderServiceImpl) UpdateOrderStatus(
	ctx context.Context,
	sellerID uint,
	orderID uint,
	req model.UpdateOrderStatusRequest,
) (*model.UpdateStatusResponse, error) {
	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	target, err := s.validateUpdateOrderStatusInput(order, sellerID, req)
	if err != nil {
		return nil, err
	}

	prev := order.Status
	now := time.Now().UTC()
	err = db.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.applyUpdateOrderStatusTx(txCtx, order, sellerID, prev, target, now, req)
	})
	if err != nil {
		return nil, err
	}

	return &model.UpdateStatusResponse{
		ID:             order.ID,
		OrderNumber:    order.OrderNumber,
		PreviousStatus: prev,
		Status:         target,
		TransactionID:  req.TransactionID,
		UpdatedAt:      now,
	}, nil
}

// validateUpdateOrderStatusInput verifies seller access, transition validity, and required fields.
func (s *OrderServiceImpl) validateUpdateOrderStatusInput(
	order *entity.Order,
	sellerID uint,
	req model.UpdateOrderStatusRequest,
) (entity.OrderStatus, error) {
	if order == nil || order.SellerID == nil || *order.SellerID != sellerID {
		return "", orderError.ErrOrderNotFound
	}

	target := normalizeOrderStatus(req.Status)
	if !target.IsValid() {
		return "", orderError.ErrInvalidOrderStatus
	}
	if !orderUtils.IsValidTransition(order.Status, target) {
		return "", orderError.ErrInvalidStatusTransition(order.Status.String(), target.String())
	}

	required := orderUtils.RequiredFieldsForTransition(order.Status, target)
	for _, field := range required {
		switch field {
		case "transactionId":
			if req.TransactionID == nil || strings.TrimSpace(*req.TransactionID) == "" {
				return "", orderError.ErrTransactionIDRequired
			}
		case "failureReason":
			if req.FailureReason == nil || strings.TrimSpace(*req.FailureReason) == "" {
				return "", orderError.ErrFailureReasonRequired
			}
		}
	}
	return target, nil
}

// applyUpdateOrderStatusTx applies order status updates and all dependent side effects in one transaction.
func (s *OrderServiceImpl) applyUpdateOrderStatusTx(
	txCtx context.Context,
	order *entity.Order,
	sellerID uint,
	prev, target entity.OrderStatus,
	now time.Time,
	req model.UpdateOrderStatusRequest,
) error {
	if err := s.orderRepo.UpdateOrderStatus(txCtx, order.ID, target); err != nil {
		return err
	}
	if target == entity.ORDER_STATUS_CONFIRMED {
		if err := s.orderRepo.UpdateOrderPaidAt(txCtx, order.ID, now); err != nil {
			return err
		}
	}
	if req.TransactionID != nil && strings.TrimSpace(*req.TransactionID) != "" {
		if err := s.orderRepo.UpdateOrderTransactionID(
			txCtx,
			order.ID,
			strings.TrimSpace(*req.TransactionID),
		); err != nil {
			return err
		}
	}
	if err := s.updateReservationForOrderStatus(txCtx, sellerID, order.ID, target); err != nil {
		return err
	}
	if target == entity.ORDER_STATUS_FAILED {
		if err := s.reactivateCartForOrder(txCtx, order.ID); err != nil {
			return err
		}
	}
	return s.createSellerStatusHistoryEntry(txCtx, order.ID, sellerID, prev, target, req)
}

// updateReservationForOrderStatus maps order status transition to reservation side effects.
func (s *OrderServiceImpl) updateReservationForOrderStatus(
	txCtx context.Context,
	sellerID, orderID uint,
	target entity.OrderStatus,
) error {
	resStatus := mapOrderStatusToReservationStatus(target)
	if resStatus == "" {
		return nil
	}
	return s.inventoryReserveSvc.UpdateReservationStatus(
		txCtx,
		sellerID,
		inventoryModel.UpdateReservationStatusRequest{
			ReferenceId: orderID,
			Status:      resStatus,
		},
	)
}

// createSellerStatusHistoryEntry appends order_history audit row for seller-driven transitions.
func (s *OrderServiceImpl) createSellerStatusHistoryEntry(
	txCtx context.Context,
	orderID, sellerID uint,
	prev, target entity.OrderStatus,
	req model.UpdateOrderStatusRequest,
) error {
	note := req.Note
	if req.FailureReason != nil && target == entity.ORDER_STATUS_FAILED {
		note = req.FailureReason
	}
	return s.orderHistoryRepo.CreateHistoryEntry(
		txCtx,
		mapper.BuildOrderTransitionHistory(
			orderID,
			prev,
			target,
			sellerID,
			constants.SELLER_ROLE_NAME,
			req.TransactionID,
			req.FailureReason,
			note,
			req.Metadata,
		),
	)
}

// CancelOrder performs customer-initiated cancellation with reservation release.
func (s *OrderServiceImpl) CancelOrder(
	ctx context.Context,
	userID uint,
	orderID uint,
	req model.CancelOrderRequest,
) (*model.UpdateStatusResponse, error) {
	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if err := s.validateCancelOrderInput(order, userID); err != nil {
		return nil, err
	}

	prev := order.Status
	now := time.Now().UTC()
	err = db.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.applyCancelOrderTx(txCtx, order, userID, prev, req)
	})
	if err != nil {
		return nil, err
	}

	return &model.UpdateStatusResponse{
		ID:             order.ID,
		OrderNumber:    order.OrderNumber,
		PreviousStatus: prev,
		Status:         entity.ORDER_STATUS_CANCELLED,
		UpdatedAt:      now,
	}, nil
}

// validateCancelOrderInput ensures customer ownership and cancellable state.
func (s *OrderServiceImpl) validateCancelOrderInput(
	order *entity.Order,
	userID uint,
) error {
	if order == nil || order.UserID != userID {
		return orderError.ErrOrderNotFound
	}
	if order.Status != entity.ORDER_STATUS_PENDING &&
		order.Status != entity.ORDER_STATUS_CONFIRMED {
		return orderError.ErrOrderNotCancellable
	}
	return nil
}

// applyCancelOrderTx applies cancellation updates, reservation release, and audit history.
func (s *OrderServiceImpl) applyCancelOrderTx(
	txCtx context.Context,
	order *entity.Order,
	userID uint,
	prev entity.OrderStatus,
	req model.CancelOrderRequest,
) error {
	if err := s.orderRepo.UpdateOrderStatus(txCtx, order.ID, entity.ORDER_STATUS_CANCELLED); err != nil {
		return err
	}

	orderSellerID := uint(0)
	if order.SellerID != nil {
		orderSellerID = *order.SellerID
	}
	if err := s.inventoryReserveSvc.UpdateReservationStatus(txCtx, orderSellerID,
		inventoryModel.UpdateReservationStatusRequest{
			ReferenceId: order.ID,
			Status:      inventoryEntity.ResCancelled,
		}); err != nil {
		return err
	}

	if prev == entity.ORDER_STATUS_PENDING {
		if err := s.reactivateCartForOrder(txCtx, order.ID); err != nil {
			return err
		}
	}

	return s.orderHistoryRepo.CreateHistoryEntry(
		txCtx,
		mapper.BuildOrderTransitionHistory(
			order.ID,
			prev,
			entity.ORDER_STATUS_CANCELLED,
			userID,
			constants.CUSTOMER_ROLE_NAME,
			nil,
			nil,
			req.Reason,
			nil,
		),
	)
}

func (s *OrderServiceImpl) loadAndValidateAddresses(
	ctx context.Context,
	userID, shippingAddressID, billingAddressID uint,
) (*userModel.AddressResponse, *userModel.AddressResponse, error) {
	shipping, err := s.addressSvc.GetAddressByID(ctx, shippingAddressID, userID)
	if err != nil {
		if strings.Contains(
			strings.ToLower(err.Error()),
			strings.ToLower(userConstant.ADDRESS_NOT_FOUND_MSG),
		) {
			return nil, nil, orderError.ErrAddressNotFound
		}
		return nil, nil, err
	}
	billing, err := s.addressSvc.GetAddressByID(ctx, billingAddressID, userID)
	if err != nil {
		if strings.Contains(
			strings.ToLower(err.Error()),
			strings.ToLower(userConstant.ADDRESS_NOT_FOUND_MSG),
		) {
			return nil, nil, orderError.ErrAddressNotFound
		}
		return nil, nil, err
	}
	return shipping, billing, nil
}

func buildReservationItems(
	cartItems []model.CartItemWithPricingResponse,
) []inventoryModel.ReservationItem {
	result := make([]inventoryModel.ReservationItem, 0, len(cartItems))
	for _, item := range cartItems {
		result = append(result, inventoryModel.ReservationItem{
			VariantID:        item.VariantID,
			ReservedQuantity: uint(item.Quantity),
		})
	}
	return result
}

func normalizeFulfillmentType(v entity.FulfillmentType) (entity.FulfillmentType, error) {
	if strings.TrimSpace(v.String()) == "" {
		return entity.DIRECTSHIP, nil
	}
	normalized := entity.FulfillmentType(strings.ToLower(strings.TrimSpace(v.String())))
	if !normalized.IsValid() {
		return "", orderError.ErrInvalidFulfillmentType
	}
	return normalized, nil
}

func normalizeOrderStatus(status entity.OrderStatus) entity.OrderStatus {
	return entity.OrderStatus(strings.ToLower(strings.TrimSpace(status.String())))
}

// normalizeCreateOrderStatus normalizes optional request status and defaults to pending.
func normalizeCreateOrderStatus(status *entity.OrderStatus) (entity.OrderStatus, error) {
	if status == nil {
		return entity.ORDER_STATUS_PENDING, nil
	}
	normalized := normalizeOrderStatus(*status)
	if !normalized.IsValid() {
		return "", orderError.ErrInvalidOrderStatus
	}
	return normalized, nil
}

func shouldReserveInventoryOnCreate(status entity.OrderStatus) bool {
	return status == entity.ORDER_STATUS_PENDING || status == entity.ORDER_STATUS_CONFIRMED
}

func mapOrderStatusToReservationStatus(
	status entity.OrderStatus,
) inventoryEntity.ReservationStatus {
	switch status {
	case entity.ORDER_STATUS_CONFIRMED:
		return inventoryEntity.ResConfirmed
	case entity.ORDER_STATUS_FAILED, entity.ORDER_STATUS_CANCELLED:
		return inventoryEntity.ResCancelled
	case entity.ORDER_STATUS_COMPLETED:
		return inventoryEntity.ResFulfilled
	default:
		return ""
	}
}

func (s *OrderServiceImpl) reactivateCartForOrder(ctx context.Context, orderID uint) error {
	return s.cartSvc.ReactivateCartByOrderID(ctx, orderID)
}

func canAccessOrder(order *entity.Order, userID uint, role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case constants.CUSTOMER_ROLE_NAME:
		return order.UserID == userID
	case constants.SELLER_ROLE_NAME:
		return order.SellerID != nil && *order.SellerID == userID
	default:
		return true
	}
}

func shouldIncludeCustomer(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case constants.SELLER_ROLE_NAME, constants.ADMIN_ROLE_NAME:
		return true
	default:
		return false
	}
}

func (s *OrderServiceImpl) buildOrderCustomer(
	ctx context.Context,
	userID uint,
) (*model.OrderCustomerResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &model.OrderCustomerResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Phone:     strPtr(user.Phone),
	}, nil
}

func strPtr(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
