package service

import (
	"context"
	"fmt"

	"ecommerce-be/common/db"
	"ecommerce-be/common/log"
	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/utils/helper"
	productModel "ecommerce-be/product/model"
	"ecommerce-be/product/service"

	"gorm.io/gorm"
)

// InventoryServiceImpl implements the InventoryService interface
type InventoryServiceImpl struct {
	inventoryRepo       repository.InventoryRepository
	transactionService  InventoryTransactionService
	locationRepo        repository.LocationRepository
	variantQueryService service.VariantQueryService
	bulkHelper          *BulkInventoryHelper
}

// NewInventoryService creates a new instance of InventoryService
func NewInventoryService(
	inventoryRepo repository.InventoryRepository,
	transactionService InventoryTransactionService,
	locationRepo repository.LocationRepository,
	variantQueryService service.VariantQueryService,
	bulkHelper *BulkInventoryHelper,
) *InventoryServiceImpl {
	return &InventoryServiceImpl{
		inventoryRepo:       inventoryRepo,
		transactionService:  transactionService,
		locationRepo:        locationRepo,
		variantQueryService: variantQueryService,
		bulkHelper:          bulkHelper,
	}
}

// ============================================================================
// Single Inventory Management
// ============================================================================

// ManageInventory manages inventory quantity, reserved quantity, or threshold
// Uses BulkManageInventory under the hood for consistency
func (s *InventoryServiceImpl) ManageInventory(
	ctx context.Context,
	req model.ManageInventoryRequest,
	sellerID uint,
	userID uint,
) (*model.ManageInventoryResponse, error) {
	bulkReq := model.BulkManageInventoryRequest{
		Items: []model.ManageInventoryRequest{req},
	}

	bulkResponse, err := s.BulkManageInventory(ctx, bulkReq, sellerID, userID)
	if err != nil {
		return nil, err
	}

	return s.extractSingleResponse(bulkResponse)
}

// extractSingleResponse extracts single response from bulk response
func (s *InventoryServiceImpl) extractSingleResponse(
	bulkResponse *model.BulkManageInventoryResponse,
) (*model.ManageInventoryResponse, error) {
	if bulkResponse == nil || len(bulkResponse.Results) == 0 {
		return nil, fmt.Errorf("inventory operation failed: no results returned")
	}

	result := bulkResponse.Results[0]
	if !result.Success {
		return nil, fmt.Errorf("inventory operation failed: %s", result.Error)
	}

	return result.Response, nil
}

// buildTransactionParams builds CreateTransactionParams from inventory operation data
func (s *InventoryServiceImpl) buildTransactionParams(
	inventory *entity.Inventory,
	req model.ManageInventoryRequest,
	previousQuantity int,
	previousReserved int,
	quantityChange int,
	userID uint,
) model.CreateTransactionParams {
	updatesReserved := req.TransactionType.UpdatesReservedQuantity()
	beforeQty, afterQty := s.determineBeforeAfterQuantities(
		inventory, previousQuantity, previousReserved, updatesReserved,
	)
	referenceType := helper.DetermineReferenceType(req.TransactionType)

	return model.CreateTransactionParams{
		InventoryID:     inventory.ID,
		TransactionType: req.TransactionType,
		QuantityChange:  quantityChange,
		BeforeQuantity:  beforeQty,
		AfterQuantity:   afterQty,
		PerformedBy:     userID,
		Reference:       req.Reference,
		ReferenceType:   referenceType,
		Reason:          req.Reason,
		Note:            req.Note,
	}
}

// determineBeforeAfterQuantities determines the before/after quantities for transaction
func (s *InventoryServiceImpl) determineBeforeAfterQuantities(
	inventory *entity.Inventory,
	previousQuantity int,
	previousReserved int,
	updatesReserved bool,
) (beforeQty int, afterQty int) {
	if updatesReserved {
		return previousReserved, inventory.ReservedQuantity
	}
	return previousQuantity, inventory.Quantity
}

// ============================================================================
// Bulk Inventory Management
// ============================================================================

