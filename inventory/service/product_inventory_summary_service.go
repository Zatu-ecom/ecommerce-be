package service

import (
	"context"

	errors "ecommerce-be/inventory/error"
	"ecommerce-be/inventory/factory"
	"ecommerce-be/inventory/mapper"
	"ecommerce-be/inventory/model"
	"ecommerce-be/inventory/repository"
	"ecommerce-be/inventory/utils/helper"
	productMapper "ecommerce-be/product/mapper"
	productModel "ecommerce-be/product/model"
	productService "ecommerce-be/product/service"
)

// ProductInventorySummaryService provides inventory summary operations grouped by product
type ProductInventorySummaryService interface {
	// GetProductsAtLocation retrieves products with inventory summary at a specific location
	GetProductsAtLocation(
		ctx context.Context,
		sellerID uint,
		locationID uint,
		params model.ProductsAtLocationParams,
		filter model.ProductsAtLocationFilter,
	) (*model.ProductsAtLocationResponse, error)

	// GetVariantInventoryAtLocation retrieves variant-level inventory for a product at a location
	GetVariantInventoryAtLocation(
		ctx context.Context,
		sellerID uint,
		productID uint,
		locationID uint,
		filter model.VariantInventoryFilter,
	) (*model.VariantInventoryResponse, error)
}

// ProductInventorySummaryServiceImpl implements ProductInventorySummaryService
type ProductInventorySummaryServiceImpl struct {
	locationService LocationService
	inventoryRepo   repository.InventoryRepository
	variantService  productService.VariantQueryService
}

// NewProductInventorySummaryService creates a new instance
func NewProductInventorySummaryService(
	locationService LocationService,
	inventoryRepo repository.InventoryRepository,
	variantService productService.VariantQueryService,
) *ProductInventorySummaryServiceImpl {
	return &ProductInventorySummaryServiceImpl{
		locationService: locationService,
		inventoryRepo:   inventoryRepo,
		variantService:  variantService,
	}
}

// ============================================================================
// API 2: GetProductsAtLocation
// ============================================================================

