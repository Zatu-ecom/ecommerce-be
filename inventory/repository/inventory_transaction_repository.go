package repository

import (
	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/model"

	"gorm.io/gorm"
)

// InventoryTransactionRepository defines the interface for inventory transaction operations
type InventoryTransactionRepository interface {
	Create(transaction *entity.InventoryTransaction) error
	CreateBatch(transactions []*entity.InventoryTransaction) error
	FindByInventoryID(inventoryID uint) ([]entity.InventoryTransaction, error)
	FindByReferenceID(referenceID string) ([]entity.InventoryTransaction, error)
	FindByFilter(filter model.ListTransactionsFilter) ([]entity.InventoryTransaction, int64, error)
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

// FindByFilter finds transactions based on filter criteria with pagination
func (r *InventoryTransactionRepositoryImpl) FindByFilter(
	filter model.ListTransactionsFilter,
) ([]entity.InventoryTransaction, int64, error) {
	var transactions []entity.InventoryTransaction
	var total int64

	query := r.db.Model(&entity.InventoryTransaction{})

	// Join with inventory table to filter by variant/location and for seller isolation
	query = query.Joins("JOIN inventory ON inventory.id = inventory_transaction.inventory_id")
	query = query.Joins("JOIN location ON location.id = inventory.location_id")

	// Seller isolation - filter by locations belonging to seller
	if filter.SellerID > 0 {
		query = query.Where("location.seller_id = ?", filter.SellerID)
	}

	// Filter by inventory IDs
	if len(filter.InventoryIDs) > 0 {
		query = query.Where("inventory_transaction.inventory_id IN ?", filter.InventoryIDs)
	}

	// Filter by variant IDs (through inventory)
	if len(filter.VariantIDs) > 0 {
		query = query.Where("inventory.variant_id IN ?", filter.VariantIDs)
	}

	// Filter by location IDs (through inventory)
	if len(filter.LocationIDs) > 0 {
		query = query.Where("inventory.location_id IN ?", filter.LocationIDs)
	}

	// Filter by transaction types
	if len(filter.Types) > 0 {
		query = query.Where("inventory_transaction.type IN ?", filter.Types)
	}

	// Filter by reference ID
	if filter.ReferenceID != nil {
		query = query.Where("inventory_transaction.reference_id = ?", *filter.ReferenceID)
	}

	// Filter by reference type
	if filter.ReferenceType != nil {
		query = query.Where("inventory_transaction.reference_type = ?", *filter.ReferenceType)
	}

	// Filter by performed by
	if filter.PerformedBy != nil {
		query = query.Where("inventory_transaction.performed_by = ?", *filter.PerformedBy)
	}

	// Filter by date range
	if filter.CreatedFrom != nil {
		query = query.Where("inventory_transaction.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("inventory_transaction.created_at <= ?", *filter.CreatedTo)
	}

	// Count total before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortColumn := "inventory_transaction." + filter.SortBy
	sortOrder := filter.SortOrder
	query = query.Order(sortColumn + " " + sortOrder)

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Select only transaction fields to avoid ambiguity
	query = query.Select("inventory_transaction.*")

	if err := query.Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
