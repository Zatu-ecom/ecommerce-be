package factory

import (
	"time"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils"
)

// ProductOptionFactory handles the creation of product option entities from requests
type ProductOptionFactory struct{}

// NewProductOptionFactory creates a new instance of ProductOptionFactory
func NewProductOptionFactory() *ProductOptionFactory {
	return &ProductOptionFactory{}
}

// CreateOptionFromRequest creates a ProductOption entity from a create request
func (f *ProductOptionFactory) CreateOptionFromRequest(
	productID uint,
	req model.ProductOptionCreateRequest,
) *entity.ProductOption {
	return &entity.ProductOption{
		ProductID:   productID,
		Name:        utils.NormalizeToSnakeCase(req.Name),
		DisplayName: req.DisplayName,
		Position:    req.Position,
	}
}

// UpdateOptionEntity updates an existing ProductOption entity from an update request
func (f *ProductOptionFactory) UpdateOptionEntity(
	option *entity.ProductOption,
	req model.ProductOptionUpdateRequest,
) *entity.ProductOption {
	if req.DisplayName != "" {
		option.DisplayName = req.DisplayName
	}
	if req.Position != 0 || req.Position != option.Position {
		option.Position = req.Position
	}
	return option
}

// BuildProductOptionResponse builds ProductOptionResponse from entity
func (f *ProductOptionFactory) BuildProductOptionResponse(
	option *entity.ProductOption,
	productID uint,
) *model.ProductOptionResponse {
	response := &model.ProductOptionResponse{
		ID:          option.ID,
		ProductID:   productID,
		Name:        option.Name,
		DisplayName: option.DisplayName,
		Position:    option.Position,
		CreatedAt:   option.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   option.UpdatedAt.Format(time.RFC3339),
	}

	// Convert values if present
	if len(option.Values) > 0 {
		valueFactory := NewProductOptionValueFactory()
		values := make([]model.ProductOptionValueResponse, 0, len(option.Values))
		for _, val := range option.Values {
			valueResp := valueFactory.BuildProductOptionValueResponse(&val)
			values = append(values, *valueResp)
		}
		response.Values = values
	}

	return response
}
