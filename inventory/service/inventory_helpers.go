package service

import (
	"fmt"

	"ecommerce-be/common/helper"
	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	productModel "ecommerce-be/product/model"
	"ecommerce-be/product/service"
)

// ============================================================================
// Bulk Inventory Helper (Batch operations with dependencies)
// ============================================================================

// BulkInventoryHelper provides helper methods for bulk inventory operations
type BulkInventoryHelper struct {
	inventoryRepo       repository.InventoryRepository
	locationRepo        repository.LocationRepository
	variantQueryService service.VariantQueryService
}

// NewBulkInventoryHelper creates a new instance of BulkInventoryHelper
func NewBulkInventoryHelper(
	inventoryRepo repository.InventoryRepository,
	locationRepo repository.LocationRepository,
	variantQueryService service.VariantQueryService,
) *BulkInventoryHelper {
	return &BulkInventoryHelper{
		inventoryRepo:       inventoryRepo,
		locationRepo:        locationRepo,
		variantQueryService: variantQueryService,
	}
}

// ============================================================================
// ID Extraction Helpers
// ============================================================================

// ExtractUniqueLocationIDs extracts unique location IDs from bulk request
func (h *BulkInventoryHelper) ExtractUniqueLocationIDs(
	items []model.ManageInventoryRequest,
) []uint {
	locationMap := make(map[uint]bool)
	for _, item := range items {
		locationMap[item.LocationID] = true
	}

	locationIDs := make([]uint, 0, len(locationMap))
	for id := range locationMap {
		locationIDs = append(locationIDs, id)
	}
	return locationIDs
}

// ExtractUniqueVariantIDs extracts unique variant IDs from bulk request
func (h *BulkInventoryHelper) ExtractUniqueVariantIDs(items []model.ManageInventoryRequest) []uint {
	variantMap := make(map[uint]bool)
	for _, item := range items {
		variantMap[item.VariantID] = true
	}

	variantIDs := make([]uint, 0, len(variantMap))
	for id := range variantMap {
		variantIDs = append(variantIDs, id)
	}
	return variantIDs
}

// ============================================================================
// Batch Validation Helpers
// ============================================================================

// BatchValidateLocations validates multiple locations and returns full location details
func (h *BulkInventoryHelper) BatchValidateLocations(
	locationIDs []uint,
	sellerID uint,
) (map[uint]*entity.Location, error) {
	locations, err := h.locationRepo.FindByIDs(locationIDs, sellerID)
	if err != nil {
		return nil, err
	}

	validMap := make(map[uint]*entity.Location)
	for i := range locations {
		validMap[locations[i].ID] = &locations[i]
	}
	return validMap, nil
}

// BatchGetInventories fetches existing inventory records in a single query
func (h *BulkInventoryHelper) BatchGetInventories(
	variantIDs []uint,
	locationIDs []uint,
) (map[string]*entity.Inventory, error) {
	inventories, err := h.inventoryRepo.FindByVariantAndLocationBatch(variantIDs, locationIDs)
	if err != nil {
		return nil, err
	}

	inventoryMap := make(map[string]*entity.Inventory)
	for i := range inventories {
		key := BuildInventoryKey(inventories[i].VariantID, inventories[i].LocationID)
		inventoryMap[key] = &inventories[i]
	}
	return inventoryMap, nil
}

// ============================================================================
// Inventory Preparation Helpers
// ============================================================================

// GetOrPrepareInventory gets existing inventory from map or prepares a new one
func (h *BulkInventoryHelper) GetOrPrepareInventory(
	existingMap map[string]*entity.Inventory,
	variantID uint,
	locationID uint,
) (*entity.Inventory, bool) {
	key := BuildInventoryKey(variantID, locationID)
	if existing, found := existingMap[key]; found {
		return existing, false
	}

	newInventory := &entity.Inventory{
		VariantID:        variantID,
		LocationID:       locationID,
		Quantity:         0,
		ReservedQuantity: 0,
		Threshold:        0,
	}
	existingMap[key] = newInventory
	return newInventory, true
}

