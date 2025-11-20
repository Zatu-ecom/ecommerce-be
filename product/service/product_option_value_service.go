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

// ProductOptionValueService defines the interface for product option value-related business logic
type ProductOptionValueService interface {
	AddOptionValue(
		productID uint,
		optionID uint,
		sellerID uint,
		req model.ProductOptionValueRequest,
	) (*model.ProductOptionValueResponse, error)
	UpdateOptionValue(
		productID uint,
		optionID uint,
		valueID uint,
		sellerID uint,
		req model.ProductOptionValueUpdateRequest,
	) (*model.ProductOptionValueResponse, error)
	DeleteOptionValue(
		productID uint,
		optionID uint,
		sellerID uint,
		valueID uint,
	) error
	BulkAddOptionValues(
		productID uint,
		optionID uint,
		sellerID uint,
		req model.ProductOptionValueBulkAddRequest,
	) ([]model.ProductOptionValueResponse, error)
	BulkUpdateOptionValues(
		productID uint,
		optionID uint,
		sellerID uint,
		req model.ProductOptionValueBulkUpdateRequest,
	) (*model.BulkUpdateResponse, error)
}

// ProductOptionValueServiceImpl implements the ProductOptionValueService interface
type ProductOptionValueServiceImpl struct {
	optionRepo  repositories.ProductOptionRepository
	productRepo repositories.ProductRepository
}

// NewProductOptionValueService creates a new instance of ProductOptionValueService
func NewProductOptionValueService(
	optionRepo repositories.ProductOptionRepository,
	productRepo repositories.ProductRepository,
) ProductOptionValueService {
	return &ProductOptionValueServiceImpl{
		optionRepo:  optionRepo,
		productRepo: productRepo,
	}
}

/***********************************************
 *    AddOptionValue adds a value to an option  *
 ***********************************************/
func (s *ProductOptionValueServiceImpl) AddOptionValue(
	productID uint,
	optionID uint,
	sellerID uint,
	req model.ProductOptionValueRequest,
) (*model.ProductOptionValueResponse, error) {
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
	if err := validator.ValidateSellerProductAndOption(sellerID, productID, product, option); err != nil {
		return nil, err
	}

	// Fetch existing option values for uniqueness check
	existingValues, err := s.optionRepo.FindOptionValuesByOptionID(optionID)
	if err != nil {
		return nil, err
	}

	// Validate value uniqueness
	if err := validator.ValidateProductOptionValueUniqueness(req.Value, existingValues); err != nil {
		return nil, err
	}

	// Create option value entity using factory
	optionValue := factory.CreateOptionValueFromRequest(optionID, req)

	// Create option value
	if err := s.optionRepo.CreateOptionValue(optionValue); err != nil {
		return nil, err
	}

	// Convert to response
	response := factory.BuildProductOptionValueResponse(optionValue)
	return response, nil
}

/***********************************************
 *    UpdateOptionValue updates a value         *
 ***********************************************/
func (s *ProductOptionValueServiceImpl) UpdateOptionValue(
	productID uint,
	optionID uint,
	valueID uint,
	sellerID uint,
	req model.ProductOptionValueUpdateRequest,
) (*model.ProductOptionValueResponse, error) {
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
	if err := validator.ValidateSellerProductAndOption(sellerID, productID, product, option); err != nil {
		return nil, err
	}

	// Fetch option value to update
	optionValue, err := s.optionRepo.FindOptionValueByID(valueID)
	if err != nil {
		return nil, err
	}

	// Validate option value belongs to option
	if err := validator.ValidateProductOptionValueBelongsToOption(optionID, optionValue); err != nil {
		return nil, err
	}

	// Update entity using factory
	factory.UpdateOptionValueEntity(optionValue, req)

	// Update option value
	if err := s.optionRepo.UpdateOptionValue(optionValue); err != nil {
		return nil, err
	}

	// Convert to response
	response := factory.BuildProductOptionValueResponse(optionValue)
	return response, nil
}

/***********************************************
 *    DeleteOptionValue deletes a value         *
 ***********************************************/
