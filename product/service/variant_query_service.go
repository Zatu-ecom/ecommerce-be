package service

import (
	"ecommerce-be/product/entity"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"
)

// VariantQueryService defines the interface for variant read operations
type VariantQueryService interface {
	// GetVariantByID retrieves detailed information about a specific variant
	GetVariantByID(
		productID,
		variantID uint,
		sellerID uint,
	) (*model.VariantDetailResponse, error)

	// FindVariantByOptions finds a variant based on selected options
	FindVariantByOptions(
		productID uint,
		optionValues map[string]string,
		sellerID *uint,
	) (*model.VariantResponse, error)

	// GetProductVariantsWithOptions retrieves all variants with their selected option values
	// Optimized single query to prevent N+1 issues when fetching variant details
	GetProductVariantsWithOptions(productID uint) ([]model.VariantDetailResponse, error)

	// GetProductVariantAggregation retrieves aggregated variant data for a single product
	GetProductVariantAggregation(productID uint) (*mapper.VariantAggregation, error)

	// GetProductsVariantAggregations retrieves aggregated variant data for multiple products
	// This is optimized for batch operations to prevent N+1 queries
	GetProductsVariantAggregations(productIDs []uint) (map[uint]*mapper.VariantAggregation, error)
}

// VariantQueryServiceImpl implements the VariantQueryService interface
type VariantQueryServiceImpl struct {
	variantRepo      repositories.VariantRepository
	optionService    ProductOptionService
	validatorService ProductValidatorService
}

// NewVariantQueryService creates a new instance of VariantQueryService
func NewVariantQueryService(
	variantRepo repositories.VariantRepository,
	optionService ProductOptionService,
	validatorService ProductValidatorService,
) VariantQueryService {
	return &VariantQueryServiceImpl{
		variantRepo:      variantRepo,
		optionService:    optionService,
		validatorService: validatorService,
	}
}

// GetVariantByID retrieves detailed information about a specific variant
func (s *VariantQueryServiceImpl) GetVariantByID(
	productID, variantID uint,
	sellerID uint,
) (*model.VariantDetailResponse, error) {
	// Get product and validate seller access using validator service
	product, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Find the variant (already validates it belongs to product via query with both IDs)
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Build response using helper method (reduces code duplication)
	return s.buildVariantDetailResponse(variant, product, productID, sellerID)
}

// FindVariantByOptions finds a variant based on selected options
func (s *VariantQueryServiceImpl) FindVariantByOptions(
	productID uint,
	optionValues map[string]string,
	sellerID *uint,
) (*model.VariantResponse, error) {
	// Validate seller access FIRST (security priority)
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Then validate input options structure (efficiency)
	if err := validator.ValidateVariantOptions(optionValues); err != nil {
		return nil, err
	}

	// Find the variant by options
	variant, err := s.variantRepo.FindVariantByOptions(productID, optionValues)
	if err != nil {
		return nil, err
	}

	// Get variant option values
	variantOptionValues, err := s.variantRepo.GetVariantOptionValues(variant.ID)
	if err != nil {
		return nil, err
	}

	// Get product options and build response using factory (reduces duplication)
	optionsResponse, err := s.optionService.GetAvailableOptions(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Use factory method instead of duplicate buildVariantOptions logic
	selectedOptions := factory.BuildVariantOptionResponsesFromAvailableOptions(
		variantOptionValues,
		optionsResponse,
	)

	// Map to response
	return factory.BuildVariantResponse(variant, selectedOptions), nil
}

// GetProductVariantsWithOptions retrieves all variants with their selected option values
// Optimized with a single query to prevent N+1 issues when fetching variant details
// Returns complete variant information including selected options for each variant
func (s *VariantQueryServiceImpl) GetProductVariantsWithOptions(
	productID uint,
) ([]model.VariantDetailResponse, error) {
	variantsWithOptions, err := s.variantRepo.GetProductVariantsWithOptions(productID)
	if err != nil {
		return nil, err
	}

	// Return empty slice for no results (not nil) - consistent error handling
	if len(variantsWithOptions) == 0 {
		return []model.VariantDetailResponse{}, nil
	}

	// Use factory to build variants detail response
	return factory.BuildVariantsDetailResponseFromMapper(variantsWithOptions), nil
}

// GetProductVariantAggregation retrieves aggregated variant data for a single product
// Returns summary information about all variants for a product
func (s *VariantQueryServiceImpl) GetProductVariantAggregation(
	productID uint,
) (*mapper.VariantAggregation, error) {
	return s.variantRepo.GetProductVariantAggregation(productID)
}

// GetProductsVariantAggregations retrieves aggregated variant data for multiple products
// This is optimized for batch operations to prevent N+1 queries
func (s *VariantQueryServiceImpl) GetProductsVariantAggregations(
	productIDs []uint,
) (map[uint]*mapper.VariantAggregation, error) {
	return s.variantRepo.GetProductsVariantAggregations(productIDs)
}

// buildVariantDetailResponse builds the variant detail response from variant data
// This helper reduces code duplication and provides consistent response building
func (s *VariantQueryServiceImpl) buildVariantDetailResponse(
	variant *entity.ProductVariant,
	product *entity.Product,
	productID uint,
	sellerID uint,
) (*model.VariantDetailResponse, error) {
	// Get variant option values
	variantOptionValues, err := s.variantRepo.GetVariantOptionValues(variant.ID)
	if err != nil {
		return nil, err
	}

	// Get product options
	sellerIDPtr := &sellerID
	optionsResponse, err := s.optionService.GetAvailableOptions(productID, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Use factory method to build selected options (eliminates code duplication)
	selectedOptions := factory.BuildVariantOptionResponsesFromAvailableOptions(
		variantOptionValues,
		optionsResponse,
	)

	// Build and return response
	return factory.BuildVariantDetailResponse(variant, product, selectedOptions), nil
}
