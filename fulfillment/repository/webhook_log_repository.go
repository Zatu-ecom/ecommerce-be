package repository

import (
	"context"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/fulfillment/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WebhookLogRepository records verified webhook deliveries. Unverifiable or
// unlocatable pushes are NEVER persisted (service rule); the UNIQUE
// (provider_code, event_id) index turns replays into no-ops at insert time.
type WebhookLogRepository interface {
	// Create inserts the received row. A unique violation means a replay:
	// the caller treats it as a no-op (returns nil, duplicate=true).
	Create(ctx context.Context, log *entity.FulfillmentWebhookLog) (duplicate bool, err error)
	MarkProcessed(ctx context.Context, id uint, shipmentID uint) error
	MarkFailed(ctx context.Context, id uint, errMsg string) error
	MarkIgnored(ctx context.Context, id uint) error
	FindFailedSince(ctx context.Context, since time.Time, limit int) ([]entity.FulfillmentWebhookLog, error)
	// FindBySeller lists verified rows linked to the seller's shipments
	// (unknown-AWB pushes were never stored, so they never appear).
	FindBySeller(ctx context.Context, filter WebhookLogFilter) ([]entity.FulfillmentWebhookLog, error)
	// CountBySeller counts the rows FindBySeller would return.
	CountBySeller(ctx context.Context, filter WebhookLogFilter) (int64, error)
}

// WebhookLogFilter scopes the seller log list. Zero values unfiltered;
// SortBy is created_at only (audit order).
type WebhookLogFilter struct {
	SellerID uint
	Status   string
	AWB      string
	Limit    int
	Offset   int
}

type WebhookLogRepositoryImpl struct{}

func NewWebhookLogRepository() WebhookLogRepository {
	return &WebhookLogRepositoryImpl{}
}

func (r *WebhookLogRepositoryImpl) Create(
	ctx context.Context,
	log *entity.FulfillmentWebhookLog,
) (bool, error) {
	// ON CONFLICT DO NOTHING: a replay must not even raise — Postgres
	// aborts the whole transaction on ANY error, so catching a unique
	// violation and continuing would still doom the commit. No row means
	// duplicate (log.ID stays zero).
	if err := db.DB(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(log).Error; err != nil {
		return false, err
	}
	return log.ID == 0, nil
}

func (r *WebhookLogRepositoryImpl) MarkProcessed(
	ctx context.Context,
	id uint,
	shipmentID uint,
) error {
	now := time.Now().UTC()
	return db.DB(ctx).
		Model(&entity.FulfillmentWebhookLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       "applied",
			"shipment_id":  shipmentID,
			"processed_at": now,
		}).Error
}

func (r *WebhookLogRepositoryImpl) MarkFailed(
	ctx context.Context,
	id uint,
	errMsg string,
) error {
	now := time.Now().UTC()
	return db.DB(ctx).
		Model(&entity.FulfillmentWebhookLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        "failed",
			"error_message": errMsg,
			"processed_at":  now,
		}).Error
}

func (r *WebhookLogRepositoryImpl) MarkIgnored(ctx context.Context, id uint) error {
	now := time.Now().UTC()
	return db.DB(ctx).
		Model(&entity.FulfillmentWebhookLog{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": "ignored", "processed_at": now}).Error
}

func (r *WebhookLogRepositoryImpl) FindFailedSince(
	ctx context.Context,
	since time.Time,
	limit int,
) ([]entity.FulfillmentWebhookLog, error) {
	var logs []entity.FulfillmentWebhookLog
	err := db.DB(ctx).
		Where("status = ? AND created_at >= ?", "failed", since).
		Order("created_at ASC").
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *WebhookLogRepositoryImpl) FindBySeller(
	ctx context.Context,
	filter WebhookLogFilter,
) ([]entity.FulfillmentWebhookLog, error) {
	var logs []entity.FulfillmentWebhookLog
	err := applyWebhookLogFilter(db.DB(ctx), filter).
		Order("fulfillment_webhook_log.created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *WebhookLogRepositoryImpl) CountBySeller(
	ctx context.Context,
	filter WebhookLogFilter,
) (int64, error) {
	var total int64
	err := applyWebhookLogFilter(db.DB(ctx).Model(&entity.FulfillmentWebhookLog{}), filter).
		Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

// applyWebhookLogFilter joins the shipment owner: only rows linked to the
// seller's boxes are visible (unlinked rows belong to no seller).
func applyWebhookLogFilter(query *gorm.DB, filter WebhookLogFilter) *gorm.DB {
	query = query.
		Joins("JOIN fulfillment_shipment ON fulfillment_shipment.id = fulfillment_webhook_log.shipment_id").
		Where("fulfillment_shipment.seller_id = ?", filter.SellerID)
	if filter.Status != "" {
		query = query.Where("fulfillment_webhook_log.status = ?", filter.Status)
	}
	if filter.AWB != "" {
		query = query.Where("fulfillment_webhook_log.awb = ?", filter.AWB)
	}
	return query
}
