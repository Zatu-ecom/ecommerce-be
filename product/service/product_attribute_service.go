package service

import (
	"context"

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
		ctx context.Context,
		productID uint,
		sellerID uint,
		req model.AddProductAttributeRequest,
	) (*model.ProductAttributeDetailResponse, error)

	UpdateProductAttribute(
		ctx context.Context,
		productID uint,
		attributeID uint,
		sellerID uint,
		req model.UpdateProductAttributeRequest,
	) (*model.ProductAttributeDetailResponse, error)

	DeleteProductAttribute(
		ctx context.Context,
		productID uint,
		attributeID uint,
		sellerID uint,
	) error

	GetProductAttributes(
		ctx context.Context,
		productID uint,
	) (*model.ProductAttributesListResponse, error)

	BulkUpdateProductAttributes(
		ctx context.Context,
		productID uint,
		sellerID uint,
		req model.BulkUpdateProductAttributesRequest,
	) (*model.BulkUpdateProductAttributesResponse, error)

	// CreateProductAttributesBulk creates multiple product attributes in bulk
	// Returns models for immediate use in responses
	CreateProductAttributesBulk(
		ctx context.Context,
		productID uint,
		sellerID uint,
		requests []model.ProductAttributeRequest,
	) ([]model.ProductAttributeResponse, error)

	// DeleteAttributesByProductID deletes all product attributes for a product
	DeleteAttributesByProductID(ctx context.Context, productID uint) error
}

// ProductAttributeServiceImpl implements the ProductAttributeService interface
type ProductAttributeServiceImpl struct {
	productAttrRepo  repositories.ProductAttributeRepository
	productRepo      repositories.ProductRepository
	attributeRepo    repositories.AttributeDefinitionRepository
	validatorService ProductValidatorService
}

// NewProductAttributeService creates a new instance of ProductAttributeService
func NewProductAttributeService(
	productAttrRepo repositories.ProductAttributeRepository,
	productRepo repositories.ProductRepository,
	attributeRepo repositories.AttributeDefinitionRepository,
	validatorService ProductValidatorService,
) ProductAttributeService {
	return &ProductAttributeServiceImpl{
		productAttrRepo:  productAttrRepo,
		productRepo:      productRepo,
		attributeRepo:    attributeRepo,
		validatorService: validatorService,
	}
}

