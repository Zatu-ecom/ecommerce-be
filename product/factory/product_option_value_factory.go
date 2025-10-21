package factory

import (
	"time"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils"
)

// ProductOptionValueFactory handles the creation of product option value entities from requests
type ProductOptionValueFactory struct{}

// NewProductOptionValueFactory creates a new instance of ProductOptionValueFactory
func NewProductOptionValueFactory() *ProductOptionValueFactory {
	return &ProductOptionValueFactory{}
}

// CreateOptionValueFromRequest creates a ProductOptionValue entity from a request
func (f *ProductOptionValueFactory) CreateOptionValueFromRequest(
	optionID uint,
	req model.ProductOptionValueRequest,
) *entity.ProductOptionValue {
	return &entity.ProductOptionValue{
		OptionID:    optionID,
		Value:       utils.ToLowerTrimmed(req.Value),
		DisplayName: req.DisplayName,
		ColorCode:   req.ColorCode,
		Position:    req.Position,
	}
}

// CreateOptionValuesFromRequests creates multiple ProductOptionValue entities from requests
func (f *ProductOptionValueFactory) CreateOptionValuesFromRequests(
	optionID uint,
	requests []model.ProductOptionValueRequest,
) []entity.ProductOptionValue {
	if len(requests) == 0 {
		return nil
	}

	optionValues := make([]entity.ProductOptionValue, 0, len(requests))
	for _, req := range requests {
		optionValue := f.CreateOptionValueFromRequest(optionID, req)
		optionValues = append(optionValues, *optionValue)
	}
	return optionValues
}

// UpdateOptionValueEntity updates an existing ProductOptionValue entity from an update request
func (f *ProductOptionValueFactory) UpdateOptionValueEntity(
	optionValue *entity.ProductOptionValue,
	req model.ProductOptionValueUpdateRequest,
) *entity.ProductOptionValue {
	if req.DisplayName != "" {
		optionValue.DisplayName = req.DisplayName
	}
	if req.ColorCode != "" {
		optionValue.ColorCode = req.ColorCode
	}
	if req.Position != 0 {
		optionValue.Position = req.Position
	}
	return optionValue
}

// BuildProductOptionValueResponse builds ProductOptionValueResponse from entity
func (f *ProductOptionValueFactory) BuildProductOptionValueResponse(
	value *entity.ProductOptionValue,
) *model.ProductOptionValueResponse {
	return &model.ProductOptionValueResponse{
		ID:          value.ID,
		OptionID:    value.OptionID,
		Value:       value.Value,
		DisplayName: value.DisplayName,
		ColorCode:   value.ColorCode,
		Position:    value.Position,
		CreatedAt:   value.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   value.UpdatedAt.Format(time.RFC3339),
	}
}