// BulkManageInventory manages multiple inventory records in a single transaction
func (s *InventoryServiceImpl) BulkManageInventory(
	ctx context.Context,
	req model.BulkManageInventoryRequest,
	sellerID uint,
	userID uint,
) (*model.BulkManageInventoryResponse, error) {
	// Phase 1-4: Batch fetch all required data
	batchData, err := s.prepareBulkData(ctx, req.Items, sellerID)
	if err != nil {
		return nil, err
	}

	// Phase 5: Process all items and prepare bulk operations
	collector := s.processAllBulkItems(req.Items, batchData)

	// Phase 6: Execute all DB operations in a single transaction
	transactions, err := s.executeBulkDBOperations(ctx, collector, req.Items, userID)
	if err != nil {
		return s.buildBulkResponse(collector), nil
	}

	// Phase 7: Build final response with transaction IDs
	s.updateResultsWithTransactionIDs(collector, req.Items, transactions)

	log.InfoWithContext(ctx, fmt.Sprintf(
		"Bulk inventory operation completed: %d success, %d failed",
		collector.SuccessCount, collector.FailureCount,
	))

	return s.buildBulkResponse(collector), nil
}

// bulkBatchData holds pre-fetched data for bulk operations
type bulkBatchData struct {
	validLocations       map[uint]*entity.Location
	validVariants        map[uint]*productModel.VariantDetailResponse
	existingInventoryMap map[string]*entity.Inventory
}

// prepareBulkData fetches all required data for bulk operation in batch
func (s *InventoryServiceImpl) prepareBulkData(
	ctx context.Context,
	items []model.ManageInventoryRequest,
	sellerID uint,
) (*bulkBatchData, error) {
	locationIDs := s.bulkHelper.ExtractUniqueLocationIDs(items)
	variantIDs := s.bulkHelper.ExtractUniqueVariantIDs(items)

	validLocations, err := s.bulkHelper.BatchValidateLocations(locationIDs, sellerID)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to batch validate locations", err)
		return nil, err
	}

	validVariants, err := s.bulkHelper.GetVariantDetails(ctx, &sellerID, variantIDs)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to batch validate variants", err)
		return nil, err
	}

	existingInventoryMap, err := s.bulkHelper.BatchGetInventories(variantIDs, locationIDs)
	if err != nil {
		log.ErrorWithContext(ctx, "Failed to batch fetch inventories", err)
		return nil, err
	}

	return &bulkBatchData{
		validLocations:       validLocations,
		validVariants:        validVariants,
		existingInventoryMap: existingInventoryMap,
	}, nil
}

// processAllBulkItems processes all items and collects operations for batch execution
func (s *InventoryServiceImpl) processAllBulkItems(
	items []model.ManageInventoryRequest,
	batchData *bulkBatchData,
) *BulkOperationCollector {
	collector := NewBulkOperationCollector(len(items))

	for i, item := range items {
		s.processSingleBulkItem(i, item, batchData, collector)
	}

	return collector
}

// processSingleBulkItem processes a single item in bulk operation
func (s *InventoryServiceImpl) processSingleBulkItem(
	index int,
	item model.ManageInventoryRequest,
	batchData *bulkBatchData,
	collector *BulkOperationCollector,
) {
	// Validate request
	if err := helper.ValidateManageRequest(item); err != nil {
		collector.AddFailure(item, err.Error())
		return
	}

	// Validate location
	if errMsg := s.validateBulkItemLocation(item, batchData.validLocations); errMsg != "" {
		collector.AddFailure(item, errMsg)
		return
	}

	// Validate variant
	if _, valid := batchData.validVariants[item.VariantID]; !valid {
		collector.AddFailure(item, "Variant not found")
		return
	}

	// Process inventory changes
	s.applyBulkItemChanges(index, item, batchData.existingInventoryMap, collector)
}

// validateBulkItemLocation validates location for a bulk item
func (s *InventoryServiceImpl) validateBulkItemLocation(
	item model.ManageInventoryRequest,
	validLocations map[uint]*entity.Location,
) string {
	location, valid := validLocations[item.LocationID]
	if !valid {
		return "Location not found or unauthorized"
	}
	if !location.IsActive {
		return "Location is inactive"
	}
	return ""
}

