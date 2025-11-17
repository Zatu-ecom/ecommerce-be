package service

import (
	"math"

	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
)

// ProductQueryService defines the interface for product query operations
// This service handles all read-only product operations with optimized queries
type ProductQueryService interface {
	GetAllProducts(
		page, limit int,
		filters map[string]interface{},
	) (*model.ProductsResponse, error)
	GetProductByID(
		id uint,
		sellerID *uint,
	) (*model.ProductResponse, error)
	SearchProducts(
		query string,
		filters map[string]interface{},
		page, limit int,
	) (*model.SearchResponse, error)
	GetProductFilters(
		sellerID *uint,
	) (*model.ProductFilters, error)
	GetRelatedProductsScored(
		productID uint,
		limit int,
		page int,
		strategies string,
		sellerID *uint,
	) (*model.RelatedProductsScoredResponse, error)
}

// ProductQueryServiceImpl implements the ProductQueryService interface
type ProductQueryServiceImpl struct {
	productRepo             repositories.ProductRepository
	variantService          VariantService
	categoryService         CategoryService
	productAttributeService ProductAttributeService
	productOptionService    ProductOptionService
}

// NewProductQueryService creates a new instance of ProductQueryService
func NewProductQueryService(
	productRepo repositories.ProductRepository,
	variantService VariantService,
	categoryService CategoryService,
	productAttributeService ProductAttributeService,
	productOptionService ProductOptionService,
) ProductQueryService {
	return &ProductQueryServiceImpl{
		productRepo:             productRepo,
		variantService:          variantService,
		categoryService:         categoryService,
		productAttributeService: productAttributeService,
		productOptionService:    productOptionService,
	}
}

/*
 * GetAllProducts - Retrieve all products with pagination
 * Optimized with batch variant aggregation to prevent N+1 queries
 */
func (s *ProductQueryServiceImpl) GetAllProducts(
	page,
	limit int,
	filters map[string]interface{},
) (*model.ProductsResponse, error) {
	// Validate and set default pagination values
	page, limit = s.validatePaginationParams(page, limit)

	// Fetch products from repository with filters
	products, total, err := s.productRepo.FindAll(filters, page, limit)
	if err != nil {
		return nil, err
	}

	// Build product responses with variant data using batch aggregation
	// This prevents N+1 queries by fetching all variant data in a single query
	productsResponse, err := s.buildProductResponsesWithVariants(products)
	if err != nil {
		return nil, err
	}

	return &model.ProductsResponse{
		Products:   productsResponse,
		Pagination: s.buildPaginationResponse(page, limit, total),
	}, nil
}

/*
 * Helper Methods for Building Product Responses
 */

// buildProductResponsesWithVariants builds ProductResponse list from products with variant data
// Performs batch variant aggregation for optimal performance - single query for all products
func (s *ProductQueryServiceImpl) buildProductResponsesWithVariants(
	products []entity.Product,
) ([]model.ProductResponse, error) {
	if len(products) == 0 {
		return []model.ProductResponse{}, nil
	}

	// Extract product IDs for batch variant aggregation
	productIDs := make([]uint, len(products))
	for i, product := range products {
		productIDs[i] = product.ID
	}

	// Fetch variant aggregations for all products in ONE query via VariantService
	// This is the key optimization to prevent N+1 queries
	variantAggs, err := s.variantService.GetProductsVariantAggregations(productIDs)
	if err != nil {
		return nil, err
	}

	// Build response models with variant data using factory
	productsResponse := make([]model.ProductResponse, 0, len(products))
	for _, product := range products {
		// Get variant aggregation for this product
		variantAgg := variantAggs[product.ID]
		if variantAgg == nil {
			// Skip products without variants (shouldn't happen per business rules)
			continue
		}

		// Use factory to build product response
		productResp := factory.BuildProductResponse(&product, variantAgg)
		productsResponse = append(productsResponse, productResp)
	}

	return productsResponse, nil
}

/*
 * GetProductByID - Retrieve product by ID with complete details
 * Includes variants, options, attributes, and package options
 * Uses service layer to prevent N+1 queries
 */
func (s *ProductQueryServiceImpl) GetProductByID(
	id uint,
	sellerID *uint,
) (*model.ProductResponse, error) {
	// Fetch product entity
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Multi-tenant check: If seller ID is provided, verify product belongs to that seller
	// If sellerID is nil (admin role or no seller context), skip this check
	if sellerID != nil && product.SellerID != *sellerID {
		return nil, prodErrors.ErrProductNotFound
	}

	// Build detailed product response using service dependencies
	return s.buildDetailedProductResponse(product)
}

