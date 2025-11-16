package service

import (
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"
)

// ProductAttributeService defines the interface for product attribute business logic
type ProductAttributeService interface {
	AddProductAttribute(
		productID uint,
		sellerID uint,
		req model.AddProductAttributeRequest,
	) (*model.ProductAttributeDetailResponse, error)

	UpdateProductAttribute(
		productID uint,
		attributeID uint,
		sellerID uint,
		req model.UpdateProductAttributeRequest,
	) (*model.ProductAttributeDetailResponse, error)

	DeleteProductAttribute(
		productID uint,
		attributeID uint,
		sellerID uint,
	) error

	GetProductAttributes(
		productID uint,
	) (*model.ProductAttributesListResponse, error)

	BulkUpdateProductAttributes(
		productID uint,
		sellerID uint,
		req model.BulkUpdateProductAttributesRequest,
	) (*model.BulkUpdateProductAttributesResponse, error)
}

// ProductAttributeServiceImpl implements the ProductAttributeService interface
type ProductAttributeServiceImpl struct {
	productAttrRepo repositories.ProductAttributeRepository
	productRepo     repositories.ProductRepository
	attributeRepo   repositories.AttributeDefinitionRepository
	factory         *factory.ProductAttributeFactory
}

// NewProductAttributeService creates a new instance of ProductAttributeService
func NewProductAttributeService(
	productAttrRepo repositories.ProductAttributeRepository,
	productRepo repositories.ProductRepository,
	attributeRepo repositories.AttributeDefinitionRepository,
) ProductAttributeService {
	return &ProductAttributeServiceImpl{
		productAttrRepo: productAttrRepo,
		productRepo:     productRepo,
		attributeRepo:   attributeRepo,
		factory:         factory.NewProductAttributeFactory(),
	}
}

// AddProductAttribute adds a new attribute to a product
func (s *ProductAttributeServiceImpl) AddProductAttribute(
	productID uint,
	sellerID uint,
	req model.AddProductAttributeRequest,
) (*model.ProductAttributeDetailResponse, error) {
	// Verify product exists and user has access
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, prodErrors.ErrProductNotFound
	}

	// Check if user has permission to modify this product
	if sellerID != 0 && product.SellerID != sellerID {
		return nil, prodErrors.ErrUnauthorizedAttributeAccess
	}

	// Verify attribute definition exists
	attributeDef, err := s.attributeRepo.FindByID(req.AttributeDefinitionID)
	if err != nil {
		return nil, prodErrors.ErrAttributeNotFound
	}

	// Validate request
	if err := validator.ValidateProductAttributeAddRequest(
		req.AttributeDefinitionID,
		req.Value,
		attributeDef.AllowedValues,
	); err != nil {
		return nil, err
	}

	// Check if attribute already exists for this product
	exists, err := s.productAttrRepo.ExistsByProductIDAndAttributeID(
		productID,
		req.AttributeDefinitionID,
	)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, prodErrors.ErrProductAttributeExists
	}

	// Create product attribute entity using factory
	productAttribute := s.factory.CreateFromRequest(productID, req)

	// Save to database
	if err := s.productAttrRepo.Create(productAttribute); err != nil {
		return nil, err
	}

	// Fetch with preloaded data
	createdAttr, err := s.productAttrRepo.FindByID(productAttribute.ID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	return s.factory.BuildDetailResponse(createdAttr), nil
}

// UpdateProductAttribute updates an existing product attribute
func (s *ProductAttributeServiceImpl) UpdateProductAttribute(
	productID uint,
	attributeID uint,
	sellerID uint,
	req model.UpdateProductAttributeRequest,
) (*model.ProductAttributeDetailResponse, error) {
	// Verify product exists and user has access
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, prodErrors.ErrProductNotFound
	}

	// Check if user has permission to modify this product
	if sellerID != 0 && product.SellerID != sellerID {
		return nil, prodErrors.ErrUnauthorizedAttributeAccess
	}

	// Find the product attribute
	productAttribute, err := s.productAttrRepo.FindByID(attributeID)
	if err != nil {
		return nil, err
	}

	// Verify the attribute belongs to this product
	if productAttribute.ProductID != productID {
		return nil, prodErrors.ErrProductAttributeNotFound
	}

	// Fetch attribute definition for validation
	attributeDef, err := s.attributeRepo.FindByID(productAttribute.AttributeDefinitionID)
	if err != nil {
		return nil, err
	}

	// Validate request
	if err := validator.ValidateProductAttributeUpdateRequest(req.Value, attributeDef.AllowedValues); err != nil {
		return nil, err
	}

	// Update entity using factory
	s.factory.UpdateEntity(productAttribute, req)

	// Save to database
	if err := s.productAttrRepo.Update(productAttribute); err != nil {
		return nil, err
	}

	// Fetch updated data
	updatedAttr, err := s.productAttrRepo.FindByID(attributeID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	return s.factory.BuildDetailResponse(updatedAttr), nil
}

