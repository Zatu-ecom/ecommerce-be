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
}

// ProductOptionServiceImpl implements the ProductOptionService interface
type ProductOptionServiceImpl struct {
	optionRepo    repositories.ProductOptionRepository
	productRepo   repositories.ProductRepository
}

// NewProductOptionService creates a new instance of ProductOptionService
func NewProductOptionService(
	optionRepo repositories.ProductOptionRepository,
	productRepo repositories.ProductRepository,
) ProductOptionService {
	return &ProductOptionServiceImpl{
		optionRepo:    optionRepo,
		productRepo:   productRepo,
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
	// Fetch product for validation
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Validate product exists
	if err := validator.ValidateProductExists(product); err != nil {
		return nil, err
	}

	// Validate product belongs to seller
	if err := validator.ValidateProductBelongsToSeller(product, sellerId); err != nil {
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
	// Fetch product for validation
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Fetch option for validation
	option, err := s.optionRepo.FindOptionByID(optionID)
	if err != nil {
		return nil, err
	}

	// Validate product and option
	if err := validator.ValidateSellerProductAndOption(sellerId, productID, product, option); err != nil {
		return nil, err
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
	// Fetch product for validation
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return err
	}

	// Fetch option for validation
	option, err := s.optionRepo.FindOptionByID(optionID)
	if err != nil {
		return err
	}

	// Validate product and option
	if err := validator.ValidateSellerProductAndOption(sellerId, productID, product, option); err != nil {
		return err
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
 *         BulkUpdateOptions                   *
 ***********************************************/
func (s *ProductOptionServiceImpl) BulkUpdateOptions(
	productID uint,
	sellerId uint,
	req model.ProductOptionBulkUpdateRequest,
) (*model.BulkUpdateResponse, error) {
	// Fetch product for validation
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Validate product exists
	if err := validator.ValidateProductExists(product); err != nil {
		return nil, err
	}

	// Validate product belongs to seller
	if err := validator.ValidateProductBelongsToSeller(product, sellerId); err != nil {
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
