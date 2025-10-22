package service

import (
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
		req model.ProductOptionCreateRequest,
	) (*model.ProductOptionResponse, error)
	UpdateOption(
		productID uint,
		optionID uint,
		req model.ProductOptionUpdateRequest,
	) (*model.ProductOptionResponse, error)
	DeleteOption(productID uint, optionID uint) error

	// GetAvailableOptions retrieves all available options and their values for a product
	GetAvailableOptions(productID uint, sellerID *uint) (*model.GetAvailableOptionsResponse, error)
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
	req model.ProductOptionCreateRequest,
) (*model.ProductOptionResponse, error) {
	// Validate product exists
	if err := s.optionValidator.ValidateProductExists(productID); err != nil {
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
	req model.ProductOptionUpdateRequest,
) (*model.ProductOptionResponse, error) {
	// Validate product and option
	if err := s.valueValidator.ValidateProductAndOption(productID, optionID); err != nil {
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
	optionID uint,
) error {
	// Validate product and option
	if err := s.valueValidator.ValidateProductAndOption(productID, optionID); err != nil {
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
				ValueID:          value.ID,
				Value:            value.Value,
				DisplayName: utils.GetDisplayNameOrDefault(value.DisplayName, value.Value),
				VariantCount:     variantCounts[value.ID],
				Position:         value.Position,
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
