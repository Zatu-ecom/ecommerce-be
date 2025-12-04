package service

import (
	"context"

	"ecommerce-be/common"
	"ecommerce-be/common/helper"
	"ecommerce-be/inventory/entity"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	userModel "ecommerce-be/user/model"
	userService "ecommerce-be/user/service"
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
	ListTransactions(
		ctx context.Context,
		filter model.ListTransactionsFilter,
	) (*model.ListTransactionsResponse, error)
}

// InventoryTransactionServiceImpl implements the InventoryTransactionService interface
type InventoryTransactionServiceImpl struct {
	transactionRepo  repository.InventoryTransactionRepository
	inventoryRepo    repository.InventoryRepository
	locationRepo     repository.LocationRepository
	userQueryService userService.UserQueryService
}

// NewInventoryTransactionService creates a new instance of InventoryTransactionService
func NewInventoryTransactionService(
	transactionRepo repository.InventoryTransactionRepository,
	inventoryRepo repository.InventoryRepository,
	locationRepo repository.LocationRepository,
	userQueryService userService.UserQueryService,
) InventoryTransactionService {
	return &InventoryTransactionServiceImpl{
		transactionRepo:  transactionRepo,
		inventoryRepo:    inventoryRepo,
		locationRepo:     locationRepo,
		userQueryService: userQueryService,
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
// Transaction Listing
// ============================================================================

// ListTransactions retrieves transactions based on filters
func (s *InventoryTransactionServiceImpl) ListTransactions(
	ctx context.Context,
	filter model.ListTransactionsFilter,
) (*model.ListTransactionsResponse, error) {
	// Set defaults for pagination
	filter.SetDefaults()

	// Fetch transactions from repository
	transactions, total, err := s.transactionRepo.FindByFilter(filter)
	if err != nil {
		return nil, err
	}

	// Build response with enriched data
	return s.buildListResponse(ctx, transactions, total, filter)
}

// buildListResponse builds the list transactions response with enriched data
func (s *InventoryTransactionServiceImpl) buildListResponse(
	ctx context.Context,
	transactions []entity.InventoryTransaction,
	total int64,
	filter model.ListTransactionsFilter,
) (*model.ListTransactionsResponse, error) {
	if len(transactions) == 0 {
		return &model.ListTransactionsResponse{
			Transactions: []model.TransactionResponse{},
			Pagination:   common.NewPaginationResponse(filter.Page, filter.PageSize, total),
		}, nil
	}

	// Collect unique IDs for batch lookups
	inventoryIDs := s.collectUniqueInventoryIDs(transactions)
	userIDs := s.collectUniqueUserIDs(transactions)

	// Batch fetch inventory data (for variant/location info)
	inventoryMap, err := s.fetchInventoryMap(inventoryIDs)
	if err != nil {
		return nil, err
	}

	// Batch fetch location names
	locationMap, err := s.fetchLocationMap(filter.SellerID)
	if err != nil {
		return nil, err
	}

	// Batch fetch user names
	userMap, err := s.fetchUserMap(ctx, userIDs, filter.SellerID)
	if err != nil {
		return nil, err
	}

	// Build response
	responses := make([]model.TransactionResponse, len(transactions))
	for i, txn := range transactions {
		responses[i] = s.buildTransactionResponse(txn, inventoryMap, locationMap, userMap)
	}

	return &model.ListTransactionsResponse{
		Transactions: responses,
		Pagination:   common.NewPaginationResponse(filter.Page, filter.PageSize, total),
	}, nil
}

// buildTransactionResponse builds a single transaction response
func (s *InventoryTransactionServiceImpl) buildTransactionResponse(
	txn entity.InventoryTransaction,
	inventoryMap map[uint]*entity.Inventory,
	locationMap map[uint]string,
	userMap map[uint]string,
) model.TransactionResponse {
	response := model.TransactionResponse{
		ID:             txn.ID,
		InventoryID:    txn.InventoryID,
		Type:           txn.Type,
		Quantity:       txn.Quantity,
		BeforeQuantity: txn.BeforeQuantity,
		AfterQuantity:  txn.AfterQuantity,
		PerformedBy:    txn.PerformedBy,
		ReferenceID:    txn.ReferenceID,
		ReferenceType:  txn.ReferenceType,
		Reason:         txn.Reason,
		Note:           txn.Note,
		CreatedAt:      txn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Enrich with inventory data
	if inv, ok := inventoryMap[txn.InventoryID]; ok {
		response.VariantID = inv.VariantID
		response.LocationID = inv.LocationID
		response.LocationName = locationMap[inv.LocationID]
	}

	// Enrich with user name
	if name, ok := userMap[txn.PerformedBy]; ok {
		response.PerformedByName = name
	}

	return response
}

// collectUniqueInventoryIDs extracts unique inventory IDs from transactions
func (s *InventoryTransactionServiceImpl) collectUniqueInventoryIDs(
	transactions []entity.InventoryTransaction,
) []uint {
	idMap := make(map[uint]bool)
	for _, txn := range transactions {
		idMap[txn.InventoryID] = true
	}

	ids := make([]uint, 0, len(idMap))
	for id := range idMap {
		ids = append(ids, id)
	}
	return ids
}

// collectUniqueUserIDs extracts unique user IDs from transactions
func (s *InventoryTransactionServiceImpl) collectUniqueUserIDs(
	transactions []entity.InventoryTransaction,
) []uint {
	idMap := make(map[uint]bool)
	for _, txn := range transactions {
		idMap[txn.PerformedBy] = true
	}

	ids := make([]uint, 0, len(idMap))
	for id := range idMap {
		ids = append(ids, id)
	}
	return ids
}

// fetchInventoryMap fetches inventory records and returns as map
func (s *InventoryTransactionServiceImpl) fetchInventoryMap(
	inventoryIDs []uint,
) (map[uint]*entity.Inventory, error) {
	result := make(map[uint]*entity.Inventory)
	if len(inventoryIDs) == 0 {
		return result, nil
	}

	inventories, err := s.inventoryRepo.FindByIDs(inventoryIDs)
	if err != nil {
		return nil, err
	}

	for i := range inventories {
		result[inventories[i].ID] = &inventories[i]
	}
	return result, nil
}

// fetchLocationMap fetches all seller's locations and returns name map
func (s *InventoryTransactionServiceImpl) fetchLocationMap(
	sellerID uint,
) (map[uint]string, error) {
	result := make(map[uint]string)
	if sellerID == 0 {
		return result, nil
	}

	locations, err := s.locationRepo.FindAll(sellerID, nil)
	if err != nil {
		return nil, err
	}

	for _, loc := range locations {
		result[loc.ID] = loc.Name
	}
	return result, nil
}

// fetchUserMap fetches user names using UserQueryService with batch processing
// TODO: When moving to microservices, replace this with HTTP call to User Service
//
// Uses helper.BatchFetch for concurrent batch fetching with goroutines.
// This prevents memory issues and improves performance for large user sets.
func (s *InventoryTransactionServiceImpl) fetchUserMap(
	ctx context.Context,
	userIDs []uint,
	sellerID uint,
) (map[uint]string, error) {
	if len(userIDs) == 0 || s.userQueryService == nil {
		return make(map[uint]string), nil
	}

	// Prepare seller ID pointer once
	var sellerIDPtr *uint
	if sellerID > 0 {
		sellerIDPtr = &sellerID
	}

	const batchSize = 100

	// Use generic batch fetcher with closure for fetch logic
	return helper.BatchFetch(ctx, userIDs, batchSize, func(batchIDs []uint) (map[uint]string, error) {
		return s.fetchUserBatchByIDs(batchIDs, sellerIDPtr)
	})
}

// fetchUserBatchByIDs fetches a batch of users by IDs
func (s *InventoryTransactionServiceImpl) fetchUserBatchByIDs(
	userIDs []uint,
	sellerIDPtr *uint,
) (map[uint]string, error) {
	filter := userModel.ListUsersFilter{
		BaseListParams: common.BaseListParams{
			Page:     1,
			PageSize: len(userIDs),
		},
		IDs: userIDs,
	}

	response, err := s.userQueryService.ListUsers(filter, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]string)
	for _, user := range response.Users {
		result[user.ID] = user.Name
	}
	return result, nil
}