// BuildInventoryKey creates a composite key for inventory lookup
func BuildInventoryKey(variantID uint, locationID uint) string {
	return fmt.Sprintf("%d:%d", variantID, locationID)
}

// ============================================================================
// Result Building Helpers
// ============================================================================

// BuildFailureResult creates a failure result for bulk operations
func BuildFailureResult(
	item model.ManageInventoryRequest,
	errorMsg string,
) model.BulkInventoryItemResult {
	return model.BulkInventoryItemResult{
		VariantID:  item.VariantID,
		LocationID: item.LocationID,
		Success:    false,
		Error:      errorMsg,
	}
}

// BuildSuccessPlaceholder creates a success placeholder result
func BuildSuccessPlaceholder(item model.ManageInventoryRequest) model.BulkInventoryItemResult {
	return model.BulkInventoryItemResult{
		VariantID:  item.VariantID,
		LocationID: item.LocationID,
		Success:    true,
	}
}

// FindResultIndex finds the index of a result in the results slice
func FindResultIndex(
	results []model.BulkInventoryItemResult,
	variantID uint,
	locationID uint,
	successOnly bool,
) int {
	for i, r := range results {
		if r.VariantID == variantID && r.LocationID == locationID {
			if successOnly && !r.Success {
				continue
			}
			return i
		}
	}
	return -1
}

// ============================================================================
// Variant Fetching Helpers (Concurrent Batch Processing)
// ============================================================================

// variantBatchResult holds the result from a single batch fetch operation
type variantBatchResult struct {
	variants map[uint]*productModel.VariantDetailResponse
	err      error
}

// GetVariantDetails fetches variant details in batches using goroutines
func (h *BulkInventoryHelper) GetVariantDetails(
	sellerID *uint,
	variantIDs []uint,
) (map[uint]*productModel.VariantDetailResponse, error) {
	if len(variantIDs) == 0 {
		return make(map[uint]*productModel.VariantDetailResponse), nil
	}

	const batchSize = 100
	totalBatches := (len(variantIDs) + batchSize - 1) / batchSize
	resultsChan := make(chan variantBatchResult, totalBatches)

	h.launchVariantBatchFetchers(variantIDs, sellerID, batchSize, resultsChan)
	return h.collectVariantBatchResults(resultsChan, totalBatches)
}

// launchVariantBatchFetchers launches goroutines to fetch variants in batches
func (h *BulkInventoryHelper) launchVariantBatchFetchers(
	variantIDs []uint,
	sellerID *uint,
	batchSize int,
	resultsChan chan<- variantBatchResult,
) {
	totalBatches := (len(variantIDs) + batchSize - 1) / batchSize

	for i := 0; i < totalBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(variantIDs) {
			end = len(variantIDs)
		}

		batchIDs := variantIDs[start:end]
		go h.fetchVariantBatch(batchIDs, i+1, sellerID, resultsChan)
	}
}

// fetchVariantBatch fetches a single batch of variants and sends result to channel
func (h *BulkInventoryHelper) fetchVariantBatch(
	batchIDs []uint,
	batchNum int,
	sellerID *uint,
	resultsChan chan<- variantBatchResult,
) {
	variantIDsStr := helper.JoinToCommaSeparated(batchIDs)
	filters := &productModel.ListVariantsRequest{
		IDs:      variantIDsStr,
		Page:     1,
		PageSize: len(batchIDs),
	}

	response, err := h.variantQueryService.ListVariants(filters, sellerID, nil)
	if err != nil {
		resultsChan <- variantBatchResult{
			err: fmt.Errorf("batch %d failed: %w", batchNum, err),
		}
		return
	}

	variantMap := make(map[uint]*productModel.VariantDetailResponse)
	for i := range response.Variants {
		variantMap[response.Variants[i].ID] = &response.Variants[i]
	}

	resultsChan <- variantBatchResult{variants: variantMap, err: nil}
}

