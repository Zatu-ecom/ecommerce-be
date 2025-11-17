package factory

import (
	"time"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// CreateFromRequest creates an AttributeDefinition entity from a create request
func CreateFromRequest(
	req model.AttributeDefinitionCreateRequest,
) *entity.AttributeDefinition {
	return &entity.AttributeDefinition{
		Key:           req.Key,
		Name:          req.Name,
		Unit:          req.Unit,
		AllowedValues: req.AllowedValues,
		BaseEntity:    helper.NewBaseEntity(),
	}
}

// UpdateEntity updates an existing AttributeDefinition entity from an update request
func UpdateEntity(
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
func CreateCategoryAttributeFromConfig(
	categoryID uint,
	config model.CategoryAttributeConfig,
) *entity.CategoryAttribute {
	return &entity.CategoryAttribute{
		CategoryID:            categoryID,
		AttributeDefinitionID: config.AttributeDefinitionID,
		BaseEntity:            helper.NewBaseEntity(),
	}
}

// BuildAttributeResponse builds AttributeDefinitionResponse from entity
func BuildAttributeResponse(
	attribute *entity.AttributeDefinition,
) *model.AttributeDefinitionResponse {
	return &model.AttributeDefinitionResponse{
		ID:            attribute.ID,
		Key:           attribute.Key,
		Name:          attribute.Name,
		Unit:          attribute.Unit,
		AllowedValues: attribute.AllowedValues,
		CreatedAt:     helper.FormatTimestamp(attribute.CreatedAt),
	}
}

// BuildProductAttributeResponse builds ProductAttributeResponse from entity
func BuildProductAttributeResponse(
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
func BuildProductAttributesResponse(
	productAttributes []entity.ProductAttribute,
) []model.ProductAttributeResponse {
	responses := make([]model.ProductAttributeResponse, 0, len(productAttributes))
	for _, productAttribute := range productAttributes {
		responses = append(responses, *BuildProductAttributeResponse(&productAttribute))
	}
	return responses
}
