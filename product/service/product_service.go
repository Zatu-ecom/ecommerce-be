package service

import (
	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
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
	productRepo             repositories.ProductRepository
	categoryRepo            repositories.CategoryRepository
	productQueryService     ProductQueryService
	validatorService        ProductValidatorService
	variantService          VariantService
	productOptionService    ProductOptionService
	productAttributeService ProductAttributeService
}

// NewProductService creates a new instance of ProductService
func NewProductService(
	productRepo repositories.ProductRepository,
	categoryRepo repositories.CategoryRepository,
	productQueryService ProductQueryService,
	validatorService ProductValidatorService,
	variantService VariantService,
	productOptionService ProductOptionService,
	productAttributeService ProductAttributeService,
) ProductService {
	return &ProductServiceImpl{
		productRepo:             productRepo,
		categoryRepo:            categoryRepo,
		productQueryService:     productQueryService,
		validatorService:        validatorService,
		variantService:          variantService,
		productOptionService:    productOptionService,
		productAttributeService: productAttributeService,
	}
}

/***********************************************
 *    CreateProduct creates a new product      *
 ***********************************************/

type productCreationResult struct {
	product        *entity.Product
	category       *entity.Category
	options        []model.ProductOptionDetailResponse
	variants       []model.VariantDetailResponse
	attributes     []model.ProductAttributeResponse
	packageOptions []entity.PackageOption
}

func (s *ProductServiceImpl) CreateProduct(
	req model.ProductCreateRequest,
	sellerID uint,
) (*model.ProductResponse, error) {
	var result productCreationResult

	err := db.Atomic(func(tx *gorm.DB) error {
		return s.executeProductCreation(&result, req, sellerID)
	})
	if err != nil {
		return nil, err
	}

	result.product.Category = result.category
	return s.buildProductResponseFromModels(
		result.product,
		result.variants,
		result.options,
		result.attributes,
		result.packageOptions,
	), nil
}

// executeProductCreation orchestrates all product creation steps in transaction
func (s *ProductServiceImpl) executeProductCreation(
	result *productCreationResult,
	req model.ProductCreateRequest,
	sellerID uint,
) error {
	// Validate and create base product
	if err := s.validateAndCreateProduct(result, req, sellerID); err != nil {
		return err
	}

	// Create all associated entities
	return s.createProductAssociations(result, req, sellerID)
}

// validateAndCreateProduct validates request and creates base product entity
func (s *ProductServiceImpl) validateAndCreateProduct(
	result *productCreationResult,
	req model.ProductCreateRequest,
	sellerID uint,
) error {
	category, err := s.categoryRepo.FindByID(req.CategoryID)
	if err != nil {
		return err
	}

	if err := validator.ValidateProductCreateRequest(req, category); err != nil {
		return err
	}

	product := factory.CreateProductFromRequest(req, sellerID)
	if err := s.productRepo.Create(product); err != nil {
		return err
	}

	result.product = product
	result.category = category
	return nil
}

// createProductAssociations creates options, variants, attributes, and package options
func (s *ProductServiceImpl) createProductAssociations(
	result *productCreationResult,
	req model.ProductCreateRequest,
	sellerID uint,
) error {
	productID := result.product.ID

	// Create options if provided
	if len(req.Options) > 0 {
		options, err := s.productOptionService.CreateOptionsBulk(productID, sellerID, req.Options)
		if err != nil {
			return err
		}
		result.options = options
	}

	// Create variants (required)
	variants, err := s.variantService.CreateVariantsBulk(productID, sellerID, req.Variants)
	if err != nil {
		return err
	}
	result.variants = variants

	// Create attributes if provided
	if len(req.Attributes) > 0 {
		attributes, err := s.productAttributeService.CreateProductAttributesBulk(
			productID,
			sellerID,
			req.Attributes,
		)
		if err != nil {
			return err
		}
		result.attributes = attributes
	}

	// Create package options if provided
	if len(req.PackageOptions) > 0 {
		packageOptions, err := s.createPackageOption(productID, req.PackageOptions)
		if err != nil {
			return err
		}
		result.packageOptions = packageOptions
	}

	return nil
}

