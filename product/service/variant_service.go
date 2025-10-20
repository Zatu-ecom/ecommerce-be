package service

import (
	"errors"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"
)

// VariantService defines the interface for variant-related business logic
type VariantService interface {
	// GetVariantByID retrieves detailed information about a specific variant
	GetVariantByID(productID, variantID uint) (*model.VariantDetailResponse, error)

	// FindVariantByOptions finds a variant based on selected options
	FindVariantByOptions(
		productID uint,
		optionValues map[string]string,
	) (*model.VariantResponse, error)

	// CreateVariant creates a new variant for a product
	CreateVariant(productID uint, request *model.CreateVariantRequest) (*model.VariantDetailResponse, error)

	// UpdateVariant updates an existing variant
	UpdateVariant(productID, variantID uint, request *model.UpdateVariantRequest) (*model.VariantDetailResponse, error)

	// DeleteVariant deletes a variant
	DeleteVariant(productID, variantID uint) error

	// UpdateVariantStock updates the stock for a variant
	UpdateVariantStock(productID, variantID uint, request *model.UpdateVariantStockRequest) (*model.UpdateVariantStockResponse, error)

	// BulkUpdateVariants updates multiple variants at once
	BulkUpdateVariants(productID uint, request *model.BulkUpdateVariantsRequest) (*model.BulkUpdateVariantsResponse, error)
}

// VariantServiceImpl implements the VariantService interface
type VariantServiceImpl struct {
	variantRepo repositories.VariantRepository
	productRepo repositories.ProductRepository
}

// NewVariantService creates a new instance of VariantService
func NewVariantService(
	variantRepo repositories.VariantRepository,
	productRepo repositories.ProductRepository,
) VariantService {
	return &VariantServiceImpl{
		variantRepo: variantRepo,
		productRepo: productRepo,
	}
}

/***********************************************
 *                GetVariantByID               *
 ***********************************************/
func (s *VariantServiceImpl) GetVariantByID(
	productID, variantID uint,
) (*model.VariantDetailResponse, error) {
	// Validate that the product exists
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Find the variant by product ID and variant ID
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Get all option values for this variant
	variantOptionValues, err := s.variantRepo.GetVariantOptionValues(variantID)
	if err != nil {
		return nil, err
	}

	// Get product options and option values to build the response
	selectedOptions, err := s.buildVariantOptions(variantOptionValues)
	if err != nil {
		return nil, err
	}

	// Map to detailed response
	response := utils.ConvertVariantToDetailResponse(variant, product, selectedOptions)

	return response, nil
}

/***********************************************
 *            FindVariantByOptions             *
 ***********************************************/
func (s *VariantServiceImpl) FindVariantByOptions(
	productID uint,
	optionValues map[string]string,
) (*model.VariantResponse, error) {
	// Validate input
	if err := utils.ValidateVariantOptions(optionValues); err != nil {
		return nil, err
	}

	// Validate that the product exists
	_, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Find the variant by options
	variant, err := s.variantRepo.FindVariantByOptions(productID, optionValues)
	if err != nil {
		// If variant not found, return error
		// The handler can fetch available options if needed for the error response
		return nil, err
	}

	// Get all option values for this variant
	variantOptionValues, err := s.variantRepo.GetVariantOptionValues(variant.ID)
	if err != nil {
		return nil, err
	}

	// Get product options and option values to build the response
	selectedOptions, err := s.buildVariantOptions(variantOptionValues)
	if err != nil {
		return nil, err
	}

	// Map to response
	response := utils.ConvertVariantToResponse(variant, selectedOptions)

	return response, nil
}

/***********************************************
 *    Helper Methods                           *
 ***********************************************/

// buildVariantOptions builds the variant option response objects
func (s *VariantServiceImpl) buildVariantOptions(
	variantOptionValues []entity.VariantOptionValue,
) ([]model.VariantOptionResponse, error) {
	if len(variantOptionValues) == 0 {
		return []model.VariantOptionResponse{}, nil
	}

	// Collect all unique option IDs and option value IDs
	optionIDMap := make(map[uint]bool)
	optionValueIDMap := make(map[uint]bool)

	for _, vov := range variantOptionValues {
		optionIDMap[vov.OptionID] = true
		optionValueIDMap[vov.OptionValueID] = true
	}

	// Fetch all product options for this product
	productOptions := []entity.ProductOption{}
	for optionID := range optionIDMap {
		option, err := s.variantRepo.GetProductOptionByID(optionID)
		if err != nil {
			continue
		}
		productOptions = append(productOptions, *option)
	}

	// Fetch all option values
	optionValues := []entity.ProductOptionValue{}
	for _, vov := range variantOptionValues {
		optionValue, err := s.variantRepo.GetOptionValueByID(vov.OptionValueID)
		if err != nil {
			continue
		}
		optionValues = append(optionValues, *optionValue)
	}

	// Map to response
	return utils.ConvertVariantOptionValues(variantOptionValues, productOptions, optionValues), nil
}

