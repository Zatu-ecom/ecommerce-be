package service

import (
	"math"
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// ProductService defines the interface for product-related business logic
type ProductService interface {
	CreateProduct(req model.ProductCreateRequest, sellerID uint) (*model.ProductResponse, error)
	UpdateProduct(id uint, req model.ProductUpdateRequest) (*model.ProductResponse, error)
	DeleteProduct(id uint) error
	GetAllProducts(page, limit int, filters map[string]interface{}) (*model.ProductsResponse, error)
	GetProductByID(id uint) (*model.ProductResponse, error)
	SearchProducts(
		query string,
		filters map[string]interface{},
		page, limit int,
	) (*model.SearchResponse, error)
	GetProductFilters() (*model.ProductFilters, error)
	GetRelatedProducts(productID uint, limit int) (*model.RelatedProductsResponse, error)
}

// ProductServiceImpl implements the ProductService interface
type ProductServiceImpl struct {
	productRepo   repositories.ProductRepository
	categoryRepo  repositories.CategoryRepository
	attributeRepo repositories.AttributeDefinitionRepository
	variantRepo   repositories.VariantRepository
	optionRepo    repositories.ProductOptionRepository
}

// NewProductService creates a new instance of ProductService
func NewProductService(
	productRepo repositories.ProductRepository,
	categoryRepo repositories.CategoryRepository,
	attributeRepo repositories.AttributeDefinitionRepository,
	variantRepo repositories.VariantRepository,
	optionRepo repositories.ProductOptionRepository,
) ProductService {
	return &ProductServiceImpl{
		productRepo:   productRepo,
		categoryRepo:  categoryRepo,
		attributeRepo: attributeRepo,
		variantRepo:   variantRepo,
		optionRepo:    optionRepo,
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
		// Validate request
		if err := s.validateProductCreateRequest(req); err != nil {
			return err
		}

		// Create product entity
		product = &entity.Product{
			Name:             req.Name,
			CategoryID:       req.CategoryID,
			Brand:            req.Brand,
			BaseSKU:          req.BaseSKU,
			ShortDescription: req.ShortDescription,
			LongDescription:  req.LongDescription,
			Tags:             req.Tags,
			SellerID:         sellerID,
		}

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
		if err := s.createProductVariants(product.ID, req); err != nil {
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
	return s.GetProductByID(product.ID)
}

// Helper to validate product creation request
func (s *ProductServiceImpl) validateProductCreateRequest(req model.ProductCreateRequest) error {
	// Check if base SKU already exists
	existingProduct, err := s.productRepo.FindBySKU(req.BaseSKU)
	if err != nil && err.Error() != utils.PRODUCT_NOT_FOUND_MSG {
		return err
	}
	if existingProduct != nil {
		return prodErrors.ErrProductExists
	}

	// Validate category exists
	category, err := s.categoryRepo.FindByID(req.CategoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return prodErrors.ErrInvalidCategory
	}

	// Validate variants are provided (PRD: at least one variant required)
	if !req.AutoGenerateVariants && len(req.Variants) == 0 {
		return prodErrors.ErrValidation.WithMessage("at least one variant is required or enable autoGenerateVariants")
	}

	// If auto-generating, validate default settings
	if req.AutoGenerateVariants {
		if len(req.Options) == 0 {
			return prodErrors.ErrValidation.WithMessage("options are required when autoGenerateVariants is true")
		}
		if req.DefaultVariantSettings == nil {
			return prodErrors.ErrValidation.WithMessage("defaultVariantSettings is required when autoGenerateVariants is true")
		}
	}

	// If variants provided, validate each variant SKU is unique
	if len(req.Variants) > 0 {
		skuMap := make(map[string]bool)
		for _, variant := range req.Variants {
			if skuMap[variant.SKU] {
				return prodErrors.ErrValidation.WithMessagef("duplicate variant SKU: %s", variant.SKU)
			}
			skuMap[variant.SKU] = true
		}
	}

	return nil
}

// Helper to create product options (PRD Section 3.1.3)
func (s *ProductServiceImpl) createProductOptions(
	productID uint,
	optionReqs []model.ProductOptionCreateRequest,
) error {
	for i, optionReq := range optionReqs {
		// Create product option
		option := &entity.ProductOption{
			ProductID:   productID,
			Name:        optionReq.Name,
			DisplayName: optionReq.DisplayName,
			Position:    optionReq.Position,
		}
		if option.Position == 0 {
			option.Position = i + 1
		}

		if err := s.optionRepo.CreateOption(option); err != nil {
			return err
		}

		// Create option values
		for j, valueReq := range optionReq.Values {
			optionValue := &entity.ProductOptionValue{
				OptionID:    option.ID,
				Value:       valueReq.Value,
				DisplayName: valueReq.DisplayName,
				ColorCode:   valueReq.ColorCode,
				Position:    valueReq.Position,
			}
			if optionValue.Position == 0 {
				optionValue.Position = j + 1
			}

			if err := s.optionRepo.CreateOptionValue(optionValue); err != nil {
				return err
			}
		}
	}
	return nil
}

// Helper to create product variants (PRD Section 3.1.3)
func (s *ProductServiceImpl) createProductVariants(
	productID uint,
	req model.ProductCreateRequest,
) error {
	if req.AutoGenerateVariants {
		return s.autoGenerateVariants(productID, req)
	}
	return s.createManualVariants(productID, req.Variants)
}

// Helper to auto-generate all variant combinations
func (s *ProductServiceImpl) autoGenerateVariants(
	productID uint,
	req model.ProductCreateRequest,
) error {
	// Get all product options with values
	productOptions, err := s.optionRepo.FindOptionsByProductID(productID)
	if err != nil {
		return err
	}

	// Generate all combinations
	combinations := s.generateOptionCombinations(productOptions)

	// Create a variant for each combination
	for i, combo := range combinations {
		// Generate SKU from combination
		sku := req.BaseSKU
		for _, opt := range combo {
			sku += "-" + opt.Value
		}

		// Create variant
		variant := &entity.ProductVariant{
			ProductID: productID,
			SKU:       sku,
			Price:     req.DefaultVariantSettings.Price,
			Stock:     req.DefaultVariantSettings.Stock,
			InStock:   req.DefaultVariantSettings.Stock > 0,
			IsPopular: req.DefaultVariantSettings.IsPopular,
			IsDefault: i == 0, // First variant is default
		}

		if err := s.variantRepo.CreateVariant(variant); err != nil {
			return err
		}

		// Link variant to option values
		var vovs []entity.VariantOptionValue
		for _, opt := range combo {
			vovs = append(vovs, entity.VariantOptionValue{
				VariantID:     variant.ID,
				OptionID:      opt.OptionID,
				OptionValueID: opt.ValueID,
			})
		}
		if err := s.variantRepo.CreateVariantOptionValues(vovs); err != nil {
			return err
		}
	}

	return nil
}

// Helper to generate all combinations of option values
func (s *ProductServiceImpl) generateOptionCombinations(options []entity.ProductOption) [][]struct {
	OptionID uint
	ValueID  uint
	Value    string
} {
	if len(options) == 0 {
		return nil
	}

	// Start with first option's values
	var result [][]struct {
		OptionID uint
		ValueID  uint
		Value    string
	}

	for _, value := range options[0].Values {
		result = append(result, []struct {
			OptionID uint
			ValueID  uint
			Value    string
		}{
			{
				OptionID: options[0].ID,
				ValueID:  value.ID,
				Value:    value.Value,
			},
		})
	}

	// Cartesian product with remaining options
	for i := 1; i < len(options); i++ {
		var newResult [][]struct {
			OptionID uint
			ValueID  uint
			Value    string
		}

		for _, combo := range result {
			for _, value := range options[i].Values {
				newCombo := make([]struct {
					OptionID uint
					ValueID  uint
					Value    string
				}, len(combo)+1)
				copy(newCombo, combo)
				newCombo[len(combo)] = struct {
					OptionID uint
					ValueID  uint
					Value    string
				}{
					OptionID: options[i].ID,
					ValueID:  value.ID,
					Value:    value.Value,
				}
				newResult = append(newResult, newCombo)
			}
		}

		result = newResult
	}

	return result
}

// Helper to create manual variants
func (s *ProductServiceImpl) createManualVariants(
	productID uint,
	variantReqs []model.CreateVariantRequest,
) error {
	// Get all product options to map option names to IDs
	productOptions, err := s.optionRepo.FindOptionsByProductID(productID)
	if err != nil && len(variantReqs) > 0 && len(variantReqs[0].Options) > 0 {
		return prodErrors.ErrValidation.WithMessage("product options not found, but variants require options")
	}

	optionMap := make(map[string]*entity.ProductOption)
	for i := range productOptions {
		optionMap[productOptions[i].Name] = &productOptions[i]
	}

	for i, variantReq := range variantReqs {
		// Determine default values
		inStock := true
		isPopular := false
		isDefault := false

		if variantReq.InStock != nil {
			inStock = *variantReq.InStock
		}
		if variantReq.IsPopular != nil {
			isPopular = *variantReq.IsPopular
		}
		if variantReq.IsDefault != nil {
			isDefault = *variantReq.IsDefault
		}

		// First variant is default if not specified
		if i == 0 && variantReq.IsDefault == nil {
			isDefault = true
		}

		// Create variant
		variant := &entity.ProductVariant{
			ProductID: productID,
			SKU:       variantReq.SKU,
			Price:     variantReq.Price,
			Stock:     variantReq.Stock,
			InStock:   inStock,
			IsPopular: isPopular,
			IsDefault: isDefault,
			Images:    variantReq.Images,
		}

		if err := s.variantRepo.CreateVariant(variant); err != nil {
			return err
		}

		// Link variant to option values
		var vovs []entity.VariantOptionValue
		for _, optInput := range variantReq.Options {
			option, exists := optionMap[optInput.OptionName]
			if !exists {
				return prodErrors.ErrValidation.WithMessagef("option not found: %s", optInput.OptionName)
			}

			// Find the option value ID
			var valueID uint
			for _, val := range option.Values {
				if val.Value == optInput.Value {
					valueID = val.ID
					break
				}
			}
			if valueID == 0 {
				return prodErrors.ErrValidation.WithMessagef("option value not found: %s for option: %s", 
					optInput.Value, optInput.OptionName)
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

// Helper to flatten []*entity.ProductAttribute to []entity.ProductAttribute
func flattenAttributes(attrs []*entity.ProductAttribute) []entity.ProductAttribute {
	var result []entity.ProductAttribute
	for _, attr := range attrs {
		result = append(result, *attr)
	}
	return result
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
	categoryInfo := utils.ConvertCategoryToHierarchyInfo(category, parentCategory)

	// Get product attributes
	productAttributes, err := s.attributeRepo.FindProductAttributeByProductID(product.ID)
	var attributes []model.ProductAttributeResponse
	if err == nil {
		attributes = utils.ConvertProductAttributesEntityToResponse(productAttributes)
	}

	// Get package options
	packageOptions, err := s.productRepo.FindPackageOptionByProductID(product.ID)
	var packageOptionResponses []model.PackageOptionResponse
	if err == nil {
		packageOptionResponses = utils.ConvertPackageOptionsEntityToResponse(packageOptions)
	}

	// Get variant aggregation for summary info
	variantAgg, err := s.variantRepo.GetProductVariantAggregation(product.ID)
	if err != nil {
		return nil, err
	}

	// Build base response (PRD Section 3.1.2)
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
		TotalStock:       variantAgg.TotalStock,
		InStock:          variantAgg.InStock,
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

	// Fetch full product options and variants for PRD Section 3.1.2
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
				ValueID:          value.ID,
				Value:            value.Value,
				ValueDisplayName: value.DisplayName,
				ColorCode:        value.ColorCode,
				VariantCount:     0, // Can be populated from variant counts map if needed
			})
		}

		result = append(result, optionResp)
	}

	return result
}

// buildVariantsDetailResponse converts variant entities with options to response format (PRD Section 3.1.2)
func (s *ProductServiceImpl) buildVariantsDetailResponse(
	variantsWithOptions []repositories.VariantWithOptions,
) []model.VariantDetailResponse {
	result := make([]model.VariantDetailResponse, 0, len(variantsWithOptions))

	for _, vwo := range variantsWithOptions {
		variantResp := model.VariantDetailResponse{
			ID:              vwo.Variant.ID,
			SKU:             vwo.Variant.SKU,
			Price:           vwo.Variant.Price,
			Stock:           vwo.Variant.Stock,
			InStock:         vwo.Variant.InStock,
			Images:          vwo.Variant.Images,
			IsDefault:       vwo.Variant.IsDefault,
			IsPopular:       vwo.Variant.IsPopular,
			SelectedOptions: make([]model.VariantOptionResponse, 0, len(vwo.SelectedOptions)),
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

// processAttributesForBulkOperations processes attributes and prepares bulk operations
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
			s.processExistingAttribute(attribute, attr.Value, operations)
		} else {
			attribute = s.createNewAttributeDefinition(attr, operations)
			attributeMap[attr.Key] = attribute
		}

		// Prepare product attribute for bulk creation
		productAttribute := &entity.ProductAttribute{
			ProductID:             productID,
			AttributeDefinitionID: attribute.ID,
			Value:                 attr.Value,
			SortOrder:             attr.SortOrder,
			AttributeDefinition:   attribute,
		}
		operations.productAttributesToCreate = append(
			operations.productAttributesToCreate,
			productAttribute,
		)
	}

	return operations
}

// processExistingAttribute processes an existing attribute definition
func (s *ProductServiceImpl) processExistingAttribute(
	attribute *entity.AttributeDefinition,
	value string,
	operations *BulkOperations,
) {
	// Check if value already exists using map for O(1) lookup
	valueMap := make(map[string]bool)
	for _, val := range attribute.AllowedValues {
		valueMap[val] = true
	}

	// Only add if value doesn't exist
	if !valueMap[value] {
		attribute.AllowedValues = append(attribute.AllowedValues, value)
		operations.attributesToUpdate = append(operations.attributesToUpdate, attribute)
	}
}

// createNewAttributeDefinition creates a new attribute definition
func (s *ProductServiceImpl) createNewAttributeDefinition(
	attr model.ProductAttributeRequest,
	operations *BulkOperations,
) *entity.AttributeDefinition {
	attribute := &entity.AttributeDefinition{
		Key:           attr.Key,
		Name:          attr.Name,
		Unit:          attr.Unit,
		AllowedValues: []string{attr.Value},
	}
	operations.attributesToCreate = append(operations.attributesToCreate, attribute)
	return attribute
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

func (s *ProductServiceImpl) createPackageOption(
	parentID uint,
	options []model.PackageOptionRequest,
) ([]entity.PackageOption, error) {
	var packageOptions []entity.PackageOption
	for _, option := range options {
		packageOption := entity.PackageOption{
			Name:        option.Name,
			Description: option.Description,
			Price:       option.Price,
			Quantity:    option.Quantity,
			ProductID:   parentID,
			BaseEntity: db.BaseEntity{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		packageOptions = append(packageOptions, packageOption)
	}

	return packageOptions, s.productRepo.CreatePackageOptions(packageOptions)
}

/********************************************************
 *		UpdateProduct updates an existing product 		*
 *		Note: Price, images, stock are managed at variant level
 ********************************************************/
func (s *ProductServiceImpl) UpdateProduct(
	id uint,
	req model.ProductUpdateRequest,
) (*model.ProductResponse, error) {
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Validate and update category
	if req.CategoryID != 0 {
		category, err := s.categoryRepo.FindByID(req.CategoryID)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, prodErrors.ErrInvalidCategory
		}
		product.CategoryID = req.CategoryID
	}

	// Update basic product fields only
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Brand != "" {
		product.Brand = req.Brand
	}
	if req.ShortDescription != "" {
		product.ShortDescription = req.ShortDescription
	}
	if req.LongDescription != "" {
		product.LongDescription = req.LongDescription
	}
	if len(req.Tags) > 0 {
		product.Tags = req.Tags
	}

	product.UpdatedAt = time.Now()

	// Save updated product
	if err := s.productRepo.Update(product); err != nil {
		return nil, err
	}

	// TODO: Update attributes and package options if provided in request

	// Build response with variant data
	return s.buildProductResponseFromEntity(product)
}

/**********************************************************
*      Deletes a product and all associated data        *
*      Implements PRD Section 3.1.5                     *
*      Cascading deletes:                               *
*      - Variants                                       *
*      - Variant option values                         *
*      - Product options                               *
*      - Product option values                         *
*      - Product attributes                            *
*      - Package options                               *
***********************************************************/
func (s *ProductServiceImpl) DeleteProduct(id uint) error {
	// Verify product exists
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return err
	}
	if product == nil {
		return prodErrors.ErrProductNotFound
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

/*************************************************************************
*       GetAllProducts gets all products with pagination and filters     *
*       Now includes variant data for each product                      *
**************************************************************************/
func (s *ProductServiceImpl) GetAllProducts(
	page,
	limit int,
	filters map[string]interface{},
) (*model.ProductsResponse, error) {
	// Set default values
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	products, total, err := s.productRepo.FindAll(filters, page, limit)
	if err != nil {
		return nil, err
	}

	// Extract product IDs for batch variant aggregation
	productIDs := make([]uint, len(products))
	for i, product := range products {
		productIDs[i] = product.ID
	}

	// Get variant aggregations for all products
	variantAggs, err := s.variantRepo.GetProductsVariantAggregations(productIDs)
	if err != nil {
		return nil, err
	}

	// Convert to response models with variant data
	productsResponse := make([]model.ProductResponse, 0, len(products))
	for _, product := range products {
		// Build category hierarchy
		categoryInfo := model.CategoryHierarchyInfo{
			ID:   product.Category.ID,
			Name: product.Category.Name,
		}
		if product.Category.Parent != nil {
			categoryInfo.Parent = &model.CategoryInfo{
				ID:   product.Category.Parent.ID,
				Name: product.Category.Parent.Name,
			}
		}

		// Get variant aggregation for this product
		variantAgg := variantAggs[product.ID]
		if variantAgg == nil {
			// Skip products without variants (shouldn't happen)
			continue
		}

		// Build response according to PRD Section 3.1.1
		productResp := model.ProductResponse{
			ID:               product.ID,
			Name:             product.Name,
			CategoryID:       product.CategoryID,
			Category:         categoryInfo,
			Brand:            product.Brand,
			SKU:              product.BaseSKU,
			ShortDescription: product.ShortDescription,
			LongDescription:  product.LongDescription,
			Tags:             product.Tags,
			SellerID:         product.SellerID,
			HasVariants:      variantAgg.HasVariants,
			TotalStock:       variantAgg.TotalStock,
			InStock:          variantAgg.InStock,
			CreatedAt:        product.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
		}

		// Set price range (PRD Section 3.1.1)
		if variantAgg.HasVariants {
			productResp.PriceRange = &model.PriceRange{
				Min: variantAgg.MinPrice,
				Max: variantAgg.MaxPrice,
			}
		}

		// Add images if available
		if variantAgg.MainImage != "" {
			productResp.Images = []string{variantAgg.MainImage}
		}

		// Add variant preview (PRD Section 3.1.1)
		if variantAgg.TotalVariants > 0 {
			variantPreview := &model.VariantPreview{
				TotalVariants: variantAgg.TotalVariants,
				Options:       []model.OptionPreview{},
			}

			for _, optionName := range variantAgg.OptionNames {
				optionValues := variantAgg.OptionValues[optionName]
				variantPreview.Options = append(variantPreview.Options, model.OptionPreview{
					Name:            optionName,
					DisplayName:     optionName,
					AvailableValues: optionValues,
				})
			}

			productResp.VariantPreview = variantPreview
		}

		productsResponse = append(productsResponse, productResp)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	hasNext := page < totalPages
	hasPrev := page > 1

	pagination := model.PaginationResponse{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   int(total),
		ItemsPerPage: limit,
		HasNext:      hasNext,
		HasPrev:      hasPrev,
	}

	return &model.ProductsResponse{
		Products:   productsResponse,
		Pagination: pagination,
	}, nil
}

/*****************************************************************************
*        GetProductByID gets a product by ID with detailed information       *
*        Now includes complete variant data                                  *
******************************************************************************/
func (s *ProductServiceImpl) GetProductByID(id uint) (*model.ProductResponse, error) {
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Use the helper method that fetches variant data
	return s.buildProductResponseFromEntity(product)
}

/**********************************************************************************
*     SearchProducts searches for products with the given query and filters       *
***********************************************************************************/
func (s *ProductServiceImpl) SearchProducts(
	query string,
	filters map[string]interface{},
	page, limit int,
) (*model.SearchResponse, error) {
	// Set default values
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	products, total, err := s.productRepo.Search(query, filters, page, limit)
	if err != nil {
		return nil, err
	}

	// Convert to search results
	var searchResults []model.SearchResult
	for _, product := range products {
		result := utils.ConvertProductToSearchResult(&product)
		searchResults = append(searchResults, *result)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	pagination := model.PaginationResponse{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   int(total),
		ItemsPerPage: limit,
		HasNext:      page < totalPages,
		HasPrev:      page > 1,
	}

	return &model.SearchResponse{
		Query:      query,
		Results:    searchResults,
		Pagination: pagination,
		SearchTime: "0.05s", // Placeholder
	}, nil
}

/****************************************************************************
*        GetProductFilters gets available filters for product search		*
*****************************************************************************/
func (s *ProductServiceImpl) GetProductFilters() (*model.ProductFilters, error) {
	brands, categories, attributes, err := s.productRepo.GetProductFilters()
	if err != nil {
		return nil, err
	}
	filters := &model.ProductFilters{
		Brands:     utils.ConvertBrandsToFilters(brands),
		Categories: s.convertCategoriesToFilters(categories),
		Attributes: utils.ConvertAttributesToFilters(attributes),
	}

	return filters, nil
}

func (s *ProductServiceImpl) convertCategoriesToFilters(
	categories []mapper.CategoryWithProductCount,
) []model.CategoryFilter {
	mp := make(map[uint]model.CategoryFilter)
	var categoryFilter []model.CategoryFilter
	for _, category := range categories {
		mp[category.CategoryID] = utils.ConvertCategoriesToFilters(category)

		if category.ParentID == nil || *category.ParentID == 0 {
			categoryFilter = append(
				categoryFilter,
				utils.ConvertCategoriesToFilters(category),
			)
		}
	}

	for _, category := range categories {
		if category.ParentID != nil && *category.ParentID != 0 {
			parentFilter, exist := mp[*category.ParentID]
			if exist {
				parentFilter.Children = append(
					parentFilter.Children,
					utils.ConvertCategoriesToFilters(category),
				)
			} else {
				categoryFilter = append(
					categoryFilter,
					utils.ConvertCategoriesToFilters(category),
				)
			}
		}
	}

	return categoryFilter
}

/*****************************************************************************
*      GetRelatedProducts gets products related to a specific product        *
******************************************************************************/
func (s *ProductServiceImpl) GetRelatedProducts(
	productID uint,
	limit int,
) (*model.RelatedProductsResponse, error) {
	// Get the product to find its category
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Set default limit
	if limit < 1 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	// Find related products in the same category
	relatedProducts, err := s.productRepo.FindRelated(product.CategoryID, productID, limit)
	if err != nil {
		return nil, err
	}

	// Convert to response models
	var relatedProductsResponse []model.RelatedProductResponse
	for _, relatedProduct := range relatedProducts {
		r := utils.ConvertProductToRelatedProduct(&relatedProduct)
		relatedProductsResponse = append(relatedProductsResponse, *r)
	}

	return &model.RelatedProductsResponse{
		RelatedProducts: relatedProductsResponse,
	}, nil
}