// applyBulkItemChanges applies inventory changes for a bulk item
func (s *InventoryServiceImpl) applyBulkItemChanges(
	index int,
	item model.ManageInventoryRequest,
	existingMap map[string]*entity.Inventory,
	collector *BulkOperationCollector,
) {
	inventory, isNew := s.bulkHelper.GetOrPrepareInventory(
		existingMap, item.VariantID, item.LocationID,
	)

	quantityChange, err := helper.CalculateQuantityChange(item, inventory.Quantity, isNew)
	if err != nil {
		collector.AddFailure(item, err.Error())
		return
	}

	previousQuantity := inventory.Quantity
	previousReserved := inventory.ReservedQuantity

	if err := helper.ApplyInventoryChanges(inventory, item, quantityChange); err != nil {
		collector.AddFailure(item, err.Error())
		return
	}
	helper.UpdateThreshold(inventory, item.Threshold)

	collector.AddSuccess(
		item,
		index,
		inventory,
		previousQuantity,
		previousReserved,
		quantityChange,
		isNew,
	)
}

// executeBulkDBOperations executes all DB operations in a single transaction
func (s *InventoryServiceImpl) executeBulkDBOperations(
	ctx context.Context,
	collector *BulkOperationCollector,
	items []model.ManageInventoryRequest,
	userID uint,
) ([]*entity.InventoryTransaction, error) {
	if !collector.HasPendingOperations() {
		return nil, nil
	}

	var transactions []*entity.InventoryTransaction

	err := db.Atomic(func(tx *gorm.DB) error {
		if err := s.batchSaveInventories(collector); err != nil {
			return err
		}

		transactionParams := s.buildBulkTransactionParams(collector, items, userID)
		var err error
		transactions, err = s.transactionService.CreateTransactionBatch(ctx, transactionParams)
		if err != nil {
			return fmt.Errorf("failed to batch create transactions: %w", err)
		}

		return nil
	})
	if err != nil {
		log.ErrorWithContext(ctx, "Bulk transaction failed, rolling back", err)
		collector.MarkAllSuccessAsFailed("Transaction failed: " + err.Error())
		return nil, err
	}

	return transactions, nil
}

// batchSaveInventories saves all inventories in batch
func (s *InventoryServiceImpl) batchSaveInventories(collector *BulkOperationCollector) error {
	if len(collector.InventoriesToCreate) > 0 {
		if err := s.inventoryRepo.CreateBatch(collector.InventoriesToCreate); err != nil {
			return fmt.Errorf("failed to batch create inventories: %w", err)
		}
	}

	if len(collector.InventoriesToUpdate) > 0 {
		if err := s.inventoryRepo.UpdateBatch(collector.InventoriesToUpdate); err != nil {
			return fmt.Errorf("failed to batch update inventories: %w", err)
		}
	}

	return nil
}

// buildBulkTransactionParams builds all transaction params for bulk operation
func (s *InventoryServiceImpl) buildBulkTransactionParams(
	collector *BulkOperationCollector,
	items []model.ManageInventoryRequest,
	userID uint,
) []model.CreateTransactionParams {
	params := make([]model.CreateTransactionParams, 0, len(collector.PendingResults))

	for _, pending := range collector.PendingResults {
		item := items[pending.Index]

		param := s.buildTransactionParams(
			pending.Inventory, item, pending.PreviousQuantity,
			pending.PreviousReserved, pending.QuantityChange, userID,
		)
		params = append(params, param)
	}

	return params
}

// updateResultsWithTransactionIDs updates results with actual transaction IDs
func (s *InventoryServiceImpl) updateResultsWithTransactionIDs(
	collector *BulkOperationCollector,
	items []model.ManageInventoryRequest,
	transactions []*entity.InventoryTransaction,
) {
	if transactions == nil {
		return
	}

	for i, pending := range collector.PendingResults {
		item := items[pending.Index]
		transaction := transactions[i]

		resultIndex := FindResultIndex(collector.Results, item.VariantID, item.LocationID, true)
		if resultIndex >= 0 {
			collector.Results[resultIndex].Response = helper.BuildManageResponse(
				pending.Inventory, pending.PreviousQuantity,
				pending.QuantityChange, transaction.ID,
			)
		}
	}
}

// buildBulkResponse builds the final bulk response
func (s *InventoryServiceImpl) buildBulkResponse(
	collector *BulkOperationCollector,
) *model.BulkManageInventoryResponse {
	return &model.BulkManageInventoryResponse{
		SuccessCount: collector.SuccessCount,
		FailureCount: collector.FailureCount,
		Results:      collector.Results,
	}
}