// buildProductResponseFromModels combines models from different services into final product response
// Uses factory builder for base response, then adds detailed fields from services
func (s *ProductServiceImpl) buildProductResponseFromModels(
	product *entity.Product,
	variants []model.VariantDetailResponse,
	options []model.ProductOptionDetailResponse,
	attributes []model.ProductAttributeResponse,
	packageOptions []entity.PackageOption,
) *model.ProductResponse {
	// Calculate variant aggregation from models
	variantAgg := calculateVariantAggFromModels(variants)

	// Use factory builder for base response
	response := factory.BuildProductResponse(product, variantAgg)

	// Add detailed fields from services (not included in base builder)
	response.Options = options
	response.Variants = variants
	response.Attributes = attributes
	response.PackageOptions = factory.BuildPackageOptionResponses(packageOptions)

	return &response
}

// calculateVariantAggFromModels calculates aggregation data from variant models
func calculateVariantAggFromModels(
	variants []model.VariantDetailResponse,
) *mapper.VariantAggregation {
	agg := &mapper.VariantAggregation{
		HasVariants:   len(variants) > 0,
		TotalVariants: len(variants),
		AllowPurchase: false,
		OptionNames:   []string{},
		OptionValues:  make(map[string][]string),
	}

	if len(variants) == 0 {
		return agg
	}

	minPrice := variants[0].Price
	maxPrice := variants[0].Price
	optionValuesMap := make(map[string]map[string]bool) // optionName -> set of unique values

	for _, v := range variants {
		if v.Price < minPrice {
			minPrice = v.Price
		}
		if v.Price > maxPrice {
			maxPrice = v.Price
		}
		if v.AllowPurchase {
			agg.AllowPurchase = true
		}
		if agg.MainImage == "" && len(v.Images) > 0 {
			agg.MainImage = v.Images[0]
		}

		// Collect unique option values
		for _, opt := range v.SelectedOptions {
			if optionValuesMap[opt.OptionName] == nil {
				optionValuesMap[opt.OptionName] = make(map[string]bool)
			}
			optionValuesMap[opt.OptionName][opt.Value] = true
		}
	}

	agg.MinPrice = minPrice
	agg.MaxPrice = maxPrice

	// Build OptionNames and OptionValues
	for optName, valuesSet := range optionValuesMap {
		agg.OptionNames = append(agg.OptionNames, optName)
		values := []string{}
		for v := range valuesSet {
			values = append(values, v)
		}
		agg.OptionValues[optName] = values
	}

	return agg
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

	// Return updated product with full details
	return s.productQueryService.GetProductByID(product.ID, sellerId)
}

/***************************************************
* Deletes a product and all associated data        *
* Implements PRD Section 3.1.5                     *
* Cascading deletes handled by respective services:*
* - Variants (via VariantService)                  *
* - Product options (via ProductOptionService)     *
* - Product attributes (via ProductAttributeService)*
* - Package options (direct)                       *
****************************************************/
func (s *ProductServiceImpl) DeleteProduct(
	id uint,
	sellerId *uint,
) error {
	// Verify product exists and validate ownership
	_, err := s.validatorService.GetAndValidateProductOwnership(id, sellerId)
	if err != nil {
		return err
	}

	// Use atomic transaction to delete everything
	return db.Atomic(func(tx *gorm.DB) error {
		// Delete variants and their associated data (variant_option_values)
		if err := s.variantService.DeleteVariantsByProductID(id); err != nil {
			return err
		}

		// Delete product options and their values
		if err := s.productOptionService.DeleteOptionsByProductID(id); err != nil {
			return err
		}

		// Delete product attributes
		if err := s.productAttributeService.DeleteAttributesByProductID(id); err != nil {
			return err
		}

		// Delete package options (no separate service yet)
		if err := s.productRepo.DeletePackageOptionsByProductID(id); err != nil {
			return err
		}

		// Finally, delete the product itself
		return s.productRepo.Delete(id)
	})
}