/***********************************************
 *              CreateVariant                  *
 ***********************************************/
func (s *VariantServiceImpl) CreateVariant(
	productID uint,
	request *model.CreateVariantRequest,
) (*model.VariantDetailResponse, error) {
	// Validate that the product exists
	_, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Check if SKU already exists
	_ , _ = s.variantRepo.FindVariantByID(0) // Check by SKU needed
	// TODO: Add check for duplicate SKU in actual implementation

	// Validate options and get option value IDs
	optionValueIDs := make(map[uint]uint) // optionID -> optionValueID
	optionsMap := make(map[string]string) // For checking duplicate combination
	variantOptionValues := []entity.VariantOptionValue{}

	for _, optionInput := range request.Options {
		// Get option by name
		option, err := s.variantRepo.GetProductOptionByName(productID, optionInput.OptionName)
		if err != nil {
			return nil, err
		}

		// Get option value by value string
		optionValue, err := s.variantRepo.GetProductOptionValueByValue(option.ID, optionInput.Value)
		if err != nil {
			return nil, err
		}

		optionValueIDs[option.ID] = optionValue.ID
		optionsMap[optionInput.OptionName] = optionInput.Value
	}

	// Check if variant with these options already exists
	existingVariant, _ := s.variantRepo.FindVariantByOptions(productID, optionsMap)
	if existingVariant != nil {
		return nil, errors.New(utils.VARIANT_OPTION_COMBINATION_EXISTS_MSG)
	}

	// Set default values
	inStock := true
	if request.InStock != nil {
		inStock = *request.InStock
	}

	isPopular := false
	if request.IsPopular != nil {
		isPopular = *request.IsPopular
	}

	isDefault := false
	if request.IsDefault != nil {
		isDefault = *request.IsDefault
	}

	// Create variant entity
	variant := &entity.ProductVariant{
		ProductID: productID,
		SKU:       request.SKU,
		Price:     request.Price,
		Stock:     request.Stock,
		Images:    request.Images,
		InStock:   inStock,
		IsPopular: isPopular,
		IsDefault: isDefault,
	}

	// Save variant
	if err := s.variantRepo.CreateVariant(variant); err != nil {
		return nil, err
	}

	// Create variant option value associations
	for optionID, optionValueID := range optionValueIDs {
		variantOptionValues = append(variantOptionValues, entity.VariantOptionValue{
			VariantID:     variant.ID,
			OptionID:      optionID,
			OptionValueID: optionValueID,
		})
	}

	if err := s.variantRepo.CreateVariantOptionValues(variantOptionValues); err != nil {
		return nil, err
	}

	// Return the created variant details
	return s.GetVariantByID(productID, variant.ID)
}

/***********************************************
 *              UpdateVariant                  *
 ***********************************************/
func (s *VariantServiceImpl) UpdateVariant(
	productID, variantID uint,
	request *model.UpdateVariantRequest,
) (*model.VariantDetailResponse, error) {
	// Validate that the product exists
	_, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Get existing variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if request.SKU != nil {
		variant.SKU = *request.SKU
	}

	if request.Price != nil {
		variant.Price = *request.Price
	}

	if request.Stock != nil {
		variant.Stock = *request.Stock
	}

	if request.Images != nil {
		variant.Images = request.Images
	}

	if request.InStock != nil {
		variant.InStock = *request.InStock
	}

	if request.IsPopular != nil {
		variant.IsPopular = *request.IsPopular
	}

	if request.IsDefault != nil {
		variant.IsDefault = *request.IsDefault
	}

	// Save updated variant
	if err := s.variantRepo.UpdateVariant(variant); err != nil {
		return nil, err
	}

	// Return updated variant details
	return s.GetVariantByID(productID, variant.ID)
}

/***********************************************
 *                DeleteVariant                *
 ***********************************************/
func (s *VariantServiceImpl) DeleteVariant(productID, variantID uint) error {
	// Validate that the product exists
	if err := s.validateProductExists(productID); err != nil {
		return err
	}

	// Get existing variant to ensure it belongs to the product
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return err
	}

	// Check if this is the last variant - cannot delete
	count, err := s.variantRepo.CountVariantsByProductID(productID)
	if err != nil {
		return err
	}

	if count <= 1 {
		return errors.New(utils.LAST_VARIANT_DELETE_NOT_ALLOWED_MSG)
	}

	// Delete variant option values first (foreign key constraint)
	if err := s.variantRepo.DeleteVariantOptionValues(variant.ID); err != nil {
		return err
	}

	// Delete the variant
	return s.variantRepo.DeleteVariant(variant.ID)
}

/***********************************************
 *            UpdateVariantStock               *
 ***********************************************/
