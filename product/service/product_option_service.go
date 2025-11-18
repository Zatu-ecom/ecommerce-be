package service

import (
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"
	"ecommerce-be/product/utils/helper"
	"ecommerce-be/product/validator"
)

// ProductOptionService defines the interface for product option-related business logic
type ProductOptionService interface {
	CreateOption(
		productID uint,
		sellerId uint,
		req model.ProductOptionCreateRequest,
	) (*model.ProductOptionResponse, error)
	UpdateOption(
		productID uint,
		optionID uint,
		sellerId uint,
		req model.ProductOptionUpdateRequest,
	) (*model.ProductOptionResponse, error)
	DeleteOption(productID uint, sellerId uint, optionID uint) error
	BulkUpdateOptions(
		productID uint,
		sellerId uint,
		req model.ProductOptionBulkUpdateRequest,
	) (*model.BulkUpdateResponse, error)

	// GetAvailableOptions retrieves all available options and their values for a product
	GetAvailableOptions(
		productID uint,
		sellerID *uint,
	) (*model.GetAvailableOptionsResponse, error)

	// GetProductOptionsWithVariantCounts retrieves all options with their values and variant counts
	// Used for detailed product views to show available options
	GetProductOptionsWithVariantCounts(
		productID uint,
		sellerID *uint,
	) ([]model.ProductOptionDetailResponse, error)

	// GetProductsOptionsWithValues retrieves all options with their values for multiple products
	// Batch operation to prevent N+1 queries
	GetProductsOptionsWithValues(productIDs []uint) (map[uint][]entity.ProductOption, error)

	// DeleteOptionsByProductID deletes all product options and their values for a product
	// Handles cascade deletion of option_values
	DeleteOptionsByProductID(productID uint) error

	// CreateOptionsBulk creates multiple product options with their values in bulk
	// Returns models for immediate use in responses
	CreateOptionsBulk(
		productID uint,
		sellerID uint,
		requests []model.ProductOptionCreateRequest,
	) ([]model.ProductOptionDetailResponse, error)
}

// ProductOptionServiceImpl implements the ProductOptionService interface
type ProductOptionServiceImpl struct {
	optionRepo       repositories.ProductOptionRepository
	validatorService ProductValidatorService
}

// NewProductOptionService creates a new instance of ProductOptionService
func NewProductOptionService(
	optionRepo repositories.ProductOptionRepository,
	validatorService ProductValidatorService,
) ProductOptionService {
	return &ProductOptionServiceImpl{
		optionRepo:       optionRepo,
		validatorService: validatorService,
	}
}

/***********************************************
 *    CreateOption creates a new product option *
 ***********************************************/
func (s *ProductOptionServiceImpl) CreateOption(
	productID uint,
	sellerId uint,
	req model.ProductOptionCreateRequest,
) (*model.ProductOptionResponse, error) {
	// Validate product exists and seller has access using validator service
	sellerIDPtr := &sellerId
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Fetch existing options for uniqueness validation
	existingOptions, err := s.optionRepo.FindOptionsByProductID(productID)
	if err != nil {
		return nil, err
	}

	// Validate option name uniqueness
	if err := validator.ValidateProductOptionNameUniqueness(req.Name, existingOptions); err != nil {
		return nil, err
	}

	// Create option entity using factory
	option := factory.CreateOptionFromRequest(productID, req)

	// Create option
	if err := s.optionRepo.CreateOption(option); err != nil {
		return nil, err
	}

	// Create option values if provided
	if len(req.Values) > 0 {
		// Extract values for validation
		values := make([]string, len(req.Values))
		for i, v := range req.Values {
			values[i] = v.Value
		}

		// Fetch existing option values (should be empty for new option, but check anyway)
		existingOptionValues, err := s.optionRepo.FindOptionValuesByOptionID(option.ID)
		if err != nil {
			return nil, err
		}

		// Validate bulk values
		if err := validator.ValidateBulkProductOptionValuesUniqueness(values, existingOptionValues); err != nil {
			return nil, err
		}

		// Create option values using factory
		optionValues := factory.CreateOptionValuesFromRequests(option.ID, req.Values)
		if err := s.optionRepo.CreateOptionValues(optionValues); err != nil {
			return nil, err
		}
	}

	// Fetch created option with values
	createdOption, err := s.optionRepo.FindOptionByID(option.ID)
	if err != nil {
		return nil, err
	}

	// Convert to response
	response := factory.BuildProductOptionResponse(createdOption, productID)
	return response, nil
}