// collectVariantBatchResults collects results from all batch goroutines
func (h *BulkInventoryHelper) collectVariantBatchResults(
	resultsChan <-chan variantBatchResult,
	totalBatches int,
) (map[uint]*productModel.VariantDetailResponse, error) {
	allVariants := make(map[uint]*productModel.VariantDetailResponse)
	var firstError error

	for i := 0; i < totalBatches; i++ {
		result := <-resultsChan
		if result.err != nil {
			if firstError == nil {
				firstError = result.err
			}
			continue
		}

		for id, variant := range result.variants {
			allVariants[id] = variant
		}
	}

	if firstError != nil {
		return nil, firstError
	}
	return allVariants, nil
}

// ============================================================================
// Bulk Operation Data Structures
// ============================================================================

// PendingBulkResult tracks successful items for post-DB-save processing
type PendingBulkResult struct {
	Index            int
	Inventory        *entity.Inventory
	PreviousQuantity int
	PreviousReserved int
	QuantityChange   int
	IsNew            bool
}

// BulkOperationCollector collects items for batch DB operations
type BulkOperationCollector struct {
	InventoriesToCreate []*entity.Inventory
	InventoriesToUpdate []*entity.Inventory
	PendingResults      []PendingBulkResult
	Results             []model.BulkInventoryItemResult
	SuccessCount        int
	FailureCount        int
}

// NewBulkOperationCollector creates a new collector with pre-allocated slices
func NewBulkOperationCollector(capacity int) *BulkOperationCollector {
	return &BulkOperationCollector{
		InventoriesToCreate: make([]*entity.Inventory, 0),
		InventoriesToUpdate: make([]*entity.Inventory, 0),
		PendingResults:      make([]PendingBulkResult, 0),
		Results:             make([]model.BulkInventoryItemResult, 0, capacity),
		SuccessCount:        0,
		FailureCount:        0,
	}
}

// AddFailure adds a failure result
func (c *BulkOperationCollector) AddFailure(item model.ManageInventoryRequest, errorMsg string) {
	c.Results = append(c.Results, BuildFailureResult(item, errorMsg))
	c.FailureCount++
}

// AddSuccess adds a success placeholder and tracks pending result
func (c *BulkOperationCollector) AddSuccess(
	item model.ManageInventoryRequest,
	index int,
	inventory *entity.Inventory,
	previousQty int,
	previousReserved int,
	quantityChange int,
	isNew bool,
) {
	c.Results = append(c.Results, BuildSuccessPlaceholder(item))
	c.PendingResults = append(c.PendingResults, PendingBulkResult{
		Index:            index,
		Inventory:        inventory,
		PreviousQuantity: previousQty,
		PreviousReserved: previousReserved,
		QuantityChange:   quantityChange,
		IsNew:            isNew,
	})

	if isNew {
		c.InventoriesToCreate = append(c.InventoriesToCreate, inventory)
	} else {
		c.InventoriesToUpdate = append(c.InventoriesToUpdate, inventory)
	}
	c.SuccessCount++
}

// MarkAllSuccessAsFailed marks all successful results as failed (for transaction rollback)
func (c *BulkOperationCollector) MarkAllSuccessAsFailed(errorMsg string) {
	for i := range c.Results {
		if c.Results[i].Success {
			c.Results[i].Success = false
			c.Results[i].Error = errorMsg
			c.Results[i].Response = nil
			c.SuccessCount--
			c.FailureCount++
		}
	}
}

// HasPendingOperations checks if there are any pending DB operations
func (c *BulkOperationCollector) HasPendingOperations() bool {
	return len(c.InventoriesToCreate) > 0 || len(c.InventoriesToUpdate) > 0
}