// buildDetailedProductResponse builds a complete ProductResponse with all details
// Uses service layer dependencies to fetch related data efficiently
func (s *ProductQueryServiceImpl) buildDetailedProductResponse(
	product *entity.Product,
) (*model.ProductResponse, error) {
	// Get variant aggregation for summary info using VariantService
	variantAgg, err := s.variantService.GetProductVariantAggregation(product.ID)
	if err != nil {
		return nil, err
	}

	// Use factory to build base product response with variant aggregation
	response := factory.BuildProductResponse(product, variantAgg)

	// Enhance with additional details for the detailed view

	// Get product attributes using ProductAttributeService
	attrResponse, err := s.productAttributeService.GetProductAttributes(product.ID)
	if err == nil && attrResponse != nil {
		// Use factory to convert ProductAttributeDetailResponse to ProductAttributeResponse
		response.Attributes = factory.ConvertDetailListToSimpleAttributeResponses(
			attrResponse.Attributes,
		)
	}

	// Get package options from repository (no service exists for this yet)
	packageOptions, err := s.productRepo.FindPackageOptionByProductID(product.ID)
	if err == nil {
		response.PackageOptions = factory.BuildPackageOptionResponses(packageOptions)
	}

	// Get all product options with their values using VariantService
	productOptions, _, err := s.variantService.GetProductOptionsWithVariantCounts(product.ID)
	if err == nil && len(productOptions) > 0 {
		// Use factory to build options detail response
		response.Options = factory.BuildProductOptionsDetailResponse(productOptions)
	}

	// Get all variants with their selected option values using VariantService
	// This is optimized with a single query to prevent N+1 issues
	variants, err := s.variantService.GetProductVariantsWithOptions(product.ID)
	if err == nil && len(variants) > 0 {
		// Use factory to build variants detail response
		response.Variants = variants
	}

	return &response, nil
}

/*
 * SearchProducts - Search products with query and filters
 * Optimized with batch variant aggregation like GetAllProducts
 */
func (s *ProductQueryServiceImpl) SearchProducts(
	query string,
	filters map[string]interface{},
	page, limit int,
) (*model.SearchResponse, error) {
	// Validate and set default pagination values
	page, limit = s.validatePaginationParams(page, limit)

	// Fetch products from repository with search query and filters
	products, total, err := s.productRepo.Search(query, filters, page, limit)
	if err != nil {
		return nil, err
	}

	// Build product responses with variant data using batch aggregation
	// Reuses the same optimization as GetAllProducts to prevent N+1 queries
	productsResponse, err := s.buildProductResponsesWithVariants(products)
	if err != nil {
		return nil, err
	}

	// Convert to search results with additional search metadata
	searchResults := make([]model.SearchResult, 0, len(productsResponse))
	for _, productResp := range productsResponse {
		searchResult := model.SearchResult{
			ProductResponse: productResp,
			RelevanceScore:  0.8,              // TODO: Implement actual relevance scoring
			MatchedFields:   []string{"name"}, // TODO: Track which fields matched
		}
		searchResults = append(searchResults, searchResult)
	}

	return &model.SearchResponse{
		Query:      query,
		Results:    searchResults,
		Pagination: s.buildPaginationResponse(page, limit, total),
		SearchTime: "0.05s", // Placeholder
	}, nil
}

/*
 * GetProductFilters - Get available filters for product search
 * Multi-tenant: filters based on sellerID if provided
 * Includes variant-based filters (price, options, stock)
 */
func (s *ProductQueryServiceImpl) GetProductFilters(
	sellerID *uint,
) (*model.ProductFilters, error) {
	// Fetch all filter data from repository in single call
	brands, categories, attributes, priceRange, variantOptions, stockStatus, err := s.productRepo.GetProductFilters(
		sellerID,
	)
	if err != nil {
		return nil, err
	}

	// Build filters using factory methods
	filters := &model.ProductFilters{
		Brands:       factory.BuildBrandFilters(brands),
		Categories:   s.buildCategoryFiltersHierarchy(categories),
		Attributes:   factory.BuildAttributeFilters(attributes),
		PriceRange:   factory.BuildPriceRangeFilter(priceRange),
		VariantTypes: factory.BuildVariantTypeFilters(variantOptions),
		StockStatus:  factory.BuildStockStatusFilter(stockStatus),
	}

	return filters, nil
}

// buildCategoryFiltersHierarchy builds hierarchical category filters
// Groups categories by parent-child relationships
func (s *ProductQueryServiceImpl) buildCategoryFiltersHierarchy(
	categories []mapper.CategoryWithProductCount,
) []model.CategoryFilter {
	// Create a map for quick lookup and build all category filters
	categoryMap := make(map[uint]model.CategoryFilter)
	var rootCategories []model.CategoryFilter

	// First pass: create all category filters
	for _, category := range categories {
		categoryFilter := factory.BuildCategoryFilter(category)
		categoryMap[category.CategoryID] = categoryFilter

		// Collect root categories (no parent or parent is 0)
		if category.ParentID == nil || *category.ParentID == 0 {
			rootCategories = append(rootCategories, categoryFilter)
		}
	}

	// Second pass: build hierarchy by adding children to parents
	for _, category := range categories {
		if category.ParentID != nil && *category.ParentID != 0 {
			parentFilter, exists := categoryMap[*category.ParentID]
			if exists {
				// Add this category as a child of its parent
				childFilter := categoryMap[category.CategoryID]
				parentFilter.Children = append(parentFilter.Children, childFilter)
				categoryMap[*category.ParentID] = parentFilter
			} else {
				// Parent not found, treat as root category
				rootCategories = append(rootCategories, categoryMap[category.CategoryID])
			}
		}
	}

	// Update root categories with their children
	for i, root := range rootCategories {
		if updated, exists := categoryMap[root.ID]; exists {
			rootCategories[i] = updated
		}
	}

	return rootCategories
}

