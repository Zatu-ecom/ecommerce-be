package service

import (
	"context"

	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/factory"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/validator"
)

// AttributeDefinitionService defines the interface for attribute definition business logic
type AttributeDefinitionService interface {
	CreateAttribute(
		ctx context.Context,
		req model.AttributeDefinitionCreateRequest,
	) (*model.AttributeDefinitionResponse, error)
	UpdateAttribute(
		ctx context.Context,
		id uint,
		req model.AttributeDefinitionUpdateRequest,
	) (*model.AttributeDefinitionResponse, error)
	DeleteAttribute(ctx context.Context, id uint) error
	GetAllAttributes(ctx context.Context) (*model.AttributeDefinitionsResponse, error)
	GetAttributeByID(ctx context.Context, id uint) (*model.AttributeDefinitionResponse, error)
	GetAttributeByKey(ctx context.Context, key string) (*model.AttributeDefinitionResponse, error)
	CreateCategoryAttributeDefinition(
		ctx context.Context,
		categoryID uint,
		req model.AttributeDefinitionCreateRequest,
	) (*model.AttributeDefinitionResponse, error)
}

// AttributeDefinitionServiceImpl implements the AttributeDefinitionService interface
type AttributeDefinitionServiceImpl struct {
	attributeRepo repositories.AttributeDefinitionRepository
}

// NewAttributeDefinitionService creates a new instance of AttributeDefinitionService
func NewAttributeDefinitionService(
	attributeRepo repositories.AttributeDefinitionRepository,
) AttributeDefinitionService {
	return &AttributeDefinitionServiceImpl{
		attributeRepo: attributeRepo,
	}
}

// CreateAttribute creates a new attribute definition
func (s *AttributeDefinitionServiceImpl) CreateAttribute(
	ctx context.Context,
	req model.AttributeDefinitionCreateRequest,
) (*model.AttributeDefinitionResponse, error) {
	// Validate attribute key format
	if err := validator.ValidateKey(req.Key); err != nil {
		return nil, err
	}

	// Validate allowed values if provided
	if len(req.AllowedValues) > 0 {
		if err := validator.ValidateAllowedValues(req.AllowedValues); err != nil {
			return nil, err
		}
	}

	// Check if attribute with same key already exists
	existingAttribute, err := s.attributeRepo.FindByKey(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if existingAttribute != nil {
		return nil, prodErrors.ErrAttributeExists
	}

	// Create attribute entity using factory
	attribute := factory.CreateFromRequest(req)

	// Save attribute to database
	if err := s.attributeRepo.Create(ctx, attribute); err != nil {
		return nil, err
	}

	// Build response using converter
	attributeResponse := factory.BuildAttributeResponse(attribute)
	return attributeResponse, nil
}

// UpdateAttribute updates an existing attribute definition
func (s *AttributeDefinitionServiceImpl) UpdateAttribute(
	ctx context.Context,
	id uint,
	req model.AttributeDefinitionUpdateRequest,
) (*model.AttributeDefinitionResponse, error) {
	// Find existing attribute
	attribute, err := s.attributeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate allowed values if provided
	if len(req.AllowedValues) > 0 {
		if err := validator.ValidateAllowedValues(req.AllowedValues); err != nil {
			return nil, err
		}
	}

	// Update attribute using factory
	attribute = factory.UpdateEntity(attribute, req)

	// Save updated attribute
	if err := s.attributeRepo.Update(ctx, attribute); err != nil {
		return nil, err
	}

	// Build response using converter
	attributeResponse := factory.BuildAttributeResponse(attribute)
	return attributeResponse, nil
}

// DeleteAttribute soft deletes an attribute definition
func (s *AttributeDefinitionServiceImpl) DeleteAttribute(ctx context.Context, id uint) error {
	return s.attributeRepo.Delete(ctx, id)
}

// GetAllAttributes gets all active attribute definitions
func (s *AttributeDefinitionServiceImpl) GetAllAttributes(ctx context.Context) (*model.AttributeDefinitionsResponse, error) {
	attributes, err := s.attributeRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	var attributesResponse []model.AttributeDefinitionResponse
	for _, attribute := range attributes {
		ar := factory.BuildAttributeResponse(&attribute)
		attributesResponse = append(attributesResponse, *ar)
	}

	return &model.AttributeDefinitionsResponse{
		Attributes: attributesResponse,
	}, nil
}

// GetAttributeByID gets an attribute definition by ID
func (s *AttributeDefinitionServiceImpl) GetAttributeByID(
	ctx context.Context,
	id uint,
) (*model.AttributeDefinitionResponse, error) {
	attribute, err := s.attributeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	attributeResponse := factory.BuildAttributeResponse(attribute)
	return attributeResponse, nil
}

// GetAttributeByKey gets an attribute definition by key
func (s *AttributeDefinitionServiceImpl) GetAttributeByKey(
	ctx context.Context,
	key string,
) (*model.AttributeDefinitionResponse, error) {
	attribute, err := s.attributeRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if attribute == nil {
		return nil, prodErrors.ErrAttributeNotFound
	}

	attributeResponse := factory.BuildAttributeResponse(attribute)
	return attributeResponse, nil
}

func (s *AttributeDefinitionServiceImpl) CreateCategoryAttributeDefinition(
	ctx context.Context,
	categoryID uint,
	req model.AttributeDefinitionCreateRequest,
) (*model.AttributeDefinitionResponse, error) {
	// Validate attribute key format
	if err := validator.ValidateKey(req.Key); err != nil {
		return nil, err
	}

	// Validate allowed values if provided
	if len(req.AllowedValues) > 0 {
		if err := validator.ValidateAllowedValues(req.AllowedValues); err != nil {
			return nil, err
		}
	}

	// Check if attribute with same key already exists
	existingAttribute, err := s.attributeRepo.FindByKey(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if existingAttribute != nil {
		return nil, prodErrors.ErrAttributeExists
	}

	// Create attribute entity using factory
	attribute := factory.CreateFromRequest(req)

	if err := s.attributeRepo.CreateCategoryAttributeDefinition(ctx, attribute, categoryID); err != nil {
		return nil, err
	}

	attributeResponse := factory.BuildAttributeResponse(attribute)
	return attributeResponse, nil
}
