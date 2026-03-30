package service

import (
	"context"
	"strings"

	"ecommerce-be/common"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/helper"
	"ecommerce-be/order/entity"
	orderError "ecommerce-be/order/error"
	"ecommerce-be/order/factory"
	"ecommerce-be/order/model"
	userModel "ecommerce-be/user/model"
	userConstant "ecommerce-be/user/utils/constant"
)

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
		Phone: helper.StringPtr(
			strings.TrimSpace(user.Phone),
		),
	}, nil
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
