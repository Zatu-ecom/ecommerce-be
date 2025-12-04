package repository

import (
	"errors"

	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"

	"gorm.io/gorm"
)

// InventoryRepository defines the interface for inventory-related database operations
type InventoryRepository interface {
	FindByVariantAndLocation(variantID uint, locationID uint) (*entity.Inventory, error)
	FindByVariantAndLocationBatch(variantIDs []uint, locationIDs []uint) ([]entity.Inventory, error)
	FindByIDs(ids []uint) ([]entity.Inventory, error)
	Create(inventory *entity.Inventory) error
	CreateBatch(inventories []*entity.Inventory) error
	Update(inventory *entity.Inventory) error
	UpdateBatch(inventories []*entity.Inventory) error
	FindByVariantID(variantID uint) ([]entity.Inventory, error)
	FindByLocationID(locationID uint) ([]entity.Inventory, error)
	Exists(variantID uint, locationID uint) (bool, error)
}

// InventoryRepositoryImpl implements the InventoryRepository interface
type InventoryRepositoryImpl struct {
	db *gorm.DB
}

// NewInventoryRepository creates a new instance of InventoryRepository
func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &InventoryRepositoryImpl{db: db}
}

// FindByVariantAndLocation finds inventory for a specific variant at a specific location
func (r *InventoryRepositoryImpl) FindByVariantAndLocation(
	variantID uint,
	locationID uint,
) (*entity.Inventory, error) {
	var inventory entity.Inventory
	result := r.db.Where("variant_id = ? AND location_id = ?", variantID, locationID).
		First(&inventory)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, invErrors.ErrInventoryNotFound
		}
		return nil, result.Error
	}
	return &inventory, nil
}

// Create creates a new inventory record
func (r *InventoryRepositoryImpl) Create(inventory *entity.Inventory) error {
	return r.db.Create(inventory).Error
}

// CreateBatch creates multiple inventory records in a single query
func (r *InventoryRepositoryImpl) CreateBatch(inventories []*entity.Inventory) error {
	if len(inventories) == 0 {
		return nil
	}
	return r.db.Create(inventories).Error
}

// Update updates an existing inventory record
func (r *InventoryRepositoryImpl) Update(inventory *entity.Inventory) error {
	return r.db.Save(inventory).Error
}

// UpdateBatch updates multiple inventory records in a single transaction
func (r *InventoryRepositoryImpl) UpdateBatch(inventories []*entity.Inventory) error {
	if len(inventories) == 0 {
		return nil
	}
	for _, inv := range inventories {
		if err := r.db.Save(inv).Error; err != nil {
			return err
		}
	}
	return nil
}

// FindByVariantID finds all inventory records for a variant across all locations
func (r *InventoryRepositoryImpl) FindByVariantID(variantID uint) ([]entity.Inventory, error) {
	var inventories []entity.Inventory
	result := r.db.Where("variant_id = ?", variantID).Find(&inventories)
	if result.Error != nil {
		return nil, result.Error
	}
	return inventories, nil
}

// FindByLocationID finds all inventory records at a specific location
func (r *InventoryRepositoryImpl) FindByLocationID(locationID uint) ([]entity.Inventory, error) {
	var inventories []entity.Inventory
	result := r.db.Where("location_id = ?", locationID).Find(&inventories)
	if result.Error != nil {
		return nil, result.Error
	}
	return inventories, nil
}

// FindByVariantAndLocationBatch finds multiple inventory records in one query (bulk query - avoids N+1)
func (r *InventoryRepositoryImpl) FindByVariantAndLocationBatch(
	variantIDs []uint,
	locationIDs []uint,
) ([]entity.Inventory, error) {
	var inventories []entity.Inventory
	if len(variantIDs) == 0 || len(locationIDs) == 0 {
		return inventories, nil
	}
	
	result := r.db.Where("variant_id IN ? AND location_id IN ?", variantIDs, locationIDs).
		Find(&inventories)
	if result.Error != nil {
		return nil, result.Error
	}
	return inventories, nil
}

// Exists checks if inventory exists for a variant at a location
func (r *InventoryRepositoryImpl) Exists(variantID uint, locationID uint) (bool, error) {
	var count int64
	result := r.db.Model(&entity.Inventory{}).
		Where("variant_id = ? AND location_id = ?", variantID, locationID).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// FindByIDs finds multiple inventory records by IDs (bulk query - avoids N+1)
func (r *InventoryRepositoryImpl) FindByIDs(ids []uint) ([]entity.Inventory, error) {
	var inventories []entity.Inventory
	if len(ids) == 0 {
		return inventories, nil
	}

	result := r.db.Where("id IN ?", ids).Find(&inventories)
	if result.Error != nil {
		return nil, result.Error
	}
	return inventories, nil
}
