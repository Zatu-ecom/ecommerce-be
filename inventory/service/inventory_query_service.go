package service

import (
	"context"

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
			InventoryResponse: model.InventoryResponse{
				ID:                inv.ID,
				VariantID:         inv.VariantID,
				LocationID:        inv.LocationID,
				Quantity:          inv.Quantity,
				ReservedQuantity:  inv.ReservedQuantity,
				Threshold:         inv.Threshold,
				AvailableQuantity: inv.Quantity - inv.ReservedQuantity,
				BelowThreshold:    inv.Quantity < inv.Threshold,
			},
			LocationName: location.Name,
			LocationType: string(location.Type),
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
		responses = append(responses, model.InventoryResponse{
			ID:                inv.ID,
			VariantID:         inv.VariantID,
			LocationID:        inv.LocationID,
			Quantity:          inv.Quantity,
			ReservedQuantity:  inv.ReservedQuantity,
			Threshold:         inv.Threshold,
			AvailableQuantity: inv.Quantity - inv.ReservedQuantity,
			BelowThreshold:    inv.Quantity < inv.Threshold,
		})
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
