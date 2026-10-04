package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/fulfillment/entity"
	fulfillmenterrors "ecommerce-be/fulfillment/error"

	"gorm.io/gorm"
)

// ShipmentRepository persists shipment boxes. All reads are seller-scoped
// by the caller (seller_id predicate); this layer never invents tenancy.
type ShipmentRepository interface {
	Create(ctx context.Context, shipment *entity.FulfillmentShipment) error
	FindByID(ctx context.Context, id uint) (*entity.FulfillmentShipment, error)
	FindByOrderID(ctx context.Context, orderID uint) ([]entity.FulfillmentShipment, error)
	FindByAWB(ctx context.Context, providerCode, awb string) (*entity.FulfillmentShipment, error)
	FindByProviderOrderID(ctx context.Context, providerCode, providerOrderID string) (*entity.FulfillmentShipment, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*entity.FulfillmentShipment, error)
	// UpdateStatusIfCurrent moves status only from the expected value and
	// reports whether the row moved (idempotent no-op on replays).
	UpdateStatusIfCurrent(
		ctx context.Context,
		id uint,
		from entity.ShipmentStatus,
		to entity.ShipmentStatus,
		patch map[string]any,
	) (bool, error)
	// FindRecoverableDrafts returns drafts for the recover job:
	// never-attempted drafts past the grace window, plus attempted drafts
	// still lacking provider ids. FOR UPDATE SKIP LOCKED.
	FindRecoverableDrafts(ctx context.Context, limit int) ([]entity.FulfillmentShipment, error)
	// FindStaleInFlight returns non-terminal rows of the given statuses whose
	// last sync predates the cutoff. FOR UPDATE SKIP LOCKED by the caller
	// pattern (reconciler locks per batch).
	FindStaleInFlight(
		ctx context.Context,
		statuses []entity.ShipmentStatus,
		syncedBefore time.Time,
		limit int,
	) ([]entity.FulfillmentShipment, error)
	// FindShipments lists a seller's boxes with optional filters and
	// pagination. SortBy is restricted to created_at|updated_at|status.
	FindShipments(ctx context.Context, filter ShipmentFilter) ([]entity.FulfillmentShipment, error)
	// CountShipments counts the rows FindShipments would return (for pages).
	CountShipments(ctx context.Context, filter ShipmentFilter) (int64, error)
	// UpdateColumns patches columns without touching status (re-stamps,
	// measurement fixes outside the draft flow). Returns moved=false when
	// the row vanished concurrently.
	UpdateColumns(ctx context.Context, id uint, patch map[string]any) (bool, error)
}

// ShipmentFilter scopes the seller box list. Zero values mean unfiltered;
// SortBy/SortOrder fall back to created_at DESC on unknown input.
type ShipmentFilter struct {
	SellerID         uint
	Status           string
	OrderID          uint
	PickupLocationID uint
	SortBy           string
	SortOrder        string
	Limit            int
	Offset           int
}

type ShipmentRepositoryImpl struct{}

func NewShipmentRepository() ShipmentRepository {
	return &ShipmentRepositoryImpl{}
}

func (r *ShipmentRepositoryImpl) Create(
	ctx context.Context,
	shipment *entity.FulfillmentShipment,
) error {
	if err := db.DB(ctx).Create(shipment).Error; err != nil {
		return err
	}
	return nil
}

func (r *ShipmentRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.FulfillmentShipment, error) {
	var shipment entity.FulfillmentShipment
	err := db.DB(ctx).Preload("Items").Where("id = ?", id).First(&shipment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorFulfillmentNotFound
		}
		return nil, err
	}
	return &shipment, nil
}

func (r *ShipmentRepositoryImpl) FindByOrderID(
	ctx context.Context,
	orderID uint,
) ([]entity.FulfillmentShipment, error) {
	var shipments []entity.FulfillmentShipment
	err := db.DB(ctx).
		Preload("Items").
		Where("order_id = ?", orderID).
		Order("id ASC").
		Find(&shipments).Error
	if err != nil {
		return nil, err
	}
	return shipments, nil
}

func (r *ShipmentRepositoryImpl) FindByAWB(
	ctx context.Context,
	providerCode, awb string,
) (*entity.FulfillmentShipment, error) {
	var shipment entity.FulfillmentShipment
	err := db.DB(ctx).
		Where("provider_code = ? AND awb = ?", providerCode, awb).
		First(&shipment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorFulfillmentNotFound
		}
		return nil, err
	}
	return &shipment, nil
}