// AddProductAttribute adds a new attribute to a product
func (s *ProductAttributeServiceImpl) AddProductAttribute(
	ctx context.Context,
	productID uint,
	sellerID uint,
	req model.AddProductAttributeRequest,
) (*model.ProductAttributeDetailResponse, error) {
	// Validate product ownership using validator service (eliminates duplication)
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Verify attribute definition exists
	attributeDef, err := s.attributeRepo.FindByID(ctx, req.AttributeDefinitionID)
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
		ctx,
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
	productAttribute := factory.BuildProductAttributeFromCreateRequest(productID, req)

	// Save to database
	if err := s.productAttrRepo.Create(ctx, productAttribute); err != nil {
		return nil, err
	}

	// Fetch with preloaded data
	createdAttr, err := s.productAttrRepo.FindByID(ctx, productAttribute.ID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	return factory.BuildProductAttributeDetailResponse(createdAttr), nil
}

// UpdateProductAttribute updates an existing product attribute
func (s *ProductAttributeServiceImpl) UpdateProductAttribute(
	ctx context.Context,
	productID uint,
	attributeID uint,
	sellerID uint,
	req model.UpdateProductAttributeRequest,
) (*model.ProductAttributeDetailResponse, error) {
	// Validate product ownership using validator service (eliminates duplication)
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Find the product attribute
	productAttribute, err := s.productAttrRepo.FindByID(ctx, attributeID)
	if err != nil {
		return nil, err
	}

	// Verify the attribute belongs to this product
	if productAttribute.ProductID != productID {
		return nil, prodErrors.ErrProductAttributeNotFound
	}

	// Fetch attribute definition for validation
	attributeDef, err := s.attributeRepo.FindByID(ctx, productAttribute.AttributeDefinitionID)
	if err != nil {
		return nil, err
	}

	// Validate request
	if err := validator.ValidateProductAttributeUpdateRequest(req.Value, attributeDef.AllowedValues); err != nil {
		return nil, err
	}

	// Update entity using factory
	factory.BuildProductAttributeFromUpdateRequest(productAttribute, req)

	// Save to database
	if err := s.productAttrRepo.Update(ctx, productAttribute); err != nil {
		return nil, err
	}

	// Fetch updated data
	updatedAttr, err := s.productAttrRepo.FindByID(ctx, attributeID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	return factory.BuildProductAttributeDetailResponse(updatedAttr), nil
}

// DeleteProductAttribute removes an attribute from a product
func (s *ProductAttributeServiceImpl) DeleteProductAttribute(
	ctx context.Context,
	productID uint,
	attributeID uint,
	sellerID uint,
) error {
	// Validate product ownership using validator service (eliminates duplication)
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return err
	}

	// Find the product attribute
	productAttribute, err := s.productAttrRepo.FindByID(ctx, attributeID)
	if err != nil {
		return err
	}

	// Verify the attribute belongs to this product
	if productAttribute.ProductID != productID {
		return prodErrors.ErrProductAttributeNotFound
	}

	// Delete from database
	return s.productAttrRepo.Delete(ctx, attributeID)
}

// GetProductAttributes retrieves all attributes for a product
func (s *ProductAttributeServiceImpl) GetProductAttributes(
	ctx context.Context,
	productID uint,
) (*model.ProductAttributesListResponse, error) {
	// Verify product exists
	_, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, prodErrors.ErrProductNotFound
	}

	// Get all product attributes
	productAttributes, err := s.productAttrRepo.FindAllByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	// Build response using factory
	return factory.BuildProductAttributesListResponse(productID, productAttributes), nil
}

// BulkUpdateProductAttributes updates multiple attributes for a product
func (s *ProductAttributeServiceImpl) BulkUpdateProductAttributes(
	ctx context.Context,
	productID uint,
	sellerID uint,
	req model.BulkUpdateProductAttributesRequest,
) (*model.BulkUpdateProductAttributesResponse, error) {
	// Validate product ownership using validator service (eliminates duplication)
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Track updated attributes
	updatedAttributes := make([]*entity.ProductAttribute, 0, len(req.Attributes))
	updatedCount := 0

	// Process each attribute update
	for _, attrUpdate := range req.Attributes {
		// Fetch the existing attribute
		productAttribute, err := s.productAttrRepo.FindByID(ctx, attrUpdate.AttributeID)
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
		attributeDef, err := s.attributeRepo.FindByID(ctx, productAttribute.AttributeDefinitionID)
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
		if err := s.productAttrRepo.Update(ctx, productAttribute); err != nil {
			return nil, err
		}

		updatedAttributes = append(updatedAttributes, productAttribute)
		updatedCount++
	}

	// Build response
	attributeResponses := make([]model.ProductAttributeDetailResponse, len(updatedAttributes))
	for i, attr := range updatedAttributes {
		attributeResponses[i] = *factory.BuildProductAttributeDetailResponse(attr)
	}

	return &model.BulkUpdateProductAttributesResponse{
		UpdatedCount: updatedCount,
		Attributes:   attributeResponses,
	}, nil
}

/***********************************************
 *      CreateProductAttributesBulk            *
 ***********************************************/
