package service

import (
	"math"

	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
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
}

// NewProductQueryService creates a new instance of ProductQueryService
func NewProductQueryService(
	productRepo repositories.ProductRepository,
	variantService VariantService,
	categoryService CategoryService,
	productAttributeService ProductAttributeService,
) ProductQueryService {
	return &ProductQueryServiceImpl{
		productRepo:             productRepo,
		variantService:          variantService,
		categoryService:         categoryService,
		productAttributeService: productAttributeService,
	}
}

/*******************************************************************
 * GetAllProducts - Retrieve all products with pagination          *
 * Optimized with batch variant aggregation to prevent N+1 queries *
 *******************************************************************/
func (s *ProductQueryServiceImpl) GetAllProducts(
	page,
	limit int,
	filters map[string]interface{},
) (*model.ProductsResponse, error) {
	// Validate and set default pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

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

	// Calculate pagination metadata
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	hasNext := page < totalPages
	hasPrev := page > 1

	pagination := model.PaginationResponse{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   int(total),
		ItemsPerPage: limit,
		HasNext:      hasNext,
		HasPrev:      hasPrev,
	}

	return &model.ProductsResponse{
		Products:   productsResponse,
		Pagination: pagination,
	}, nil
}

/*******************************************************************
 * Helper Methods for Building Product Responses                   *
 *******************************************************************/

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

/*******************************************************************
 * GetProductByID - Retrieve product by ID with complete details   *
 * Includes variants, options, attributes, and package options     *
 * Uses service layer to prevent N+1 queries                       *
 *******************************************************************/
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
	variantsWithOptions, err := s.variantService.GetProductVariantsWithOptions(product.ID)
	if err == nil && len(variantsWithOptions) > 0 {
		// Use factory to build variants detail response
		response.Variants = factory.BuildVariantsDetailResponseFromMapper(variantsWithOptions)
	}

	return &response, nil
}

/*******************************************************************
 * Stub Methods - To be implemented in subsequent refactoring      *
 *******************************************************************/

// SearchProducts - Stub implementation
func (s *ProductQueryServiceImpl) SearchProducts(
	query string,
	filters map[string]interface{},
	page, limit int,
) (*model.SearchResponse, error) {
	// TODO: Implement in next refactoring phase
	return nil, nil
}

// GetProductFilters - Stub implementation
func (s *ProductQueryServiceImpl) GetProductFilters(
	sellerID *uint,
) (*model.ProductFilters, error) {
	// TODO: Implement in next refactoring phase
	return nil, nil
}

// GetRelatedProductsScored - Stub implementation
func (s *ProductQueryServiceImpl) GetRelatedProductsScored(
	productID uint,
	limit int,
	page int,
	strategies string,
	sellerID *uint,
) (*model.RelatedProductsScoredResponse, error) {
	// TODO: Implement in next refactoring phase
	return nil, nil
}
