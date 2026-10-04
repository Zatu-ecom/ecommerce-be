package repository

import (
	"context"
	"errors"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/fulfillment/entity"

	"gorm.io/gorm"
)

// NDRRepository persists failed-delivery rounds. The partial unique index
// (shipment_id WHERE action_taken IS NULL) enforces a single open round;
// concurrent double-opens surface as unique violations, which the service
// treats as idempotent replays.
type NDRRepository interface {
	Create(ctx context.Context, ndr *entity.FulfillmentNDR) error
	FindByShipmentID(ctx context.Context, shipmentID uint) ([]entity.FulfillmentNDR, error)
	FindOpenByShipment(ctx context.Context, shipmentID uint) (*entity.FulfillmentNDR, error)
	MaxAttemptNo(ctx context.Context, shipmentID uint) (int, error)
	MarkActed(ctx context.Context, id uint, action string) error
	// FindStaleOpen returns open rounds older than the cutoff for the
	// escalation sweep, oldest first.
	FindStaleOpen(ctx context.Context, olderThan time.Time, limit int) ([]entity.FulfillmentNDR, error)
}

type NDRRepositoryImpl struct{}

func NewNDRRepository() NDRRepository {
	return &NDRRepositoryImpl{}
}

func (r *NDRRepositoryImpl) Create(
	ctx context.Context,
	ndr *entity.FulfillmentNDR,
) error {
	return db.DB(ctx).Create(ndr).Error
}

func (r *NDRRepositoryImpl) FindByShipmentID(
	ctx context.Context,
	shipmentID uint,
) ([]entity.FulfillmentNDR, error) {
	var rounds []entity.FulfillmentNDR
	err := db.DB(ctx).
		Where("shipment_id = ?", shipmentID).
		Order("attempt_no ASC").
		Find(&rounds).Error
	if err != nil {
		return nil, err
	}
	return rounds, nil
}

func (r *NDRRepositoryImpl) FindOpenByShipment(
	ctx context.Context,
	shipmentID uint,
) (*entity.FulfillmentNDR, error) {
	var round entity.FulfillmentNDR
	err := db.DB(ctx).
		Where("shipment_id = ? AND action_taken IS NULL", shipmentID).
		First(&round).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &round, nil
}

func (r *NDRRepositoryImpl) MaxAttemptNo(
	ctx context.Context,
	shipmentID uint,
) (int, error) {
	var max *int
	err := db.DB(ctx).
		Model(&entity.FulfillmentNDR{}).
		Select("MAX(attempt_no)").
		Where("shipment_id = ?", shipmentID).
		Scan(&max).Error
	if err != nil {
		return 0, err
	}
	if max == nil {
		return 0, nil
	}
	return *max, nil
}

func (r *NDRRepositoryImpl) MarkActed(
	ctx context.Context,
	id uint,
	action string,
) error {
	return db.DB(ctx).
		Model(&entity.FulfillmentNDR{}).
		Where("id = ? AND action_taken IS NULL", id).
		Updates(map[string]any{
			"action_taken": action,
			"acted_at":     time.Now().UTC(),
		}).Error
}

func (r *NDRRepositoryImpl) FindStaleOpen(
	ctx context.Context,
	olderThan time.Time,
	limit int,
) ([]entity.FulfillmentNDR, error) {
	var rounds []entity.FulfillmentNDR
	err := db.DB(ctx).
		Where("action_taken IS NULL AND created_at < ?", olderThan).
		Order("created_at ASC").
		Limit(limit).
		Find(&rounds).Error
	if err != nil {
		return nil, err
	}
	return rounds, nil
}
