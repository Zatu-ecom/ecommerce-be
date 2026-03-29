package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/order/entity"
)

type OrderHistoryRepository interface {
	CreateHistoryEntry(ctx context.Context, entry *entity.OrderHistory) error
	FindHistoryByOrderID(ctx context.Context, orderID uint) ([]entity.OrderHistory, error)
}

type OrderHistoryRepositoryImpl struct{}

func NewOrderHistoryRepository() OrderHistoryRepository {
	return &OrderHistoryRepositoryImpl{}
}

func (r *OrderHistoryRepositoryImpl) CreateHistoryEntry(
	ctx context.Context,
	entry *entity.OrderHistory,
) error {
	return db.DB(ctx).Create(entry).Error
}

func (r *OrderHistoryRepositoryImpl) FindHistoryByOrderID(
	ctx context.Context,
	orderID uint,
) ([]entity.OrderHistory, error) {
	var rows []entity.OrderHistory
	if err := db.DB(ctx).
		Where("order_id = ?", orderID).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
