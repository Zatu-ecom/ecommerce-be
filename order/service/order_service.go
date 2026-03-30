package service

import (
	"context"

	inventoryService "ecommerce-be/inventory/service"
	"ecommerce-be/order/entity"
	"ecommerce-be/order/model"
	"ecommerce-be/order/repository"
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