/***********************************************
 *    UpdateOption updates an existing option   *
 ***********************************************/
func (s *ProductOptionServiceImpl) UpdateOption(
	productID uint,
	optionID uint,
	sellerId uint,
	req model.ProductOptionUpdateRequest,
) (*model.ProductOptionResponse, error) {
	// Validate product exists and seller has access using validator service
	sellerIDPtr := &sellerId
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Fetch option for validation
	option, err := s.optionRepo.FindOptionByID(optionID)
	if err != nil {
		return nil, err
	}

	// Validate option belongs to product (ownership already validated by validator service)
	if option == nil || option.ProductID != productID {
		return nil, prodErrors.ErrProductOptionMismatch
	}

	// Update option entity using factory
	option = factory.UpdateOptionEntity(option, req)

	// Update option
	if err := s.optionRepo.UpdateOption(option); err != nil {
		return nil, err
	}

	// Fetch updated option with values
	updatedOption, err := s.optionRepo.FindOptionByID(optionID)
	if err != nil {
		return nil, err
	}

	// Convert to response
	response := factory.BuildProductOptionResponse(updatedOption, productID)
	return response, nil
}

/***********************************************
 *    DeleteOption deletes a product option     *
 ***********************************************/
func (s *ProductOptionServiceImpl) DeleteOption(
	productID uint,
	sellerId uint,
	optionID uint,
) error {
	// Validate product exists and seller has access using validator service
	sellerIDPtr := &sellerId
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerIDPtr)
	if err != nil {
		return err
	}

	// Fetch option for validation
	option, err := s.optionRepo.FindOptionByID(optionID)
	if err != nil {
		return err
	}

	// Validate option belongs to product (ownership already validated by validator service)
	if option == nil || option.ProductID != productID {
		return prodErrors.ErrProductOptionMismatch
	}

	// Check if option is in use by variants
	inUse, variantIDs, err := s.optionRepo.CheckOptionInUse(optionID)
	if err != nil {
		return err
	}

	// Validate option is not in use
	if err := validator.ValidateProductOptionNotInUse(inUse, len(variantIDs)); err != nil {
		return err
	}

	// Delete option (cascade deletes option values)
	return s.optionRepo.DeleteOption(optionID)
}

// DeleteOptionsByProductID deletes all product options and their values for a product
// Handles cascade deletion of option_values
func (s *ProductOptionServiceImpl) DeleteOptionsByProductID(productID uint) error {
	// Get all product options
	productOptions, err := s.optionRepo.FindOptionsByProductID(productID)
	if err != nil {
		return nil // If error or no options, nothing to delete
	}

	if len(productOptions) == 0 {
		return nil
	}

	// Delete each option and its values
	for _, option := range productOptions {
		// Delete option values (CASCADE should handle, but explicit is safer)
		if err := s.optionRepo.DeleteOptionValuesByOptionID(option.ID); err != nil {
			return err
		}

		// Delete the option itself
		if err := s.optionRepo.DeleteOption(option.ID); err != nil {
			return err
		}
	}

	return nil
}

/***********************************************
 *            GetAvailableOptions              *
 ***********************************************/
