package service

import (
	"context"

	"ecommerce-be/common/helper"
	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
)

// InventoryTransactionService defines the interface for transaction operations
type InventoryTransactionService interface {
	// CreateTransaction creates a single inventory transaction
	// Uses CreateTransactionBatch under the hood
	CreateTransaction(
		ctx context.Context,
		params model.CreateTransactionParams,
	) (*entity.InventoryTransaction, error)

	// CreateTransactionBatch creates multiple inventory transactions in batch
	CreateTransactionBatch(
		ctx context.Context,
		params []model.CreateTransactionParams,
	) ([]*entity.InventoryTransaction, error)

	// ListTransactions retrieves transactions based on filters
	// TODO: Implement filtering and pagination in a future PR
	ListTransactions(
		ctx context.Context,
		filter model.ListTransactionsFilter,
	) (*model.ListTransactionsResponse, error)
}

// InventoryTransactionServiceImpl implements the InventoryTransactionService interface
type InventoryTransactionServiceImpl struct {
	transactionRepo repository.InventoryTransactionRepository
}

// NewInventoryTransactionService creates a new instance of InventoryTransactionService
func NewInventoryTransactionService(
	transactionRepo repository.InventoryTransactionRepository,
) InventoryTransactionService {
	return &InventoryTransactionServiceImpl{
		transactionRepo: transactionRepo,
	}
}

// ============================================================================
// Transaction Creation
// ============================================================================

// CreateTransaction creates a single inventory transaction
// Uses CreateTransactionBatch under the hood for consistency
func (s *InventoryTransactionServiceImpl) CreateTransaction(
	ctx context.Context,
	params model.CreateTransactionParams,
) (*entity.InventoryTransaction, error) {
	transactions, err := s.CreateTransactionBatch(ctx, []model.CreateTransactionParams{params})
	if err != nil {
		return nil, err
	}
	if len(transactions) == 0 {
		return nil, nil
	}
	return transactions[0], nil
}

// CreateTransactionBatch creates multiple inventory transactions in batch
func (s *InventoryTransactionServiceImpl) CreateTransactionBatch(
	ctx context.Context,
	params []model.CreateTransactionParams,
) ([]*entity.InventoryTransaction, error) {
	if len(params) == 0 {
		return []*entity.InventoryTransaction{}, nil
	}

	transactions := s.buildTransactionsFromParams(params)

	if err := s.transactionRepo.CreateBatch(transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}

// buildTransactionsFromParams converts params to transaction entities
func (s *InventoryTransactionServiceImpl) buildTransactionsFromParams(
	params []model.CreateTransactionParams,
) []*entity.InventoryTransaction {
	transactions := make([]*entity.InventoryTransaction, len(params))

	for i, p := range params {
		transactions[i] = &entity.InventoryTransaction{
			InventoryID:    p.InventoryID,
			Type:           p.TransactionType,
			Quantity:       p.QuantityChange,
			BeforeQuantity: p.BeforeQuantity,
			AfterQuantity:  p.AfterQuantity,
			PerformedBy:    p.PerformedBy,
			ReferenceID:    p.Reference,
			ReferenceType:  helper.StringPtr(p.ReferenceType),
			Reason:         p.Reason,
			Note:           p.Note,
		}
	}

	return transactions
}

// ============================================================================
// Transaction Listing (Stub for future implementation)
// ============================================================================

// ListTransactions retrieves transactions based on filters
// TODO: Implement filtering and pagination in a future PR
func (s *InventoryTransactionServiceImpl) ListTransactions(
	ctx context.Context,
	filter model.ListTransactionsFilter,
) (*model.ListTransactionsResponse, error) {
	// For now, return empty response
	// This will be fully implemented when filters are defined
	return &model.ListTransactionsResponse{
		Transactions: []entity.InventoryTransaction{},
		Total:        0,
		Page:         filter.Page,
		PageSize:     filter.PageSize,
	}, nil
}
