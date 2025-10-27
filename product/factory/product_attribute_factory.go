package factory

import (
	"time"

	commonEntity "ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
)

// ProductAttributeFactory handles the creation of product attribute entities and responses
type ProductAttributeFactory struct{}

// NewProductAttributeFactory creates a new instance of ProductAttributeFactory
func NewProductAttributeFactory() *ProductAttributeFactory {
	return &ProductAttributeFactory{}
}

// CreateFromRequest creates a ProductAttribute entity from an add request
func (f *ProductAttributeFactory) CreateFromRequest(
	productID uint,
	req model.AddProductAttributeRequest,
) *entity.ProductAttribute {
	now := time.Now()
	return &entity.ProductAttribute{
		ProductID:             productID,
		AttributeDefinitionID: req.AttributeDefinitionID,
		Value:                 req.Value,
		SortOrder:             req.SortOrder,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// UpdateEntity updates an existing ProductAttribute entity from an update request
func (f *ProductAttributeFactory) UpdateEntity(
	productAttribute *entity.ProductAttribute,
	req model.UpdateProductAttributeRequest,
) *entity.ProductAttribute {
	productAttribute.Value = req.Value
	productAttribute.SortOrder = req.SortOrder
	productAttribute.UpdatedAt = time.Now()

	return productAttribute
}

// BuildDetailResponse builds ProductAttributeDetailResponse from entity
func (f *ProductAttributeFactory) BuildDetailResponse(
	productAttribute *entity.ProductAttribute,
) *model.ProductAttributeDetailResponse {
	response := &model.ProductAttributeDetailResponse{
		ID:                    productAttribute.ID,
		ProductID:             productAttribute.ProductID,
		AttributeDefinitionID: productAttribute.AttributeDefinitionID,
		Value:                 productAttribute.Value,
		SortOrder:             productAttribute.SortOrder,
		CreatedAt:             productAttribute.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             productAttribute.UpdatedAt.Format(time.RFC3339),
	}

	// Add attribute definition details if loaded
	if productAttribute.AttributeDefinition != nil {
		response.AttributeKey = productAttribute.AttributeDefinition.Key
		response.AttributeName = productAttribute.AttributeDefinition.Name
		response.Unit = productAttribute.AttributeDefinition.Unit
	}

	return response
}

// BuildListResponse builds ProductAttributesListResponse from entities
func (f *ProductAttributeFactory) BuildListResponse(
	productID uint,
	productAttributes []entity.ProductAttribute,
) *model.ProductAttributesListResponse {
	attributes := make([]model.ProductAttributeDetailResponse, 0, len(productAttributes))

	for i := range productAttributes {
		attributes = append(attributes, *f.BuildDetailResponse(&productAttributes[i]))
	}

	return &model.ProductAttributesListResponse{
		ProductID:  productID,
		Attributes: attributes,
		Total:      len(attributes),
	}
}