// GetProductsAtLocation retrieves products with inventory summary at a specific location
func (s *ProductInventorySummaryServiceImpl) GetProductsAtLocation(
	ctx context.Context,
	sellerID uint,
	locationID uint,
	params model.ProductsAtLocationParams,
	filter model.ProductsAtLocationFilter,
) (*model.ProductsAtLocationResponse, error) {
	params.SetDefaults()

	// 1. Validate location
	location, err := s.locationService.GetLocationByID(ctx, locationID, sellerID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch variant inventories at location
	variantInventories, err := s.inventoryRepo.GetVariantInventoriesAtLocation(ctx, locationID)
	if err != nil {
		return nil, err
	}

	if len(variantInventories) == 0 {
		return factory.BuildEmptyProductsAtLocationResponse(
			locationID, location.Name, params.Page, params.PageSize,
		), nil
	}

	// 3. Get product info for variants
	variantIDs := extractVariantIDs(variantInventories)
	productInfoRows, err := s.variantService.GetProductBasicInfoByVariantIDs(ctx, variantIDs, &sellerID)
	if err != nil {
		return nil, err
	}

	// 4. Aggregate and filter
	productSummaries := s.aggregateByProduct(variantInventories, productInfoRows, filter, params)

	// 5. Paginate and return
	totalCount := len(productSummaries)
	paginatedProducts := paginate(productSummaries, params.Page, params.PageSize)

	return factory.BuildProductsAtLocationResponse(
		locationID,
		location.Name,
		paginatedProducts,
		params.Page,
		params.PageSize,
		int64(totalCount),
	), nil
}

// ============================================================================
// API 3: GetVariantInventoryAtLocation
// ============================================================================

// inventorySortFields are fields that can be sorted at DB level
var inventorySortFields = map[string]bool{
	"quantity":          true,
	"reservedQuantity":  true,
	"availableQuantity": true,
	"threshold":         true,
}

// GetVariantInventoryAtLocation retrieves variant-level inventory for a product at a location
func (s *ProductInventorySummaryServiceImpl) GetVariantInventoryAtLocation(
	ctx context.Context,
	sellerID uint,
	productID uint,
	locationID uint,
	filter model.VariantInventoryFilter,
) (*model.VariantInventoryResponse, error) {
	// 1. Validate location
	location, err := s.locationService.GetLocationByID(ctx, locationID, sellerID)
	if err != nil {
		return nil, err
	}

	// 2. Get variants from product service
	variants, err := s.variantService.GetProductVariantsWithOptions(ctx, productID)
	if err != nil {
		return nil, err
	}
	if len(variants) == 0 {
		return nil, errors.ErrProductInventoryNotFound
	}

	// 3. Get product info
	variantIDs := extractVariantIDsFromVariants(variants)
	productInfoRows, err := s.variantService.GetProductBasicInfoByVariantIDs(ctx, variantIDs, &sellerID)
	if err != nil {
		return nil, err
	}
	productName, baseSKU, categoryID := extractProductInfo(productInfoRows)

	// 4. Get inventory with appropriate sorting
	variantInventories, err := s.getInventoriesWithSort(ctx, locationID, variantIDs, filter)
	if err != nil {
		return nil, err
	}
	if len(variantInventories) == 0 {
		return nil, errors.ErrProductInventoryNotFound
	}

	// 5. Build variants with inventory
	variantsWithInventory, summary := s.buildVariantsResponse(
		variants,
		variantInventories,
		locationID,
		filter,
	)

	// 6. Apply filters
	filteredVariants := helper.ApplyVariantFilters(variantsWithInventory, filter)

	return factory.BuildVariantInventoryResponse(
		productID, productName, categoryID, baseSKU,
		locationID, location.Name, filteredVariants, summary, filter,
	), nil
}

// ============================================================================
// Private Helper Methods
// ============================================================================

// getInventoriesWithSort fetches inventories with DB or in-memory sorting based on sort field
func (s *ProductInventorySummaryServiceImpl) getInventoriesWithSort(
	ctx context.Context,
	locationID uint,
	variantIDs []uint,
	filter model.VariantInventoryFilter,
) ([]mapper.VariantInventoryRow, error) {
	if inventorySortFields[filter.SortBy] {
		return s.inventoryRepo.GetVariantInventoriesAtLocationWithSort(
			ctx, locationID, variantIDs, filter.SortBy, filter.SortOrder,
		)
	}
	return s.inventoryRepo.GetVariantInventoriesAtLocationWithSort(
		ctx, locationID, variantIDs, "", "",
	)
}

// buildVariantsResponse builds variant responses with inventory, respecting sort order
func (s *ProductInventorySummaryServiceImpl) buildVariantsResponse(
	variants []productModel.VariantDetailResponse,
	inventories []mapper.VariantInventoryRow,
	locationID uint,
	filter model.VariantInventoryFilter,
) ([]model.VariantWithInventory, model.VariantInventorySummary) {
	if inventorySortFields[filter.SortBy] {
		// DB sorted - maintain inventory order
		return s.buildInInventoryOrder(variants, inventories, locationID)
	}
	// Need in-memory sort by variant fields
	result, summary := s.buildInVariantOrder(variants, inventories, locationID)
	helper.SortVariantsWithInventory(result, filter.SortBy, filter.SortOrder)
	return result, summary
}

// buildInInventoryOrder builds responses maintaining DB sort order
func (s *ProductInventorySummaryServiceImpl) buildInInventoryOrder(
	variants []productModel.VariantDetailResponse,
	sortedInventories []mapper.VariantInventoryRow,
	locationID uint,
) ([]model.VariantWithInventory, model.VariantInventorySummary) {
	variantMap := buildVariantMap(variants)
	result := make([]model.VariantWithInventory, 0, len(sortedInventories))
	accumulator := &factory.VariantSummaryAccumulator{}

	for _, inv := range sortedInventories {
		variant, exists := variantMap[inv.VariantID]
		if !exists {
			continue
		}
		accumulator.AddInventory(inv)
		result = append(result, factory.BuildVariantWithInventory(variant, inv, locationID))
	}

	return result, accumulator.Build()
}

// buildInVariantOrder builds responses in variant order (for in-memory sorting)
func (s *ProductInventorySummaryServiceImpl) buildInVariantOrder(
	variants []productModel.VariantDetailResponse,
	inventories []mapper.VariantInventoryRow,
	locationID uint,
) ([]model.VariantWithInventory, model.VariantInventorySummary) {
	inventoryMap := buildInventoryMap(inventories)
	result := make([]model.VariantWithInventory, 0, len(inventoryMap))
	accumulator := &factory.VariantSummaryAccumulator{}

	for _, variant := range variants {
		inv, exists := inventoryMap[variant.ID]
		if !exists {
			continue
		}
		accumulator.AddInventory(inv)
		result = append(result, factory.BuildVariantWithInventory(variant, inv, locationID))
	}

	return result, accumulator.Build()
}

// aggregateByProduct groups variant inventory data by product
func (s *ProductInventorySummaryServiceImpl) aggregateByProduct(
	variantInventories []mapper.VariantInventoryRow,
	productInfoRows []productMapper.VariantBasicInfoRow,
	filter model.ProductsAtLocationFilter,
	params model.ProductsAtLocationParams,
) []model.ProductInventorySummary {
	variantToProduct := buildVariantToProductMap(productInfoRows)
	aggregates := make(map[uint]*factory.ProductSummaryAccumulator)

	// Aggregate by product
	for _, inv := range variantInventories {
		productInfo, exists := variantToProduct[inv.VariantID]
		if !exists {
			continue
		}

		agg, ok := aggregates[productInfo.ProductID]
		if !ok {
			agg = &factory.ProductSummaryAccumulator{
				ProductID:   productInfo.ProductID,
				ProductName: productInfo.ProductName,
				CategoryID:  productInfo.CategoryID,
				BaseSKU:     productInfo.BaseSKU,
			}
			aggregates[productInfo.ProductID] = agg
		}
		agg.AddInventory(inv)
	}

	// Build and filter summaries
	summaries := make([]model.ProductInventorySummary, 0, len(aggregates))
	for _, agg := range aggregates {
		summary := agg.Build()
		if s.matchesFilter(summary, filter) {
			summaries = append(summaries, summary)
		}
	}

	// Sort using helper
	helper.SortProductSummaries(summaries, params.SortBy, params.SortOrder)
	return summaries
}

// matchesFilter checks if a product summary matches the filter criteria
func (s *ProductInventorySummaryServiceImpl) matchesFilter(
	summary model.ProductInventorySummary,
	filter model.ProductsAtLocationFilter,
) bool {
	if filter.StockStatus != nil && summary.StockStatus != *filter.StockStatus {
		return false
	}
	if filter.LowStockOnly != nil && *filter.LowStockOnly &&
		summary.StockStatus == model.StockStatusInStock {
		return false
	}
	if filter.Search != nil && *filter.Search != "" &&
		!helper.ContainsIgnoreCase(summary.ProductName, *filter.Search) {
		return false
	}
	if filter.CategoryID != nil && summary.CategoryID != *filter.CategoryID {
		return false
	}
	return true
}

// ============================================================================
// Utility Functions
// ============================================================================

func extractVariantIDs(inventories []mapper.VariantInventoryRow) []uint {
	ids := make([]uint, len(inventories))
	for i, inv := range inventories {
		ids[i] = inv.VariantID
	}
	return ids
}

func extractVariantIDsFromVariants(variants []productModel.VariantDetailResponse) []uint {
	ids := make([]uint, len(variants))
	for i, v := range variants {
		ids[i] = v.ID
	}
	return ids
}

func extractProductInfo(
	rows []productMapper.VariantBasicInfoRow,
) (name, sku string, categoryID uint) {
	if len(rows) > 0 {
		return rows[0].ProductName, rows[0].BaseSKU, rows[0].CategoryID
	}
	return name, sku, categoryID
}

func buildVariantMap(
	variants []productModel.VariantDetailResponse,
) map[uint]productModel.VariantDetailResponse {
	m := make(map[uint]productModel.VariantDetailResponse, len(variants))
	for _, v := range variants {
		m[v.ID] = v
	}
	return m
}

func buildInventoryMap(
	inventories []mapper.VariantInventoryRow,
) map[uint]mapper.VariantInventoryRow {
	m := make(map[uint]mapper.VariantInventoryRow, len(inventories))
	for _, inv := range inventories {
		m[inv.VariantID] = inv
	}
	return m
}

func buildVariantToProductMap(
	rows []productMapper.VariantBasicInfoRow,
) map[uint]productMapper.VariantBasicInfoRow {
	m := make(map[uint]productMapper.VariantBasicInfoRow, len(rows))
	for _, row := range rows {
		m[row.VariantID] = row
	}
	return m
}

func paginate(
	summaries []model.ProductInventorySummary,
	page, pageSize int,
) []model.ProductInventorySummary {
	total := len(summaries)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return []model.ProductInventorySummary{}
	}
	if end > total {
		end = total
	}
	return summaries[start:end]
}
