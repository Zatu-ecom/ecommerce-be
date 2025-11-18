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
	// TODO: Remove these once DeleteProduct is refactored to use services
	variantRepo   repositories.VariantRepository
	optionRepo    repositories.ProductOptionRepository
	attributeRepo repositories.AttributeDefinitionRepository
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
	// TODO: Remove these once DeleteProduct is refactored
	variantRepo repositories.VariantRepository,
	optionRepo repositories.ProductOptionRepository,
	attributeRepo repositories.AttributeDefinitionRepository,
) ProductService {
	return &ProductServiceImpl{
		productRepo:             productRepo,
		categoryRepo:            categoryRepo,
		productQueryService:     productQueryService,
		validatorService:        validatorService,
		variantService:          variantService,
		productOptionService:    productOptionService,
		productAttributeService: productAttributeService,
		variantRepo:             variantRepo,
		optionRepo:              optionRepo,
		attributeRepo:           attributeRepo,
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
	var category *entity.Category
	var optionsModel []model.ProductOptionDetailResponse
	var variantsModel []model.VariantDetailResponse
	var attributesModel []model.ProductAttributeResponse
	var packageOptions []entity.PackageOption

	err := db.Atomic(func(tx *gorm.DB) error {
		// Fetch category for validation
		var err error
		category, err = s.categoryRepo.FindByID(req.CategoryID)
		if err != nil {
			return err
		}

		// Validate request
		if err := validator.ValidateProductCreateRequest(req, category); err != nil {
			return err
		}

		// Create product
		product = factory.CreateProductFromRequest(req, sellerID)
		if err := s.productRepo.Create(product); err != nil {
			return err
		}

		// Create options (returns models)
		if len(req.Options) > 0 {
			optionsModel, err = s.productOptionService.CreateOptionsBulk(product.ID, sellerID, req.Options)
			if err != nil {
				return err
			}
		}

		// Create variants (returns models)
		variantsModel, err = s.variantService.CreateVariantsBulk(product.ID, sellerID, req.Variants)
		if err != nil {
			return err
		}

		// Create attributes (returns models)
		if len(req.Attributes) > 0 {
			attributesModel, err = s.productAttributeService.CreateProductAttributesBulk(product.ID, sellerID, req.Attributes)
			if err != nil {
				return err
			}
		}

		// Create package options
		if len(req.PackageOptions) > 0 {
			packageOptions, err = s.createPackageOption(product.ID, req.PackageOptions)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build final response from models (no extra DB query!)
	product.Category = category
	return s.buildProductResponseFromModels(product, variantsModel, optionsModel, attributesModel, packageOptions), nil
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
func calculateVariantAggFromModels(variants []model.VariantDetailResponse) *mapper.VariantAggregation {
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
	// Verify product exists and validate ownership using validator service
	_, err := s.validatorService.GetAndValidateProductOwnership(id, sellerId)
	if err != nil {
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
