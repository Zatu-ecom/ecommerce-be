package repository

import (
	"ecommerce-be/inventory/entity"

	"gorm.io/gorm"
)

// InventoryTransactionRepository defines the interface for inventory transaction operations
type InventoryTransactionRepository interface {
	Create(transaction *entity.InventoryTransaction) error
	CreateBatch(transactions []*entity.InventoryTransaction) error
	FindByInventoryID(inventoryID uint) ([]entity.InventoryTransaction, error)
	FindByReferenceID(referenceID string) ([]entity.InventoryTransaction, error)
}

// InventoryTransactionRepositoryImpl implements the InventoryTransactionRepository interface
type InventoryTransactionRepositoryImpl struct {
	db *gorm.DB
}

// NewInventoryTransactionRepository creates a new instance of InventoryTransactionRepository
func NewInventoryTransactionRepository(db *gorm.DB) InventoryTransactionRepository {
	return &InventoryTransactionRepositoryImpl{db: db}
}

// Create creates a new inventory transaction record
func (r *InventoryTransactionRepositoryImpl) Create(
	transaction *entity.InventoryTransaction,
) error {
	return r.db.Create(transaction).Error
}

// CreateBatch creates multiple inventory transaction records in a single query
func (r *InventoryTransactionRepositoryImpl) CreateBatch(
	transactions []*entity.InventoryTransaction,
) error {
	if len(transactions) == 0 {
		return nil
	}
	return r.db.Create(transactions).Error
}

// FindByInventoryID finds all transactions for a specific inventory
func (r *InventoryTransactionRepositoryImpl) FindByInventoryID(
	inventoryID uint,
) ([]entity.InventoryTransaction, error) {
	var transactions []entity.InventoryTransaction
	result := r.db.Where("inventory_id = ?", inventoryID).
		Order("created_at DESC").
		Find(&transactions)

	if result.Error != nil {
		return nil, result.Error
	}
	return transactions, nil
}

// FindByReferenceID finds all transactions with a specific reference ID
func (r *InventoryTransactionRepositoryImpl) FindByReferenceID(
	referenceID string,
) ([]entity.InventoryTransaction, error) {
	var transactions []entity.InventoryTransaction
	result := r.db.Where("reference_id = ?", referenceID).
		Order("created_at DESC").
		Find(&transactions)

	if result.Error != nil {
		return nil, result.Error
	}
	return transactions, nil
}
