package service

import (
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"
)

// VariantService defines the interface for variant-related business logic
type VariantService interface {
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

	// CreateVariant creates a new variant for a product
	CreateVariant(
		productID uint,
		sellerID uint,
		request *model.CreateVariantRequest,
	) (*model.VariantDetailResponse, error)

	// UpdateVariant updates an existing variant
	UpdateVariant(
		productID, variantID uint, sellerID uint,
		request *model.UpdateVariantRequest,
	) (*model.VariantDetailResponse, error)

	// DeleteVariant deletes a variant
	DeleteVariant(productID, variantID uint, sellerID uint) error

	// UpdateVariantStock updates the stock for a variant
	UpdateVariantStock(
		productID, variantID, sellerID uint,
		request *model.UpdateVariantStockRequest,
	) (*model.UpdateVariantStockResponse, error)

	// BulkUpdateVariants updates multiple variants at once
	BulkUpdateVariants(
		productID, sellerID uint,
		request *model.BulkUpdateVariantsRequest,
	) (*model.BulkUpdateVariantsResponse, error)
}

// VariantServiceImpl implements the VariantService interface
type VariantServiceImpl struct {
	variantRepo repositories.VariantRepository
	productRepo repositories.ProductRepository
	validator   *validator.VariantValidator
	factory     *factory.VariantFactory
}

// NewVariantService creates a new instance of VariantService
func NewVariantService(
	variantRepo repositories.VariantRepository,
	productRepo repositories.ProductRepository,
) VariantService {
	return &VariantServiceImpl{
		variantRepo: variantRepo,
		productRepo: productRepo,
		validator:   validator.NewVariantValidator(variantRepo, productRepo),
		factory:     factory.NewVariantFactory(),
	}
}

/***********************************************
 *          Private Helper Methods             *
 ***********************************************/

// handleDefaultVariantLogic ensures only one variant per product is default
// If isDefault is true, it unsets all other defaults for the product
// This maintains the business rule: "Only one variant can be default per product"
func (s *VariantServiceImpl) handleDefaultVariantLogic(productID uint, isDefault bool) error {
	if isDefault {
		// Unset all existing defaults for this product before setting new default
		if err := s.variantRepo.UnsetAllDefaultVariantsForProduct(productID); err != nil {
			return err
		}
	}
	return nil
}

/***********************************************
 *                GetVariantByID               *
 ***********************************************/
func (s *VariantServiceImpl) GetVariantByID(
	productID, variantID uint,
	sellerID uint,
) (*model.VariantDetailResponse, error) {
	// Validate that the product exists
	if err := s.validator.ValidateProductAndSeller(productID, sellerID); err != nil {
		return nil, err
	}

	// Get product to validate seller access
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Validate seller access: if sellerID is provided (non-admin), check ownership
	if sellerID != 0 && product.SellerID != sellerID {
		return nil, prodErrors.ErrProductNotFound
	}

	// Validate variant belongs to product
	if err := s.validator.ValidateVariantBelongsToProduct(productID, variantID); err != nil {
		return nil, err
	}

	// Find the variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Get all option values for this variant
	variantOptionValues, err := s.variantRepo.GetVariantOptionValues(variantID)
	if err != nil {
		return nil, err
	}

	// Get product options and option values to build the response
	selectedOptions, err := s.buildVariantOptions(variantOptionValues)
	if err != nil {
		return nil, err
	}

	// Map to detailed response
	response := s.factory.BuildVariantDetailResponse(variant, product, selectedOptions)

	return response, nil
}

/***********************************************
 *            FindVariantByOptions             *
 ***********************************************/
func (s *VariantServiceImpl) FindVariantByOptions(
	productID uint,
	optionValues map[string]string,
	sellerID *uint,
) (*model.VariantResponse, error) {
	// Validate input
	if err := s.validator.ValidateVariantOptions(optionValues); err != nil {
		return nil, err
	}

	// Validate that the product exists and validate seller access
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Validate seller access: if sellerID is provided (non-admin), check ownership
	if sellerID != nil && product.SellerID != *sellerID {
		return nil, prodErrors.ErrProductNotFound
	}

	// Find the variant by options
	variant, err := s.variantRepo.FindVariantByOptions(productID, optionValues)
	if err != nil {
		// If variant not found, return error
		// The handler can fetch available options if needed for the error response
		return nil, err
	}

	// Get all option values for this variant
	variantOptionValues, err := s.variantRepo.GetVariantOptionValues(variant.ID)
	if err != nil {
		return nil, err
	}

	// Get product options and option values to build the response
	selectedOptions, err := s.buildVariantOptions(variantOptionValues)
	if err != nil {
		return nil, err
	}

	// Map to response
	response := s.factory.BuildVariantResponse(variant, selectedOptions)

	return response, nil
}

/***********************************************
 *    Helper Methods                           *
 ***********************************************/

