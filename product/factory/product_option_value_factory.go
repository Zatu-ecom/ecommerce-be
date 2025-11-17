package factory

import (
	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// ProductOptionValueFactory handles the creation of product option value entities from requests
// Stateless factory - all methods are pure functions

// CreateOptionValueFromRequest creates a ProductOptionValue entity from a request
func CreateOptionValueFromRequest(
	optionID uint,
	req model.ProductOptionValueRequest,
) *entity.ProductOptionValue {
	return &entity.ProductOptionValue{
		OptionID:    optionID,
		Value:       helper.ToLowerTrimmed(req.Value),
		DisplayName: req.DisplayName,
		ColorCode:   req.ColorCode,
		Position:    req.Position,
	}
}

// CreateOptionValuesFromRequests creates multiple ProductOptionValue entities from requests
func CreateOptionValuesFromRequests(
	optionID uint,
	requests []model.ProductOptionValueRequest,
) []entity.ProductOptionValue {
	if len(requests) == 0 {
		return nil
	}

	optionValues := make([]entity.ProductOptionValue, 0, len(requests))
	for j, req := range requests {
		optionValue := CreateOptionValueFromRequest(optionID, req)
		optionValue.Position = helper.GetPositionOrDefault(req.Position, j+1)
		optionValues = append(optionValues, *optionValue)
	}
	return optionValues
}

// UpdateOptionValueEntity updates an existing ProductOptionValue entity from an update request
func UpdateOptionValueEntity(
	optionValue *entity.ProductOptionValue,
	req model.ProductOptionValueUpdateRequest,
) *entity.ProductOptionValue {
	if req.DisplayName != nil {
		optionValue.DisplayName = *req.DisplayName
	}
	if req.ColorCode != nil {
		optionValue.ColorCode = *req.ColorCode
	}
	if req.Position != nil {
		optionValue.Position = *req.Position
	}
	return optionValue
}

// BuildProductOptionValueResponse builds ProductOptionValueResponse from entity
func BuildProductOptionValueResponse(
	value *entity.ProductOptionValue,
) *model.ProductOptionValueResponse {
	return &model.ProductOptionValueResponse{
		ID:          value.ID,
		OptionID:    value.OptionID,
		Value:       value.Value,
		DisplayName: value.DisplayName,
		ColorCode:   value.ColorCode,
		Position:    value.Position,
		CreatedAt:   helper.FormatTimestamp(value.CreatedAt),
		UpdatedAt:   helper.FormatTimestamp(value.UpdatedAt),
	}
}
