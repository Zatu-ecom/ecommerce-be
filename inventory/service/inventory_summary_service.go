package service

import (
	"context"

	factory "ecommerce-be/inventory/factory"
	"ecommerce-be/inventory/mapper"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	productService "ecommerce-be/product/service"
)

type InventorySummaryService interface {
	// GetLocationsSummary retrieves locations with inventory summary (microservice-ready)
	GetLocationsSummary(
		c context.Context,
		sellerID uint,
		filter model.LocationsFilter,
	) (*model.LocationsSummaryResponse, error)
}

type InventorySummaryServiceImpl struct {
	locationService LocationService // Reuse existing service (DRY)
	inventoryRepo   repository.InventoryRepository
	variantService  productService.VariantQueryService
}

// NewInventorySummaryService creates a new instance of InventorySummaryService
func NewInventorySummaryService(
	locationService LocationService, // Inject location service
	inventoryRepo repository.InventoryRepository,
	variantService productService.VariantQueryService,
) *InventorySummaryServiceImpl {
	return &InventorySummaryServiceImpl{
		locationService: locationService,
		inventoryRepo:   inventoryRepo,
		variantService:  variantService,
	}
}

// GetLocationsSummary retrieves locations with inventory summary (microservice-ready)
//
// Efficient implementation: Only 3 queries total regardless of location count
//  1. Call LocationService.GetAllLocations (handles pagination, filtering, addresses)
//  2. Batch fetch inventory summaries for all locations
//  3. Collect all variant IDs and batch fetch product counts
func (s *InventorySummaryServiceImpl) GetLocationsSummary(
	c context.Context,
	sellerID uint,
	filter model.LocationsFilter,
) (*model.LocationsSummaryResponse, error) {
	// Set pagination defaults
	filter.SetDefaults()

	// 1. Reuse existing LocationService.GetAllLocations (DRY principle)
	// This handles: pagination, filtering, address loading, response building
	locationsResp, err := s.locationService.GetAllLocations(c, sellerID, filter)
	if err != nil {
		return nil, err
	}

	// Extract locations array from paginated response
	locationResponses := locationsResp.Locations

	if len(locationResponses) == 0 {
		return &model.LocationsSummaryResponse{
			Locations:  []model.LocationSummaryResponse{},
			Pagination: locationsResp.Pagination, // Reuse pagination from location service
		}, nil
	}

	// 2. Extract location IDs for batch inventory query
	locationIDs := make([]uint, len(locationResponses))
	for i, loc := range locationResponses {
		locationIDs[i] = loc.ID
	}

	// 3. Batch fetch inventory summaries for all locations (single query)
	inventorySummaries, err := s.inventoryRepo.GetLocationInventorySummaryBatch(locationIDs)
	if err != nil {
		return nil, err
	}

	// 4. Collect all variant IDs for batch product count query
	allVariantIDs := s.collectAllVariantIDs(inventorySummaries)

	// 5. Batch fetch product count for all variants (single query)
	var totalProductCount uint
	if len(allVariantIDs) > 0 {
		totalProductCount, _ = s.variantService.GetProductCountByVariantIDs(
			allVariantIDs,
			&sellerID,
		)
	}

	// 6. Build summary responses by combining location + inventory data
	summaryResponses := s.buildSummaryResponses(
		locationResponses,
		inventorySummaries,
		totalProductCount,
	)

	// 7. Return response with pagination metadata from location service
	return &model.LocationsSummaryResponse{
		Locations:  summaryResponses,
		Pagination: locationsResp.Pagination, // Reuse pagination from location service
	}, nil
}

// collectAllVariantIDs extracts all variant IDs from inventory summaries
func (s *InventorySummaryServiceImpl) collectAllVariantIDs(
	inventorySummaries map[uint]*mapper.LocationInventorySummaryAggregate,
) []uint {
	variantIDSet := make(map[uint]bool)
	for _, summary := range inventorySummaries {
		for _, variantID := range summary.VariantIDs {
			variantIDSet[variantID] = true
		}
	}

	variantIDs := make([]uint, 0, len(variantIDSet))
	for variantID := range variantIDSet {
		variantIDs = append(variantIDs, variantID)
	}
	return variantIDs
}

// buildSummaryResponses combines location data with inventory summaries
func (s *InventorySummaryServiceImpl) buildSummaryResponses(
	locationResponses []model.LocationResponse,
	inventorySummaries map[uint]*mapper.LocationInventorySummaryAggregate,
	totalProductCount uint,
) []model.LocationSummaryResponse {
	summaryResponses := make([]model.LocationSummaryResponse, 0, len(locationResponses))

	for _, locationResp := range locationResponses {
		// Get inventory summary for this location
		invSummary := inventorySummaries[locationResp.ID]
		if invSummary == nil {
			// No inventory for this location
			invSummary = &mapper.LocationInventorySummaryAggregate{
				VariantIDs: []uint{},
			}
		}

		// Use pre-calculated total product count (already fetched in batch)
		// Note: This is total across all locations, not per-location
		// If per-location count is needed, we'd need to track variant→product mapping
		productCount := totalProductCount

		// Calculate stock status based on low/out of stock counts
		stockStatus := s.calculateStockStatus(
			invSummary.LowStockCount,
			invSummary.OutOfStockCount,
		)

		// Calculate average stock value
		var averageStockValue float64
		if invSummary.VariantCount > 0 {
			averageStockValue = float64(invSummary.TotalStock) / float64(invSummary.VariantCount)
		}

		// Build summary response using factory
		summaryResponse := factory.BuildLocationSummaryResponse(
			locationResp,
			productCount,
			*invSummary,
			averageStockValue,
			stockStatus,
		)

		summaryResponses = append(summaryResponses, summaryResponse)
	}

	return summaryResponses
}

// calculateStockStatus determines overall stock status for a location
// Based on presence of out-of-stock or low-stock items
func (s *InventorySummaryServiceImpl) calculateStockStatus(
	lowStockCount, outOfStockCount uint,
) model.StockStatus {
	// If any item is out of stock, status is OUT_OF_STOCK
	if outOfStockCount > 0 {
		return model.StockStatusOutOfStock
	}

	// If any item is low stock, status is LOW_STOCK
	if lowStockCount > 0 {
		return model.StockStatusLowStock
	}

	// All items have healthy stock
	return model.StockStatusInStock
}
