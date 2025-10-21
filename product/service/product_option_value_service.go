package service

import (
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"
)

// ProductOptionValueService defines the interface for product option value-related business logic
type ProductOptionValueService interface {
	AddOptionValue(
		productID uint,
		optionID uint,
		req model.ProductOptionValueRequest,
	) (*model.ProductOptionValueResponse, error)
	UpdateOptionValue(
		productID uint,
		optionID uint,
		valueID uint,
		req model.ProductOptionValueUpdateRequest,
	) (*model.ProductOptionValueResponse, error)
	DeleteOptionValue(productID uint, optionID uint, valueID uint) error
	BulkAddOptionValues(
		productID uint,
		optionID uint,
		req model.ProductOptionValueBulkAddRequest,
	) ([]model.ProductOptionValueResponse, error)
}

// ProductOptionValueServiceImpl implements the ProductOptionValueService interface
type ProductOptionValueServiceImpl struct {
	optionRepo  repositories.ProductOptionRepository
	productRepo repositories.ProductRepository
	validator   *validator.ProductOptionValueValidator
	factory     *factory.ProductOptionValueFactory
}

// NewProductOptionValueService creates a new instance of ProductOptionValueService
func NewProductOptionValueService(
	optionRepo repositories.ProductOptionRepository,
	productRepo repositories.ProductRepository,
) ProductOptionValueService {
	return &ProductOptionValueServiceImpl{
		optionRepo:  optionRepo,
		productRepo: productRepo,
		validator:   validator.NewProductOptionValueValidator(optionRepo, productRepo),
		factory:     factory.NewProductOptionValueFactory(),
	}
}

/***********************************************
 *    AddOptionValue adds a value to an option  *
 ***********************************************/
func (s *ProductOptionValueServiceImpl) AddOptionValue(
	productID uint,
	optionID uint,
	req model.ProductOptionValueRequest,
) (*model.ProductOptionValueResponse, error) {
	// Validate product and option
	if err := s.validator.ValidateProductAndOption(productID, optionID); err != nil {
		return nil, err
	}

	// Validate value uniqueness
	if err := s.validator.ValidateOptionValueUniqueness(optionID, req.Value); err != nil {
		return nil, err
	}

	// Create option value entity using factory
	optionValue := s.factory.CreateOptionValueFromRequest(optionID, req)

	// Create option value
	if err := s.optionRepo.CreateOptionValue(optionValue); err != nil {
		return nil, err
	}

	// Convert to response
	response := s.factory.BuildProductOptionValueResponse(optionValue)
	return response, nil
}

/***********************************************
 *    UpdateOptionValue updates a value         *
 ***********************************************/
func (s *ProductOptionValueServiceImpl) UpdateOptionValue(
	productID uint,
	optionID uint,
	valueID uint,
	req model.ProductOptionValueUpdateRequest,
) (*model.ProductOptionValueResponse, error) {
	// Validate product and option
	if err := s.validator.ValidateProductAndOption(productID, optionID); err != nil {
		return nil, err
	}

	// Validate option value belongs to option
	if err := s.validator.ValidateOptionValueBelongsToOption(valueID, optionID); err != nil {
		return nil, err
	}

	// Fetch option value to update
	optionValue, err := s.optionRepo.FindOptionValueByID(valueID)
	if err != nil {
		return nil, err
	}

	// Update entity using factory
	s.factory.UpdateOptionValueEntity(optionValue, req)

	// Update option value
	if err := s.optionRepo.UpdateOptionValue(optionValue); err != nil {
		return nil, err
	}

	// Convert to response
	response := s.factory.BuildProductOptionValueResponse(optionValue)
	return response, nil
}

/***********************************************
 *    DeleteOptionValue deletes a value         *
 ***********************************************/
func (s *ProductOptionValueServiceImpl) DeleteOptionValue(
	productID uint,
	optionID uint,
	valueID uint,
) error {
	// Validate product and option
	if err := s.validator.ValidateProductAndOption(productID, optionID); err != nil {
		return err
	}

	// Validate option value belongs to option
	if err := s.validator.ValidateOptionValueBelongsToOption(valueID, optionID); err != nil {
		return err
	}

	// Check if value is being used by any variants
	if err := s.validator.ValidateOptionValueNotInUse(valueID); err != nil {
		return err
	}

	// Delete option value
	return s.optionRepo.DeleteOptionValue(valueID)
}

/***********************************************
 *    BulkAddOptionValues adds multiple values  *
 ***********************************************/
func (s *ProductOptionValueServiceImpl) BulkAddOptionValues(
	productID uint,
	optionID uint,
	req model.ProductOptionValueBulkAddRequest,
) ([]model.ProductOptionValueResponse, error) {
	// Validate product and option
	if err := s.validator.ValidateProductAndOption(productID, optionID); err != nil {
		return nil, err
	}

	// Extract values for validation
	values := make([]string, len(req.Values))
	for i, v := range req.Values {
		values[i] = v.Value
	}

	// Validate bulk uniqueness (checks both existing values and within batch)
	if err := s.validator.ValidateBulkOptionValuesUniqueness(optionID, values); err != nil {
		return nil, err
	}

	// Create option values using factory
	valuesToCreate := s.factory.CreateOptionValuesFromRequests(optionID, req.Values)

	// Bulk create option values
	if err := s.optionRepo.CreateOptionValues(valuesToCreate); err != nil {
		return nil, err
	}

	// Convert to response
	var responses []model.ProductOptionValueResponse
	for i := range valuesToCreate {
		response := s.factory.BuildProductOptionValueResponse(&valuesToCreate[i])
		responses = append(responses, *response)
	}

	return responses, nil
}