func (s *ProductOptionServiceImpl) GetAvailableOptions(
	productID uint,
	sellerID *uint,
) (*model.GetAvailableOptionsResponse, error) {
	// Validate that the product exists and validate seller access using validator service
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Get all options with variant counts
	options, variantCounts, err := s.optionRepo.GetProductOptionsWithVariantCounts(productID)
	if err != nil {
		return nil, err
	}

	// Convert to response model
	optionResponses := make([]model.ProductOptionDetailResponse, 0, len(options))

	for _, option := range options {
		values := make([]model.OptionValueResponse, 0, len(option.Values))

		for _, value := range option.Values {
			valueResponse := model.OptionValueResponse{
				ValueID:      value.ID,
				Value:        value.Value,
				DisplayName:  helper.GetDisplayNameOrDefault(value.DisplayName, value.Value),
				VariantCount: variantCounts[value.ID],
				Position:     value.Position,
			}

			// Add color code if it exists
			if value.ColorCode != "" {
				valueResponse.ColorCode = value.ColorCode
			}

			values = append(values, valueResponse)
		}

		optionResponse := model.ProductOptionDetailResponse{
			OptionID:          option.ID,
			OptionName:        option.Name,
			OptionDisplayName: helper.GetDisplayNameOrDefault(option.DisplayName, option.Name),
			Position:          option.Position,
			Values:            values,
		}

		optionResponses = append(optionResponses, optionResponse)
	}

	return &model.GetAvailableOptionsResponse{
		ProductID: productID,
		Options:   optionResponses,
	}, nil
}

/***********************************************
 *    GetProductOptionsWithVariantCounts       *
 ***********************************************/
// GetProductOptionsWithVariantCounts retrieves all options with their values and variant counts
// Used for detailed product views to show available options
func (s *ProductOptionServiceImpl) GetProductOptionsWithVariantCounts(
	productID uint,
	sellerID *uint,
) ([]model.ProductOptionDetailResponse, error) {
	// Validate product exists and seller has access using validator service
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Get options with variant counts from repository
	productOptions, variantCounts, err := s.optionRepo.GetProductOptionsWithVariantCounts(productID)
	if err != nil {
		return nil, err
	}

	// Convert to response model using factory
	options := factory.BuildProductOptionsDetailResponse(productOptions, variantCounts)
	return options, nil
}

/***********************************************
 *         BulkUpdateOptions                   *
 ***********************************************/
func (s *ProductOptionServiceImpl) BulkUpdateOptions(
	productID uint,
	sellerId uint,
	req model.ProductOptionBulkUpdateRequest,
) (*model.BulkUpdateResponse, error) {
	// Validate product exists and seller has access using validator service
	sellerIDPtr := &sellerId
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Get all existing options for this product
	existingOptions, err := s.optionRepo.FindOptionsByProductID(productID)
	if err != nil {
		return nil, err
	}

	// Create a map of existing option IDs for validation and lookup
	existingOptionMap := make(map[uint]*entity.ProductOption)
	for i := range existingOptions {
		existingOptionMap[existingOptions[i].ID] = &existingOptions[i]
	}

	// Validate all option IDs belong to this product and prepare updates
	optionsToUpdate := make([]*entity.ProductOption, 0, len(req.Options))
	for _, update := range req.Options {
		existingOption, exists := existingOptionMap[update.OptionID]
		if !exists {
			return nil, prodErrors.ErrProductOptionNotFound
		}

		// Create updated option entity
		updatedOption := &entity.ProductOption{
			ProductID:   productID,
			Name:        existingOption.Name, // Name cannot be changed
			DisplayName: update.DisplayName,
			Position:    update.Position,
		}
		updatedOption.ID = update.OptionID

		// If DisplayName is empty, keep the existing one
		if update.DisplayName == "" {
			updatedOption.DisplayName = existingOption.DisplayName
		}

		optionsToUpdate = append(optionsToUpdate, updatedOption)
	}

	// Perform bulk update
	if err := s.optionRepo.BulkUpdateOptions(optionsToUpdate); err != nil {
		return nil, err
	}

	return &model.BulkUpdateResponse{
		UpdatedCount: len(optionsToUpdate),
		Message:      utils.PRODUCT_OPTIONS_BULK_UPDATED_MSG,
	}, nil
}