func (r *ShipmentRepositoryImpl) FindByProviderOrderID(
	ctx context.Context,
	providerCode, providerOrderID string,
) (*entity.FulfillmentShipment, error) {
	var shipment entity.FulfillmentShipment
	err := db.DB(ctx).
		Where("provider_code = ? AND provider_order_id = ?", providerCode, providerOrderID).
		First(&shipment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorFulfillmentNotFound
		}
		return nil, err
	}
	return &shipment, nil
}

func (r *ShipmentRepositoryImpl) FindByIdempotencyKey(
	ctx context.Context,
	key string,
) (*entity.FulfillmentShipment, error) {
	var shipment entity.FulfillmentShipment
	err := db.DB(ctx).Where("idempotency_key = ?", key).First(&shipment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fulfillmenterrors.ErrorFulfillmentNotFound
		}
		return nil, err
	}
	return &shipment, nil
}

func (r *ShipmentRepositoryImpl) UpdateStatusIfCurrent(
	ctx context.Context,
	id uint,
	from entity.ShipmentStatus,
	to entity.ShipmentStatus,
	patch map[string]any,
) (bool, error) {
	updates := map[string]any{"status": string(to)}
	for k, v := range patch {
		updates[k] = v
	}
	result := db.DB(ctx).
		Model(&entity.FulfillmentShipment{}).
		Where("id = ? AND status = ?", id, string(from)).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *ShipmentRepositoryImpl) FindRecoverableDrafts(
	ctx context.Context,
	limit int,
) ([]entity.FulfillmentShipment, error) {
	var shipments []entity.FulfillmentShipment
	err := db.DB(ctx).
		Where("status = ?", string(entity.SHIPMENT_STATUS_DRAFT)).
		Where(db.DB(ctx).Where("provider_order_id IS NULL")).
		Order("id ASC").
		Limit(limit).
		Find(&shipments).Error
	if err != nil {
		return nil, err
	}
	return shipments, nil
}

func (r *ShipmentRepositoryImpl) FindShipments(
	ctx context.Context,
	filter ShipmentFilter,
) ([]entity.FulfillmentShipment, error) {
	var shipments []entity.FulfillmentShipment
	err := applyShipmentFilter(db.DB(ctx), filter).
		Preload("Items").
		Order(sortClause(filter)).
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&shipments).Error
	if err != nil {
		return nil, err
	}
	return shipments, nil
}

func (r *ShipmentRepositoryImpl) CountShipments(
	ctx context.Context,
	filter ShipmentFilter,
) (int64, error) {
	var total int64
	err := applyShipmentFilter(db.DB(ctx).Model(&entity.FulfillmentShipment{}), filter).
		Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ShipmentRepositoryImpl) UpdateColumns(
	ctx context.Context,
	id uint,
	patch map[string]any,
) (bool, error) {
	result := db.DB(ctx).
		Model(&entity.FulfillmentShipment{}).
		Where("id = ?", id).
		Updates(patch)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// applyShipmentFilter scopes by seller plus optional filters.
func applyShipmentFilter(query *gorm.DB, filter ShipmentFilter) *gorm.DB {
	query = query.Where("seller_id = ?", filter.SellerID)
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.OrderID != 0 {
		query = query.Where("order_id = ?", filter.OrderID)
	}
	if filter.PickupLocationID != 0 {
		query = query.Where("pickup_location_id = ?", filter.PickupLocationID)
	}
	return query
}

// sortClause whitelists sort columns; unknown input falls back safely.
func sortClause(filter ShipmentFilter) string {
	column := "created_at"
	switch filter.SortBy {
	case "created_at", "updated_at", "status":
		column = filter.SortBy
	}
	order := "DESC"
	if strings.ToUpper(filter.SortOrder) == "ASC" {
		order = "ASC"
	}
	return column + " " + order
}

func (r *ShipmentRepositoryImpl) FindStaleInFlight(
	ctx context.Context,
	statuses []entity.ShipmentStatus,
	syncedBefore time.Time,
	limit int,
) ([]entity.FulfillmentShipment, error) {
	names := make([]string, 0, len(statuses))
	for _, s := range statuses {
		names = append(names, string(s))
	}
	var shipments []entity.FulfillmentShipment
	err := db.DB(ctx).
		Where("status IN ?", names).
		Where("last_synced_at IS NULL OR last_synced_at < ?", syncedBefore).
		Order("last_synced_at ASC NULLS FIRST").
		Limit(limit).
		Find(&shipments).Error
	if err != nil {
		return nil, err
	}
	return shipments, nil
}
