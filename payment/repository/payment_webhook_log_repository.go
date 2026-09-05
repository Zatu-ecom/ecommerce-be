package repository

import (
	"context"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/payment/entity"
)

// PaymentWebhookLogRepository handles data access for the webhook audit trail.
type PaymentWebhookLogRepository interface {
	Create(ctx context.Context, log *entity.PaymentWebhookLog) error
	MarkProcessed(ctx context.Context, id uint, transactionID, refundID *uint) error
	MarkIgnored(ctx context.Context, id uint) error
	MarkFailed(ctx context.Context, id uint, errorMessage string) error
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
	errorMessage string,
) error {
	return db.DB(ctx).
		Model(&entity.PaymentWebhookLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        entity.WebhookStatusFailed,
			"error_message": errorMessage,
			"processed_at":  time.Now().UTC(),
		}).
		Error
}
