package validator

import (
	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
)

// ProductValidator handles validation logic for product operations
type ProductValidator struct {
	productRepo  repositories.ProductRepository
	categoryRepo repositories.CategoryRepository
	optionRepo   repositories.ProductOptionRepository
}

// NewProductValidator creates a new product validator
func NewProductValidator(
	productRepo repositories.ProductRepository,
	categoryRepo repositories.CategoryRepository,
	optionRepo repositories.ProductOptionRepository,
) *ProductValidator {
	return &ProductValidator{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		optionRepo:   optionRepo,
	}
}

// ValidateCategoryExists checks if a category exists
func (v *ProductValidator) ValidateCategoryExists(categoryID uint) (*entity.Category, error) {
	category, err := v.categoryRepo.FindByID(categoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, prodErrors.ErrInvalidCategory
	}
	return category, nil
}

// ValidateProductExists checks if a product exists by ID
func (v *ProductValidator) ValidateProductExistsAndCheckProductOwnership(
	productID uint,
	sellerID *uint,
) (*entity.Product, error) {
	product, err := v.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}
	if product == nil || (sellerID != nil && product.SellerID != *sellerID) {
		return nil, prodErrors.ErrUnauthorizedProductAccess
	}

	return product, nil
}

// ValidateVariantRequirements validates variant-related requirements for product creation
func (v *ProductValidator) ValidateVariantRequirements(req model.ProductCreateRequest) error {
	// At least one variant is required
	if len(req.Variants) == 0 {
		return commonError.ErrValidation.WithMessage("at least one variant is required")
	}

	return nil
}

// ValidateVariantSKUsUnique checks that all variant SKUs in the request are unique
func (v *ProductValidator) ValidateVariantSKUsUnique(variants []model.CreateVariantRequest) error {
	skuMap := make(map[string]bool)
	for _, variant := range variants {
		if skuMap[variant.SKU] {
			return commonError.ErrValidation.WithMessagef("duplicate variant SKU: %s", variant.SKU)
		}
		skuMap[variant.SKU] = true
	}
	return nil
}

// ValidateProductCreateRequest validates the entire product creation request
// This is the main validation orchestrator for product creation
func (v *ProductValidator) ValidateProductCreateRequest(req model.ProductCreateRequest) error {
	// Validate category exists
	if _, err := v.ValidateCategoryExists(req.CategoryID); err != nil {
		return err
	}

	// Validate variant requirements
	if err := v.ValidateVariantRequirements(req); err != nil {
		return err
	}

	// Validate variant SKUs are unique if manual variants provided
	if len(req.Variants) > 0 {
		if err := v.ValidateVariantSKUsUnique(req.Variants); err != nil {
			return err
		}
	}

	return nil
}

// ValidateProductUpdateRequest validates product update request
func (v *ProductValidator) ValidateProductUpdateRequest(
	productID uint,
	sellerID *uint,
	req model.ProductUpdateRequest,
) (*entity.Product, error) {
	// Verify product exists
	product, err := v.ValidateProductExistsAndCheckProductOwnership(productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Validate category if being updated
	if req.CategoryID != 0 {
		if _, err := v.ValidateCategoryExists(req.CategoryID); err != nil {
			return nil, err
		}
	}

	return product, nil
}
