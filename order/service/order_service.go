package service

import (
	"context"

	commonModel "ecommerce-be/common/model"
	inventoryService "ecommerce-be/inventory/service"
	"ecommerce-be/order/entity"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"
	promotionService "ecommerce-be/promotion/service"
	userFactory "ecommerce-be/user/factory"
	userModel "ecommerce-be/user/model"
	userRepository "ecommerce-be/user/repository"
	userService "ecommerce-be/user/service"
)

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

	// AttachTransactionID links our internal payment transaction id to the order.
	AttachTransactionID(ctx context.Context, orderID, sellerID uint, transactionID string) error
	// ConfirmPaymentByTransactionID confirms a pending order (pending → confirmed + paid_at).
	ConfirmPaymentByTransactionID(ctx context.Context, transactionID string) error
	// FailPaymentByTransactionID fails a pending order (pending → failed).
	FailPaymentByTransactionID(ctx context.Context, transactionID, reason string) error
}

type OrderServiceImpl struct {
	cartSvc             CartService
	orderRepo           repository.OrderRepository
	orderHistoryRepo    repository.OrderHistoryRepository
	inventoryReserveSvc inventoryService.InventoryReservationService
	addressSvc          userService.AddressService
	userRepo            userRepository.UserRepository
	userSvc             userService.UserService
	couponApplySvc      promotionService.CouponApplyService
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

// orderCurrency resolves the seller's base currency for order Money rendering.
// Order presentation stays in the seller's base currency until FX (FR-010).
func (s *OrderServiceImpl) orderCurrency(ctx context.Context, sellerID uint) (commonModel.CurrencyInfo, error) {
	ccy, err := s.userSvc.GetSellerDefaultCurrency(ctx, sellerID)
	if err != nil {
		return commonModel.CurrencyInfo{}, err
	}
	return userFactory.ToCurrencyInfo(ccy), nil
}

func NewOrderService(
	cartSvc CartService,
	orderRepo repository.OrderRepository,
	orderHistoryRepo repository.OrderHistoryRepository,
	inventoryReserveSvc inventoryService.InventoryReservationService,
	addressSvc userService.AddressService,
	userRepo userRepository.UserRepository,
	userSvc userService.UserService,
	couponApplySvc promotionService.CouponApplyService,
) OrderService {
	return &OrderServiceImpl{
		cartSvc:             cartSvc,
		orderRepo:           orderRepo,
		orderHistoryRepo:    orderHistoryRepo,
		inventoryReserveSvc: inventoryReserveSvc,
		addressSvc:          addressSvc,
		userRepo:            userRepo,
		userSvc:             userSvc,
		couponApplySvc:      couponApplySvc,
	}
}
