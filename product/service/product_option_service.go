package service

import (
	"errors"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
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
	GetAvailableOptions(productID uint) (*model.GetAvailableOptionsResponse, error)
}

// ProductOptionServiceImpl implements the ProductOptionService interface
type ProductOptionServiceImpl struct {
	optionRepo  repositories.ProductOptionRepository
	productRepo repositories.ProductRepository
}

// NewProductOptionService creates a new instance of ProductOptionService
func NewProductOptionService(
	optionRepo repositories.ProductOptionRepository,
	productRepo repositories.ProductRepository,
) ProductOptionService {
	return &ProductOptionServiceImpl{
		optionRepo:  optionRepo,
		productRepo: productRepo,
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
	product, err := s.validateProductExists(productID)
	if err != nil {
		return nil, err
	}

	// Normalize option name
	normalizedName := utils.NormalizeToSnakeCase(req.Name)

	// Check if option name is unique
	if err := s.checkOptionNameUniqueness(productID, normalizedName); err != nil {
		return nil, err
	}

	// Create option entity
	option := &entity.ProductOption{
		ProductID:   productID,
		Name:        normalizedName,
		DisplayName: req.DisplayName,
		Position:    req.Position,
	}

	// Create option
	if err := s.optionRepo.CreateOption(option); err != nil {
		return nil, err
	}

	// Create option values if provided
	if err := s.validateAndCreateOptionValues(option.ID, req.Values); err != nil {
		return nil, err
	}

	// Fetch created option with values
	createdOption, err := s.optionRepo.FindOptionByID(option.ID)
	if err != nil {
		return nil, err
	}

	// Convert to response
	response := utils.ConvertProductOptionToResponse(createdOption, product.ID)
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
	// Validate product exists
	if _, err := s.validateProductExists(productID); err != nil {
		return nil, err
	}

	// Validate option belongs to product
	option, err := s.validateOptionBelongsToProduct(productID, optionID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.DisplayName != "" {
		option.DisplayName = req.DisplayName
	}
	if req.Position != 0 || req.Position != option.Position {
		option.Position = req.Position
	}

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
	response := utils.ConvertProductOptionToResponse(updatedOption, productID)
	return response, nil
}

/***********************************************
 *    DeleteOption deletes a product option     *
 ***********************************************/
func (s *ProductOptionServiceImpl) DeleteOption(
	productID uint,
	optionID uint,
) error {
	// Validate product exists
	if _, err := s.validateProductExists(productID); err != nil {
		return err
	}

	// Validate option belongs to product
	option, err := s.validateOptionBelongsToProduct(productID, optionID)
	if err != nil {
		return err
	}

	// Check if option is being used by any variants
	inUse, variantIDs, err := s.optionRepo.CheckOptionInUse(optionID)
	if err != nil {
		return err
	}

	if inUse {
		// Return error with details about affected variants
		return &OptionInUseError{
			OptionID:         optionID,
			OptionName:       option.Name,
			VariantCount:     len(variantIDs),
			AffectedVariants: variantIDs,
		}
	}

	// Delete option (cascade deletes option values)
	return s.optionRepo.DeleteOption(optionID)
}

/***********************************************
 *    Custom Errors                             *
 ***********************************************/

// OptionInUseError represents an error when trying to delete an option that's in use
type OptionInUseError struct {
	OptionID         uint
	OptionName       string
	VariantCount     int
	AffectedVariants []uint
}

func (e *OptionInUseError) Error() string {
	return utils.PRODUCT_OPTION_IN_USE_MSG
}

/***********************************************
 *            GetAvailableOptions              *
 ***********************************************/
func (s *ProductOptionServiceImpl) GetAvailableOptions(
	productID uint,
) (*model.GetAvailableOptionsResponse, error) {
	// Validate that the product exists
	_, err := s.productRepo.FindByID(productID)
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
				ValueID:          value.ID,
				Value:            value.Value,
				ValueDisplayName: utils.GetDisplayNameOrDefault(value.DisplayName, value.Value),
				VariantCount:     variantCounts[value.ID],
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
 *          Helper Methods                      *
 ***********************************************/

// validateProductExists validates that a product exists
func (s *ProductOptionServiceImpl) validateProductExists(productID uint) (*entity.Product, error) {
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.PRODUCT_NOT_FOUND_MSG)
		}
		return nil, err
	}
	return product, nil
}

// checkOptionNameUniqueness checks if an option name is unique for a product
func (s *ProductOptionServiceImpl) checkOptionNameUniqueness(
	productID uint,
	normalizedName string,
) error {
	existingOptions, err := s.optionRepo.FindOptionsByProductID(productID)
	if err != nil {
		return err
	}

	for _, opt := range existingOptions {
		if opt.Name == normalizedName {
			return errors.New(utils.PRODUCT_OPTION_NAME_EXISTS_MSG)
		}
	}
	return nil
}

// validateAndCreateOptionValues validates and creates option values
func (s *ProductOptionServiceImpl) validateAndCreateOptionValues(
	optionID uint,
	valueRequests []model.ProductOptionValueRequest,
) error {
	if len(valueRequests) == 0 {
		return nil
	}

	// Get existing values from DB
	existingValues, err := s.optionRepo.FindOptionValuesByOptionID(optionID)
	if err != nil {
		return err
	}

	// Create a map of existing values for quick lookup
	existingValueMap := make(map[string]bool)
	for _, val := range existingValues {
		existingValueMap[val.Value] = true
	}

	var optionValues []entity.ProductOptionValue
	valueSet := make(map[string]bool) // Track unique values in current request

	for _, valueReq := range valueRequests {
		optionValue := utils.ConvertProductOptionValueRequestToEntity(valueReq, optionID)
		optionValue.Value = utils.ToLowerTrimmed(optionValue.Value)

		// Check if value already exists in DB
		if existingValueMap[optionValue.Value] {
			return errors.New(utils.PRODUCT_OPTION_VALUE_EXISTS_MSG + ": " + optionValue.Value)
		}

		// Check for duplicate values in the same request
		if valueSet[optionValue.Value] {
			return errors.New(
				utils.PRODUCT_OPTION_VALUE_DUPLICATE_IN_BATCH_MSG + ": " + optionValue.Value,
			)
		}
		valueSet[optionValue.Value] = true

		optionValues = append(optionValues, optionValue)
	}

	return s.optionRepo.CreateOptionValues(optionValues)
}

// validateOptionBelongsToProduct validates that an option belongs to a product
func (s *ProductOptionServiceImpl) validateOptionBelongsToProduct(
	productID uint,
	optionID uint,
) (*entity.ProductOption, error) {
	option, err := s.optionRepo.FindOptionByID(optionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.PRODUCT_OPTION_NOT_FOUND_MSG)
		}
		return nil, err
	}

	if option.ProductID != productID {
		return nil, errors.New(utils.PRODUCT_OPTION_PRODUCT_MISMATCH_MSG)
	}

	return option, nil
}
