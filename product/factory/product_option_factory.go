package factory

import (
	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// CreateOptionFromRequest creates a ProductOption entity from a create request
func CreateOptionFromRequest(
	productID uint,
	req model.ProductOptionCreateRequest,
) *entity.ProductOption {
	return &entity.ProductOption{
		ProductID:   productID,
		Name:        helper.NormalizeToSnakeCase(req.Name),
		DisplayName: req.DisplayName,
		Position:    req.Position,
	}
}

// UpdateOptionEntity updates an existing ProductOption entity from an update request
func UpdateOptionEntity(
	option *entity.ProductOption,
	req model.ProductOptionUpdateRequest,
) *entity.ProductOption {
	if req.DisplayName != nil {
		option.DisplayName = *req.DisplayName
	}
	if req.Position != nil {
		option.Position = *req.Position
	}
	return option
}

// BuildProductOptionResponse builds ProductOptionResponse from entity
func BuildProductOptionResponse(
	option *entity.ProductOption,
	productID uint,
) *model.ProductOptionResponse {
	response := &model.ProductOptionResponse{
		ID:          option.ID,
		ProductID:   productID,
		Name:        option.Name,
		DisplayName: option.DisplayName,
		Position:    option.Position,
		CreatedAt:   helper.FormatTimestamp(option.CreatedAt),
		UpdatedAt:   helper.FormatTimestamp(option.UpdatedAt),
	}

	// Convert values if present
	if len(option.Values) > 0 {
		values := make([]model.ProductOptionValueResponse, 0, len(option.Values))
		for _, val := range option.Values {
			valueResp := BuildProductOptionValueResponse(&val)
			values = append(values, *valueResp)
		}
		response.Values = values
	}

	return response
}

// BuildProductOptionDetailResponse builds ProductOptionDetailResponse from entity
func BuildProductOptionDetailResponse(
	option *entity.ProductOption,
	variantCountMp map[uint]int,
) *model.ProductOptionDetailResponse {
	response := &model.ProductOptionDetailResponse{
		OptionID:          option.ID,
		OptionName:        option.Name,
		OptionDisplayName: option.DisplayName,
		Position:          option.Position,
		Values:            make([]model.OptionValueResponse, 0, len(option.Values)),
	}

	// Convert option values
	for _, value := range option.Values {
		variantCount := 0
		if count, exists := variantCountMp[value.ID]; exists {
			variantCount = count
		}
		response.Values = append(response.Values, model.OptionValueResponse{
			ValueID:      value.ID,
			Value:        value.Value,
			DisplayName:  value.DisplayName,
			ColorCode:    value.ColorCode,
			Position:     value.Position,
			VariantCount: variantCount,
		})
	}

	return response
}

// BuildProductOptionsDetailResponse builds multiple ProductOptionDetailResponse from entities
func BuildProductOptionsDetailResponse(
	options []entity.ProductOption,
	variantCount map[uint]int,
) []model.ProductOptionDetailResponse {
	result := make([]model.ProductOptionDetailResponse, 0, len(options))
	for _, option := range options {
		result = append(result, *BuildProductOptionDetailResponse(&option, variantCount))
	}
	return result
}