/*
 * GetRelatedProductsScored - Get related products with scoring
 * Uses stored procedure for multi-strategy matching
 * Optimized to avoid N+1 queries
 */
func (s *ProductQueryServiceImpl) GetRelatedProductsScored(
	productID uint,
	limit int,
	page int,
	strategies string,
	sellerID *uint,
) (*model.RelatedProductsScoredResponse, error) {
	// Validate and set defaults
	page, limit = s.validatePaginationParams(page, limit)
	if strategies == "" {
		strategies = "all"
	}

	// Calculate offset for pagination
	offset := (page - 1) * limit

	// Call repository method that uses stored procedure for scoring
	scoredResults, totalCount, err := s.productRepo.FindRelatedScored(
		productID,
		sellerID,
		limit,
		offset,
		strategies,
	)
	if err != nil {
		return nil, err
	}

	// If no results, return empty response with metadata
	if len(scoredResults) == 0 {
		return factory.BuildRelatedProductsScoredResponse(
			[]model.RelatedProductItemScored{},
			[]string{},
			0,
			s.buildPaginationResponse(page, limit, 0),
			8,
		), nil
	}

	// Build related items using batch operation and factory methods
	relatedItems, strategiesUsedMap, totalScore, err := s.buildRelatedProductItems(scoredResults)
	if err != nil {
		return nil, err
	}

	// Build strategies used list
	strategiesUsed := make([]string, 0, len(strategiesUsedMap))
	for strategy := range strategiesUsedMap {
		strategiesUsed = append(strategiesUsed, strategy)
	}

	// Calculate average score
	avgScore := float64(totalScore) / float64(len(relatedItems))

	return factory.BuildRelatedProductsScoredResponse(
		relatedItems,
		strategiesUsed,
		avgScore,
		s.buildPaginationResponse(page, limit, totalCount),
		8,
	), nil
}

// buildRelatedProductItems builds related product items with batch optimization
// Returns the items, strategies map, and total score for metadata calculation
func (s *ProductQueryServiceImpl) buildRelatedProductItems(
	scoredResults []mapper.RelatedProductScored,
) ([]model.RelatedProductItemScored, map[string]bool, int, error) {
	// Extract product IDs for batch fetching options (optimization to avoid N+1)
	productIDs := make([]uint, len(scoredResults))
	for i, result := range scoredResults {
		productIDs[i] = result.ProductID
	}

	// Batch fetch all options with values for all products at once
	// This prevents N+1 queries by fetching all data in a single query
	productOptionsMap, err := s.productOptionService.GetProductsOptionsWithValues(productIDs)
	if err != nil {
		// If fetching options fails, continue without options preview
		productOptionsMap = make(map[uint][]entity.ProductOption)
	}

	// Build response items using factory
	relatedItems := make([]model.RelatedProductItemScored, 0, len(scoredResults))
	strategiesUsedMap := make(map[string]bool)
	totalScore := 0

	for _, result := range scoredResults {
		// Get options preview from pre-fetched data
		var optionsPreview []model.OptionPreview
		if options, exists := productOptionsMap[result.ProductID]; exists {
			// Use factory method to build options preview
			optionsPreview = factory.BuildOptionsPreview(options)
		}

		// Use factory to build the complete scored item
		scoredItem := factory.BuildRelatedProductItemScored(&result, optionsPreview)

		relatedItems = append(relatedItems, scoredItem)
		strategiesUsedMap[result.StrategyUsed] = true
		totalScore += result.FinalScore
	}

	return relatedItems, strategiesUsedMap, totalScore, nil
}

// validatePaginationParams validates and normalizes pagination parameters
// Returns normalized page and limit values
func (s *ProductQueryServiceImpl) validatePaginationParams(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// buildPaginationResponse builds a standard pagination response
// Helper to avoid code duplication across query methods
func (s *ProductQueryServiceImpl) buildPaginationResponse(
	page, limit int,
	total int64,
) model.PaginationResponse {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return model.PaginationResponse{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   int(total),
		ItemsPerPage: limit,
		HasNext:      page < totalPages,
		HasPrev:      page > 1,
	}
}
