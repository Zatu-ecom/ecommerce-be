package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/order/entity"
)

// OrderRepository handles database operations for orders
type OrderRepository interface {
	HasPastOrders(ctx context.Context, userID uint) (bool, error)
}

// OrderRepositoryImpl implements OrderRepository
type OrderRepositoryImpl struct{}

// NewOrderRepository creates a new OrderRepository
func NewOrderRepository() OrderRepository {
	return &OrderRepositoryImpl{}
}

// HasPastOrders returns true if the user has any order that isn't pending, cancelled, or failed
func (r *OrderRepositoryImpl) HasPastOrders(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := db.DB(ctx).Model(&entity.Order{}).
		Where("user_id = ? AND status NOT IN ?", userID, []entity.OrderStatus{
			entity.ORDER_STATUS_PENDING,
			entity.ORDER_STATUS_CANCELLED,
			entity.ORDER_STATUS_FAILED,
		}).
		Count(&count).Error

	return count > 0, err
}
