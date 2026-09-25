package repository

import (
	"context"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/common/helper"
	"ecommerce-be/payment/entity"
)

// SellerWebhookLog is one seller-scoped webhook delivery: the log row plus the
// linked transaction's public id (resolved by join, no N+1).
type SellerWebhookLog struct {
	entity.PaymentWebhookLog `gorm:"embedded"`
	PublicTransactionID      string `gorm:"column:public_transaction_id"`
}

// PaymentWebhookLogRepository handles data access for the webhook audit trail.
type PaymentWebhookLogRepository interface {
	Create(ctx context.Context, log *entity.PaymentWebhookLog) error
	MarkProcessed(ctx context.Context, id uint, transactionID, refundID *uint) error
	MarkIgnored(ctx context.Context, id uint) error
	// MarkFailed records a failed apply. Transaction/refund linkage is set
	// whenever the transaction was located (even on failure) so sellers and
	// operators can trace the failed delivery.
	MarkFailed(ctx context.Context, id uint, transactionID, refundID *uint, errorMessage string) error
	// FindBySellerID lists a seller's webhook deliveries (paginated, newest
	// first). Only logs linked to the seller's transactions are visible;
	// unmatched deliveries (NULL transaction) are never listed. Optional
	// status/eventType narrow the result; pageSize caps at 100.
	FindBySellerID(
		ctx context.Context,
		sellerID uint,
		page, pageSize int,
		status, eventType string,
	) ([]SellerWebhookLog, int64, error)
}

type PaymentWebhookLogRepositoryImpl struct{}

func NewPaymentWebhookLogRepository() PaymentWebhookLogRepository {
	return &PaymentWebhookLogRepositoryImpl{}
}

func (r *PaymentWebhookLogRepositoryImpl) Create(
	ctx context.Context,
	log *entity.PaymentWebhookLog,
) error {
	return db.DB(ctx).Create(log).Error
}

func (r *PaymentWebhookLogRepositoryImpl) MarkProcessed(
	ctx context.Context,
	id uint,
	transactionID, refundID *uint,
) error {
	now := time.Now().UTC()
	return db.DB(ctx).
		Model(&entity.PaymentWebhookLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         entity.WebhookStatusProcessed,
			"processed_at":   now,
			"transaction_id": transactionID,
			"refund_id":      refundID,
		}).
		Error
}

func (r *PaymentWebhookLogRepositoryImpl) MarkIgnored(
	ctx context.Context,
	id uint,
) error {
	return db.DB(ctx).
		Model(&entity.PaymentWebhookLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       entity.WebhookStatusIgnored,
			"processed_at": time.Now().UTC(),
		}).
		Error
}

func (r *PaymentWebhookLogRepositoryImpl) MarkFailed(
	ctx context.Context,
	id uint,
	transactionID, refundID *uint,
	errorMessage string,
) error {
	return db.DB(ctx).
		Model(&entity.PaymentWebhookLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         entity.WebhookStatusFailed,
			"error_message":  errorMessage,
			"processed_at":   time.Now().UTC(),
			"transaction_id": transactionID,
			"refund_id":      refundID,
		}).
		Error
}

func (r *PaymentWebhookLogRepositoryImpl) FindBySellerID(
	ctx context.Context,
	sellerID uint,
	page, pageSize int,
	status, eventType string,
) ([]SellerWebhookLog, int64, error) {
	var logs []SellerWebhookLog
	var total int64

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := db.DB(ctx).Model(&entity.PaymentWebhookLog{}).
		Joins("JOIN payment_transaction AS txn ON txn.id = payment_webhook_log.transaction_id").
		Where("txn.seller_id = ?", sellerID)
	if status != "" {
		query = query.Where("payment_webhook_log.status = ?", status)
	}
	if eventType != "" {
		query = query.Where("payment_webhook_log.event_type = ?", eventType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := helper.CalculateOffset(page, pageSize)
	err := query.
		Select("payment_webhook_log.*, txn.transaction_id AS public_transaction_id").
		Order("payment_webhook_log.created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Scan(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
