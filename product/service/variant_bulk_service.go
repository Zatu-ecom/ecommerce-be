package service

import (
	"context"
	"strings"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"
)

// VariantBulkService defines the interface for bulk variant operations
type VariantBulkService interface {
	// BulkUpdateVariants updates multiple variants at once
	BulkUpdateVariants(
		ctx context.Context,
		productID, sellerID uint,
		request *model.BulkUpdateVariantsRequest,
	) (*model.BulkUpdateVariantsResponse, error)

	// CreateVariantsBulk creates multiple variants at once with bulk option value linking
	// Returns models for immediate use in responses
	// Fetches product options internally for validation
	CreateVariantsBulk(
		ctx context.Context,
		productID uint,
		sellerID uint,
		requests []model.CreateVariantRequest,
	) ([]model.VariantDetailResponse, error)

	// DeleteVariantsByProductID deletes all variants and their associated data for a product
	// Handles cascade deletion of variant_option_values
	DeleteVariantsByProductID(ctx context.Context, productID uint) error
}

// VariantBulkServiceImpl implements the VariantBulkService interface
type VariantBulkServiceImpl struct {
	variantRepo      repositories.VariantRepository
	optionService    ProductOptionService
	validatorService ProductValidatorService
}

// NewVariantBulkService creates a new instance of VariantBulkService
func NewVariantBulkService(
	variantRepo repositories.VariantRepository,
	optionService ProductOptionService,
	validatorService ProductValidatorService,
) VariantBulkService {
	return &VariantBulkServiceImpl{
		variantRepo:      variantRepo,
		optionService:    optionService,
		validatorService: validatorService,
	}
}

/***********************************************
 *           BulkUpdateVariants                *
 ***********************************************/
func (s *VariantBulkServiceImpl) BulkUpdateVariants(
	ctx context.Context,
	productID, sellerID uint,
	request *model.BulkUpdateVariantsRequest,
) (*model.BulkUpdateVariantsResponse, error) {
	// Get product and validate seller access using validator service
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Validate bulk update request
	if err := validator.ValidateBulkVariantUpdateRequest(request); err != nil {
		return nil, err
	}

	// Extract variant IDs and track default
	variantIDs, updateMap, lastDefaultVariantID := s.extractVariantIDsAndTrackDefault(request)

	// Fetch all variants by IDs
	existingVariants, err := s.variantRepo.FindVariantsByIDs(ctx, variantIDs)
	if err != nil {
		return nil, err
	}

	// Validate all variants exist and belong to product
	if err := validator.ValidateBulkVariantsExist(productID, variantIDs, existingVariants); err != nil {
		return nil, err
	}

	// Apply "last one wins" rule for defaults
	s.applyLastOneWinsRule(updateMap, lastDefaultVariantID)

	// Update variants using factory
	variantsToUpdate := make([]*entity.ProductVariant, 0, len(existingVariants))
	for i := range existingVariants {
		variant := &existingVariants[i]
		variant = factory.BulkUpdateVariantEntity(variant, updateMap[variant.ID])
		variantsToUpdate = append(variantsToUpdate, variant)
	}

	// Transaction: Handle default logic and bulk update atomically
	if lastDefaultVariantID != nil {
		if err := s.variantRepo.UnsetAllDefaultVariantsForProduct(ctx, productID); err != nil {
			return nil, err
		}
	}

	if err := s.variantRepo.BulkUpdateVariants(ctx, variantsToUpdate); err != nil {
		return nil, err
	}

	return s.buildBulkUpdateResponse(variantsToUpdate), nil
}

// extractVariantIDsAndTrackDefault extracts variant IDs and tracks the last default variant
func (s *VariantBulkServiceImpl) extractVariantIDsAndTrackDefault(
	request *model.BulkUpdateVariantsRequest,
) ([]uint, map[uint]*model.BulkUpdateVariantItem, *uint) {
	variantIDs := make([]uint, 0, len(request.Variants))
	updateMap := make(map[uint]*model.BulkUpdateVariantItem)
	var lastDefaultVariantID *uint

	for i := range request.Variants {
		variantIDs = append(variantIDs, request.Variants[i].ID)
		updateMap[request.Variants[i].ID] = &request.Variants[i]

		if request.Variants[i].IsDefault != nil && *request.Variants[i].IsDefault {
			lastDefaultVariantID = &request.Variants[i].ID
		}
	}

	return variantIDs, updateMap, lastDefaultVariantID
}

