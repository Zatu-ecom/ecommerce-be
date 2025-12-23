package db

import (
	"context"

	"gorm.io/gorm"
)

// txKey is the context key for storing transaction
type txKey struct{}

// DB returns transaction if in transaction, otherwise connection pool
// Use this in all repository methods to get the correct *gorm.DB
//
// Example:
//
//	func (r *ProductRepository) Create(ctx context.Context, product *entity.Product) error {
//	    return db.DB(ctx).Create(product).Error
//	}
func DB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return GetDB()
}

// IsInTransaction checks if the context is currently in a transaction
func IsInTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(txKey{}).(*gorm.DB)
	return ok
}

// WithTransaction executes fn within a database transaction
// The transaction is stored in context and can be retrieved via GetDBFromContext
// Supports nested calls - if already in a transaction, reuses the existing one
//
// Example:
//
//	err := db.WithTransaction(ctx, func(ctx context.Context) error {
//	    if err := s.orderRepo.Create(ctx, order); err != nil {
//	        return err // Rollback
//	    }
//	    if err := s.inventoryService.Reserve(ctx, items); err != nil {
//	        return err // Rollback
//	    }
//	    return nil // Commit
//	})
func WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Check if already in transaction (nested call)
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx) // Reuse existing transaction
	}

	// Start new transaction
	return GetDB().Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

// WithTransactionResult executes fn within a transaction and returns a result
// Use when you need to return a value from the transaction
//
// Example:
//
//	order, err := db.WithTransactionResult(ctx, func(ctx context.Context) (*entity.Order, error) {
//	    order := &entity.Order{...}
//	    if err := s.orderRepo.Create(ctx, order); err != nil {
//	        return nil, err
//	    }
//	    return order, nil
//	})
func WithTransactionResult[T any](
	ctx context.Context,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	var result T

	// Check if already in transaction (nested call)
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}

	// Start new transaction
	err := GetDB().Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		var fnErr error
		result, fnErr = fn(txCtx)
		return fnErr
	})

	return result, err
}
