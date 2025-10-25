package factory

import (
	"time"

	commonEntity "ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
)

// AttributeFactory handles the creation of attribute entities from requests
type AttributeFactory struct{}

// NewAttributeFactory creates a new instance of AttributeFactory
func NewAttributeFactory() *AttributeFactory {
	return &AttributeFactory{}
}

// CreateFromRequest creates an AttributeDefinition entity from a create request
func (f *AttributeFactory) CreateFromRequest(
	req model.AttributeDefinitionCreateRequest,
) *entity.AttributeDefinition {
	now := time.Now()
	return &entity.AttributeDefinition{
		Key:           req.Key,
		Name:          req.Name,
		Unit:          req.Unit,
		AllowedValues: req.AllowedValues,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// UpdateEntity updates an existing AttributeDefinition entity from an update request
func (f *AttributeFactory) UpdateEntity(
	attribute *entity.AttributeDefinition,
	req model.AttributeDefinitionUpdateRequest,
) *entity.AttributeDefinition {
	attribute.Name = req.Name
	attribute.Unit = req.Unit
	attribute.AllowedValues = req.AllowedValues
	attribute.UpdatedAt = time.Now()

	return attribute
}

// CreateCategoryAttributeFromConfig creates a CategoryAttribute entity from config
func (f *AttributeFactory) CreateCategoryAttributeFromConfig(
	categoryID uint,
	config model.CategoryAttributeConfig,
) *entity.CategoryAttribute {
	now := time.Now()
	return &entity.CategoryAttribute{
		CategoryID:            categoryID,
		AttributeDefinitionID: config.AttributeDefinitionID,
		IsRequired:            config.IsRequired,
		IsSearchable:          config.IsSearchable,
		IsFilterable:          config.IsFilterable,
		DefaultValue:          config.DefaultValue,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// BuildAttributeResponse builds AttributeDefinitionResponse from entity
func (f *AttributeFactory) BuildAttributeResponse(
	attribute *entity.AttributeDefinition,
) *model.AttributeDefinitionResponse {
	return &model.AttributeDefinitionResponse{
		ID:            attribute.ID,
		Key:           attribute.Key,
		Name:          attribute.Name,
		Unit:          attribute.Unit,
		AllowedValues: attribute.AllowedValues,
		CreatedAt:     attribute.CreatedAt.Format(time.RFC3339),
	}
}

// BuildProductAttributeResponse builds ProductAttributeResponse from entity
func (f *AttributeFactory) BuildProductAttributeResponse(
	productAttribute *entity.ProductAttribute,
) *model.ProductAttributeResponse {
	return &model.ProductAttributeResponse{
		ID:        productAttribute.ID,
		Key:       productAttribute.AttributeDefinition.Key,
		Value:     productAttribute.Value,
		Name:      productAttribute.AttributeDefinition.Name,
		Unit:      productAttribute.AttributeDefinition.Unit,
		SortOrder: productAttribute.SortOrder,
	}
}

// BuildProductAttributesResponse builds multiple ProductAttributeResponse from entities
func (f *AttributeFactory) BuildProductAttributesResponse(
	productAttributes []entity.ProductAttribute,
) []model.ProductAttributeResponse {
	responses := make([]model.ProductAttributeResponse, 0, len(productAttributes))
	for _, productAttribute := range productAttributes {
		responses = append(responses, *f.BuildProductAttributeResponse(&productAttribute))
	}
	return responses
}
