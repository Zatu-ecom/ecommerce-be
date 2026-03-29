package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/common/helper"
	"ecommerce-be/order/entity"
	"ecommerce-be/order/model"

	"gorm.io/gorm"
)

// OrderRepository handles database operations for orders.
type OrderRepository interface {
	HasPastOrders(ctx context.Context, userID uint) (bool, error)

	CreateOrder(ctx context.Context, order *entity.Order) error
	CreateOrderItems(ctx context.Context, items []entity.OrderItem) error
	CreateOrderAddresses(ctx context.Context, addresses []entity.OrderAddress) error
	CreateOrderAppliedPromotions(ctx context.Context, promos []entity.OrderAppliedPromotion) error
	CreateOrderItemAppliedPromotions(
		ctx context.Context,
		promos []entity.OrderItemAppliedPromotion,
	) error

	FindOrderByID(ctx context.Context, orderID uint) (*entity.Order, error)
	FindOrdersByUserID(
		ctx context.Context,
		userID uint,
		filters model.ListOrdersFilter,
	) ([]entity.Order, int64, error)
	FindOrdersBySellerID(
		ctx context.Context,
		sellerID uint,
		filters model.ListOrdersFilter,
	) ([]entity.Order, int64, error)
	FindAllOrders(
		ctx context.Context,
		filters model.ListOrdersFilter,
	) ([]entity.Order, int64, error)

	UpdateOrderStatus(ctx context.Context, orderID uint, status entity.OrderStatus) error
	UpdateOrderTransactionID(ctx context.Context, orderID uint, txnID string) error
	UpdateOrderPaidAt(ctx context.Context, orderID uint, paidAt time.Time) error
}

// OrderRepositoryImpl implements OrderRepository.
type OrderRepositoryImpl struct{}

// NewOrderRepository creates a new OrderRepository.
func NewOrderRepository() OrderRepository {
	return &OrderRepositoryImpl{}
}

// HasPastOrders returns true if the user has any order that isn't pending, cancelled, or failed.
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

func (r *OrderRepositoryImpl) CreateOrder(ctx context.Context, order *entity.Order) error {
	return db.DB(ctx).Create(order).Error
}

func (r *OrderRepositoryImpl) CreateOrderItems(
	ctx context.Context,
	items []entity.OrderItem,
) error {
	if len(items) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&items).Error
}

func (r *OrderRepositoryImpl) CreateOrderAddresses(
	ctx context.Context,
	addresses []entity.OrderAddress,
) error {
	if len(addresses) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&addresses).Error
}

func (r *OrderRepositoryImpl) CreateOrderAppliedPromotions(
	ctx context.Context,
	promos []entity.OrderAppliedPromotion,
) error {
	if len(promos) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&promos).Error
}

func (r *OrderRepositoryImpl) CreateOrderItemAppliedPromotions(
	ctx context.Context,
	promos []entity.OrderItemAppliedPromotion,
) error {
	if len(promos) == 0 {
		return nil
	}
	return db.DB(ctx).Create(&promos).Error
}

func (r *OrderRepositoryImpl) FindOrderByID(
	ctx context.Context,
	orderID uint,
) (*entity.Order, error) {
	var order entity.Order
	err := db.DB(ctx).
		Preload("Items").
		Preload("Addresses").
		Preload("AppliedPromotions").
		Preload("AppliedCoupons").
		Preload("ItemAppliedPromotions").
		Where("id = ?", orderID).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepositoryImpl) FindOrdersByUserID(
	ctx context.Context,
	userID uint,
	filters model.ListOrdersFilter,
) ([]entity.Order, int64, error) {
	query := db.DB(ctx).Model(&entity.Order{}).Where("user_id = ?", userID)
	return r.findOrdersWithFilters(query, filters)
}

func (r *OrderRepositoryImpl) FindOrdersBySellerID(
	ctx context.Context,
	sellerID uint,
	filters model.ListOrdersFilter,
) ([]entity.Order, int64, error) {
	query := db.DB(ctx).Model(&entity.Order{}).Where("seller_id = ?", sellerID)
	return r.findOrdersWithFilters(query, filters)
}

func (r *OrderRepositoryImpl) FindAllOrders(
	ctx context.Context,
	filters model.ListOrdersFilter,
) ([]entity.Order, int64, error) {
	query := db.DB(ctx).Model(&entity.Order{})
	return r.findOrdersWithFilters(query, filters)
}

func (r *OrderRepositoryImpl) findOrdersWithFilters(
	query *gorm.DB,
	filters model.ListOrdersFilter,
) ([]entity.Order, int64, error) {
	var orders []entity.Order
	var total int64

	filtered := applyOrderFilters(query, filters)
	if err := filtered.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filters.Page
	pageSize := filters.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := helper.CalculateOffset(page, pageSize)

	if err := filtered.
		Order(normalizeOrderSortBy(filters.SortBy) + " " + normalizeOrderSortOrder(filters.SortOrder)).
		Limit(pageSize).
		Offset(offset).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func applyOrderFilters(query *gorm.DB, filters model.ListOrdersFilter) *gorm.DB {
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if filters.FromDate != nil {
		query = query.Where("placed_at >= ?", filters.FromDate.UTC())
	}
	if filters.ToDate != nil {
		query = query.Where("placed_at <= ?", filters.ToDate.UTC())
	}

	if filters.Search != nil {
		search := strings.TrimSpace(*filters.Search)
		if search != "" {
			pattern := "%" + search + "%"
			query = query.Where(
				`order_number ILIKE ? OR transaction_id ILIKE ? OR CAST(id AS TEXT) ILIKE ?`,
				pattern,
				pattern,
				pattern,
			)
		}
	}

	return query
}

func normalizeOrderSortBy(sortBy string) string {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "id":
		return "id"
	case "ordernumber", "order_number":
		return "order_number"
	case "status":
		return "status"
	case "subtotalcents", "subtotal_cents":
		return "subtotal_cents"
	case "discountcents", "discount_cents":
		return "discount_cents"
	case "shippingcents", "shipping_cents":
		return "shipping_cents"
	case "taxcents", "tax_cents":
		return "tax_cents"
	case "totalcents", "total_cents":
		return "total_cents"
	case "placedat", "placed_at":
		return "placed_at"
	case "paidat", "paid_at":
		return "paid_at"
	case "updatedat", "updated_at":
		return "updated_at"
	default:
		return "created_at"
	}
}

func normalizeOrderSortOrder(sortOrder string) string {
	if strings.EqualFold(strings.TrimSpace(sortOrder), "asc") {
		return "asc"
	}
	return "desc"
}

func (r *OrderRepositoryImpl) UpdateOrderStatus(
	ctx context.Context,
	orderID uint,
	status entity.OrderStatus,
) error {
	return db.DB(ctx).
		Model(&entity.Order{}).
		Where("id = ?", orderID).
		Update("status", status).
		Error
}

func (r *OrderRepositoryImpl) UpdateOrderTransactionID(
	ctx context.Context,
	orderID uint,
	txnID string,
) error {
	return db.DB(ctx).
		Model(&entity.Order{}).
		Where("id = ?", orderID).
		Update("transaction_id", txnID).
		Error
}

func (r *OrderRepositoryImpl) UpdateOrderPaidAt(
	ctx context.Context,
	orderID uint,
	paidAt time.Time,
) error {
	return db.DB(ctx).
		Model(&entity.Order{}).
		Where("id = ?", orderID).
		Update("paid_at", paidAt.UTC()).
		Error
}
