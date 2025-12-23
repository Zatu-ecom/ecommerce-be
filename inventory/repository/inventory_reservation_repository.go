package repository

import (
	"context"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/inventory/entity"
)

type InventoryReservationRepository interface {
	CreateOrSave(
		ctx context.Context,
		reservations []*entity.InventoryReservation,
	) error
	FindByID(
		ctx context.Context,
		id uint,
	) (*entity.InventoryReservation, error)
	FindByIDs(
		ctx context.Context,
		ids []uint,
	) ([]*entity.InventoryReservation, error)
	FindByReferenceIDAndStatus(
		ctx context.Context,
		status entity.ReservationStatus,
		referenceID uint,
	) ([]*entity.InventoryReservation, error)
	UpdateStatusByReferenceID(
		ctx context.Context,
		referenceID uint,
		existingStatus entity.ReservationStatus,
		status entity.ReservationStatus,
	) error
	UpdateStatusByIDs(
		ctx context.Context,
		ids []uint,
		status entity.ReservationStatus,
	) error
}

type InventoryReservationRepositoryImpl struct{}

func NewInventoryReservationRepository() InventoryReservationRepository {
	return &InventoryReservationRepositoryImpl{}
}

func (r *InventoryReservationRepositoryImpl) CreateOrSave(
	ctx context.Context,
	reservations []*entity.InventoryReservation,
) error {
	return db.DB(ctx).Save(&reservations).Error
}

func (r *InventoryReservationRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.InventoryReservation, error) {
	var reservation entity.InventoryReservation
	err := db.DB(ctx).First(&reservation, id).Error
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

func (r *InventoryReservationRepositoryImpl) FindByIDs(
	ctx context.Context,
	ids []uint,
) ([]*entity.InventoryReservation, error) {
	var reservations []*entity.InventoryReservation
	err := db.DB(ctx).Where("id IN ?", ids).Find(&reservations).Error
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *InventoryReservationRepositoryImpl) UpdateStatus(
	ctx context.Context,
	id uint,
	status entity.ReservationStatus,
) error {
	return db.DB(ctx).Model(&entity.InventoryReservation{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *InventoryReservationRepositoryImpl) FindByReferenceIDAndStatus(
	ctx context.Context,
	status entity.ReservationStatus,
	referenceID uint,
) ([]*entity.InventoryReservation, error) {
	var reservations []*entity.InventoryReservation
	err := db.DB(ctx).
		Where("reference_id = ? AND status = ?", referenceID, status).
		Find(&reservations).
		Error
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *InventoryReservationRepositoryImpl) UpdateStatusByReferenceID(
	ctx context.Context,
	referenceID uint,
	existingStatus entity.ReservationStatus,
	status entity.ReservationStatus,
) error {
	return db.DB(ctx).Model(&entity.InventoryReservation{}).
		Where("reference_id = ? AND status = ?", referenceID, existingStatus).
		Update("status", status).Error
}

func (r *InventoryReservationRepositoryImpl) UpdateStatusByIDs(
	ctx context.Context,
	ids []uint,
	status entity.ReservationStatus,
) error {
	return db.DB(ctx).Model(&entity.InventoryReservation{}).
		Where("id IN ?", ids).
		Update("status", status).
		Update("updated_at", time.Now().UTC()).Error
}
