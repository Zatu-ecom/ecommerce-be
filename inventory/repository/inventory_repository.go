package repository

import (
	"errors"

	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
	"ecommerce-be/inventory/mapper"

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
	GetLocationInventorySummary(locationID uint) (*mapper.LocationInventorySummaryAggregate, error)
	GetLocationInventorySummaryBatch(
		locationIDs []uint,
	) (map[uint]*mapper.LocationInventorySummaryAggregate, error)
	GetVariantInventoriesAtLocation(locationID uint) ([]mapper.VariantInventoryRow, error)
	// GetVariantInventoriesAtLocationWithSort retrieves variant inventories with sorting
	// sortBy: quantity, reserved_quantity, threshold, available_quantity (computed)
	// sortOrder: asc, desc
	GetVariantInventoriesAtLocationWithSort(
		locationID uint,
		variantIDs []uint,
		sortBy string,
		sortOrder string,
	) ([]mapper.VariantInventoryRow, error)
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

// GetLocationInventorySummary aggregates inventory statistics for a location
// Returns: LocationInventorySummaryAggregate with all metrics
// Microservice-ready: Only queries inventory table, no product joins
func (r *InventoryRepositoryImpl) GetLocationInventorySummary(
	locationID uint,
) (*mapper.LocationInventorySummaryAggregate, error) {
	var result mapper.LocationInventorySummaryAggregate

	// Single aggregation query - no N+1 problem
	err := r.db.Model(&entity.Inventory{}).
		Select(
			"COUNT(DISTINCT variant_id) as variant_count",
			"COALESCE(SUM(quantity), 0) as total_stock",
			"COALESCE(SUM(reserved_quantity), 0) as total_reserved",
			"COUNT(CASE WHEN quantity > 0 AND quantity <= threshold THEN 1 END) as low_stock_count",
			"COUNT(CASE WHEN quantity = 0 THEN 1 END) as out_of_stock_count",
		).
		Where("location_id = ?", locationID).
		Scan(&result).
		Error
	if err != nil {
		return nil, err
	}

	// Get distinct variant IDs for product count calculation
	err = r.db.Model(&entity.Inventory{}).
		Select("DISTINCT variant_id").
		Where("location_id = ?", locationID).
		Pluck("variant_id", &result.VariantIDs).
		Error
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetLocationInventorySummaryBatch aggregates inventory statistics for multiple locations
// Efficient batch query - avoids N+1 problem by fetching all locations at once
// Returns: map[locationID] -> LocationInventorySummaryAggregate
func (r *InventoryRepositoryImpl) GetLocationInventorySummaryBatch(
	locationIDs []uint,
) (map[uint]*mapper.LocationInventorySummaryAggregate, error) {
	if len(locationIDs) == 0 {
		return make(map[uint]*mapper.LocationInventorySummaryAggregate), nil
	}

	// Batch aggregation query for all locations
	type AggregateResult struct {
		LocationID      uint
		VariantCount    uint
		TotalStock      uint
		TotalReserved   uint
		LowStockCount   uint
		OutOfStockCount uint
	}

	var results []AggregateResult
	err := r.db.Model(&entity.Inventory{}).
		Select(
			"location_id",
			"COUNT(DISTINCT variant_id) as variant_count",
			"COALESCE(SUM(quantity), 0) as total_stock",
			"COALESCE(SUM(reserved_quantity), 0) as total_reserved",
			"COUNT(CASE WHEN quantity > 0 AND quantity <= threshold THEN 1 END) as low_stock_count",
			"COUNT(CASE WHEN quantity = 0 THEN 1 END) as out_of_stock_count",
		).
		Where("location_id IN ?", locationIDs).
		Group("location_id").
		Scan(&results).
		Error
	if err != nil {
		return nil, err
	}

	// Batch fetch variant IDs for all locations
	type VariantIDResult struct {
		LocationID uint
		VariantID  uint
	}

	var variantResults []VariantIDResult
	err = r.db.Model(&entity.Inventory{}).
		Select("DISTINCT location_id, variant_id").
		Where("location_id IN ?", locationIDs).
		Scan(&variantResults).
		Error
	if err != nil {
		return nil, err
	}

	// Group variant IDs by location
	variantIDsByLocation := make(map[uint][]uint)
	for _, vr := range variantResults {
		variantIDsByLocation[vr.LocationID] = append(
			variantIDsByLocation[vr.LocationID],
			vr.VariantID,
		)
	}

	// Build result map
	summaryMap := make(map[uint]*mapper.LocationInventorySummaryAggregate)
	for _, aggResult := range results {
		summaryMap[aggResult.LocationID] = &mapper.LocationInventorySummaryAggregate{
			VariantCount:    aggResult.VariantCount,
			TotalStock:      aggResult.TotalStock,
			TotalReserved:   aggResult.TotalReserved,
			LowStockCount:   aggResult.LowStockCount,
			OutOfStockCount: aggResult.OutOfStockCount,
			VariantIDs:      variantIDsByLocation[aggResult.LocationID],
		}
	}

	// Add empty results for locations with no inventory
	for _, locationID := range locationIDs {
		if _, exists := summaryMap[locationID]; !exists {
			summaryMap[locationID] = &mapper.LocationInventorySummaryAggregate{
				VariantIDs: []uint{},
			}
		}
	}

	return summaryMap, nil
}

// GetVariantInventoriesAtLocation retrieves all variant inventory records at a location
// Returns flat list of variant inventory data for aggregation by product
// Microservice-ready: Only queries inventory table, no product joins
func (r *InventoryRepositoryImpl) GetVariantInventoriesAtLocation(
	locationID uint,
) ([]mapper.VariantInventoryRow, error) {
	var results []mapper.VariantInventoryRow

	err := r.db.Model(&entity.Inventory{}).
		Select(
			"variant_id",
			"quantity",
			"reserved_quantity",
			"threshold",
			"bin_location",
		).
		Where("location_id = ?", locationID).
		Scan(&results).
		Error
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil for consistency
	if results == nil {
		results = []mapper.VariantInventoryRow{}
	}

	return results, nil
}

// GetVariantInventoriesAtLocationWithSort retrieves variant inventories with DB-side sorting
func (r *InventoryRepositoryImpl) GetVariantInventoriesAtLocationWithSort(
	locationID uint,
	variantIDs []uint,
	sortBy string,
	sortOrder string,
) ([]mapper.VariantInventoryRow, error) {
	var results []mapper.VariantInventoryRow

	// Map sortBy to actual DB column names
	columnMap := map[string]string{
		"quantity":          "quantity",
		"reservedQuantity":  "reserved_quantity",
		"threshold":         "threshold",
		"availableQuantity": "(quantity - reserved_quantity)",
	}

	// Default sorting
	orderColumn := "quantity"
	if col, ok := columnMap[sortBy]; ok {
		orderColumn = col
	}

	// Validate sort order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	query := r.db.Model(&entity.Inventory{}).
		Select(
			"variant_id",
			"quantity",
			"reserved_quantity",
			"threshold",
			"bin_location",
		).
		Where("location_id = ?", locationID)

	// Filter by variant IDs if provided
	if len(variantIDs) > 0 {
		query = query.Where("variant_id IN ?", variantIDs)
	}

	// Apply ordering
	query = query.Order(orderColumn + " " + sortOrder)

	err := query.Scan(&results).Error
	if err != nil {
		return nil, err
	}

	if results == nil {
		results = []mapper.VariantInventoryRow{}
	}

	return results, nil
}
