package service

import (
	"math"
	"time"

	"ecommerce-be/common/db"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
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
	GetAllProducts(
		page, limit int,
		filters map[string]interface{},
	) (*model.ProductsResponse, error)
	GetProductByID(
		id uint,
		sellerID *uint,
	) (*model.ProductResponse, error)
	SearchProducts(
		query string,
		filters map[string]interface{},
		page, limit int,
	) (*model.SearchResponse, error)
	GetProductFilters(
		sellerID *uint,
	) (*model.ProductFilters, error)
	GetRelatedProductsScored(
		productID uint,
		limit int,
		page int,
		strategies string,
		sellerID *uint,
	) (*model.RelatedProductsScoredResponse, error)
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
	return s.GetProductByID(product.ID, nil)
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

/*****************************************************
* HELPER FUNCTIONS                                   *
******************************************************/

// buildProductResponse builds a ProductResponse from product entity and variant aggregation
// This is a shared helper to ensure consistency across GetAllProducts, GetRelatedProducts, etc.
func (s *ProductServiceImpl) buildProductResponse(
	product *entity.Product,
	variantAgg *mapper.VariantAggregation,
) model.ProductResponse {
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

	// Build base product response
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
		AllowPurchase:    variantAgg.AllowPurchase,
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}

	// Set price range
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

	// Add variant preview
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

	return productResp
}

// buildProductResponsesWithVariants builds ProductResponse list from products with variant data
// Performs batch variant aggregation for optimal performance
func (s *ProductServiceImpl) buildProductResponsesWithVariants(
	products []entity.Product,
) ([]model.ProductResponse, error) {
	if len(products) == 0 {
		return []model.ProductResponse{}, nil
	}

	// Extract product IDs for batch variant aggregation
	productIDs := make([]uint, len(products))
	for i, product := range products {
		productIDs[i] = product.ID
	}

	// Get variant aggregations for all products in one query (performance optimization)
	variantAggs, err := s.variantRepo.GetProductsVariantAggregations(productIDs)
	if err != nil {
		return nil, err
	}

	// Convert to response models with variant data
	productsResponse := make([]model.ProductResponse, 0, len(products))
	for _, product := range products {
		// Get variant aggregation for this product
		variantAgg := variantAggs[product.ID]
		if variantAgg == nil {
			// Skip products without variants (shouldn't happen)
			continue
		}

		productResp := s.buildProductResponse(&product, variantAgg)
		productsResponse = append(productsResponse, productResp)
	}

	return productsResponse, nil
}

