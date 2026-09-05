package repository

import (
	"context"
	"errors"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/common/helper"
	"ecommerce-be/payment/entity"
	paymenterrors "ecommerce-be/payment/error"

	"gorm.io/gorm"
)

// PaymentTransactionRepository handles data access for payment transactions.
type PaymentTransactionRepository interface {
	Create(ctx context.Context, transaction *entity.PaymentTransaction) error
	FindByID(ctx context.Context, id uint) (*entity.PaymentTransaction, error)
	FindByTransactionID(ctx context.Context, transactionID string) (*entity.PaymentTransaction, error)
	FindByGatewaySessionID(ctx context.Context, sessionID string) (*entity.PaymentTransaction, error)
	FindByGatewayPaymentID(ctx context.Context, paymentID string) (*entity.PaymentTransaction, error)
	FindByReference(
		ctx context.Context,
		referenceType entity.ReferenceType,
		referenceID uint,
	) ([]entity.PaymentTransaction, error)
	FindBySellerID(
		ctx context.Context,
		sellerID uint,
		page, pageSize int,
	) ([]entity.PaymentTransaction, int64, error)
	// UpdateStatusIfCurrent conditionally transitions status; returns true if a row changed.
	UpdateStatusIfCurrent(
		ctx context.Context,
		id uint,
		from, to entity.TransactionStatus,
		patch map[string]any,
	) (bool, error)
}

type PaymentTransactionRepositoryImpl struct{}

func NewPaymentTransactionRepository() PaymentTransactionRepository {
	return &PaymentTransactionRepositoryImpl{}
}

func (r *PaymentTransactionRepositoryImpl) Create(
	ctx context.Context,
	transaction *entity.PaymentTransaction,
) error {
	return db.DB(ctx).Create(transaction).Error
}

func (r *PaymentTransactionRepositoryImpl) FindByID(
	ctx context.Context,
	id uint,
) (*entity.PaymentTransaction, error) {
	var t entity.PaymentTransaction
	err := db.DB(ctx).Where("id = ?", id).First(&t).Error
	if err != nil {
		return nil, r.mapNotFound(err)
	}
	return &t, nil
}

func (r *PaymentTransactionRepositoryImpl) FindByTransactionID(
	ctx context.Context,
	transactionID string,
) (*entity.PaymentTransaction, error) {
	var t entity.PaymentTransaction
	err := db.DB(ctx).Where("transaction_id = ?", transactionID).First(&t).Error
	if err != nil {
		return nil, r.mapNotFound(err)
	}
	return &t, nil
}

func (r *PaymentTransactionRepositoryImpl) FindByGatewaySessionID(
	ctx context.Context,
	sessionID string,
) (*entity.PaymentTransaction, error) {
	var t entity.PaymentTransaction
	err := db.DB(ctx).Where("gateway_session_id = ?", sessionID).First(&t).Error
	if err != nil {
		return nil, r.mapNotFound(err)
	}
	return &t, nil
}

func (r *PaymentTransactionRepositoryImpl) FindByGatewayPaymentID(
	ctx context.Context,
	paymentID string,
) (*entity.PaymentTransaction, error) {
	var t entity.PaymentTransaction
	err := db.DB(ctx).Where("gateway_payment_id = ?", paymentID).First(&t).Error
	if err != nil {
		return nil, r.mapNotFound(err)
	}
	return &t, nil
}

func (r *PaymentTransactionRepositoryImpl) FindByReference(
	ctx context.Context,
	referenceType entity.ReferenceType,
	referenceID uint,
) ([]entity.PaymentTransaction, error) {
	var txns []entity.PaymentTransaction
	err := db.DB(ctx).
		Where("reference_type = ? AND reference_id = ?", referenceType, referenceID).
		Order("created_at DESC").
		Find(&txns).Error
	if err != nil {
		return nil, err
	}
	return txns, nil
}

func (r *PaymentTransactionRepositoryImpl) FindBySellerID(
	ctx context.Context,
	sellerID uint,
	page, pageSize int,
) ([]entity.PaymentTransaction, int64, error) {
	var transactions []entity.PaymentTransaction
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

	query := db.DB(ctx).Model(&entity.PaymentTransaction{}).
		Where("seller_id = ?", sellerID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := helper.CalculateOffset(page, pageSize)
	err := query.
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&transactions).Error
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// UpdateStatusIfCurrent transitions status only when the current row state matches `from`,
// preventing duplicate/late webhooks from regressing a terminal state.
func (r *PaymentTransactionRepositoryImpl) UpdateStatusIfCurrent(
	ctx context.Context,
	id uint,
	from, to entity.TransactionStatus,
	patch map[string]any,
) (bool, error) {
	if patch == nil {
		patch = map[string]any{}
	}
	patch["status"] = to
	patch["updated_at"] = time.Now().UTC()

	result := db.DB(ctx).
		Model(&entity.PaymentTransaction{}).
		Where("id = ? AND status = ?", id, from).
		Updates(patch)

	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (r *PaymentTransactionRepositoryImpl) mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return paymenterrors.ErrorPaymentTransactionNotFound
	}
	return err
}
