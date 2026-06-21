package service

import (
	"context"
	"sort"

	commonHelper "ecommerce-be/common/helper"
	commonError "ecommerce-be/common/error"
	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repository"
	"ecommerce-be/product/validator"
	productUtils "ecommerce-be/product/utils"
)

// ProductService defines the interface for product-related business logic
type ProductService interface {
	CreateProduct(
		ctx context.Context,
		req model.ProductCreateRequest,
		sellerID uint,
	) (*model.ProductResponse, error)
	UpdateProduct(
		ctx context.Context,
		id uint,
		sellerId *uint,
		req model.ProductUpdateRequest,
	) (*model.ProductResponse, error)
	DeleteProduct(
		ctx context.Context,
		id uint,
		sellerId *uint,
	) error
}

// ProductServiceImpl implements the ProductService interface
type ProductServiceImpl struct {
	productRepo             repository.ProductRepository
	categoryRepo            repository.CategoryRepository
	variantRepo             repository.VariantRepository
	productQueryService     ProductQueryService
	validatorService        ProductValidatorService
	variantService          VariantService
	variantBulkService      VariantBulkService
	productOptionService    ProductOptionService
	productAttributeService ProductAttributeService
	packageOptionService    PackageOptionService
}

// NewProductService creates a new instance of ProductService
func NewProductService(
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	variantRepo repository.VariantRepository,
	productQueryService ProductQueryService,
	validatorService ProductValidatorService,
	variantService VariantService,
	variantBulkService VariantBulkService,
	productOptionService ProductOptionService,
	productAttributeService ProductAttributeService,
	packageOptionService PackageOptionService,
) ProductService {
	return &ProductServiceImpl{
		productRepo:             productRepo,
		categoryRepo:            categoryRepo,
		variantRepo:             variantRepo,
		productQueryService:     productQueryService,
		validatorService:        validatorService,
		variantService:          variantService,
		variantBulkService:      variantBulkService,
		productOptionService:    productOptionService,
		productAttributeService: productAttributeService,
		packageOptionService:    packageOptionService,
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
	ctx context.Context,
	req model.ProductCreateRequest,
	sellerID uint,
) (*model.ProductResponse, error) {
	var result productCreationResult

	err := db.WithTransaction(ctx, func(txCtx context.Context) error {
		return s.executeProductCreation(txCtx, &result, req, sellerID)
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
	ctx context.Context,
	result *productCreationResult,
	req model.ProductCreateRequest,
	sellerID uint,
) error {
	// Validate and create base product
	if err := s.validateAndCreateProduct(ctx, result, req, sellerID); err != nil {
		return err
	}

	// Create all associated entities
	return s.createProductAssociations(ctx, result, req, sellerID)
}

// validateAndCreateProduct validates request and creates base product entity
func (s *ProductServiceImpl) validateAndCreateProduct(
	ctx context.Context,
	result *productCreationResult,
	req model.ProductCreateRequest,
	sellerID uint,
) error {
	category, err := s.categoryRepo.FindByID(ctx, req.CategoryID)
	if err != nil {
		return err
	}

	if err := validator.ValidateProductCreateRequest(req, category); err != nil {
		return err
	}

	product := factory.CreateProductFromRequest(req, sellerID)
	if err := s.productRepo.Create(ctx, product); err != nil {
		return err
	}

	result.product = product
	result.category = category
	return nil
}

// createProductAssociations creates options, variants, attributes, and package options
func (s *ProductServiceImpl) createProductAssociations(
	ctx context.Context,
	result *productCreationResult,
	req model.ProductCreateRequest,
	sellerID uint,
) error {
	productID := result.product.ID

	// Create options if provided
	if len(req.Options) > 0 {
		options, err := s.productOptionService.CreateOptionsBulk(
			ctx,
			productID,
			sellerID,
			req.Options,
		)
		if err != nil {
			return err
		}
		result.options = options
	}

	// Create variants (explicit or synthesized placeholder for simple products)
	variantRequests := resolveVariantCreateRequests(req)
	variants, err := s.variantBulkService.CreateVariantsBulk(ctx, productID, sellerID, variantRequests)
	if err != nil {
		return err
	}
	result.variants = variants

	// Create attributes if provided
	if len(req.Attributes) > 0 {
		attributes, err := s.productAttributeService.CreateProductAttributesBulk(
			ctx,
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
		packageOptions, err := s.packageOptionService.CreatePackageOptionsBulk(
			ctx,
			productID,
			sellerID,
			req.PackageOptions,
		)
		if err != nil {
			return err
		}
		result.packageOptions = packageOptions
	}

	return nil
}

// resolveVariantCreateRequests returns explicit variants or synthesizes a placeholder for simple products.
func resolveVariantCreateRequests(req model.ProductCreateRequest) []model.CreateVariantRequest {
	if len(req.Variants) > 0 {
		return req.Variants
	}

	allowPurchase := true
	if req.AllowPurchase != nil {
		allowPurchase = *req.AllowPurchase
	}

	isPopular := false
	if req.IsPopular != nil {
		isPopular = *req.IsPopular
	}

	return []model.CreateVariantRequest{
		{
			SKU:           req.BaseSKU,
			Price:         req.Price,
			AllowPurchase: commonHelper.BoolPtr(allowPurchase),
			IsPopular:     commonHelper.BoolPtr(isPopular),
			IsDefault:     commonHelper.BoolPtr(true),
		},
	}
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
	publicVariants := productUtils.FilterPublicVariants(variants)
	variantAgg := calculateVariantAggFromModels(variants, len(options))

	// Use factory builder for base response
	response := factory.BuildProductResponse(product, variantAgg)

	// Add detailed fields from services (not included in base builder)
	response.Options = options
	response.Variants = publicVariants
	response.Attributes = attributes
	response.PackageOptions = factory.BuildPackageOptionResponses(packageOptions)

	return &response
}

// calculateVariantAggFromModels calculates aggregation data from variant models
func calculateVariantAggFromModels(
	variants []model.VariantDetailResponse,
	productOptionsCount int,
) *mapper.VariantAggregation {
	publicVariants := productUtils.FilterPublicVariants(variants)

	agg := &mapper.VariantAggregation{
		ProductOptionsCount: productOptionsCount,
		OptionDerivedCount:  len(publicVariants),
		DefaultPrice:        productUtils.DeriveProductPrice(variants),
		AllowPurchase:       productUtils.DeriveAllowPurchase(variants),
		IsPopular:           productUtils.DeriveIsPopular(variants),
		OptionNames:         []string{},
		OptionValues:        make(map[string][]string),
	}

	if len(publicVariants) > 0 {
		minPrice := publicVariants[0].Price
		maxPrice := publicVariants[0].Price
		optionValuesMap := make(map[string]map[string]bool)

		for _, v := range publicVariants {
			if v.Price < minPrice {
				minPrice = v.Price
			}
			if v.Price > maxPrice {
				maxPrice = v.Price
			}
			for _, opt := range v.SelectedOptions {
				if optionValuesMap[opt.OptionName] == nil {
					optionValuesMap[opt.OptionName] = make(map[string]bool)
				}
				optionValuesMap[opt.OptionName][opt.Value] = true
			}
		}

		agg.MinPrice = minPrice
		agg.MaxPrice = maxPrice

		for optName, valuesSet := range optionValuesMap {
			agg.OptionNames = append(agg.OptionNames, optName)
			values := make([]string, 0, len(valuesSet))
			for v := range valuesSet {
				values = append(values, v)
			}
			agg.OptionValues[optName] = values
		}
	}

	productUtils.ApplyAggregationSemantics(agg)

	return agg
}

/************************************************************
 *	UpdateProduct updates an existing product 		        *
 *	Note: Price, images, stock are managed at variant level *
 ************************************************************/
func (s *ProductServiceImpl) UpdateProduct(
	ctx context.Context,
	id uint,
	sellerId *uint,
	req model.ProductUpdateRequest,
) (*model.ProductResponse, error) {
	// Fetch product to validate
	product, err := s.productRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Fetch category if being updated
	var category *entity.Category
	if req.CategoryID != nil && *req.CategoryID != 0 {
		category, err = s.categoryRepo.FindByID(ctx, *req.CategoryID)
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

	hasCommerceUpdate := req.Price != nil || req.AllowPurchase != nil || req.IsPopular != nil

	if hasCommerceUpdate {
		err = db.WithTransaction(ctx, func(txCtx context.Context) error {
			if err := s.productRepo.Update(txCtx, product); err != nil {
				return err
			}
			return s.applyProductCommerceUpdates(txCtx, product.ID, req)
		})
	} else if err = s.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	// TODO: Update attributes and package options if provided in request

	// Return updated product with full details
	// Note: userID is nil here as this is a seller/admin update operation
	return s.productQueryService.GetProductByID(ctx, product.ID, sellerId, nil)
}

func (s *ProductServiceImpl) applyProductCommerceUpdates(
	ctx context.Context,
	productID uint,
	req model.ProductUpdateRequest,
) error {
	variants, err := s.variantRepo.FindVariantsByProductID(ctx, productID)
	if err != nil {
		return err
	}
	if len(variants) == 0 {
		return commonError.ErrValidation.WithMessage("product has no variants")
	}

	if req.Price != nil {
		defaultVariant := findDefaultVariantEntity(variants)
		if defaultVariant == nil {
			return commonError.ErrValidation.WithMessage("default variant not found")
		}
		defaultVariant.Price = *req.Price
		if err := s.variantRepo.UpdateVariant(ctx, defaultVariant); err != nil {
			return err
		}
	}

	if req.AllowPurchase != nil || req.IsPopular != nil {
		return s.variantRepo.UpdateAllVariantsFlags(ctx, productID, req.AllowPurchase, req.IsPopular)
	}

	return nil
}

func findDefaultVariantEntity(variants []entity.ProductVariant) *entity.ProductVariant {
	if len(variants) == 0 {
		return nil
	}

	sorted := make([]entity.ProductVariant, len(variants))
	copy(sorted, variants)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	for i := range sorted {
		if sorted[i].IsDefault {
			return &sorted[i]
		}
	}

	return &sorted[0]
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
	ctx context.Context,
	id uint,
	sellerId *uint,
) error {
	// Verify product exists and validate ownership
	_, err := s.validatorService.GetAndValidateProductOwnership(ctx, id, sellerId)
	if err != nil {
		return err
	}

	// Use atomic transaction to delete everything
	return db.WithTransaction(ctx, func(txCtx context.Context) error {
		// Delete variants and their associated data (variant_option_values)
		if err := s.variantBulkService.DeleteVariantsByProductID(txCtx, id); err != nil {
			return err
		}

		// Delete product options and their values
		if err := s.productOptionService.DeleteOptionsByProductID(txCtx, id); err != nil {
			return err
		}

		// Delete product attributes
		if err := s.productAttributeService.DeleteAttributesByProductID(txCtx, id); err != nil {
			return err
		}

		// Delete package options
		if err := s.packageOptionService.DeletePackageOptionsByProductID(txCtx, id); err != nil {
			return err
		}

		// Finally, delete the product itself
		return s.productRepo.Delete(txCtx, id)
	})
}