/*******************************************************************
* GetAllProducts gets all products with pagination and filters     *
* Now includes variant data for each product                       *
********************************************************************/
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

	// Build product responses with variant data using shared helper
	productsResponse, err := s.buildProductResponsesWithVariants(products)
	if err != nil {
		return nil, err
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

/**********************************************************************
* GetProductByID gets a product by ID with detailed information       *
* Now includes complete variant data                                  *
* Multi-tenant: If sellerID is provided, verify product belongs to    *
* that seller. If nil (admin), allow access to any product.           *
***********************************************************************/
func (s *ProductServiceImpl) GetProductByID(
	id uint,
	sellerID *uint,
) (*model.ProductResponse, error) {
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Multi-tenant check: If seller ID is provided, verify product belongs to that seller
	// If sellerID is nil (admin role or no seller context), skip this check
	if sellerID != nil && product.SellerID != *sellerID {
		return nil, prodErrors.ErrProductNotFound
	}

	// Use the helper method that fetches variant data
	return s.buildProductResponseFromEntity(product)
}

/******************************************************************************
* SearchProducts searches for products with the given query and filters       *
* Now includes variant data in search results                                 *
*******************************************************************************/
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

	// Build product responses with variant data using shared helper
	productsResponse, err := s.buildProductResponsesWithVariants(products)
	if err != nil {
		return nil, err
	}

	// Convert to search results with additional search metadata
	searchResults := make([]model.SearchResult, 0, len(productsResponse))
	for _, productResp := range productsResponse {
		searchResult := model.SearchResult{
			ProductResponse: productResp,
			RelevanceScore:  0.8,              // TODO: Implement actual relevance scoring
			MatchedFields:   []string{"name"}, // TODO: Track which fields matched
		}
		searchResults = append(searchResults, searchResult)
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

/********************************************************************
* GetProductFilters gets available filters for product search		*
* Multi-tenant: If sellerID is provided, get filters for that       *
* seller's products only. If nil (admin), get all filters.          *
* Now includes variant-based filters (price, options, stock)        *
*********************************************************************/
func (s *ProductServiceImpl) GetProductFilters(sellerID *uint) (*model.ProductFilters, error) {
	brands, categories, attributes, priceRange, variantOptions, stockStatus, err := s.productRepo.GetProductFilters(
		sellerID,
	)
	if err != nil {
		return nil, err
	}

	filters := &model.ProductFilters{
		Brands:       factory.BuildBrandFilters(brands),
		Categories:   s.convertCategoriesToFilters(categories),
		Attributes:   factory.BuildAttributeFilters(attributes),
		PriceRange:   factory.BuildPriceRangeFilter(priceRange),
		VariantTypes: factory.BuildVariantTypeFilters(variantOptions),
		StockStatus:  factory.BuildStockStatusFilter(stockStatus),
	}

	return filters, nil
}

func (s *ProductServiceImpl) convertCategoriesToFilters(
	categories []mapper.CategoryWithProductCount,
) []model.CategoryFilter {
	mp := make(map[uint]model.CategoryFilter)
	var categoryFilter []model.CategoryFilter
	for _, category := range categories {
		mp[category.CategoryID] = factory.BuildCategoryFilter(category)

		if category.ParentID == nil || *category.ParentID == 0 {
			categoryFilter = append(
				categoryFilter,
				factory.BuildCategoryFilter(category),
			)
		}
	}

	for _, category := range categories {
		if category.ParentID != nil && *category.ParentID != 0 {
			parentFilter, exist := mp[*category.ParentID]
			if exist {
				parentFilter.Children = append(
					parentFilter.Children,
					factory.BuildCategoryFilter(category),
				)
			} else {
				categoryFilter = append(
					categoryFilter,
					factory.BuildCategoryFilter(category),
				)
			}
		}
	}

	return categoryFilter
}

/************************************************************************
* GetRelatedProductsScored gets products using intelligent scoring      *
* Uses stored procedure for multi-strategy matching with pagination     *
*************************************************************************/
func (s *ProductServiceImpl) GetRelatedProductsScored(
	productID uint,
	limit int,
	page int,
	strategies string,
	sellerID *uint,
) (*model.RelatedProductsScoredResponse, error) {
	// Validate and set defaults
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}
	if strategies == "" {
		strategies = "all"
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Call repository method that uses stored procedure
	scoredResults, totalCount, err := s.productRepo.FindRelatedScored(
		productID,
		sellerID,
		limit,
		offset,
		strategies,
	)
	if err != nil {
		return nil, err
	}

	// If no results, return empty response
	if len(scoredResults) == 0 {
		return &model.RelatedProductsScoredResponse{
			RelatedProducts: []model.RelatedProductItemScored{},
			Pagination: model.PaginationResponse{
				CurrentPage:  page,
				TotalPages:   0,
				TotalItems:   0,
				ItemsPerPage: limit,
				HasNext:      false,
				HasPrev:      false,
			},
			Meta: model.RelatedProductsMeta{
				StrategiesUsed:  []string{},
				AvgScore:        0,
				TotalStrategies: 8, // Total number of strategies available in the system
			},
		}, nil
	}

	// Build response items with variant options
	relatedItems := make([]model.RelatedProductItemScored, 0, len(scoredResults))
	strategiesUsedMap := make(map[string]bool)
	totalScore := 0

	for _, result := range scoredResults {
		// Build category info
		categoryInfo := model.CategoryHierarchyInfo{
			ID:   result.CategoryID,
			Name: result.CategoryName,
		}
		if result.ParentCategoryID != nil && result.ParentCategoryName != nil {
			categoryInfo.Parent = &model.CategoryInfo{
				ID:   *result.ParentCategoryID,
				Name: *result.ParentCategoryName,
			}
		}

		// Get variant options for this product
		variantOptions, err := s.optionRepo.FindOptionsByProductID(result.ProductID)
		if err != nil {
			return nil, err
		}

		// Build options preview
		optionsPreview := make([]model.OptionPreview, 0, len(variantOptions))
		for _, option := range variantOptions {
			// Get all values for this option
			optionValues, err := s.optionRepo.FindOptionValuesByOptionID(option.ID)
			if err != nil {
				return nil, err
			}

			availableValues := make([]string, 0, len(optionValues))
			for _, val := range optionValues {
				availableValues = append(availableValues, val.Value)
			}

			optionsPreview = append(optionsPreview, model.OptionPreview{
				Name:            option.Name,
				DisplayName:     option.DisplayName,
				AvailableValues: availableValues,
			})
		}

		// Build product response
		productResponse := model.ProductResponse{
			ID:               result.ProductID,
			Name:             result.ProductName,
			CategoryID:       result.CategoryID,
			Category:         categoryInfo,
			Brand:            result.Brand,
			SKU:              result.SKU,
			ShortDescription: result.ShortDescription,
			LongDescription:  result.LongDescription,
			Tags:             result.Tags,
			SellerID:         result.SellerID,
			HasVariants:      result.HasVariants,
			PriceRange: &model.PriceRange{
				Min: result.MinPrice,
				Max: result.MaxPrice,
			},
			AllowPurchase: result.AllowPurchase,
			Images:        []string{}, // Images would need to be fetched separately if needed
			VariantPreview: &model.VariantPreview{
				TotalVariants: int(result.TotalVariants),
				Options:       optionsPreview,
			},
		}

		// Create scored item
		scoredItem := model.RelatedProductItemScored{
			ProductResponse: productResponse,
			RelationReason:  result.RelationReason,
			Score:           result.FinalScore,
			StrategyUsed:    result.StrategyUsed,
		}

		relatedItems = append(relatedItems, scoredItem)
		strategiesUsedMap[result.StrategyUsed] = true
		totalScore += result.FinalScore
	}

	// Build strategies used list
	strategiesUsed := make([]string, 0, len(strategiesUsedMap))
	for strategy := range strategiesUsedMap {
		strategiesUsed = append(strategiesUsed, strategy)
	}

	// Calculate average score
	avgScore := float64(totalScore) / float64(len(relatedItems))

	// Calculate pagination
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))

	return &model.RelatedProductsScoredResponse{
		RelatedProducts: relatedItems,
		Pagination: model.PaginationResponse{
			CurrentPage:  page,
			TotalPages:   totalPages,
			TotalItems:   int(totalCount),
			ItemsPerPage: limit,
			HasNext:      page < totalPages,
			HasPrev:      page > 1,
		},
		Meta: model.RelatedProductsMeta{
			StrategiesUsed:  strategiesUsed,
			AvgScore:        avgScore,
			TotalStrategies: 8, // Total number of strategies available in the system
		},
	}, nil
}
