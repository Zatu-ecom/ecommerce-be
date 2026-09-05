package repository

import (
	"context"

	"ecommerce-be/common/db"
	"ecommerce-be/payment/entity"
)

// PaymentTransactionEventRepository handles append-only writes to the event ledger.
type PaymentTransactionEventRepository interface {
	Create(ctx context.Context, event *entity.PaymentTransactionEvent) error
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