// DeleteProductAttribute removes an attribute from a product
func (s *ProductAttributeServiceImpl) DeleteProductAttribute(
	productID uint,
	attributeID uint,
	sellerID uint,
) error {
	// Verify product exists and user has access
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return prodErrors.ErrProductNotFound
	}

	// Check if user has permission to modify this product
	if sellerID != 0 && product.SellerID != sellerID {
		return prodErrors.ErrUnauthorizedAttributeAccess
	}

	// Find the product attribute
	productAttribute, err := s.productAttrRepo.FindByID(attributeID)
	if err != nil {
		return err
	}

	// Verify the attribute belongs to this product
	if productAttribute.ProductID != productID {
		return prodErrors.ErrProductAttributeNotFound
	}

	// Delete from database
	return s.productAttrRepo.Delete(attributeID)
}

// GetProductAttributes retrieves all attributes for a product
func (s *ProductAttributeServiceImpl) GetProductAttributes(
	productID uint,
) (*model.ProductAttributesListResponse, error) {
	// Verify product exists
	_, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, prodErrors.ErrProductNotFound
	}

	// Get all product attributes
	productAttributes, err := s.productAttrRepo.FindAllByProductID(productID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	return s.factory.BuildListResponse(productID, productAttributes), nil
}

// BulkUpdateProductAttributes updates multiple attributes for a product
func (s *ProductAttributeServiceImpl) BulkUpdateProductAttributes(
	productID uint,
	sellerID uint,
	req model.BulkUpdateProductAttributesRequest,
) (*model.BulkUpdateProductAttributesResponse, error) {
	// Verify product exists and belongs to seller
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, prodErrors.ErrProductNotFound
	}

	if sellerID != 0 && product.SellerID != sellerID {
		return nil, prodErrors.ErrUnauthorizedAttributeAccess
	}

	// Track updated attributes
	updatedAttributes := make([]*entity.ProductAttribute, 0, len(req.Attributes))
	updatedCount := 0

	// Process each attribute update
	for _, attrUpdate := range req.Attributes {
		// Fetch the existing attribute
		productAttribute, err := s.productAttrRepo.FindByID(attrUpdate.AttributeID)
		if err != nil {
			// Skip attributes that don't exist (could return error instead)
			continue
		}

		// Verify the attribute belongs to this product
		if productAttribute.ProductID != productID {
			// Skip attributes that don't belong to this product
			continue
		}

		// Get attribute definition for validation
		attributeDef, err := s.attributeRepo.FindByID(productAttribute.AttributeDefinitionID)
		if err != nil {
			continue
		}

		// Validate the new value against allowed values
		if err := validator.ValidateProductAttributeValue(attrUpdate.Value, attributeDef.AllowedValues); err != nil {
			return nil, err
		}

		// Update attribute fields
		productAttribute.Value = attrUpdate.Value
		productAttribute.SortOrder = attrUpdate.SortOrder

		// Save to database
		if err := s.productAttrRepo.Update(productAttribute); err != nil {
			return nil, err
		}

		updatedAttributes = append(updatedAttributes, productAttribute)
		updatedCount++
	}

	// Build response
	attributeResponses := make([]model.ProductAttributeDetailResponse, len(updatedAttributes))
	for i, attr := range updatedAttributes {
		attributeResponses[i] = *s.factory.BuildDetailResponse(attr)
	}

	return &model.BulkUpdateProductAttributesResponse{
		UpdatedCount: updatedCount,
		Attributes:   attributeResponses,
	}, nil
}