// GetProductsOptionsWithValues retrieves all options with their values for multiple products
// Batch operation optimized to prevent N+1 queries
func (s *ProductOptionServiceImpl) GetProductsOptionsWithValues(
	productIDs []uint,
) (map[uint][]entity.ProductOption, error) {
	// Use the option repository's batch method to fetch all options at once
	return s.optionRepo.FindOptionsByProductIDs(productIDs)
}

/***********************************************
 *         CreateOptionsBulk                   *
 ***********************************************/
// CreateOptionsBulk creates multiple product options with their values in bulk
// TRUE BULK: Single INSERT for all options, single INSERT for all option values
func (s *ProductOptionServiceImpl) CreateOptionsBulk(
	productID uint,
	sellerID uint,
	requests []model.ProductOptionCreateRequest,
) ([]model.ProductOptionDetailResponse, error) {
	if len(requests) == 0 {
		return []model.ProductOptionDetailResponse{}, nil
	}

	// Validate product exists and seller has access using validator service (single query)
	sellerIDPtr := &sellerID
	_, err := s.validatorService.GetAndValidateProductOwnership(productID, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Fetch existing options for uniqueness validation (single query)
	existingOptions, err := s.optionRepo.FindOptionsByProductID(productID)
	if err != nil {
		return nil, err
	}

	// Build map of existing option names for quick lookup
	existingNames := make(map[string]bool)
	for _, opt := range existingOptions {
		existingNames[opt.Name] = true
	}

	// Validate all option names are unique (against existing and within request)
	requestNames := make(map[string]bool)
	for _, req := range requests {
		// Check against existing options
		if existingNames[req.Name] {
			return nil, prodErrors.ErrProductOptionNameExists
		}
		// Check for duplicates within request
		if requestNames[req.Name] {
			return nil, prodErrors.ErrProductOptionNameExists
		}
		requestNames[req.Name] = true
	}

	// Prepare all options for bulk insert
	optionsToCreate := make([]*entity.ProductOption, 0, len(requests))

	for _, req := range requests {
		// Create option entity using factory
		option := factory.CreateOptionFromRequest(productID, req)
		optionsToCreate = append(optionsToCreate, option)
	}

	// ✅ TRUE BULK: Create ALL options in ONE query with RETURNING
	if err := s.optionRepo.BulkCreateOptions(optionsToCreate); err != nil {
		return nil, err
	}

	// Now prepare all option values with the generated option IDs
	allOptionValues := make([]*entity.ProductOptionValue, 0)

	for i, req := range requests {
		option := optionsToCreate[i] // Has ID now from BulkCreateOptions

		if len(req.Values) > 0 {
			// Validate bulk values uniqueness within the request
			valueSet := make(map[string]bool)
			for _, v := range req.Values {
				if valueSet[v.Value] {
					return nil, prodErrors.ErrProductOptionValueExists
				}
				valueSet[v.Value] = true
			}

			// Create option values using factory
			optionValues := factory.CreateOptionValuesFromRequests(option.ID, req.Values)

			// Add to bulk insert list
			for j := range optionValues {
				allOptionValues = append(allOptionValues, &optionValues[j])
			}

			// Set values on option for return
			option.Values = optionValues
		}
	}

	// ✅ TRUE BULK: Insert ALL option values in ONE query
	if len(allOptionValues) > 0 {
		if err := s.optionRepo.BulkCreateOptionValues(allOptionValues); err != nil {
			return nil, err
		}
	}

	// Convert pointers to values for factory
	createdOptions := make([]entity.ProductOption, 0, len(optionsToCreate))
	for _, opt := range optionsToCreate {
		createdOptions = append(createdOptions, *opt)
	}

	// Convert entities to models using factory (no variant counts yet)
	emptyVariantCounts := make(map[uint]int)
	return factory.BuildProductOptionsDetailResponse(createdOptions, emptyVariantCounts), nil
}
