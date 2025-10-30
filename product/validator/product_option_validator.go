package validator

import (
	"errors"

	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"

	"gorm.io/gorm"
)

// ProductOptionValidator handles validation logic for product options
type ProductOptionValidator struct {
	optionRepo  repositories.ProductOptionRepository
	productRepo repositories.ProductRepository
}

// NewProductOptionValidator creates a new instance of ProductOptionValidator
func NewProductOptionValidator(
	optionRepo repositories.ProductOptionRepository,
	productRepo repositories.ProductRepository,
) *ProductOptionValidator {
	return &ProductOptionValidator{
		optionRepo:  optionRepo,
		productRepo: productRepo,
	}
}

// ValidateProductExists validates that a product exists
func (v *ProductOptionValidator) ValidateProductExists(productID uint) error {
	_, err := v.productRepo.FindByID(productID)
	if err != nil {
		return err
	}
	return nil
}

// ValidateOptionNameUniqueness validates that option name is unique for a product
func (v *ProductOptionValidator) ValidateOptionNameUniqueness(productID uint, name string) error {
	normalizedName := utils.NormalizeToSnakeCase(name)

	existingOptions, err := v.optionRepo.FindOptionsByProductID(productID)
	if err != nil {
		return err
	}

	for _, opt := range existingOptions {
		if opt.Name == normalizedName {
			return prodErrors.ErrProductOptionNameExists
		}
	}
	return nil
}

// ValidateOptionBelongsToProduct validates that an option belongs to a product
func (v *ProductOptionValidator) ValidateOptionBelongsToProduct(
	productID uint,
	optionID uint,
) error {
	option, err := v.optionRepo.FindOptionByID(optionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return prodErrors.ErrProductOptionNotFound
		}
		return err
	}

	if option.ProductID != productID {
		return prodErrors.ErrProductOptionMismatch
	}

	return nil
}

// ValidateOptionNotInUse validates that an option is not being used by any variants
func (v *ProductOptionValidator) ValidateOptionNotInUse(optionID uint) error {
	inUse, variantIDs, err := v.optionRepo.CheckOptionInUse(optionID)
	if err != nil {
		return err
	}

	if inUse {
		// Return custom error that can include variant details
		return prodErrors.ErrProductOptionInUse.WithMessagef(
			"%s (used by %d variants)",
			utils.PRODUCT_OPTION_IN_USE_MSG,
			len(variantIDs),
		)
	}

	return nil
}

func (v *ProductOptionValidator) ValidateProductBelongsToSeller(
	productID uint,
	sellerID uint,
) error {
	product, err := v.productRepo.FindByID(productID)
	if err != nil {
		return err
	}

	if product.SellerID != sellerID {
		return prodErrors.ErrUnauthorizedProductAccess
	}

	return nil
}
