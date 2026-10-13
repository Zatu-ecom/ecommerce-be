package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/payment/entity"
)

// PaymentTransactionEventRepository handles append-only writes to the event ledger.
type PaymentTransactionEventRepository interface {
	Create(ctx context.Context, event *entity.PaymentTransactionEvent) error
	// ListByTransactionID returns the full history ordered oldest-first.
	// Used ONLY by GET transaction detail — never by list or hot paths.
	ListByTransactionID(ctx context.Context, transactionID uint) ([]entity.PaymentTransactionEvent, error)
}

type PaymentTransactionEventRepositoryImpl struct{}

func NewPaymentTransactionEventRepository() PaymentTransactionEventRepository {
	return &PaymentTransactionEventRepositoryImpl{}
}

func (r *PaymentTransactionEventRepositoryImpl) Create(
	ctx context.Context,
	event *entity.PaymentTransactionEvent,
) error {
	return db.DB(ctx).Create(event).Error
}

func (r *PaymentTransactionEventRepositoryImpl) ListByTransactionID(
	ctx context.Context,
	transactionID uint,
) ([]entity.PaymentTransactionEvent, error) {
	var events []entity.PaymentTransactionEvent
	err := db.DB(ctx).
		Where("transaction_id = ?", transactionID).
		Order("created_at ASC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}
