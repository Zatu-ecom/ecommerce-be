package service

import (
	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils/helper"
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

	// BulkUpdateVariants updates multiple variants at once
	BulkUpdateVariants(
		productID, sellerID uint,
		request *model.BulkUpdateVariantsRequest,
	) (*model.BulkUpdateVariantsResponse, error)

	// Query methods for variant aggregations (used by ProductQueryService)
	// GetProductsVariantAggregations retrieves aggregated variant data for multiple products
	// This is optimized for batch operations to prevent N+1 queries
	GetProductsVariantAggregations(productIDs []uint) (map[uint]*mapper.VariantAggregation, error)

	// GetProductVariantAggregation retrieves aggregated variant data for a single product
	GetProductVariantAggregation(productID uint) (*mapper.VariantAggregation, error)

	// GetProductVariantsWithOptions retrieves all variants with their selected option values
	// Optimized single query to prevent N+1 issues when fetching variant details
	GetProductVariantsWithOptions(productID uint) ([]model.VariantDetailResponse, error)

	// CreateVariantsBulk creates multiple variants at once with bulk option value linking
	// Returns models for immediate use in responses
	// Fetches product options internally for validation
	CreateVariantsBulk(
		productID uint,
		sellerID uint,
		requests []model.CreateVariantRequest,
	) ([]model.VariantDetailResponse, error)
}

// VariantServiceImpl implements the VariantService interface
type VariantServiceImpl struct {
	variantRepo      repositories.VariantRepository
	optionService    ProductOptionService
	validatorService ProductValidatorService
}

