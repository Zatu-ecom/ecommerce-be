package service

import (
	"errors"

	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
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
	req model.ProductOptionValueRequest,
) (*model.ProductOptionValueResponse, error) {
	// Validate product and option
	_, err := validateProductAndOption(s.productRepo, s.optionRepo, productID, optionID)
	if err != nil {
		return nil, err
	}

	// Check if value already exists for this option
	existingValues, err := s.optionRepo.FindOptionValuesByOptionID(optionID)
	if err != nil {
		return nil, err
	}

	normalizedValue := utils.ToLowerTrimmed(req.Value)
	for _, val := range existingValues {
		if val.Value == normalizedValue {
			return nil, prodErrors.ErrProductOptionValueExists
		}
	}

	// Create option value entity
	optionValue := utils.ConvertProductOptionValueRequestToEntity(req, optionID)
	optionValue.Value = normalizedValue

	// Create option value
	if err := s.optionRepo.CreateOptionValue(&optionValue); err != nil {
		return nil, err
	}

	// Convert to response
	response := utils.ConvertProductOptionValueToResponse(&optionValue)
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
	_, err := validateProductAndOption(s.productRepo, s.optionRepo, productID, optionID)
	if err != nil {
		return nil, err
	}

	// Validate option value
	optionValue, err := validateOptionValue(s.optionRepo, optionID, valueID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.DisplayName != "" {
		optionValue.DisplayName = req.DisplayName
	}
	if req.ColorCode != "" {
		optionValue.ColorCode = req.ColorCode
	}
	if req.Position != 0 {
		optionValue.Position = req.Position
	}

	// Update option value
	if err := s.optionRepo.UpdateOptionValue(optionValue); err != nil {
		return nil, err
	}

	// Fetch updated option value
	updatedValue, err := s.optionRepo.FindOptionValueByID(valueID)
	if err != nil {
		return nil, err
	}

	// Convert to response
	response := utils.ConvertProductOptionValueToResponse(updatedValue)
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
	_, err := validateProductAndOption(s.productRepo, s.optionRepo, productID, optionID)
	if err != nil {
		return err
	}

	// Validate option value
	optionValue, err := validateOptionValue(s.optionRepo, optionID, valueID)
	if err != nil {
		return err
	}

	// Check if value is being used by any variants
	inUse, variantIDs, err := s.optionRepo.CheckOptionValueInUse(valueID)
	if err != nil {
		return err
	}

	if inUse {
		// Return error with details about affected variants
		return &OptionValueInUseError{
			OptionValueID:    valueID,
			OptionValue:      optionValue.Value,
			VariantCount:     len(variantIDs),
			AffectedVariants: variantIDs,
		}
	}

	// Delete option value
	if err := s.optionRepo.DeleteOptionValue(valueID); err != nil {
		return err
	}

	return nil
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
	_, err := validateProductAndOption(s.productRepo, s.optionRepo, productID, optionID)
	if err != nil {
		return nil, err
	}

	// Get existing values for duplicate check
	existingValues, err := s.optionRepo.FindOptionValuesByOptionID(optionID)
	if err != nil {
		return nil, err
	}

	// Create a map of existing values for quick lookup
	existingValueMap := make(map[string]bool)
	for _, val := range existingValues {
		existingValueMap[val.Value] = true
	}

	// Prepare values to create
	var valuesToCreate []entity.ProductOptionValue
	var normalizedValues []string

	for _, valueReq := range req.Values {
		normalizedValue := utils.ToLowerTrimmed(valueReq.Value)

		// Check for duplicates in existing values
		if existingValueMap[normalizedValue] {
			return nil, prodErrors.ErrProductOptionValueExists.WithMessagef("%s: %s",
				utils.PRODUCT_OPTION_VALUE_EXISTS_MSG, normalizedValue)
		}

		// Check for duplicates within the current batch
		isDuplicate := false
		for _, nv := range normalizedValues {
			if nv == normalizedValue {
				isDuplicate = true
				break
			}
		}

		if isDuplicate {
			return nil, prodErrors.ErrProductOptionValueExists.WithMessagef("%s: %s",
				utils.PRODUCT_OPTION_VALUE_DUPLICATE_IN_BATCH_MSG, normalizedValue)
		}

		normalizedValues = append(normalizedValues, normalizedValue)

		optionValue := utils.ConvertProductOptionValueRequestToEntity(valueReq, optionID)
		optionValue.Value = normalizedValue
		valuesToCreate = append(valuesToCreate, optionValue)
	}

	// Bulk create option values
	if err := s.optionRepo.CreateOptionValues(valuesToCreate); err != nil {
		return nil, err
	}

	// Convert to response
	var responses []model.ProductOptionValueResponse
	for i := range valuesToCreate {
		response := utils.ConvertProductOptionValueToResponse(&valuesToCreate[i])
		responses = append(responses, *response)
	}

	return responses, nil
}

/***********************************************
 *    Custom Errors                             *
 ***********************************************/

// OptionValueInUseError represents an error when trying to delete a value that's in use
type OptionValueInUseError struct {
	OptionValueID    uint
	OptionValue      string
	VariantCount     int
	AffectedVariants []uint
}

func (e *OptionValueInUseError) Error() string {
	return utils.PRODUCT_OPTION_VALUE_IN_USE_MSG
}


/***********************************************
 *    Helper Functions                         *
 ***********************************************/

 // validateProductAndOption validates that product and option exist and belong together
func validateProductAndOption(
	productRepo repositories.ProductRepository,
	optionRepo repositories.ProductOptionRepository,
	productID uint,
	optionID uint,
) (*entity.ProductOption, error) {
	// Validate product exists
	_, err := productRepo.FindByID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductNotFound
		}
		return nil, err
	}

	// Fetch existing option
	option, err := optionRepo.FindOptionByID(optionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionNotFound
		}
		return nil, err
	}

	// Verify option belongs to product
	if option.ProductID != productID {
		return nil, prodErrors.ErrProductOptionMismatch
	}

	return option, nil
}

// validateOptionValue validates that an option value exists and belongs to the given option
func validateOptionValue(
	optionRepo repositories.ProductOptionRepository,
	optionID uint,
	valueID uint,
) (*entity.ProductOptionValue, error) {
	// Fetch existing option value
	optionValue, err := optionRepo.FindOptionValueByID(valueID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, prodErrors.ErrProductOptionValueNotFound
		}
		return nil, err
	}

	// Verify value belongs to option
	if optionValue.OptionID != optionID {
		return nil, prodErrors.ErrProductOptionValueMismatch
	}

	return optionValue, nil
}
