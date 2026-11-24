package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/fulfillment/entity"
)

// ShipmentEventRepository appends to the immutable ledger. Rows are never
// updated or deleted; there is intentionally no update or delete method.
type ShipmentEventRepository interface {
	Create(ctx context.Context, event *entity.FulfillmentShipmentEvent) error
	FindByShipmentID(ctx context.Context, shipmentID uint) ([]entity.FulfillmentShipmentEvent, error)
}

type ShipmentEventRepositoryImpl struct{}

func NewShipmentEventRepository() ShipmentEventRepository {
	return &ShipmentEventRepositoryImpl{}
}

func (r *ShipmentEventRepositoryImpl) Create(
	ctx context.Context,
	event *entity.FulfillmentShipmentEvent,
) error {
	return db.DB(ctx).Create(event).Error
}

func (r *ShipmentEventRepositoryImpl) FindByShipmentID(
	ctx context.Context,
	shipmentID uint,
) ([]entity.FulfillmentShipmentEvent, error) {
	var events []entity.FulfillmentShipmentEvent
	err := db.DB(ctx).
		Where("shipment_id = ?", shipmentID).
		Order("created_at ASC, id ASC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}