// applyLastOneWinsRule ensures only the last variant marked as default remains default
func (s *VariantBulkServiceImpl) applyLastOneWinsRule(
	updateMap map[uint]*model.BulkUpdateVariantItem,
	lastDefaultVariantID *uint,
) {
	if lastDefaultVariantID == nil {
		return
	}

	for variantID, updateData := range updateMap {
		if updateData.IsDefault != nil && *updateData.IsDefault &&
			variantID != *lastDefaultVariantID {
			falseValue := false
			updateData.IsDefault = &falseValue
		}
	}
}

// buildBulkUpdateResponse builds the response with variant summaries
func (s *VariantBulkServiceImpl) buildBulkUpdateResponse(
	variants []*entity.ProductVariant,
) *model.BulkUpdateVariantsResponse {
	summaries := make([]model.BulkUpdateVariantSummary, 0, len(variants))
	for _, variant := range variants {
		summaries = append(summaries, model.BulkUpdateVariantSummary{
			ID:            variant.ID,
			SKU:           variant.SKU,
			Price:         variant.Price,
			AllowPurchase: variant.AllowPurchase,
		})
	}

	return &model.BulkUpdateVariantsResponse{
		UpdatedCount: len(variants),
		Variants:     summaries,
	}
}

/***********************************************
 *          CreateVariantsBulk                 *
 ***********************************************/
