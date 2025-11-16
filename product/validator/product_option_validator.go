package validator

import (
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/utils"
)

// ValidateProductExists validates that a product exists
// product should be nil if not found, otherwise the fetched product entity
func ValidateProductExists(product *entity.Product) error {
	if product == nil {
		return prodErrors.ErrProductNotFound
	}
	return nil
}

// ValidateProductOptionNameUniqueness validates that option name is unique for a product
// existingOptions should be the list of options already associated with the product
func ValidateProductOptionNameUniqueness(name string, existingOptions []entity.ProductOption) error {
	normalizedName := utils.NormalizeToSnakeCase(name)

	for _, opt := range existingOptions {
		if opt.Name == normalizedName {
			return prodErrors.ErrProductOptionNameExists
		}
	}
	return nil
}

// ValidateProductOptionBelongsToProduct validates that an option belongs to a product
// option should be the fetched option entity
func ValidateProductOptionBelongsToProduct(
	productID uint,
	option *entity.ProductOption,
) error {
	if option == nil {
		return prodErrors.ErrProductOptionNotFound
	}

	if option.ProductID != productID {
		return prodErrors.ErrProductOptionMismatch
	}

	return nil
}

// ValidateProductOptionNotInUse validates that an option is not being used by any variants
// inUse indicates if the option is being used by variants
// variantCount is the number of variants using this option
func ValidateProductOptionNotInUse(inUse bool, variantCount int) error {
	if inUse {
		// Return custom error that can include variant details
		return prodErrors.ErrProductOptionInUse.WithMessagef(
			"%s (used by %d variants)",
			utils.PRODUCT_OPTION_IN_USE_MSG,
			variantCount,
		)
	}

	return nil
}

// ValidateProductBelongsToSeller validates that a product belongs to a seller
// product should be the fetched product entity
func ValidateProductBelongsToSeller(
	product *entity.Product,
	sellerID uint,
) error {
	if product == nil {
		return prodErrors.ErrProductNotFound
	}

	if sellerID != 0 && product.SellerID != sellerID {
		return prodErrors.ErrUnauthorizedProductAccess
	}

	return nil
}
