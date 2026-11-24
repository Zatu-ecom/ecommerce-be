package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/fulfillment/entity"
)

// ShipmentItemRepository persists split lines. The Σ-quantity guard
// (Σ shipment qty <= order_item.quantity) lives in the service layer,
// which holds the order-row lock; this layer only sums.
type ShipmentItemRepository interface {
	Create(ctx context.Context, item *entity.FulfillmentShipmentItem) error
	FindByShipmentID(ctx context.Context, shipmentID uint) ([]entity.FulfillmentShipmentItem, error)
	// SumQuantityByOrderItem totals non-cancelled-box quantities for one
	// order line. Cancelled boxes are excluded by joining the shipment row.
	SumQuantityByOrderItem(ctx context.Context, orderItemID uint) (int, error)
}

type ShipmentItemRepositoryImpl struct{}

func NewShipmentItemRepository() ShipmentItemRepository {
	return &ShipmentItemRepositoryImpl{}
}

func (r *ShipmentItemRepositoryImpl) Create(
	ctx context.Context,
	item *entity.FulfillmentShipmentItem,
) error {
	return db.DB(ctx).Create(item).Error
}

func (r *ShipmentItemRepositoryImpl) FindByShipmentID(
	ctx context.Context,
	shipmentID uint,
) ([]entity.FulfillmentShipmentItem, error) {
	var items []entity.FulfillmentShipmentItem
	err := db.DB(ctx).
		Where("shipment_id = ?", shipmentID).
		Order("id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ShipmentItemRepositoryImpl) SumQuantityByOrderItem(
	ctx context.Context,
	orderItemID uint,
) (int, error) {
	var total *int
	err := db.DB(ctx).
		Model(&entity.FulfillmentShipmentItem{}).
		Select("SUM(fulfillment_shipment_item.quantity)").
		Joins("JOIN fulfillment_shipment ON fulfillment_shipment.id = fulfillment_shipment_item.shipment_id").
		Where("fulfillment_shipment_item.order_item_id = ?", orderItemID).
		Where("fulfillment_shipment.status <> ?", string(entity.SHIPMENT_STATUS_CANCELLED)).
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}