func (s *ProductOptionValueServiceImpl) DeleteOptionValue(
	productID uint,
	optionID uint,
	sellerID uint,
	valueID uint,
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
	if err := validator.ValidateSellerProductAndOption(sellerID, productID, product, option); err != nil {
		return err
	}

	// Fetch option value for validation
	optionValue, err := s.optionRepo.FindOptionValueByID(valueID)
	if err != nil {
		return err
	}

	// Validate option value belongs to option
	if err := validator.ValidateProductOptionValueBelongsToOption(optionID, optionValue); err != nil {
		return err
	}

	// Check if value is being used by any variants
	inUse, variantIDs, err := s.optionRepo.CheckOptionValueInUse(valueID)
	if err != nil {
		return err
	}

	if err := validator.ValidateProductOptionValueNotInUse(inUse, len(variantIDs)); err != nil {
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
	sellerID uint,
	req model.ProductOptionValueBulkAddRequest,
) ([]model.ProductOptionValueResponse, error) {
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
	if err := validator.ValidateSellerProductAndOption(sellerID, productID, product, option); err != nil {
		return nil, err
	}

	// Extract values for validation
	values := make([]string, len(req.Values))
	for i, v := range req.Values {
		values[i] = v.Value
	}

	// Fetch existing option values
	existingValues, err := s.optionRepo.FindOptionValuesByOptionID(optionID)
	if err != nil {
		return nil, err
	}

	// Validate bulk uniqueness (checks both existing values and within batch)
	if err := validator.ValidateBulkProductOptionValuesUniqueness(values, existingValues); err != nil {
		return nil, err
	}

	// Create option values using factory
	valuesToCreate := factory.CreateOptionValuesFromRequests(optionID, req.Values)

	// Bulk create option values
	if err := s.optionRepo.CreateOptionValues(valuesToCreate); err != nil {
		return nil, err
	}

	// Convert to response
	var responses []model.ProductOptionValueResponse
	for i := range valuesToCreate {
		response := factory.BuildProductOptionValueResponse(&valuesToCreate[i])
		responses = append(responses, *response)
	}

	return responses, nil
}

/***********************************************
 *       BulkUpdateOptionValues                *
 ***********************************************/
func (s *ProductOptionValueServiceImpl) BulkUpdateOptionValues(
	productID uint,
	optionID uint,
	sellerID uint,
	req model.ProductOptionValueBulkUpdateRequest,
) (*model.BulkUpdateResponse, error) {
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
	if err := validator.ValidateSellerProductAndOption(sellerID, productID, product, option); err != nil {
		return nil, err
	}

	// Get all existing values for this option
	existingValues, err := s.optionRepo.FindOptionValuesByOptionID(optionID)
	if err != nil {
		return nil, err
	}

	// Create a map of existing value IDs for validation and lookup
	existingValueMap := make(map[uint]*entity.ProductOptionValue)
	for i := range existingValues {
		existingValueMap[existingValues[i].ID] = &existingValues[i]
	}

	// Validate all value IDs belong to this option and prepare updates
	valuesToUpdate := make([]*entity.ProductOptionValue, 0, len(req.Values))
	for _, update := range req.Values {
		existingValue, exists := existingValueMap[update.ValueID]
		if !exists {
			return nil, prodErrors.ErrProductOptionValueNotFound
		}

		// Create updated value entity
		updatedValue := &entity.ProductOptionValue{
			OptionID:    optionID,
			Value:       existingValue.Value, // Value cannot be changed
			DisplayName: update.DisplayName,
			ColorCode:   update.ColorCode,
			Position:    update.Position,
		}
		updatedValue.ID = update.ValueID

		// If fields are empty, keep the existing ones
		if update.DisplayName == "" {
			updatedValue.DisplayName = existingValue.DisplayName
		}
		if update.ColorCode == "" {
			updatedValue.ColorCode = existingValue.ColorCode
		}

		valuesToUpdate = append(valuesToUpdate, updatedValue)
	}

	// Perform bulk update
	if err := s.optionRepo.BulkUpdateOptionValues(valuesToUpdate); err != nil {
		return nil, err
	}

	return &model.BulkUpdateResponse{
		UpdatedCount: len(valuesToUpdate),
		Message:      utils.OPTION_VALUES_BULK_UPDATED_MSG,
	}, nil
}