// NewVariantService creates a new instance of VariantService
func NewVariantService(
	variantRepo repositories.VariantRepository,
	optionService ProductOptionService,
	validatorService ProductValidatorService,
) VariantService {
	return &VariantServiceImpl{
		variantRepo:      variantRepo,
		optionService:    optionService,
		validatorService: validatorService,
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
	// Get product and validate seller access using validator service
	product, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Find the variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Validate variant belongs to product
	if err := validator.ValidateVariantBelongsToProduct(productID, variant); err != nil {
		return nil, err
	}

	// Get all option values for this variant
	variantOptionValues, err := s.variantRepo.GetVariantOptionValues(variantID)
	if err != nil {
		return nil, err
	}

	// Get product options and option values to build the response
	sellerIDPtr := &sellerID
	selectedOptions, err := s.buildVariantOptions(productID, variantOptionValues, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Map to detailed response
	response := factory.BuildVariantDetailResponse(variant, product, selectedOptions)

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
	// Validate input options structure
	if err := validator.ValidateVariantOptions(optionValues); err != nil {
		return nil, err
	}

	// Get product and validate seller access using validator service
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerID)
	if err != nil {
		return nil, err
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
	selectedOptions, err := s.buildVariantOptions(productID, variantOptionValues, sellerID)
	if err != nil {
		return nil, err
	}

	// Map to response
	response := factory.BuildVariantResponse(variant, selectedOptions)

	return response, nil
}

/***********************************************
 *    Helper Methods                           *
 ***********************************************/

// buildVariantOptions builds the variant option response objects from variant option values
// Uses ProductOptionService to get option details as models
func (s *VariantServiceImpl) buildVariantOptions(
	productID uint,
	variantOptionValues []entity.VariantOptionValue,
	sellerID *uint,
) ([]model.VariantOptionResponse, error) {
	if len(variantOptionValues) == 0 {
		return []model.VariantOptionResponse{}, nil
	}

	// Get all options with their values using service (returns models)
	optionsResponse, err := s.optionService.GetAvailableOptions(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Build lookup maps from the response models
	optionMap := make(map[uint]model.ProductOptionDetailResponse)
	valueMap := make(map[uint]model.OptionValueResponse)

	for _, opt := range optionsResponse.Options {
		optionMap[opt.OptionID] = opt
		for _, val := range opt.Values {
			valueMap[val.ValueID] = val
		}
	}

	// Build variant option responses from the selected values
	variantOptions := make([]model.VariantOptionResponse, 0, len(variantOptionValues))
	for _, vov := range variantOptionValues {
		opt, optExists := optionMap[vov.OptionID]
		val, valExists := valueMap[vov.OptionValueID]

		if optExists && valExists {
			variantOption := model.VariantOptionResponse{
				OptionID:          opt.OptionID,
				OptionName:        opt.OptionName,
				OptionDisplayName: opt.OptionDisplayName,
				ValueID:           val.ValueID,
				Value:             val.Value,
				ValueDisplayName:  val.DisplayName,
			}
			if val.ColorCode != "" {
				variantOption.ColorCode = val.ColorCode
			}
			variantOptions = append(variantOptions, variantOption)
		}
	}

	return variantOptions, nil
}

/***********************************************
 *              CreateVariant                  *
 ***********************************************/
func (s *VariantServiceImpl) CreateVariant(
	productID uint,
	sellerID uint,
	request *model.CreateVariantRequest,
) (*model.VariantDetailResponse, error) {
	// Get product and validate seller access using validator service
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Get all available options using service
	optionsResponse, err := s.optionService.GetAvailableOptions(productID, &sellerID)
	if err != nil {
		return nil, err
	}

	// Build lookup maps for validation
	optionNameToID := make(map[string]uint)
	optionValueMap := make(map[uint]map[string]uint) // optionID -> value -> valueID

	for _, opt := range optionsResponse.Options {
		optionNameToID[opt.OptionName] = opt.OptionID
		optionValueMap[opt.OptionID] = make(map[string]uint)
		for _, val := range opt.Values {
			optionValueMap[opt.OptionID][val.Value] = val.ValueID
		}
	}

	// Validate options and get option value IDs
	optionValueIDs := make(map[uint]uint) // optionID -> optionValueID
	optionsMap := make(map[string]string) // For checking duplicate combination

	for _, optionInput := range request.Options {
		// Find option ID by name
		optionID, optExists := optionNameToID[optionInput.OptionName]
		if !optExists {
			return nil, prodErrors.ErrProductOptionNotFound.WithMessagef(
				"Product option not found: %s",
				optionInput.OptionName,
			)
		}

		// Find option value ID
		valueID, valExists := optionValueMap[optionID][optionInput.Value]
		if !valExists {
			return nil, prodErrors.ErrProductOptionValueNotFound.WithMessagef(
				"Product option value not found: %s for option: %s",
				optionInput.Value,
				optionInput.OptionName,
			)
		}

		optionValueIDs[optionID] = valueID
		optionsMap[optionInput.OptionName] = optionInput.Value
	}

	// Check if variant with these options already exists
	existingVariant, _ := s.variantRepo.FindVariantByOptions(productID, optionsMap)
	if err := validator.ValidateVariantCombinationUnique(existingVariant); err != nil {
		return nil, err
	}

	// Create variant entity using factory
	variant := factory.CreateVariantFromRequest(productID, request)

	// Save variant
	if err := s.variantRepo.CreateVariant(variant); err != nil {
		return nil, err
	}

	// Create variant option value associations using factory
	variantOptionValues := factory.CreateVariantOptionValues(variant.ID, optionValueIDs)

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
	// Get product and validate seller access using validator service
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Get existing variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Validate variant belongs to product
	if err := validator.ValidateVariantBelongsToProduct(productID, variant); err != nil {
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
	variant = factory.UpdateVariantEntity(variant, request)

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
	// Get product and validate seller access using validator service
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return err
	}

	// Get variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return err
	}

	// Validate variant belongs to product
	if err := validator.ValidateVariantBelongsToProduct(productID, variant); err != nil {
		return err
	}

	// Check variant count - cannot delete if it's the last one
	variantCount, err := s.variantRepo.CountVariantsByProductID(productID)
	if err != nil {
		return err
	}

	if err := validator.ValidateCanDeleteVariant(variantCount); err != nil {
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
 *           BulkUpdateVariants                *
 ***********************************************/
func (s *VariantServiceImpl) BulkUpdateVariants(
	productID, sellerID uint,
	request *model.BulkUpdateVariantsRequest,
) (*model.BulkUpdateVariantsResponse, error) {
	// Get product and validate seller access using validator service
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Validate bulk update request
	if err := validator.ValidateBulkVariantUpdateRequest(request); err != nil {
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

	// Fetch all variants by IDs
	existingVariants, err := s.variantRepo.FindVariantsByIDs(variantIDs)
	if err != nil {
		return nil, err
	}

	// Validate all variants exist and belong to product
	if err := validator.ValidateBulkVariantsExist(productID, variantIDs, existingVariants); err != nil {
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

	// Update variants using factory
	variantsToUpdate := make([]*entity.ProductVariant, 0, len(existingVariants))
	for i := range existingVariants {
		variant := &existingVariants[i]
		updateData := updateMap[variant.ID]

		// Update variant using factory
		variant = factory.BulkUpdateVariantEntity(variant, updateData)
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
			ID:            variant.ID,
			SKU:           variant.SKU,
			Price:         variant.Price,
			AllowPurchase: variant.AllowPurchase,
		})
	}

	return &model.BulkUpdateVariantsResponse{
		UpdatedCount: len(variantsToUpdate),
		Variants:     summaries,
	}, nil
}

/***********************************************
 *    Query Methods for Variant Aggregations   *
 ***********************************************/

// GetProductsVariantAggregations retrieves aggregated variant data for multiple products
// This method is optimized for batch operations to prevent N+1 queries
// Used by ProductQueryService for efficient product listing queries
func (s *VariantServiceImpl) GetProductsVariantAggregations(
	productIDs []uint,
) (map[uint]*mapper.VariantAggregation, error) {
	return s.variantRepo.GetProductsVariantAggregations(productIDs)
}

// GetProductVariantAggregation retrieves aggregated variant data for a single product
// Returns summary information about all variants for a product
func (s *VariantServiceImpl) GetProductVariantAggregation(
	productID uint,
) (*mapper.VariantAggregation, error) {
	return s.variantRepo.GetProductVariantAggregation(productID)
}

// GetProductVariantsWithOptions retrieves all variants with their selected option values
// Optimized with a single query to prevent N+1 issues when fetching variant details
// Returns complete variant information including selected options for each variant
func (s *VariantServiceImpl) GetProductVariantsWithOptions(
	productID uint,
) ([]model.VariantDetailResponse, error) {
	variantsWithOptions, err := s.variantRepo.GetProductVariantsWithOptions(productID)
	if err == nil && len(variantsWithOptions) > 0 {
		// Use factory to build variants detail response
		response := factory.BuildVariantsDetailResponseFromMapper(variantsWithOptions)
		return response, nil
	}
	return nil, err
}

/***********************************************
 *          CreateVariantsBulk                 *
 ***********************************************/
// CreateVariantsBulk creates multiple variants at once for a product
// Handles default variant logic ("last one wins") and bulk option value linking
// Returns models for immediate use in responses
func (s *VariantServiceImpl) CreateVariantsBulk(
	productID uint,
	sellerID uint,
	requests []model.CreateVariantRequest,
) ([]model.VariantDetailResponse, error) {
	if len(requests) == 0 {
		return nil, commonError.ErrValidation.WithMessage("at least one variant is required")
	}

	// Validate product ownership
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Fetch product options for validation (service-to-service call)
	productOptions, err := s.fetchProductOptionsForValidation(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Build lookup maps for options and values
	optionMap, optionValueMap := s.buildOptionLookupMaps(productOptions)

	// Find last default variant index and handle default logic
	lastDefaultIndex := s.findLastDefaultVariantIndex(requests)
	if lastDefaultIndex != -1 {
		if err := s.handleDefaultVariantLogic(productID, true); err != nil {
			return nil, err
		}
	}

	// Validate and map variant option combinations
	variantOptionCombinations, err := s.validateAndMapVariantOptions(
		requests,
		productOptions,
		optionMap,
		optionValueMap,
	)
	if err != nil {
		return nil, err
	}

	// Create variants and link to option values
	createdVariants, err := s.createVariantsAndLinkOptions(
		productID,
		requests,
		lastDefaultIndex,
		variantOptionCombinations,
	)
	if err != nil {
		return nil, err
	}

	// Convert entities to models using factory
	return s.buildVariantDetailResponses(createdVariants, variantOptionCombinations, productOptions), nil
}

// buildOptionLookupMaps builds lookup maps for quick option and value access
func (s *VariantServiceImpl) buildOptionLookupMaps(
	productOptions []entity.ProductOption,
) (map[string]*entity.ProductOption, map[uint]map[string]uint) {
	optionMap := make(map[string]*entity.ProductOption)
	optionValueMap := make(map[uint]map[string]uint)

	for i := range productOptions {
		optionMap[productOptions[i].Name] = &productOptions[i]
		optionValueMap[productOptions[i].ID] = make(map[string]uint)

		for _, val := range productOptions[i].Values {
			optionValueMap[productOptions[i].ID][val.Value] = val.ID
		}
	}

	return optionMap, optionValueMap
}

// findLastDefaultVariantIndex finds the last variant marked as default
// Returns -1 if no variant is explicitly marked as default
func (s *VariantServiceImpl) findLastDefaultVariantIndex(
	requests []model.CreateVariantRequest,
) int {
	lastDefaultIndex := -1
	for i := range requests {
		if requests[i].IsDefault != nil && *requests[i].IsDefault {
			lastDefaultIndex = i
		}
	}
	return lastDefaultIndex
}

// validateAndMapVariantOptions validates all variant option combinations and maps them to IDs
// Returns a map of variant index to option value IDs for bulk linking
func (s *VariantServiceImpl) validateAndMapVariantOptions(
	requests []model.CreateVariantRequest,
	productOptions []entity.ProductOption,
	optionMap map[string]*entity.ProductOption,
	optionValueMap map[uint]map[string]uint,
) ([]map[uint]uint, error) {
	combinationSet := make(map[string]bool)
	variantOptionCombinations := make([]map[uint]uint, len(requests))

	for i, req := range requests {
		if len(req.Options) == 0 {
			continue
		}

		// Validate all required options are provided
		if len(productOptions) > 0 && len(req.Options) != len(productOptions) {
			return nil, commonError.ErrValidation.WithMessagef(
				"variant must specify all product options (%d required, %d provided)",
				len(productOptions),
				len(req.Options),
			)
		}

		// Map options to IDs and build combination key
		optionValueIDs, combinationKey, err := s.mapVariantOptionsToIDs(
			req.Options,
			optionMap,
			optionValueMap,
		)
		if err != nil {
			return nil, err
		}

		// Check for duplicate combinations
		if combinationSet[combinationKey] {
			return nil, prodErrors.ErrVariantCombinationExists
		}
		combinationSet[combinationKey] = true
		variantOptionCombinations[i] = optionValueIDs
	}

	return variantOptionCombinations, nil
}

// mapVariantOptionsToIDs maps variant option inputs to option and value IDs
// Returns the option value ID map and a combination key for uniqueness checking
func (s *VariantServiceImpl) mapVariantOptionsToIDs(
	options []model.VariantOptionInput,
	optionMap map[string]*entity.ProductOption,
	optionValueMap map[uint]map[string]uint,
) (map[uint]uint, string, error) {
	optionValueIDs := make(map[uint]uint)
	combinationKey := ""

	for _, optInput := range options {
		option, exists := optionMap[optInput.OptionName]
		if !exists {
			return nil, "", commonError.ErrValidation.WithMessagef(
				"option not found: %s",
				optInput.OptionName,
			)
		}

		valueID, exists := optionValueMap[option.ID][optInput.Value]
		if !exists {
			return nil, "", commonError.ErrValidation.WithMessagef(
				"option value not found: %s for option: %s",
				optInput.Value,
				optInput.OptionName,
			)
		}

		optionValueIDs[option.ID] = valueID
		combinationKey += optInput.OptionName + ":" + optInput.Value + ";"
	}

	return optionValueIDs, combinationKey, nil
}

// createVariantsAndLinkOptions creates all variants and links them to option values in bulk
// TRUE BULK: Single INSERT for all variants, single INSERT for all option values
func (s *VariantServiceImpl) createVariantsAndLinkOptions(
	productID uint,
	requests []model.CreateVariantRequest,
	lastDefaultIndex int,
	variantOptionCombinations []map[uint]uint,
) ([]entity.ProductVariant, error) {
	// Prepare all variants for bulk insert
	variantsToCreate := make([]*entity.ProductVariant, 0, len(requests))

	for i, req := range requests {
		// Determine default value based on "last one wins" rule
		isDefault := s.calculateIsDefault(i, lastDefaultIndex)

		// Create variant entity using factory
		variant := factory.CreateVariantFromRequest(productID, &req)
		variant.IsDefault = isDefault

		variantsToCreate = append(variantsToCreate, variant)
	}

	// ✅ TRUE BULK: Create ALL variants in ONE query with RETURNING
	if err := s.variantRepo.BulkCreateVariants(variantsToCreate); err != nil {
		return nil, err
	}

	// Now prepare variant option values with the generated IDs
	allVariantOptionValues := make([]entity.VariantOptionValue, 0)

	for i, variant := range variantsToCreate {
		// Prepare variant option values for bulk insert
		if optionValueIDs := variantOptionCombinations[i]; len(optionValueIDs) > 0 {
			for optionID, valueID := range optionValueIDs {
				allVariantOptionValues = append(allVariantOptionValues, entity.VariantOptionValue{
					VariantID:     variant.ID, // ID populated from BulkCreateVariants
					OptionID:      optionID,
					OptionValueID: valueID,
				})
			}
		}
	}

	// ✅ TRUE BULK: Insert all variant option values in ONE query
	if len(allVariantOptionValues) > 0 {
		if err := s.variantRepo.CreateVariantOptionValues(allVariantOptionValues); err != nil {
			return nil, err
		}
	}

	// Convert pointers back to values for return
	createdVariants := make([]entity.ProductVariant, 0, len(variantsToCreate))
	for _, v := range variantsToCreate {
		createdVariants = append(createdVariants, *v)
	}

	return createdVariants, nil
}

// calculateIsDefault determines if a variant should be default based on "last one wins" rule
func (s *VariantServiceImpl) calculateIsDefault(index, lastDefaultIndex int) bool {
	if lastDefaultIndex != -1 {
		return index == lastDefaultIndex
	}
	// No explicit defaults, first variant is default
	return index == 0
}

// buildVariantDetailResponses converts created variants to models
// Maps variant options using the combinations and product options data
func (s *VariantServiceImpl) buildVariantDetailResponses(
	variants []entity.ProductVariant,
	variantOptionCombinations []map[uint]uint,
	productOptions []entity.ProductOption,
) []model.VariantDetailResponse {
	// Build option lookup maps for quick access
	optionMap := make(map[uint]*entity.ProductOption)
	optionValueMap := make(map[uint]*entity.ProductOptionValue)

	for i := range productOptions {
		optionMap[productOptions[i].ID] = &productOptions[i]
		for j := range productOptions[i].Values {
			optionValueMap[productOptions[i].Values[j].ID] = &productOptions[i].Values[j]
		}
	}

	// Build variant responses
	responses := make([]model.VariantDetailResponse, 0, len(variants))

	for i, variant := range variants {
		variantResp := model.VariantDetailResponse{
			ID:              variant.ID,
			SKU:             variant.SKU,
			Price:           variant.Price,
			Images:          variant.Images,
			AllowPurchase:   variant.AllowPurchase,
			IsPopular:       variant.IsPopular,
			IsDefault:       variant.IsDefault,
			SelectedOptions: []model.VariantOptionResponse{},
			CreatedAt:       helper.FormatTimestamp(variant.CreatedAt),
			UpdatedAt:       helper.FormatTimestamp(variant.UpdatedAt),
		}

		// Build selected options from combinations
		if optionValueIDs := variantOptionCombinations[i]; len(optionValueIDs) > 0 {
			for optionID, valueID := range optionValueIDs {
				option := optionMap[optionID]
				value := optionValueMap[valueID]

				if option != nil && value != nil {
					optionResp := model.VariantOptionResponse{
						OptionID:          option.ID,
						OptionName:        option.Name,
						OptionDisplayName: option.DisplayName,
						ValueID:           value.ID,
						Value:             value.Value,
						ValueDisplayName:  value.DisplayName,
						ColorCode:         value.ColorCode,
					}
					variantResp.SelectedOptions = append(variantResp.SelectedOptions, optionResp)
				}
			}
		}

		responses = append(responses, variantResp)
	}

	return responses
}

// fetchProductOptionsForValidation fetches product options as entities for variant validation
// Uses the option service's available method and converts back to entities
func (s *VariantServiceImpl) fetchProductOptionsForValidation(
	productID uint,
	sellerID uint,
) ([]entity.ProductOption, error) {
	// Use option service to get available options
	sellerIDPtr := &sellerID
	optionsResponse, err := s.optionService.GetAvailableOptions(productID, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Convert response back to entities for validation
	// This is temporary - options are already validated in database
	productOptions := make([]entity.ProductOption, 0, len(optionsResponse.Options))
	for _, optResp := range optionsResponse.Options {
		opt := entity.ProductOption{
			ProductID:   productID,
			Name:        optResp.OptionName,
			DisplayName: optResp.OptionDisplayName,
			Position:    optResp.Position,
			Values:      make([]entity.ProductOptionValue, 0, len(optResp.Values)),
		}
		opt.ID = optResp.OptionID // Set BaseEntity ID

		for _, valResp := range optResp.Values {
			val := entity.ProductOptionValue{
				OptionID:    optResp.OptionID,
				Value:       valResp.Value,
				DisplayName: valResp.DisplayName,
				ColorCode:   valResp.ColorCode,
				Position:    valResp.Position,
			}
			val.ID = valResp.ValueID // Set BaseEntity ID
			opt.Values = append(opt.Values, val)
		}

		productOptions = append(productOptions, opt)
	}

	return productOptions, nil
}
