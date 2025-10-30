package service

import (
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"
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
}

// ProductOptionServiceImpl implements the ProductOptionService interface
type ProductOptionServiceImpl struct {
	optionRepo      repositories.ProductOptionRepository
	productRepo     repositories.ProductRepository
	optionValidator *validator.ProductOptionValidator
	valueValidator  *validator.ProductOptionValueValidator
	optionFactory   *factory.ProductOptionFactory
	valueFactory    *factory.ProductOptionValueFactory
}

// NewProductOptionService creates a new instance of ProductOptionService
func NewProductOptionService(
	optionRepo repositories.ProductOptionRepository,
	productRepo repositories.ProductRepository,
) ProductOptionService {
	return &ProductOptionServiceImpl{
		optionRepo:      optionRepo,
		productRepo:     productRepo,
		optionValidator: validator.NewProductOptionValidator(optionRepo, productRepo),
		valueValidator:  validator.NewProductOptionValueValidator(optionRepo, productRepo),
		optionFactory:   factory.NewProductOptionFactory(),
		valueFactory:    factory.NewProductOptionValueFactory(),
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
	// Validate product exists
	if err := s.optionValidator.ValidateProductExists(productID); err != nil {
		return nil, err
	}

	if err := s.optionValidator.ValidateProductBelongsToSeller(productID, sellerId); err != nil {
		return nil, err
	}

	// Validate option name uniqueness
	if err := s.optionValidator.ValidateOptionNameUniqueness(productID, req.Name); err != nil {
		return nil, err
	}

	// Create option entity using factory
	option := s.optionFactory.CreateOptionFromRequest(productID, req)

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

		// Validate bulk values
		if err := s.valueValidator.ValidateBulkOptionValuesUniqueness(option.ID, values); err != nil {
			return nil, err
		}

		// Create option values using factory
		optionValues := s.valueFactory.CreateOptionValuesFromRequests(option.ID, req.Values)
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
	response := s.optionFactory.BuildProductOptionResponse(createdOption, productID)
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
	// Validate product and option
	if err := s.valueValidator.ValidateProductAndOption(productID, optionID); err != nil {
		return nil, err
	}

	if err := s.optionValidator.ValidateProductBelongsToSeller(productID, sellerId); err != nil {
		return nil, err
	}

	// Fetch option
	option, err := s.optionRepo.FindOptionByID(optionID)
	if err != nil {
		return nil, err
	}

	// Update option entity using factory
	option = s.optionFactory.UpdateOptionEntity(option, req)

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
	response := s.optionFactory.BuildProductOptionResponse(updatedOption, productID)
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
	// Validate product and option
	if err := s.valueValidator.ValidateProductAndOption(productID, optionID); err != nil {
		return err
	}

	if err := s.optionValidator.ValidateProductBelongsToSeller(productID, sellerId); err != nil {
		return err
	}

	// Validate option is not in use
	if err := s.optionValidator.ValidateOptionNotInUse(optionID); err != nil {
		return err
	}

	// Delete option (cascade deletes option values)
	return s.optionRepo.DeleteOption(optionID)
}

/***********************************************
 *            GetAvailableOptions              *
 ***********************************************/
func (s *ProductOptionServiceImpl) GetAvailableOptions(
	productID uint,
	sellerID *uint,
) (*model.GetAvailableOptionsResponse, error) {
	// Validate that the product exists and validate seller access
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Validate seller access: if sellerID is provided (non-admin), check ownership
	if sellerID != nil && product.SellerID != *sellerID {
		return nil, prodErrors.ErrProductNotFound
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
				DisplayName:  utils.GetDisplayNameOrDefault(value.DisplayName, value.Value),
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
			OptionDisplayName: utils.GetDisplayNameOrDefault(option.DisplayName, option.Name),
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
 *         BulkUpdateOptions                   *
 ***********************************************/
func (s *ProductOptionServiceImpl) BulkUpdateOptions(
	productID uint,
	sellerId uint,
	req model.ProductOptionBulkUpdateRequest,
) (*model.BulkUpdateResponse, error) {
	// Validate product exists
	if err := s.optionValidator.ValidateProductExists(productID); err != nil {
		return nil, err
	}

	if err := s.optionValidator.ValidateProductBelongsToSeller(productID, sellerId); err != nil {
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
