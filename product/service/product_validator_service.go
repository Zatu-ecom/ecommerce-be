package service

import (
	"ecommerce-be/product/entity"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"
)

type ProductValidatorService interface {
	// Service-to-Service methods (for other services to use)
	// These methods centralize validation logic to avoid duplication
	GetAndValidateProductOwnership(
		productID uint,
		sellerID *uint,
	) (*entity.Product, error)
	GetAndValidateProductOwnershipNonPtr(
		productID uint,
		sellerID uint,
	) (*entity.Product, error)
	GetAndValidateProduct(
		productID uint,
	) (*entity.Product, error)
}

type ProductValidatorServiceImpl struct {
	productRepo repositories.ProductRepository
}

func NewProductValidatorService(
	productRepo repositories.ProductRepository,
) *ProductValidatorServiceImpl {
	return &ProductValidatorServiceImpl{
		productRepo: productRepo,
	}
}

/***********************************************
 *     Service-to-Service Methods              *
 *     (For other services to use)             *
 ***********************************************/

// GetAndValidateProductOwnership fetches a product and validates that the seller has access to it
// This method centralizes product ownership validation logic to avoid duplication across services
// Returns the product entity if validation passes, error otherwise
func (s *ProductValidatorServiceImpl) GetAndValidateProductOwnership(
	productID uint,
	sellerID *uint,
) (*entity.Product, error) {
	// Fetch product from repository
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Validate product exists and seller has ownership
	if err := validator.ValidateProductExistsAndOwnership(product, sellerID); err != nil {
		return nil, err
	}

	return product, nil
}

// GetAndValidateProductOwnershipNonPtr fetches a product and validates seller ownership (non-pointer sellerID)
// This method is for services that use uint instead of *uint for sellerID
// Converts uint to *uint internally and calls GetAndValidateProductOwnership
func (s *ProductValidatorServiceImpl) GetAndValidateProductOwnershipNonPtr(
	productID uint,
	sellerID uint,
) (*entity.Product, error) {
	// Convert uint to *uint
	var sellerIDPtr *uint
	if sellerID != 0 {
		sellerIDPtr = &sellerID
	}

	return s.GetAndValidateProductOwnership(productID, sellerIDPtr)
}

// GetAndValidateProduct fetches a product and validates that it exists
// This method centralizes product existence validation logic
// Returns the product entity if validation passes, error otherwise
func (s *ProductValidatorServiceImpl) GetAndValidateProduct(
	productID uint,
) (*entity.Product, error) {
	// Fetch product from repository
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Validate product exists
	if err := validator.ValidateProductExists(product); err != nil {
		return nil, err
	}

	return product, nil
}
