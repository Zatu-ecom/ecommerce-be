package service

import (
	"context"

	"ecommerce-be/inventory/entity"
	invErrors "ecommerce-be/inventory/error"
	"ecommerce-be/inventory/factory"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/validator"
)

type InventoryQueryServiceImpl struct {
	inventoryRepo repository.InventoryRepository
	locationRepo  repository.LocationRepository
}

// NewInventoryQueryService creates a new instance of InventoryQueryService
func NewInventoryQueryServiceImpl(
	inventoryRepo repository.InventoryRepository,
	locationRepo repository.LocationRepository,
) *InventoryQueryServiceImpl {
	return &InventoryQueryServiceImpl{
		inventoryRepo: inventoryRepo,
		locationRepo:  locationRepo,
	}
}

// GetInventoryByVariant retrieves inventory for a variant across all locations
func (s *InventoryQueryServiceImpl) GetInventoryByVariant(
	ctx context.Context,
	variantID uint,
	sellerID uint,
) ([]model.InventoryDetailResponse, error) {
	inventories, err := s.inventoryRepo.FindByVariantID(variantID)
	if err != nil {
		return nil, err
	}

	// Filter by seller's locations
	var responses []model.InventoryDetailResponse
	for _, inv := range inventories {
		location, err := s.locationRepo.FindByID(inv.LocationID, sellerID)
		if err != nil {
			continue // Skip if location doesn't belong to seller
		}

		responses = append(responses, model.InventoryDetailResponse{
			InventoryResponse: factory.BuildInventoryResponseFromEntity(inv),
			LocationName:      location.Name,
			LocationType:      string(location.Type),
		})
	}

	return responses, nil
}

// GetInventoryByLocation retrieves all inventory at a specific location
func (s *InventoryQueryServiceImpl) GetInventoryByLocation(
	ctx context.Context,
	locationID uint,
	sellerID uint,
) ([]model.InventoryResponse, error) {
	// Validate location belongs to seller
	if err := s.validateLocation(locationID, sellerID); err != nil {
		return nil, err
	}

	inventories, err := s.inventoryRepo.FindByLocationID(locationID)
	if err != nil {
		return nil, err
	}

	var responses []model.InventoryResponse
	for _, inv := range inventories {
		responses = append(responses, factory.BuildInventoryResponseFromEntity(inv))
	}

	return responses, nil
}

// GetInventoryByVariantAndLocationPriority retrieves inventory allocations for reservation items,
// selecting inventory from locations by priority and splitting across multiple locations when needed.
func (s *InventoryQueryServiceImpl) GetInventoryByVariantAndLocationPriority(
	ctx context.Context,
	items []model.ReservationItem,
	sellerID uint,
) ([]model.InventoryResponse, error) {
	if len(items) == 0 {
		return nil, nil
	}

	variantIDs, requestedQty := s.extractVariantRequests(items)

	locationIDs, err := s.getActiveLocationIDs(sellerID)
	if err != nil {
		return nil, err
	}

	inventories, err := s.inventoryRepo.FindByVariantAndLocationBatch(variantIDs, locationIDs)
	if err != nil {
		return nil, err
	}

	inventoryMap := s.buildInventoryMapByPriority(inventories, locationIDs)

	return s.allocateInventoryByPriority(variantIDs, requestedQty, inventoryMap)
}

// extractVariantRequests extracts variant IDs and requested quantities from reservation items
func (s *InventoryQueryServiceImpl) extractVariantRequests(
	items []model.ReservationItem,
) ([]uint, map[uint]int) {
	variantIDs := make([]uint, len(items))
	requestedQty := make(map[uint]int)
	for i, item := range items {
		variantIDs[i] = item.VariantID
		requestedQty[item.VariantID] = int(item.ReservedQuantity)
	}
	return variantIDs, requestedQty
}

// getActiveLocationIDs retrieves active location IDs sorted by priority
func (s *InventoryQueryServiceImpl) getActiveLocationIDs(sellerID uint) ([]uint, error) {
	locations, err := s.locationRepo.FindActiveByPriority(sellerID)
	if err != nil {
		return nil, err
	}
	if len(locations) == 0 {
		return nil, invErrors.ErrLocationNotFound
	}

	locationIDs := make([]uint, len(locations))
	for i, loc := range locations {
		locationIDs[i] = loc.ID
	}
	return locationIDs, nil
}

// buildInventoryMapByPriority builds a map of variantID -> []inventory sorted by location priority
func (s *InventoryQueryServiceImpl) buildInventoryMapByPriority(
	inventories []entity.Inventory,
	locationIDs []uint,
) map[uint][]*entity.Inventory {
	locationPriorityIndex := make(map[uint]int)
	for i, locID := range locationIDs {
		locationPriorityIndex[locID] = i
	}

	inventoryMap := make(map[uint][]*entity.Inventory)
	for i := range inventories {
		inv := &inventories[i]
		inventoryMap[inv.VariantID] = append(inventoryMap[inv.VariantID], inv)
	}

	// Sort each variant's inventories by location priority
	for _, invList := range inventoryMap {
		s.sortByLocationPriority(invList, locationPriorityIndex)
	}

	return inventoryMap
}

// sortByLocationPriority sorts inventory list by location priority (insertion sort for small lists)
func (s *InventoryQueryServiceImpl) sortByLocationPriority(
	invList []*entity.Inventory,
	priorityIndex map[uint]int,
) {
	for i := 1; i < len(invList); i++ {
		for j := i; j > 0 && priorityIndex[invList[j].LocationID] < priorityIndex[invList[j-1].LocationID]; j-- {
			invList[j], invList[j-1] = invList[j-1], invList[j]
		}
	}
}

// allocateInventoryByPriority allocates inventory from locations by priority for each variant
func (s *InventoryQueryServiceImpl) allocateInventoryByPriority(
	variantIDs []uint,
	requestedQty map[uint]int,
	inventoryMap map[uint][]*entity.Inventory,
) ([]model.InventoryResponse, error) {
	var responses []model.InventoryResponse

	for _, variantID := range variantIDs {
		allocated, err := s.allocateForVariant(
			requestedQty[variantID],
			inventoryMap[variantID],
		)
		if err != nil {
			return nil, err
		}
		responses = append(responses, allocated...)
	}

	return responses, nil
}

// allocateForVariant allocates inventory for a single variant from available locations
func (s *InventoryQueryServiceImpl) allocateForVariant(
	requestedQty int,
	inventories []*entity.Inventory,
) ([]model.InventoryResponse, error) {
	var responses []model.InventoryResponse
	remaining := requestedQty

	for _, inv := range inventories {
		if remaining <= 0 {
			break
		}

		// Available = Quantity - Reserved - Threshold (safety stock)
		availableQty := inv.Quantity - inv.ReservedQuantity - inv.Threshold
		if availableQty <= 0 {
			continue
		}

		allocateQty := min(availableQty, remaining)
		resp := factory.BuildInventoryResponseFromEntity(*inv)
		resp.AvailableQuantity = allocateQty
		responses = append(responses, resp)

		remaining -= allocateQty
	}

	if remaining > 0 {
		return nil, invErrors.ErrInsufficientStock
	}

	return responses, nil
}

func (s *InventoryQueryServiceImpl) validateLocation(locationID uint, sellerID uint) error {
	location, err := s.locationRepo.FindByID(locationID, sellerID)
	if err != nil {
		return err
	}

	// Validate location is active
	return validator.ValidateLocationActive(location)
}
