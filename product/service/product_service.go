package service

import (
	"time"

	"ecommerce-be/common/db"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils/helper"
	"ecommerce-be/product/validator"

	"gorm.io/gorm"
)

// ProductService defines the interface for product-related business logic
type ProductService interface {
	CreateProduct(
		req model.ProductCreateRequest,
		sellerID uint,
	) (*model.ProductResponse, error)
	UpdateProduct(
		id uint,
		sellerId *uint,
		req model.ProductUpdateRequest,
	) (*model.ProductResponse, error)
	DeleteProduct(
		id uint,
		sellerId *uint,
	) error
}

// ProductServiceImpl implements the ProductService interface
type ProductServiceImpl struct {
	productRepo         repositories.ProductRepository
	categoryRepo        repositories.CategoryRepository
	attributeRepo       repositories.AttributeDefinitionRepository
	variantRepo         repositories.VariantRepository
	productQueryService ProductQueryService
	optionRepo          repositories.ProductOptionRepository
}

// NewProductService creates a new instance of ProductService
func NewProductService(
	productRepo repositories.ProductRepository,
	categoryRepo repositories.CategoryRepository,
	attributeRepo repositories.AttributeDefinitionRepository,
	variantRepo repositories.VariantRepository,
	optionRepo repositories.ProductOptionRepository,
	productQueryService ProductQueryService,
) ProductService {
	return &ProductServiceImpl{
		productRepo:         productRepo,
		categoryRepo:        categoryRepo,
		attributeRepo:       attributeRepo,
		variantRepo:         variantRepo,
		optionRepo:          optionRepo,
		productQueryService: productQueryService,
	}
}

/***********************************************
 *    CreateProduct creates a new product      *
 *    Implements PRD Section 3.1.3             *
 ***********************************************/
