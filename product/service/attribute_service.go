package service

import (
	"regexp"
	"time"

	commonEntity "ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	prodErrors "ecommerce-be/product/errors"
	"ecommerce-be/product/model"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/utils"
)

// AttributeDefinitionService defines the interface for attribute definition business logic
type AttributeDefinitionService interface {
	CreateAttribute(
		req model.AttributeDefinitionCreateRequest,
	) (*model.AttributeDefinitionResponse, error)
	UpdateAttribute(
		id uint,
		req model.AttributeDefinitionUpdateRequest,
	) (*model.AttributeDefinitionResponse, error)
	DeleteAttribute(id uint) error
	GetAllAttributes() (*model.AttributeDefinitionsResponse, error)
	GetAttributeByID(id uint) (*model.AttributeDefinitionResponse, error)
	GetAttributeByKey(key string) (*model.AttributeDefinitionResponse, error)
	CreateCategoryAttributeDefinition(
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
	req model.AttributeDefinitionCreateRequest,
) (*model.AttributeDefinitionResponse, error) {
	attribute, err := s.validateAttributeKeyAndConvertToEntity(req)
	if err != nil {
		return nil, err
	}

	// Save attribute to database
	if err := s.attributeRepo.Create(attribute); err != nil {
		return nil, err
	}

	// Build response using converter
	attributeResponse := utils.ConvertAttributeDefinitionToResponse(attribute)
	return attributeResponse, nil
}

// UpdateAttribute updates an existing attribute definition
func (s *AttributeDefinitionServiceImpl) UpdateAttribute(
	id uint,
	req model.AttributeDefinitionUpdateRequest,
) (*model.AttributeDefinitionResponse, error) {
	// Find existing attribute
	attribute, err := s.attributeRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Update attribute fields
	attribute.Name = req.Name
	attribute.Unit = req.Unit
	attribute.AllowedValues = req.AllowedValues
	attribute.UpdatedAt = time.Now()

	// Save updated attribute
	if err := s.attributeRepo.Update(attribute); err != nil {
		return nil, err
	}

	// Build response using converter
	attributeResponse := utils.ConvertAttributeDefinitionToResponse(attribute)
	return attributeResponse, nil
}

// DeleteAttribute soft deletes an attribute definition
func (s *AttributeDefinitionServiceImpl) DeleteAttribute(id uint) error {
	return s.attributeRepo.Delete(id)
}

// GetAllAttributes gets all active attribute definitions
func (s *AttributeDefinitionServiceImpl) GetAllAttributes() (*model.AttributeDefinitionsResponse, error) {
	attributes, err := s.attributeRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var attributesResponse []model.AttributeDefinitionResponse
	for _, attribute := range attributes {
		ar := utils.ConvertAttributeDefinitionToResponse(&attribute)
		attributesResponse = append(attributesResponse, *ar)
	}

	return &model.AttributeDefinitionsResponse{
		Attributes: attributesResponse,
	}, nil
}

// GetAttributeByID gets an attribute definition by ID
func (s *AttributeDefinitionServiceImpl) GetAttributeByID(
	id uint,
) (*model.AttributeDefinitionResponse, error) {
	attribute, err := s.attributeRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	attributeResponse := utils.ConvertAttributeDefinitionToResponse(attribute)
	return attributeResponse, nil
}

// GetAttributeByKey gets an attribute definition by key
func (s *AttributeDefinitionServiceImpl) GetAttributeByKey(
	key string,
) (*model.AttributeDefinitionResponse, error) {
	attribute, err := s.attributeRepo.FindByKey(key)
	if err != nil {
		return nil, err
	}
	if attribute == nil {
		return nil, prodErrors.ErrAttributeNotFound
	}

	attributeResponse := utils.ConvertAttributeDefinitionToResponse(attribute)
	return attributeResponse, nil
}

func (s *AttributeDefinitionServiceImpl) CreateCategoryAttributeDefinition(
	categoryID uint,
	req model.AttributeDefinitionCreateRequest,
) (*model.AttributeDefinitionResponse, error) {
	attribute, err := s.validateAttributeKeyAndConvertToEntity(req)
	if err != nil {
		return nil, err
	}

	if err := s.attributeRepo.CreateCategoryAttributeDefinition(attribute, categoryID); err != nil {
		return nil, err
	}

	attributeResponse := utils.ConvertAttributeDefinitionToResponse(attribute)
	return attributeResponse, nil
}

// isValidAttributeKey validates the attribute key format
func (s *AttributeDefinitionServiceImpl) isValidAttributeKey(key string) bool {
	// Key must contain only lowercase letters, numbers, and underscores
	matched, _ := regexp.MatchString(`^[a-z0-9_]+$`, key)
	return matched
}

// isValidDataType validates the data type
func (s *AttributeDefinitionServiceImpl) isValidDataType(dataType string) bool {
	validTypes := []string{"string", "number", "boolean", "array"}
	for _, validType := range validTypes {
		if dataType == validType {
			return true
		}
	}
	return false
}

func (s *AttributeDefinitionServiceImpl) validateAttributeKeyAndConvertToEntity(
	req model.AttributeDefinitionCreateRequest,
) (*entity.AttributeDefinition, error) {
	// Validate attribute key format
	if !s.isValidAttributeKey(req.Key) {
		return nil, prodErrors.ErrInvalidAttributeKey
	}

	// Check if attribute with same key already exists
	existingAttribute, err := s.attributeRepo.FindByKey(req.Key)
	if err != nil {
		return nil, err
	}
	if existingAttribute != nil {
		return nil, prodErrors.ErrAttributeExists
	}

	// Create attribute definition entity
	attribute := &entity.AttributeDefinition{
		Key:           req.Key,
		Name:          req.Name,
		Unit:          req.Unit,
		AllowedValues: req.AllowedValues,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	return attribute, nil
}