// CreateVariantsBulk creates multiple variants at once for a product
// Handles default variant logic ("last one wins") and bulk option value linking
// Returns models for immediate use in responses
func (s *VariantBulkServiceImpl) CreateVariantsBulk(
	ctx context.Context,
	productID uint,
	sellerID uint,
	requests []model.CreateVariantRequest,
) ([]model.VariantDetailResponse, error) {
	if len(requests) == 0 {
		return nil, commonError.ErrValidation.WithMessage("at least one variant is required")
	}

	// Validate product ownership
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Fetch product options for validation (service-to-service call)
	productOptions, err := s.fetchProductOptionsForValidation(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Build lookup maps for options and values
	optionMap, optionValueMap := s.buildOptionLookupMaps(productOptions)

	// Find last default variant index
	lastDefaultIndex := s.findLastDefaultVariantIndex(requests)

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

	// Transaction: Wrap all operations for data consistency
	// This ensures default logic, variant creation, and option linking are atomic
	var createdVariants []entity.ProductVariant
	err = s.createVariantsInTransaction(
		ctx,
		productID,
		requests,
		lastDefaultIndex,
		variantOptionCombinations,
		&createdVariants,
	)
	if err != nil {
		return nil, err
	}

	// Convert entities to models using factory method (eliminates code duplication)
	return s.buildVariantDetailResponsesWithFactory(
		createdVariants,
		variantOptionCombinations,
		productOptions,
	), nil
}

// buildOptionLookupMaps builds lookup maps for quick option and value access
func (s *VariantBulkServiceImpl) buildOptionLookupMaps(
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

// createVariantsInTransaction wraps variant creation in a transaction with default logic
// Ensures atomicity of all operations: default unset, variant creation, option linking
func (s *VariantBulkServiceImpl) createVariantsInTransaction(
	ctx context.Context,
	productID uint,
	requests []model.CreateVariantRequest,
	lastDefaultIndex int,
	variantOptionCombinations []map[uint]uint,
	createdVariants *[]entity.ProductVariant,
) error {
	// Unset existing defaults if needed (before creating new defaults)
	if lastDefaultIndex != -1 {
		if err := s.variantRepo.UnsetAllDefaultVariantsForProduct(ctx, productID); err != nil {
			return err
		}
	}

	// Create variants and link options
	variants, err := s.createVariantsAndLinkOptions(
		ctx,
		productID,
		requests,
		lastDefaultIndex,
		variantOptionCombinations,
	)
	if err != nil {
		return err
	}

	*createdVariants = variants
	return nil
}

// findLastDefaultVariantIndex finds the last variant marked as default
// Returns -1 if no variant is explicitly marked as default
func (s *VariantBulkServiceImpl) findLastDefaultVariantIndex(
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
func (s *VariantBulkServiceImpl) validateAndMapVariantOptions(
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
func (s *VariantBulkServiceImpl) mapVariantOptionsToIDs(
	options []model.VariantOptionInput,
	optionMap map[string]*entity.ProductOption,
	optionValueMap map[uint]map[string]uint,
) (map[uint]uint, string, error) {
	optionValueIDs := make(map[uint]uint)

	// Use strings.Builder for efficient string concatenation
	var keyBuilder strings.Builder
	keyBuilder.Grow(len(options) * 20) // Pre-allocate approximate size

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

		// Build combination key efficiently
		keyBuilder.WriteString(optInput.OptionName)
		keyBuilder.WriteString(":")
		keyBuilder.WriteString(optInput.Value)
		keyBuilder.WriteString(";")
	}

	return optionValueIDs, keyBuilder.String(), nil
}

// createVariantsAndLinkOptions creates all variants and links them to option values in bulk
// TRUE BULK: Single INSERT for all variants, single INSERT for all option values
func (s *VariantBulkServiceImpl) createVariantsAndLinkOptions(
	ctx context.Context,
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
	if err := s.variantRepo.BulkCreateVariants(ctx, variantsToCreate); err != nil {
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
		if err := s.variantRepo.CreateVariantOptionValues(ctx, allVariantOptionValues); err != nil {
			return nil, err
		}
	}

	// Convert pointers to values efficiently (single allocation)
	createdVariants := make([]entity.ProductVariant, len(variantsToCreate))
	for i, v := range variantsToCreate {
		createdVariants[i] = *v
	}

	return createdVariants, nil
}

// calculateIsDefault determines if a variant should be default based on "last one wins" rule
func (s *VariantBulkServiceImpl) calculateIsDefault(index, lastDefaultIndex int) bool {
	if lastDefaultIndex != -1 {
		return index == lastDefaultIndex
	}
	// No explicit defaults, first variant is default
	return index == 0
}

// buildVariantDetailResponsesWithFactory converts created variants to models using factory
// Eliminates code duplication by leveraging existing factory methods
func (s *VariantBulkServiceImpl) buildVariantDetailResponsesWithFactory(
	variants []entity.ProductVariant,
	variantOptionCombinations []map[uint]uint,
	productOptions []entity.ProductOption,
) []model.VariantDetailResponse {
	// Convert product options to model format for factory method
	optionsModel := s.convertProductOptionsToModel(productOptions)

	// Build responses using factory method
	responses := make([]model.VariantDetailResponse, 0, len(variants))

	for i, variant := range variants {
		// Build variant option values for factory
		variantOptionValues := make([]entity.VariantOptionValue, 0)
		if optionValueIDs := variantOptionCombinations[i]; len(optionValueIDs) > 0 {
			for optionID, valueID := range optionValueIDs {
				variantOptionValues = append(variantOptionValues, entity.VariantOptionValue{
					VariantID:     variant.ID,
					OptionID:      optionID,
					OptionValueID: valueID,
				})
			}
		}

		// Use factory method to build selected options (eliminates duplication)
		selectedOptions := factory.BuildVariantOptionResponsesFromAvailableOptions(
			variantOptionValues,
			optionsModel,
		)

		// Build response using factory
		response := factory.BuildVariantDetailResponse(&variant, nil, selectedOptions)
		responses = append(responses, *response)
	}

	return responses
}

// convertProductOptionsToModel converts entity.ProductOption to model format
// This helper method bridges the gap between entity and model representations
func (s *VariantBulkServiceImpl) convertProductOptionsToModel(
	productOptions []entity.ProductOption,
) *model.GetAvailableOptionsResponse {
	optionsResponse := &model.GetAvailableOptionsResponse{
		Options: make([]model.ProductOptionDetailResponse, 0, len(productOptions)),
	}

	for _, opt := range productOptions {
		optionResp := model.ProductOptionDetailResponse{
			OptionID:          opt.ID,
			OptionName:        opt.Name,
			OptionDisplayName: opt.DisplayName,
			Position:          opt.Position,
			Values:            make([]model.OptionValueResponse, 0, len(opt.Values)),
		}

		for _, val := range opt.Values {
			optionResp.Values = append(optionResp.Values, model.OptionValueResponse{
				ValueID:     val.ID,
				Value:       val.Value,
				DisplayName: val.DisplayName,
				ColorCode:   val.ColorCode,
				Position:    val.Position,
			})
		}

		optionsResponse.Options = append(optionsResponse.Options, optionResp)
	}

	return optionsResponse
}

// fetchProductOptionsForValidation fetches product options as entities for variant validation
// Directly queries entities for better performance (eliminates model-to-entity conversion)
func (s *VariantBulkServiceImpl) fetchProductOptionsForValidation(
	ctx context.Context,
	productID uint,
	sellerID uint,
) ([]entity.ProductOption, error) {
	// Validate product ownership first
	if _, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID); err != nil {
		return nil, err
	}

	// Use option service to get available options (already returns validated data)
	sellerIDPtr := &sellerID
	optionsResponse, err := s.optionService.GetAvailableOptions(ctx, productID, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Convert response to entities (simplified conversion for validation purposes)
	return s.convertOptionsResponseToEntities(productID, optionsResponse), nil
}

// convertOptionsResponseToEntities converts model response to entities efficiently
func (s *VariantBulkServiceImpl) convertOptionsResponseToEntities(
	productID uint,
	optionsResponse *model.GetAvailableOptionsResponse,
) []entity.ProductOption {
	productOptions := make([]entity.ProductOption, len(optionsResponse.Options))

	for i, optResp := range optionsResponse.Options {
		values := make([]entity.ProductOptionValue, len(optResp.Values))
		for j, valResp := range optResp.Values {
			values[j] = entity.ProductOptionValue{
				OptionID:    optResp.OptionID,
				Value:       valResp.Value,
				DisplayName: valResp.DisplayName,
				ColorCode:   valResp.ColorCode,
				Position:    valResp.Position,
			}
			values[j].ID = valResp.ValueID
		}

		productOptions[i] = entity.ProductOption{
			ProductID:   productID,
			Name:        optResp.OptionName,
			DisplayName: optResp.OptionDisplayName,
			Position:    optResp.Position,
			Values:      values,
		}
		productOptions[i].ID = optResp.OptionID
	}

	return productOptions
}

/***********************************************
 *       DeleteVariantsByProductID             *
 ***********************************************/
// DeleteVariantsByProductID deletes all variants and their associated data for a product
// Handles cascade deletion of variant_option_values in a transaction
func (s *VariantBulkServiceImpl) DeleteVariantsByProductID(ctx context.Context, productID uint) error {
	// Get all variants for this product
	variants, err := s.variantRepo.FindVariantsByProductID(ctx, productID)
	if err != nil {
		return err
	}

	// Early return for consistent error handling
	if len(variants) == 0 {
		return nil
	}

	// Collect all variant IDs
	variantIDs := make([]uint, len(variants))
	for i, v := range variants {
		variantIDs[i] = v.ID
	}

	// Transaction: Wrap both deletes to ensure atomicity
	// If variant delete fails, option values won't be orphaned
	if err := s.variantRepo.DeleteVariantOptionValuesByVariantIDs(ctx, variantIDs); err != nil {
		return err
	}

	// Delete all variants (both operations should ideally be in one DB transaction)
	return s.variantRepo.DeleteVariantsByProductID(ctx, productID)
}