// buildVariantOptions builds the variant option response objects
func (s *VariantServiceImpl) buildVariantOptions(
	variantOptionValues []entity.VariantOptionValue,
) ([]model.VariantOptionResponse, error) {
	if len(variantOptionValues) == 0 {
		return []model.VariantOptionResponse{}, nil
	}

	// Collect all unique option IDs and option value IDs
	optionIDMap := make(map[uint]bool)
	optionValueIDMap := make(map[uint]bool)

	for _, vov := range variantOptionValues {
		optionIDMap[vov.OptionID] = true
		optionValueIDMap[vov.OptionValueID] = true
	}

	// Fetch all product options for this product
	productOptions := []entity.ProductOption{}
	for optionID := range optionIDMap {
		option, err := s.variantRepo.GetProductOptionByID(optionID)
		if err != nil {
			continue
		}
		productOptions = append(productOptions, *option)
	}

	// Fetch all option values
	optionValues := []entity.ProductOptionValue{}
	for _, vov := range variantOptionValues {
		optionValue, err := s.variantRepo.GetOptionValueByID(vov.OptionValueID)
		if err != nil {
			continue
		}
		optionValues = append(optionValues, *optionValue)
	}

	// Map to response
	return s.factory.BuildVariantOptionResponses(
		variantOptionValues,
		productOptions,
		optionValues,
	), nil
}

/***********************************************
 *              CreateVariant                  *
 ***********************************************/
func (s *VariantServiceImpl) CreateVariant(
	productID uint,
	sellerID uint,
	request *model.CreateVariantRequest,
) (*model.VariantDetailResponse, error) {
	// Validate that the product exists
	if err := s.validator.ValidateProductAndSeller(productID, sellerID); err != nil {
		return nil, err
	}

	// Validate options and get option value IDs
	optionValueIDs := make(map[uint]uint) // optionID -> optionValueID
	optionsMap := make(map[string]string) // For checking duplicate combination

	for _, optionInput := range request.Options {
		// Validate option exists and get option ID
		optionID, err := s.validator.ValidateOptionExists(productID, optionInput.OptionName)
		if err != nil {
			return nil, err
		}

		// Validate option value exists and get option value ID
		optionValueID, err := s.validator.ValidateOptionValueExists(*optionID, optionInput.Value)
		if err != nil {
			return nil, err
		}

		optionValueIDs[*optionID] = *optionValueID
		optionsMap[optionInput.OptionName] = optionInput.Value
	}

	// Check if variant with these options already exists
	if err := s.validator.ValidateVariantCombinationUnique(productID, optionsMap); err != nil {
		return nil, err
	}

	// Create variant entity using factory
	variant := s.factory.CreateVariantFromRequest(productID, request)

	// Save variant
	if err := s.variantRepo.CreateVariant(variant); err != nil {
		return nil, err
	}

	// Create variant option value associations using factory
	variantOptionValues := s.factory.CreateVariantOptionValues(variant.ID, optionValueIDs)

	if err := s.variantRepo.CreateVariantOptionValues(variantOptionValues); err != nil {
		return nil, err
	}

	// Return the created variant details (no seller validation needed for create response)
	return s.GetVariantByID(productID, variant.ID, sellerID)
}

/***********************************************
 *              UpdateVariant                  *
 ***********************************************/
func (s *VariantServiceImpl) UpdateVariant(
	productID,
	variantID uint,
	sellerID uint,
	request *model.UpdateVariantRequest,
) (*model.VariantDetailResponse, error) {
	// Validate that the product exists
	if err := s.validator.ValidateProductAndSeller(productID, sellerID); err != nil {
		return nil, err
	}

	// Validate variant belongs to product
	if err := s.validator.ValidateVariantBelongsToProduct(productID, variantID); err != nil {
		return nil, err
	}

	// Get existing variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Handle default variant logic BEFORE updating
	// If setting this variant as default, unset all other defaults for the product
	if request.IsDefault != nil && *request.IsDefault {
		if err := s.handleDefaultVariantLogic(productID, true); err != nil {
			return nil, err
		}
	}

	// Update variant using factory
	variant = s.factory.UpdateVariantEntity(variant, request)

	// Save updated variant
	if err := s.variantRepo.UpdateVariant(variant); err != nil {
		return nil, err
	}

	// Return updated variant details (no seller validation needed for update response)
	return s.GetVariantByID(productID, variant.ID, sellerID)
}

/***********************************************
 *                DeleteVariant                *
 ***********************************************/
func (s *VariantServiceImpl) DeleteVariant(productID, variantID uint, sellerID uint) error {
	// Validate that the product exists
	if err := s.validator.ValidateProductAndSeller(productID, sellerID); err != nil {
		return err
	}

	// Validate variant belongs to product
	if err := s.validator.ValidateVariantBelongsToProduct(productID, variantID); err != nil {
		return err
	}

	// Check if this is the last variant - cannot delete
	if err := s.validator.ValidateCanDeleteVariant(productID); err != nil {
		return err
	}

	// Delete variant option values first (foreign key constraint)
	if err := s.variantRepo.DeleteVariantOptionValues(variantID); err != nil {
		return err
	}

	// Delete the variant
	return s.variantRepo.DeleteVariant(variantID)
}