func (s *ProductServiceImpl) CreateProduct(
	req model.ProductCreateRequest,
	sellerID uint,
) (*model.ProductResponse, error) {
	var product *entity.Product

	err := db.Atomic(func(tx *gorm.DB) error {
		// Fetch category for validation
		category, err := s.categoryRepo.FindByID(req.CategoryID)
		if err != nil {
			return err
		}

		// Validate request using validator
		if err := validator.ValidateProductCreateRequest(req, category); err != nil {
			return err
		}

		// Create product entity using factory
		product = factory.CreateProductFromRequest(req, sellerID)

		if err := s.productRepo.Create(product); err != nil {
			return err
		}

		// Create product options if provided (PRD Section 3.1.3)
		if len(req.Options) > 0 {
			if err := s.createProductOptions(product.ID, req.Options); err != nil {
				return err
			}
		}

		// Create variants (PRD requirement: at least one variant)
		if err := s.createProductVariants(product.ID, req.Variants); err != nil {
			return err
		}

		// Create product attributes
		if len(req.Attributes) > 0 {
			_, err := s.createProductAttributes(product.ID, req.Attributes)
			if err != nil {
				return err
			}
		}

		// Create package options
		if len(req.PackageOptions) > 0 {
			_, err := s.createPackageOption(product.ID, req.PackageOptions)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Return created product with full details
	// No seller check needed here since we just created it with this sellerID
	return s.productQueryService.GetProductByID(product.ID, nil)
}

// Helper to create product options (uses factory for entity creation)
func (s *ProductServiceImpl) createProductOptions(
	productID uint,
	optionReqs []model.ProductOptionCreateRequest,
) error {
	for _, optionReq := range optionReqs {
		// Create product option using factory
		options := factory.CreateProductOptionsFromRequests(
			productID,
			[]model.ProductOptionCreateRequest{optionReq},
		)
		option := options[0]

		if err := s.optionRepo.CreateOption(option); err != nil {
			return err
		}

		// Create option values using factory
		if len(optionReq.Values) > 0 {
			values := factory.CreateOptionValuesFromRequests(option.ID, optionReq.Values)
			for _, value := range values {
				if err := s.optionRepo.CreateOptionValue(&value); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Helper to create manual variants
func (s *ProductServiceImpl) createProductVariants(
	productID uint,
	variantReqs []model.CreateVariantRequest,
) error {
	// Get all product options to map option names to IDs
	productOptions, err := s.optionRepo.FindOptionsByProductID(productID)
	if err != nil && len(variantReqs) > 0 && len(variantReqs[0].Options) > 0 {
		return commonError.ErrValidation.WithMessage(
			"product options not found, but variants require options",
		)
	}

	optionMap := make(map[string]*entity.ProductOption)
	for i := range productOptions {
		optionMap[productOptions[i].Name] = &productOptions[i]
	}

	// Apply "last one wins" rule for multiple default variants
	// Find the last variant that has isDefault=true
	lastDefaultIndex := -1
	for i, variantReq := range variantReqs {
		if variantReq.IsDefault != nil && *variantReq.IsDefault {
			lastDefaultIndex = i
		}
	}

	for i, variantReq := range variantReqs {
		// Determine default values
		allowPurchase := true
		isPopular := false
		isDefault := false

		if variantReq.AllowPurchase != nil {
			allowPurchase = *variantReq.AllowPurchase
		}
		if variantReq.IsPopular != nil {
			isPopular = *variantReq.IsPopular
		}

		// Apply multiple default validation: "last one wins" rule
		// If multiple variants are marked as default, only the last one remains default
		if lastDefaultIndex != -1 {
			// There are explicitly marked defaults, only set default for the last one
			isDefault = (i == lastDefaultIndex)
		} else {
			// No explicit defaults, first variant is default by convention
			if i == 0 {
				isDefault = true
			}
		}

		// Create variant
		variant := &entity.ProductVariant{
			ProductID:     productID,
			SKU:           variantReq.SKU,
			Price:         variantReq.Price,
			AllowPurchase: allowPurchase,
			IsPopular:     isPopular,
			IsDefault:     isDefault,
			Images:        variantReq.Images,
		}

		if err := s.variantRepo.CreateVariant(variant); err != nil {
			return err
		}

		// Validate that variant provides all required options
		// If product has options defined, variants with options must specify all of them
		if len(productOptions) > 0 && len(variantReq.Options) > 0 {
			if len(variantReq.Options) != len(productOptions) {
				return commonError.ErrValidation.WithMessagef(
					"variant must specify all product options (%d required, %d provided)",
					len(productOptions),
					len(variantReq.Options),
				)
			}
		}

		// Link variant to option values
		var vovs []entity.VariantOptionValue
		for _, optInput := range variantReq.Options {
			option, exists := optionMap[optInput.OptionName]
			if !exists {
				return commonError.ErrValidation.WithMessagef(
					"option not found: %s",
					optInput.OptionName,
				)
			}

			// Find the option value ID
			// Normalize input value to lowercase for comparison (values are stored in lowercase)
			normalizedValue := helper.ToLowerTrimmed(optInput.Value)
			var valueID uint
			for _, val := range option.Values {
				if val.Value == normalizedValue {
					valueID = val.ID
					break
				}
			}
			if valueID == 0 {
				return commonError.ErrValidation.WithMessagef(
					"option value not found: %s for option: %s",
					optInput.Value,
					optInput.OptionName,
				)
			}

			vovs = append(vovs, entity.VariantOptionValue{
				VariantID:     variant.ID,
				OptionID:      option.ID,
				OptionValueID: valueID,
			})
		}

		if len(vovs) > 0 {
			if err := s.variantRepo.CreateVariantOptionValues(vovs); err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper to build product response from entity (fetches variant data)
// Implements PRD Section 3.1.2 - Get Product by ID with full details
func (s *ProductServiceImpl) buildProductResponseFromEntity(
	product *entity.Product,
) (*model.ProductResponse, error) {
	// Get category with parent
	category, err := s.categoryRepo.FindByID(product.CategoryID)
	if err != nil {
		return nil, err
	}

	var parentCategory *entity.Category
	if category.ParentID != nil && *category.ParentID != 0 {
		if pc, err := s.categoryRepo.FindByID(*category.ParentID); err == nil {
			parentCategory = pc
		}
	}
	categoryInfo := factory.BuildCategoryHierarchyInfo(category, parentCategory)

	// Get product attributes
	productAttributes, err := s.attributeRepo.FindProductAttributeByProductID(product.ID)
	var attributes []model.ProductAttributeResponse
	if err == nil {
		attributes = factory.BuildProductAttributesResponse(productAttributes)
	}

	// Get package options
	packageOptions, err := s.productRepo.FindPackageOptionByProductID(product.ID)
	var packageOptionResponses []model.PackageOptionResponse
	if err == nil {
		packageOptionResponses = factory.BuildPackageOptionResponses(packageOptions)
	}

	// Get variant aggregation for summary info
	variantAgg, err := s.variantRepo.GetProductVariantAggregation(product.ID)
	if err != nil {
		return nil, err
	}

	response := &model.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         *categoryInfo,
		Brand:            product.Brand,
		SKU:              product.BaseSKU,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Tags:             product.Tags,
		SellerID:         product.SellerID,
		HasVariants:      variantAgg.HasVariants,
		AllowPurchase:    variantAgg.AllowPurchase,
		Attributes:       attributes,
		PackageOptions:   packageOptionResponses,
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}

	// Set price range
	if variantAgg.HasVariants {
		response.PriceRange = &model.PriceRange{
			Min: variantAgg.MinPrice,
			Max: variantAgg.MaxPrice,
		}
	}

	// Fetch full product options and variants
	// Using single optimized query for performance (read-heavy operation)

	// Get all product options with their values
	productOptions, _, err := s.variantRepo.GetProductOptionsWithVariantCounts(product.ID)
	if err == nil && len(productOptions) > 0 {
		response.Options = s.buildProductOptionsResponse(productOptions)
	}

	// Get all variants with their selected option values in a single query
	variantsWithOptions, err := s.variantRepo.GetProductVariantsWithOptions(product.ID)
	if err == nil && len(variantsWithOptions) > 0 {
		response.Variants = s.buildVariantsDetailResponse(variantsWithOptions)
	}

	// Set main product images from default variant (already fetched in aggregation)
	if variantAgg.MainImage != "" {
		response.Images = []string{variantAgg.MainImage}
	}

	return response, nil
}

// buildProductOptionsResponse converts entity options to response format (PRD Section 3.1.2)
func (s *ProductServiceImpl) buildProductOptionsResponse(
	options []entity.ProductOption,
) []model.ProductOptionDetailResponse {
	result := make([]model.ProductOptionDetailResponse, 0, len(options))

	for _, option := range options {
		optionResp := model.ProductOptionDetailResponse{
			OptionID:          option.ID,
			OptionName:        option.Name,
			OptionDisplayName: option.DisplayName,
			Position:          option.Position,
			Values:            make([]model.OptionValueResponse, 0, len(option.Values)),
		}

		// Convert option values
		for _, value := range option.Values {
			optionResp.Values = append(optionResp.Values, model.OptionValueResponse{
				ValueID:     value.ID,
				Value:       value.Value,
				DisplayName: value.DisplayName,
				ColorCode:   value.ColorCode,
				Position:    value.Position,
				// VariantCount:     0, // Can be populated from variant counts map if needed
			})
		}

		result = append(result, optionResp)
	}

	return result
}

// buildVariantsDetailResponse converts variant entities with options to response format (PRD Section 3.1.2)
func (s *ProductServiceImpl) buildVariantsDetailResponse(
	variantsWithOptions []mapper.VariantWithOptions,
) []model.VariantDetailResponse {
	result := make([]model.VariantDetailResponse, 0, len(variantsWithOptions))

	for _, vwo := range variantsWithOptions {
		variantResp := model.VariantDetailResponse{
			ID:              vwo.Variant.ID,
			SKU:             vwo.Variant.SKU,
			Price:           vwo.Variant.Price,
			AllowPurchase:   vwo.Variant.AllowPurchase,
			Images:          vwo.Variant.Images,
			IsDefault:       vwo.Variant.IsDefault,
			IsPopular:       vwo.Variant.IsPopular,
			SelectedOptions: make([]model.VariantOptionResponse, 0, len(vwo.SelectedOptions)),
			CreatedAt:       vwo.Variant.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       vwo.Variant.UpdatedAt.Format(time.RFC3339),
		}

		// Convert selected options
		for _, selOpt := range vwo.SelectedOptions {
			variantResp.SelectedOptions = append(
				variantResp.SelectedOptions,
				model.VariantOptionResponse{
					OptionID:          selOpt.OptionID,
					OptionName:        selOpt.OptionName,
					OptionDisplayName: selOpt.OptionDisplayName,
					ValueID:           selOpt.ValueID,
					Value:             selOpt.Value,
					ValueDisplayName:  selOpt.ValueDisplayName,
					ColorCode:         selOpt.ColorCode,
				},
			)
		}

		result = append(result, variantResp)
	}

	return result
}

// createProductAttributes creates product attributes for a given product
func (s *ProductServiceImpl) createProductAttributes(
	productID uint,
	attributes []model.ProductAttributeRequest,
) ([]*entity.ProductAttribute, error) {
	// Extract unique keys and fetch existing attributes
	keys := s.extractUniqueKeys(attributes)
	attributeMap, err := s.attributeRepo.FindByKeys(keys)
	if err != nil {
		return nil, err
	}

	// Process attributes and prepare bulk operations
	operations := s.processAttributesForBulkOperations(productID, attributes, attributeMap)

	// Execute all bulk operations
	if err = s.executeBulkOperations(operations); err != nil {
		return nil, err
	}

	return operations.productAttributesToCreate, nil
}

// extractUniqueKeys extracts unique keys from attribute requests
func (s *ProductServiceImpl) extractUniqueKeys(
	attributes []model.ProductAttributeRequest,
) []string {
	keys := make([]string, 0, len(attributes))
	keySet := make(map[string]bool)
	for _, attr := range attributes {
		if !keySet[attr.Key] {
			keys = append(keys, attr.Key)
			keySet[attr.Key] = true
		}
	}
	return keys
}

// BulkOperations holds all bulk operations to be executed
type BulkOperations struct {
	attributesToUpdate        []*entity.AttributeDefinition
	attributesToCreate        []*entity.AttributeDefinition
	productAttributesToCreate []*entity.ProductAttribute
}

// processAttributesForBulkOperations processes attributes and prepares bulk operations using factory
func (s *ProductServiceImpl) processAttributesForBulkOperations(
	productID uint,
	attributes []model.ProductAttributeRequest,
	attributeMap map[string]*entity.AttributeDefinition,
) *BulkOperations {
	operations := &BulkOperations{
		attributesToUpdate:        make([]*entity.AttributeDefinition, 0),
		attributesToCreate:        make([]*entity.AttributeDefinition, 0),
		productAttributesToCreate: make([]*entity.ProductAttribute, 0),
	}

	for _, attr := range attributes {
		attribute, exists := attributeMap[attr.Key]

		if exists {
			// Update existing attribute using factory
			if factory.UpdateAttributeDefinitionValues(attribute, attr.Value) {
				operations.attributesToUpdate = append(operations.attributesToUpdate, attribute)
			}
		} else {
			// Create new attribute definition using factory
			attribute = factory.CreateNewAttributeDefinition(attr)
			operations.attributesToCreate = append(operations.attributesToCreate, attribute)
			attributeMap[attr.Key] = attribute
		}
	}

	// Create product attributes using factory
	productAttributes := factory.CreateProductAttributesFromRequests(
		productID,
		attributes,
		attributeMap,
	)
	operations.productAttributesToCreate = productAttributes

	return operations
}

// executeBulkOperations executes all bulk database operations
func (s *ProductServiceImpl) executeBulkOperations(operations *BulkOperations) error {
	// Bulk create new attribute definitions
	if len(operations.attributesToCreate) > 0 {
		if err := s.attributeRepo.CreateBulk(operations.attributesToCreate); err != nil {
			return err
		}
	}

	// Bulk update modified attributes
	if len(operations.attributesToUpdate) > 0 {
		if err := s.attributeRepo.UpdateBulk(operations.attributesToUpdate); err != nil {
			return err
		}
	}

	// Bulk create product attributes
	if len(operations.productAttributesToCreate) > 0 {
		if err := s.attributeRepo.CreateProductAttributesBulk(operations.productAttributesToCreate); err != nil {
			return err
		}
	}

	return nil
}

// createPackageOption creates package options using factory
func (s *ProductServiceImpl) createPackageOption(
	parentID uint,
	options []model.PackageOptionRequest,
) ([]entity.PackageOption, error) {
	// Create package options using factory
	packageOptions := factory.CreatePackageOptionsFromRequests(parentID, options)
	return packageOptions, s.productRepo.CreatePackageOptions(packageOptions)
}

/************************************************************
 *	UpdateProduct updates an existing product 		        *
 *	Note: Price, images, stock are managed at variant level *
 ************************************************************/
func (s *ProductServiceImpl) UpdateProduct(
	id uint,
	sellerId *uint,
	req model.ProductUpdateRequest,
) (*model.ProductResponse, error) {
	// Fetch product to validate
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Fetch category if being updated
	var category *entity.Category
	if req.CategoryID != nil && *req.CategoryID != 0 {
		category, err = s.categoryRepo.FindByID(*req.CategoryID)
		if err != nil {
			return nil, err
		}
	}

	// Validate product exists and category if provided
	if err := validator.ValidateProductUpdateRequest(product, sellerId, req, category); err != nil {
		return nil, err
	}

	// Update product entity using factory
	product = factory.CreateProductEntityFromUpdateRequest(product, req)

	// Clear preloaded associations to avoid GORM sync issues
	// When CategoryID is updated but Category is preloaded, GORM may not update correctly
	product.Category = nil

	// Save updated product
	if err := s.productRepo.Update(product); err != nil {
		return nil, err
	}

	// TODO: Update attributes and package options if provided in request

	// Build response with variant data
	return s.buildProductResponseFromEntity(product)
}

/***************************************************
* Deletes a product and all associated data        *
* Implements PRD Section 3.1.5                     *
* Cascading deletes:                               *
* - Variants                                       *
* - Variant option values                          *
* - Product options                                *
* - Product option values                          *
* - Product attributes                             *
* - Package options                                *
****************************************************/
func (s *ProductServiceImpl) DeleteProduct(
	id uint,
	sellerId *uint,
) error {
	// Fetch product to validate
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return err
	}

	// Verify product exists and ownership
	if err := validator.ValidateProductExistsAndOwnership(product, sellerId); err != nil {
		return err
	}

	// Use atomic transaction to delete everything
	return db.Atomic(func(tx *gorm.DB) error {
		// Step 1: Get all variants for this product
		variants, err := s.variantRepo.FindVariantsByProductID(id)
		if err != nil {
			return err
		}

		// Step 2: Delete variant option values for all variants
		if len(variants) > 0 {
			variantIDs := make([]uint, len(variants))
			for i, v := range variants {
				variantIDs[i] = v.ID
			}

			// Delete all variant option values
			if err := s.variantRepo.DeleteVariantOptionValuesByVariantIDs(variantIDs); err != nil {
				return err
			}

			// Delete all variants
			if err := s.variantRepo.DeleteVariantsByProductID(id); err != nil {
				return err
			}
		}

		// Step 3: Get all product options
		productOptions, err := s.optionRepo.FindOptionsByProductID(id)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		// Step 4: Delete product option values and options
		if len(productOptions) > 0 {
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
		}

		// Step 5: Delete product attributes
		if err := s.attributeRepo.DeleteProductAttributesByProductID(id); err != nil {
			return err
		}

		// Step 6: Delete package options
		if err := s.productRepo.DeletePackageOptionsByProductID(id); err != nil {
			return err
		}

		// Step 7: Finally, delete the product itself
		if err := s.productRepo.Delete(id); err != nil {
			return err
		}

		return nil
	})
}
