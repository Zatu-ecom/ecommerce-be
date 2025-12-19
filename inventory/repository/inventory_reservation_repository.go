package repository

import (
	"time"

	"ecommerce-be/inventory/entity"

	"gorm.io/gorm"
)

type InventoryReservationRepository interface {
	CreateReservations(reservations []*entity.InventoryReservation) error
	FindByID(id uint) (*entity.InventoryReservation, error)
	FindByIDs(ids []uint) ([]*entity.InventoryReservation, error)
	FindByReferenceID(referenceID uint) ([]*entity.InventoryReservation, error)
	UpdateStatusByReferenceID(referenceID uint, status entity.ReservationStatus) error
	UpdateStatusByIDs(ids []uint, status entity.ReservationStatus) error
}

type InventoryReservationRepositoryImpl struct {
	db *gorm.DB
}

func NewInventoryReservationRepository(db *gorm.DB) InventoryReservationRepository {
	return &InventoryReservationRepositoryImpl{db: db}
}

func (r *InventoryReservationRepositoryImpl) CreateReservations(
	reservations []*entity.InventoryReservation,
) error {
	return r.db.Create(&reservations).Error
}

func (r *InventoryReservationRepositoryImpl) FindByID(
	id uint,
) (*entity.InventoryReservation, error) {
	var reservation entity.InventoryReservation
	err := r.db.First(&reservation, id).Error
	if err != nil {
		return nil, err
	}
	return &reservation, nil
}

func (r *InventoryReservationRepositoryImpl) FindByIDs(
	ids []uint,
) ([]*entity.InventoryReservation, error) {
	var reservations []*entity.InventoryReservation
	err := r.db.Where("id IN ?", ids).Find(&reservations).Error
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *InventoryReservationRepositoryImpl) UpdateStatus(
	id uint,
	status entity.ReservationStatus,
) error {
	return r.db.Model(&entity.InventoryReservation{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *InventoryReservationRepositoryImpl) FindByReferenceID(
	referenceID uint,
) ([]*entity.InventoryReservation, error) {
	var reservations []*entity.InventoryReservation
	err := r.db.Where("reference_id = ?", referenceID).Find(&reservations).Error
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *InventoryReservationRepositoryImpl) UpdateStatusByReferenceID(
	referenceID uint,
	status entity.ReservationStatus,
) error {
	return r.db.Model(&entity.InventoryReservation{}).
		Where("reference_id = ?", referenceID).
		Update("status", status).Error
}

func (r *InventoryReservationRepositoryImpl) UpdateStatusByIDs(
	ids []uint,
	status entity.ReservationStatus,
) error {
	return r.db.Model(&entity.InventoryReservation{}).
		Where("id IN ?", ids).
		Update("status", status).
		Update("updated_at", time.Now().UTC()).Error
}