// CreateProductAttributesBulk creates multiple product attributes in bulk
// This method handles both attribute definition creation/update and product attribute linking
// Moved from ProductService for proper separation of concerns
func (s *ProductAttributeServiceImpl) CreateProductAttributesBulk(
	ctx context.Context,
	productID uint,
	sellerID uint,
	requests []model.ProductAttributeRequest,
) ([]model.ProductAttributeResponse, error) {
	if len(requests) == 0 {
		return []model.ProductAttributeResponse{}, nil
	}

	// Validate product ownership using validator service (eliminates duplication)
	_, err := s.validatorService.GetAndValidateProductOwnershipNonPtr(ctx, productID, sellerID)
	if err != nil {
		return nil, err
	}

	// Extract unique keys and fetch existing attribute definitions (single query)
	keys := extractUniqueKeys(requests)
	attributeMap, err := s.attributeRepo.FindByKeys(ctx, keys)
	if err != nil {
		return nil, err
	}

	// Process attributes and prepare bulk operations
	operations := processAttributesForBulkOperations(productID, requests, attributeMap)

	// Execute all bulk operations
	if err = s.executeBulkOperations(ctx, operations); err != nil {
		return nil, err
	}

	// Convert entities to models using factory
	return s.convertAttributesToModels(operations.productAttributesToCreate), nil
}

// extractUniqueKeys extracts unique keys from attribute requests
func extractUniqueKeys(requests []model.ProductAttributeRequest) []string {
	keys := make([]string, 0, len(requests))
	keySet := make(map[string]bool)
	for _, attr := range requests {
		if !keySet[attr.Key] {
			keys = append(keys, attr.Key)
			keySet[attr.Key] = true
		}
	}
	return keys
}

// BulkAttributeOperations holds all bulk operations to be executed
type BulkAttributeOperations struct {
	attributesToUpdate        []*entity.AttributeDefinition
	attributesToCreate        []*entity.AttributeDefinition
	productAttributesToCreate []*entity.ProductAttribute
}

// processAttributesForBulkOperations processes attributes and prepares bulk operations using factory
func processAttributesForBulkOperations(
	productID uint,
	requests []model.ProductAttributeRequest,
	attributeMap map[string]*entity.AttributeDefinition,
) *BulkAttributeOperations {
	operations := &BulkAttributeOperations{
		attributesToUpdate:        make([]*entity.AttributeDefinition, 0),
		attributesToCreate:        make([]*entity.AttributeDefinition, 0),
		productAttributesToCreate: make([]*entity.ProductAttribute, 0),
	}

	for _, attr := range requests {
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
		requests,
		attributeMap,
	)
	operations.productAttributesToCreate = productAttributes

	return operations
}

// executeBulkOperations executes all bulk database operations
func (s *ProductAttributeServiceImpl) executeBulkOperations(
	ctx context.Context,
	operations *BulkAttributeOperations,
) error {
	// Bulk create new attribute definitions
	if len(operations.attributesToCreate) > 0 {
		if err := s.attributeRepo.CreateBulk(ctx, operations.attributesToCreate); err != nil {
			return err
		}
	}

	// Bulk update modified attributes
	if len(operations.attributesToUpdate) > 0 {
		if err := s.attributeRepo.UpdateBulk(ctx, operations.attributesToUpdate); err != nil {
			return err
		}
	}

	// BULK: Create ALL product attributes in ONE query
	if len(operations.productAttributesToCreate) > 0 {
		if err := s.productAttrRepo.BulkCreate(
			ctx,
			operations.productAttributesToCreate,
		); err != nil {
			return err
		}
	}

	return nil
}

// convertAttributesToModels converts product attribute entities to model responses
func (s *ProductAttributeServiceImpl) convertAttributesToModels(
	attributes []*entity.ProductAttribute,
) []model.ProductAttributeResponse {
	if len(attributes) == 0 {
		return []model.ProductAttributeResponse{}
	}

	responses := make([]model.ProductAttributeResponse, 0, len(attributes))
	for _, attr := range attributes {
		if attr.AttributeDefinition != nil {
			responses = append(responses, model.ProductAttributeResponse{
				Name:  attr.AttributeDefinition.Name,
				Key:   attr.AttributeDefinition.Key,
				Value: attr.Value,
				Unit:  attr.AttributeDefinition.Unit,
			})
		}
	}
	return responses
}

// DeleteAttributesByProductID deletes all product attributes for a product
func (s *ProductAttributeServiceImpl) DeleteAttributesByProductID(ctx context.Context, productID uint) error {
	return s.attributeRepo.DeleteProductAttributesByProductID(ctx, productID)
}
