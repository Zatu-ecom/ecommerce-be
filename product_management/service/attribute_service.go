package service

import (
	"errors"
	"regexp"
	"time"

	commonEntity "ecommerce-be/common/entity"
	"ecommerce-be/product_management/entity"
	"ecommerce-be/product_management/model"
	"ecommerce-be/product_management/repositories"
	"ecommerce-be/product_management/utils"
)

// AttributeDefinitionService defines the interface for attribute definition business logic
type AttributeDefinitionService interface {
	CreateAttribute(req model.AttributeDefinitionCreateRequest) (*model.AttributeDefinitionResponse, error)
	UpdateAttribute(id uint, req model.AttributeDefinitionUpdateRequest) (*model.AttributeDefinitionResponse, error)
	DeleteAttribute(id uint) error
	GetAllAttributes() (*model.AttributeDefinitionsResponse, error)
	GetAttributeByID(id uint) (*model.AttributeDefinitionResponse, error)
	GetAttributeByKey(key string) (*model.AttributeDefinitionResponse, error)
}

// AttributeDefinitionServiceImpl implements the AttributeDefinitionService interface
type AttributeDefinitionServiceImpl struct {
	attributeRepo repositories.AttributeDefinitionRepository
}

// NewAttributeDefinitionService creates a new instance of AttributeDefinitionService
func NewAttributeDefinitionService(attributeRepo repositories.AttributeDefinitionRepository) AttributeDefinitionService {
	return &AttributeDefinitionServiceImpl{
		attributeRepo: attributeRepo,
	}
}

// CreateAttribute creates a new attribute definition
func (s *AttributeDefinitionServiceImpl) CreateAttribute(req model.AttributeDefinitionCreateRequest) (*model.AttributeDefinitionResponse, error) {
	// Validate attribute key format
	if !s.isValidAttributeKey(req.Key) {
		return nil, errors.New(utils.ATTRIBUTE_KEY_FORMAT_MSG)
	}

	// Check if attribute with same key already exists
	existingAttribute, err := s.attributeRepo.FindByKey(req.Key)
	if err != nil {
		return nil, err
	}
	if existingAttribute != nil {
		return nil, errors.New(utils.ATTRIBUTE_DEFINITION_EXISTS_MSG)
	}

	// Validate data type
	if !s.isValidDataType(req.DataType) {
		return nil, errors.New(utils.ATTRIBUTE_DATA_TYPE_INVALID_MSG)
	}

	// Create attribute definition entity
	attribute := &entity.AttributeDefinition{
		Key:           req.Key,
		Name:          req.Name,
		DataType:      req.DataType,
		Unit:          req.Unit,
		Description:   req.Description,
		AllowedValues: req.AllowedValues,
		IsActive:      true,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
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
func (s *AttributeDefinitionServiceImpl) UpdateAttribute(id uint, req model.AttributeDefinitionUpdateRequest) (*model.AttributeDefinitionResponse, error) {
	// Find existing attribute
	attribute, err := s.attributeRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Validate data type
	if !s.isValidDataType(req.DataType) {
		return nil, errors.New(utils.ATTRIBUTE_DATA_TYPE_INVALID_MSG)
	}

	// Update attribute fields
	attribute.Name = req.Name
	attribute.DataType = req.DataType
	attribute.Unit = req.Unit
	attribute.Description = req.Description
	attribute.AllowedValues = req.AllowedValues
	attribute.IsActive = req.IsActive
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
func (s *AttributeDefinitionServiceImpl) GetAttributeByID(id uint) (*model.AttributeDefinitionResponse, error) {
	attribute, err := s.attributeRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	attributeResponse := utils.ConvertAttributeDefinitionToResponse(attribute)
	return attributeResponse, nil
}

// GetAttributeByKey gets an attribute definition by key
func (s *AttributeDefinitionServiceImpl) GetAttributeByKey(key string) (*model.AttributeDefinitionResponse, error) {
	attribute, err := s.attributeRepo.FindByKey(key)
	if err != nil {
		return nil, err
	}
	if attribute == nil {
		return nil, errors.New(utils.ATTRIBUTE_DEFINITION_NOT_FOUND_MSG)
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
