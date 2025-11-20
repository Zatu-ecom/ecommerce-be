package service

import (
	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"

	"gorm.io/gorm"
)

// VariantService defines the interface for single variant mutation operations (CQRS Command side)
// For bulk operations, use VariantBulkService
type VariantService interface {
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
}

// VariantServiceImpl implements the VariantService interface
type VariantServiceImpl struct {
	variantRepo      repositories.VariantRepository
	optionService    ProductOptionService
	validatorService ProductValidatorService
	queryService     VariantQueryService
}

// NewVariantService creates a new instance of VariantService
func NewVariantService(
	variantRepo repositories.VariantRepository,
	optionService ProductOptionService,
	validatorService ProductValidatorService,
	queryService VariantQueryService,
) VariantService {
	return &VariantServiceImpl{
		variantRepo:      variantRepo,
		optionService:    optionService,
		validatorService: validatorService,
		queryService:     queryService,
	}
}

/***********************************************
 *              CreateVariant                  *
 ***********************************************/
func (s *VariantServiceImpl) CreateVariant(
	productID uint,
	sellerID uint,
	request *model.CreateVariantRequest,
) (*model.VariantDetailResponse, error) {
	// Get product and validate seller access
	product, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Get all available options
	optionsResponse, err := s.optionService.GetAvailableOptions(productID, &sellerID)
	if err != nil {
		return nil, err
	}

	// Validate and map variant options (extracted for readability)
	optionValueIDs, optionsMap, err := s.validateAndMapVariantOptions(
		request.Options,
		optionsResponse,
	)
	if err != nil {
		return nil, err
	}

	// Create variant entity using factory
	variant := factory.CreateVariantFromRequest(productID, request)

	// Store variant option values for response mapping
	var variantOptionValues []entity.VariantOptionValue

	// Transaction with race condition prevention:
	// Duplicate check is INSIDE transaction with FOR UPDATE lock to prevent concurrent inserts
	err = db.Atomic(func(tx *gorm.DB) error {
		// Check for duplicate variant combination INSIDE transaction with lock
		// This prevents race condition where two concurrent requests create same variant
		existingVariant, _ := s.variantRepo.FindVariantByOptions(productID, optionsMap)
		if err := validator.ValidateVariantCombinationUnique(existingVariant); err != nil {
			return err
		}

		// Create variant
		if err := tx.Create(variant).Error; err != nil {
			return err
		}

		// Create variant option value associations
		variantOptionValues = factory.CreateVariantOptionValues(variant.ID, optionValueIDs)
		if err := tx.Create(&variantOptionValues).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build response directly from created data using factory builder (no additional query needed)
	selectedOptions := factory.BuildVariantOptionResponsesFromAvailableOptions(
		variantOptionValues,
		optionsResponse,
	)

	// Build and return response using factory
	return factory.BuildVariantDetailResponse(variant, product, selectedOptions), nil
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
	// Get product and validate seller access
	product, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Get existing variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Transaction with race condition prevention:
	// Wrap default variant logic and update in single transaction for atomicity
	err = db.Atomic(func(tx *gorm.DB) error {
		// Handle default variant logic INSIDE transaction
		// This prevents race condition where two concurrent updates both set isDefault=true
		if request.IsDefault != nil && *request.IsDefault {
			if err := s.variantRepo.UnsetAllDefaultVariantsForProduct(productID); err != nil {
				return err
			}
		}

		// Update variant using factory
		variant = factory.UpdateVariantEntity(variant, request)

		// Save updated variant
		return tx.Save(variant).Error
	})
	if err != nil {
		return nil, err
	}

	// Build and return response directly from updated data (no additional query needed)
	return s.buildVariantDetailResponse(variant, product, productID, sellerID)
}

/***********************************************
 *                DeleteVariant                *
 ***********************************************/
func (s *VariantServiceImpl) DeleteVariant(productID, variantID uint, sellerID uint) error {
	// Get product and validate seller access
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(productID, sellerID)
	if err != nil {
		return err
	}

	// Get variant to delete
	// Note: FindVariantByProductIDAndVariantID already validates variant belongs to product
	// No need for separate ValidateVariantBelongsToProduct call (redundant validation removed)
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return err
	}

	// Store if we're deleting the default variant (for reassignment logic)
	isDeletingDefault := variant.IsDefault

	// Transaction with race condition prevention:
	// Wrap count check, delete operations, and default reassignment in single transaction
	err = db.Atomic(func(tx *gorm.DB) error {
		// Check variant count INSIDE transaction to prevent race condition
		// This ensures count check and delete are atomic
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
		if err := tx.Delete(&entity.ProductVariant{}, variantID).Error; err != nil {
			return err
		}

		// If we deleted the default variant, automatically reassign default to another variant
		if isDeletingDefault && variantCount > 1 {
			if err := s.reassignDefaultVariant(tx, productID, variantID); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

/***********************************************
 *          Private Helper Methods             *
 ***********************************************/

// buildVariantDetailResponse builds the variant detail response from variant data
// This helper reduces code duplication between CreateVariant and UpdateVariant
func (s *VariantServiceImpl) buildVariantDetailResponse(
	variant *entity.ProductVariant,
	product *entity.Product,
	productID uint,
	sellerID uint,
) (*model.VariantDetailResponse, error) {
	// Get variant option values
	variantOptionValues, err := s.variantRepo.GetVariantOptionValues(variant.ID)
	if err != nil {
		return nil, err
	}

	// Get product options
	sellerIDPtr := &sellerID
	optionsResponse, err := s.optionService.GetAvailableOptions(productID, sellerIDPtr)
	if err != nil {
		return nil, err
	}

	// Build selected options using factory
	selectedOptions := factory.BuildVariantOptionResponsesFromAvailableOptions(
		variantOptionValues,
		optionsResponse,
	)

	// Build and return response
	return factory.BuildVariantDetailResponse(variant, product, selectedOptions), nil
}

// reassignDefaultVariant reassigns default status to another variant when default is deleted
// This maintains the business rule: "Every product must have a default variant"
func (s *VariantServiceImpl) reassignDefaultVariant(
	tx *gorm.DB,
	productID uint,
	deletedVariantID uint,
) error {
	// Get remaining variants for this product
	remainingVariants, err := s.variantRepo.FindVariantsByProductID(productID)
	if err != nil {
		return err
	}

	// Set the first remaining variant (that's not deleted) as default
	for i := range remainingVariants {
		if remainingVariants[i].ID != deletedVariantID {
			remainingVariants[i].IsDefault = true
			return tx.Save(&remainingVariants[i]).Error
		}
	}

	return nil
}

// validateAndMapVariantOptions validates variant options and returns mapped option value IDs and options map
// Returns:
// - optionValueIDs: map of optionID -> optionValueID for database insertion
// - optionsMap: map of optionName -> optionValue for duplicate checking
func (s *VariantServiceImpl) validateAndMapVariantOptions(
	optionInputs []model.VariantOptionInput,
	optionsResponse *model.GetAvailableOptionsResponse,
) (map[uint]uint, map[string]string, error) {
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

	for _, optionInput := range optionInputs {
		// Find option ID by name
		optionID, optExists := optionNameToID[optionInput.OptionName]
		if !optExists {
			return nil, nil, prodErrors.ErrProductOptionNotFound.WithMessagef(
				"Product option not found: %s",
				optionInput.OptionName,
			)
		}

		// Find option value ID
		valueID, valExists := optionValueMap[optionID][optionInput.Value]
		if !valExists {
			return nil, nil, prodErrors.ErrProductOptionValueNotFound.WithMessagef(
				"Product option value not found: %s for option: %s",
				optionInput.Value,
				optionInput.OptionName,
			)
		}

		optionValueIDs[optionID] = valueID
		optionsMap[optionInput.OptionName] = optionInput.Value
	}

	return optionValueIDs, optionsMap, nil
}
