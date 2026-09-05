package repository

import (
	"context"
	"errors"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"

	"gorm.io/gorm"
)

// PaymentRefundRepository handles data access for payment refunds.
type PaymentRefundRepository interface {
	Create(ctx context.Context, refund *entity.PaymentRefund) error
	FindByRefundID(ctx context.Context, refundID string) (*entity.PaymentRefund, error)
	FindByGatewayRefundID(ctx context.Context, gatewayRefundID string) (*entity.PaymentRefund, error)
	SumByTransactionAndStatuses(
		ctx context.Context,
		transactionID uint,
		statuses []entity.RefundStatus,
		sum *int64,
	) error
	UpdateStatusIfCurrent(
		ctx context.Context,
		id uint,
		from, to entity.RefundStatus,
		patch map[string]any,
	) (bool, error)
}

type PaymentRefundRepositoryImpl struct{}

func NewPaymentRefundRepository() PaymentRefundRepository {
	return &PaymentRefundRepositoryImpl{}
}

func (r *PaymentRefundRepositoryImpl) Create(
	ctx context.Context,
	refund *entity.PaymentRefund,
) error {
	return db.DB(ctx).Create(refund).Error
}

func (r *PaymentRefundRepositoryImpl) FindByRefundID(
	ctx context.Context,
	refundID string,
) (*entity.PaymentRefund, error) {
	var refund entity.PaymentRefund
	err := db.DB(ctx).Where("refund_id = ?", refundID).First(&refund).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paymenterrors.ErrorRefundNotFound
		}
		return nil, err
	}
	return &refund, nil
}

func (r *PaymentRefundRepositoryImpl) FindByGatewayRefundID(
	ctx context.Context,
	gatewayRefundID string,
) (*entity.PaymentRefund, error) {
	var refund entity.PaymentRefund
	err := db.DB(ctx).Where("gateway_refund_id = ?", gatewayRefundID).First(&refund).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, paymenterrors.ErrorRefundNotFound
		}
		return nil, err
	}
	return &refund, nil
}

func (r *PaymentRefundRepositoryImpl) SumByTransactionAndStatuses(
	ctx context.Context,
	transactionID uint,
	statuses []entity.RefundStatus,
	sum *int64,
) error {
	var result int64
	err := db.DB(ctx).
		Model(&entity.PaymentRefund{}).
		Where("transaction_id = ? AND status IN ?", transactionID, statuses).
		Select("COALESCE(SUM(amount_cents), 0)").
		Scan(&result).Error
	if err != nil {
		return err
	}
	*sum = result
	return nil
}

func (r *PaymentRefundRepositoryImpl) UpdateStatusIfCurrent(
	ctx context.Context,
	id uint,
	from, to entity.RefundStatus,
	patch map[string]any,
) (bool, error) {
	if patch == nil {
		patch = map[string]any{}
	}
	patch["status"] = to
	patch["updated_at"] = time.Now().UTC()

	result := db.DB(ctx).
		Model(&entity.PaymentRefund{}).
		Where("id = ? AND status = ?", id, from).
		Updates(patch)

	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}