func (s *VariantServiceImpl) UpdateVariantStock(
	productID, variantID uint,
	request *model.UpdateVariantStockRequest,
) (*model.UpdateVariantStockResponse, error) {
	// Validate that the product exists
	if err := s.validateProductExists(productID); err != nil {
		return nil, err
	}

	// Get existing variant
	variant, err := s.variantRepo.FindVariantByProductIDAndVariantID(productID, variantID)
	if err != nil {
		return nil, err
	}

	// Apply stock operation
	switch request.Operation {
	case "set":
		variant.Stock = request.Stock
	case "add":
		variant.Stock += request.Stock
	case "subtract":
		if variant.Stock < request.Stock {
			return nil, errors.New(utils.INSUFFICIENT_STOCK_FOR_OPERATION_MSG)
		}
		variant.Stock -= request.Stock
	default:
		return nil, errors.New(utils.INVALID_STOCK_OPERATION_MSG)
	}

	// Update InStock status based on new stock value
	variant.InStock = variant.Stock > 0

	// Save updated variant
	if err := s.variantRepo.UpdateVariant(variant); err != nil {
		return nil, err
	}

	// Return stock update response
	return &model.UpdateVariantStockResponse{
		VariantID: variant.ID,
		SKU:       variant.SKU,
		Stock:     variant.Stock,
		InStock:   variant.InStock,
	}, nil
}

/***********************************************
 *           BulkUpdateVariants                *
 ***********************************************/
func (s *VariantServiceImpl) BulkUpdateVariants(
	productID uint,
	request *model.BulkUpdateVariantsRequest,
) (*model.BulkUpdateVariantsResponse, error) {
	// Validate that the product exists
	if err := s.validateProductExists(productID); err != nil {
		return nil, err
	}

	// Validate variants list is not empty
	if len(request.Variants) == 0 {
		return nil, errors.New(utils.BULK_UPDATE_EMPTY_LIST_MSG)
	}

	// Extract all variant IDs from the request
	variantIDs := make([]uint, 0, len(request.Variants))
	updateMap := make(map[uint]*model.BulkUpdateVariantItem)

	for i := range request.Variants {
		variantIDs = append(variantIDs, request.Variants[i].ID)
		updateMap[request.Variants[i].ID] = &request.Variants[i]
	}

	// Fetch all variants by IDs to validate they exist and belong to this product
	existingVariants, err := s.variantRepo.FindVariantsByIDs(variantIDs)
	if err != nil {
		return nil, err
	}

	// Validate that all variants belong to the specified product
	if len(existingVariants) != len(variantIDs) {
		return nil, errors.New(utils.BULK_UPDATE_VARIANT_NOT_FOUND_MSG)
	}

	// Create a map for quick lookup and validate product ownership
	variantsToUpdate := make([]*entity.ProductVariant, 0, len(existingVariants))
	for i := range existingVariants {
		variant := &existingVariants[i]

		// Validate that this variant belongs to the specified product
		if variant.ProductID != productID {
			return nil, errors.New(utils.BULK_UPDATE_VARIANT_NOT_FOUND_MSG)
		}

		// Get the update data for this variant
		updateData := updateMap[variant.ID]

		// Update only the fields that are provided (not nil)
		if updateData.SKU != nil {
			variant.SKU = *updateData.SKU
		}

		if updateData.Price != nil {
			variant.Price = *updateData.Price
		}

		if updateData.Stock != nil {
			variant.Stock = *updateData.Stock
		}

		if updateData.Images != nil {
			variant.Images = updateData.Images
		}

		if updateData.InStock != nil {
			variant.InStock = *updateData.InStock
		}

		if updateData.IsPopular != nil {
			variant.IsPopular = *updateData.IsPopular
		}

		if updateData.IsDefault != nil {
			variant.IsDefault = *updateData.IsDefault
		}

		variantsToUpdate = append(variantsToUpdate, variant)
	}

	// Perform bulk update in a transaction
	if err := s.variantRepo.BulkUpdateVariants(variantsToUpdate); err != nil {
		return nil, err
	}

	// Build response with summary of updated variants
	summaries := make([]model.BulkUpdateVariantSummary, 0, len(variantsToUpdate))
	for _, variant := range variantsToUpdate {
		summaries = append(summaries, model.BulkUpdateVariantSummary{
			ID:      variant.ID,
			SKU:     variant.SKU,
			Price:   variant.Price,
			Stock:   variant.Stock,
			InStock: variant.InStock,
		})
	}

	return &model.BulkUpdateVariantsResponse{
		UpdatedCount: len(variantsToUpdate),
		Variants:     summaries,
	}, nil
}

/***********************************************
 *              Helper Methods                 *
 ***********************************************/
// validateProductExists validates that a product exists
func (s *VariantServiceImpl) validateProductExists(productID uint) error {
	_, err := s.productRepo.FindByID(productID)
	return err
}
