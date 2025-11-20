package factory

import (
	"time"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// BuildProductAttributeFromCreateRequest creates a ProductAttribute entity from an add request
func BuildProductAttributeFromCreateRequest(
	productID uint,
	req model.AddProductAttributeRequest,
) *entity.ProductAttribute {
	return &entity.ProductAttribute{
		ProductID:             productID,
		AttributeDefinitionID: req.AttributeDefinitionID,
		Value:                 req.Value,
		SortOrder:             req.SortOrder,
		BaseEntity:            helper.NewBaseEntity(),
	}
}

// BuildProductAttributeFromUpdateRequest updates an existing ProductAttribute entity from an update request
func BuildProductAttributeFromUpdateRequest(
	productAttribute *entity.ProductAttribute,
	req model.UpdateProductAttributeRequest,
) *entity.ProductAttribute {
	productAttribute.Value = req.Value
	productAttribute.SortOrder = req.SortOrder
	productAttribute.UpdatedAt = time.Now()

	return productAttribute
}

// BuildProductAttributeDetailResponse builds ProductAttributeDetailResponse from entity
func BuildProductAttributeDetailResponse(
	productAttribute *entity.ProductAttribute,
) *model.ProductAttributeDetailResponse {
	response := &model.ProductAttributeDetailResponse{
		ID:                    productAttribute.ID,
		ProductID:             productAttribute.ProductID,
		AttributeDefinitionID: productAttribute.AttributeDefinitionID,
		Value:                 productAttribute.Value,
		SortOrder:             productAttribute.SortOrder,
		CreatedAt:             helper.FormatTimestamp(productAttribute.CreatedAt),
		UpdatedAt:             helper.FormatTimestamp(productAttribute.UpdatedAt),
	}

	// Add attribute definition details if loaded
	if productAttribute.AttributeDefinition != nil {
		response.AttributeKey = productAttribute.AttributeDefinition.Key
		response.AttributeName = productAttribute.AttributeDefinition.Name
		response.Unit = productAttribute.AttributeDefinition.Unit
	}

	return response
}

// BuildProductAttributesListResponse builds ProductAttributesListResponse from entities
func BuildProductAttributesListResponse(
	productID uint,
	productAttributes []entity.ProductAttribute,
) *model.ProductAttributesListResponse {
	attributes := make([]model.ProductAttributeDetailResponse, 0, len(productAttributes))

	for i := range productAttributes {
		attributes = append(
			attributes,
			*BuildProductAttributeDetailResponse(&productAttributes[i]),
		)
	}

	return &model.ProductAttributesListResponse{
		ProductID:  productID,
		Attributes: attributes,
		Total:      len(attributes),
	}
}

// ConvertDetailToSimpleAttributeResponse converts ProductAttributeDetailResponse to ProductAttributeResponse
func ConvertDetailToSimpleAttributeResponse(
	detail *model.ProductAttributeDetailResponse,
) model.ProductAttributeResponse {
	return model.ProductAttributeResponse{
		ID:        detail.ID,
		Key:       detail.AttributeKey,
		Value:     detail.Value,
		Name:      detail.AttributeName,
		Unit:      detail.Unit,
		SortOrder: detail.SortOrder,
	}
}

// ConvertDetailListToSimpleAttributeResponses converts list of ProductAttributeDetailResponse to ProductAttributeResponse
func ConvertDetailListToSimpleAttributeResponses(
	details []model.ProductAttributeDetailResponse,
) []model.ProductAttributeResponse {
	responses := make([]model.ProductAttributeResponse, 0, len(details))
	for i := range details {
		responses = append(responses, ConvertDetailToSimpleAttributeResponse(&details[i]))
	}
	return responses
}