/***********************************************
 *            UpdateVariantStock               *
 ***********************************************/
func (s *VariantServiceImpl) UpdateVariantStock(
	productID, variantID, sellerID uint,
	request *model.UpdateVariantStockRequest,
) (*model.UpdateVariantStockResponse, error) {
	// Validate that the product exists
	if err := s.validator.ValidateProductAndSeller(productID, sellerID); err != nil {
		return nil, err
	}

	// Validate variant belongs to product
	if err := s.validator.ValidateVariantBelongsToProduct(productID, variantID); err != nil {
		return nil, err
	}

	// Get existing variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Validate stock operation
	if err := s.validator.ValidateStockOperation(request, variant.Stock); err != nil {
		return nil, err
	}

	// Apply stock operation using factory
	variant = s.factory.ApplyStockOperation(variant, request.Operation, request.Stock)

	// Save updated variant
	if err := s.variantRepo.UpdateVariant(variant); err != nil {
		return nil, err
	}

	// Return stock update response
	return &model.UpdateVariantStockResponse{
		VariantID: variant.ID,
		SKU:       variant.SKU,
		Stock:     variant.Stock,
		InStock:   variant.InStock,
	}, nil
}

/***********************************************
 *           BulkUpdateVariants                *
 ***********************************************/
func (s *VariantServiceImpl) BulkUpdateVariants(
	productID, sellerID uint,
	request *model.BulkUpdateVariantsRequest,
) (*model.BulkUpdateVariantsResponse, error) {
	// Validate that the product exists
	if err := s.validator.ValidateProductAndSeller(productID, sellerID); err != nil {
		return nil, err
	}

	// Validate bulk update request
	if err := s.validator.ValidateBulkUpdateRequest(request); err != nil {
		return nil, err
	}

	// Extract all variant IDs from the request
	variantIDs := make([]uint, 0, len(request.Variants))
	updateMap := make(map[uint]*model.BulkUpdateVariantItem)

	// Track which variants are being set to default (to apply "last one wins" logic)
	var lastDefaultVariantID *uint

	for i := range request.Variants {
		variantIDs = append(variantIDs, request.Variants[i].ID)
		updateMap[request.Variants[i].ID] = &request.Variants[i]

		// Track last variant being set to default (last one wins)
		if request.Variants[i].IsDefault != nil && *request.Variants[i].IsDefault {
			lastDefaultVariantID = &request.Variants[i].ID
		}
	}

	// Validate all variants exist and belong to product
	if err := s.validator.ValidateBulkVariantsExist(productID, variantIDs); err != nil {
		return nil, err
	}

	// Handle default variant logic BEFORE updating
	// If any variant is being set to default, unset all existing defaults
	// Then only the last variant marked as default will remain as default
	if lastDefaultVariantID != nil {
		if err := s.handleDefaultVariantLogic(productID, true); err != nil {
			return nil, err
		}

		// Apply "last one wins" rule: only keep the last variant's isDefault=true
		// Set all other variants' isDefault to false
		for variantID, updateData := range updateMap {
			if updateData.IsDefault != nil && *updateData.IsDefault {
				if variantID != *lastDefaultVariantID {
					// Not the last one, force to false
					falseValue := false
					updateData.IsDefault = &falseValue
				}
			}
		}
	}

	// Fetch all variants by IDs
	existingVariants, err := s.variantRepo.FindVariantsByIDs(variantIDs)
	if err != nil {
		return nil, err
	}

	// Update variants using factory
	variantsToUpdate := make([]*entity.ProductVariant, 0, len(existingVariants))
	for i := range existingVariants {
		variant := &existingVariants[i]
		updateData := updateMap[variant.ID]

		// Update variant using factory
		variant = s.factory.BulkUpdateVariantEntity(variant, updateData)
		variantsToUpdate = append(variantsToUpdate, variant)
	}

	// Perform bulk update in a transaction
	if err := s.variantRepo.BulkUpdateVariants(variantsToUpdate); err != nil {
		return nil, err
	}

	// Build response with summary of updated variants
	summaries := make([]model.BulkUpdateVariantSummary, 0, len(variantsToUpdate))
	for _, variant := range variantsToUpdate {
		summaries = append(summaries, model.BulkUpdateVariantSummary{
			ID:      variant.ID,
			SKU:     variant.SKU,
			Price:   variant.Price,
			Stock:   variant.Stock,
			InStock: variant.InStock,
		})
	}

	return &model.BulkUpdateVariantsResponse{
		UpdatedCount: len(variantsToUpdate),
		Variants:     summaries,
	}, nil
}
