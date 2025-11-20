package factory

import (
	"ecommerce-be/product/entity"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// VariantFactory handles the creation of product variant entities from requests
// Stateless factory - all methods are pure functions

// CreateVariantFromRequest creates a ProductVariant entity from a create request
func CreateVariantFromRequest(
	productID uint,
	req *model.CreateVariantRequest,
) *entity.ProductVariant {
	return &entity.ProductVariant{
		ProductID:     productID,
		SKU:           req.SKU,
		Price:         req.Price,
		Images:        req.Images,
		AllowPurchase: helper.GetBoolOrDefault(req.AllowPurchase, true),
		IsPopular:     helper.GetBoolOrDefault(req.IsPopular, false),
		IsDefault:     helper.GetBoolOrDefault(req.IsDefault, false),
	}
}

// UpdateVariantEntity updates an existing ProductVariant entity from an update request
func UpdateVariantEntity(
	variant *entity.ProductVariant,
	req *model.UpdateVariantRequest,
) *entity.ProductVariant {
	if req.SKU != nil {
		variant.SKU = *req.SKU
	}

	if req.Price != nil {
		variant.Price = *req.Price
	}

	if req.Images != nil {
		variant.Images = req.Images
	}

	if req.IsPopular != nil {
		variant.IsPopular = *req.IsPopular
	}

	if req.IsDefault != nil {
		variant.IsDefault = *req.IsDefault
	}

	// Apply AllowPurchase logic based on business rules:
	// - AllowPurchase is user-controlled, only apply if explicitly provided
	if req.AllowPurchase != nil {
		variant.AllowPurchase = *req.AllowPurchase
	}

	return variant
}

// BulkUpdateVariantEntity updates a variant entity from bulk update data
func BulkUpdateVariantEntity(
	variant *entity.ProductVariant,
	updateData *model.BulkUpdateVariantItem,
) *entity.ProductVariant {
	if updateData.SKU != nil {
		variant.SKU = *updateData.SKU
	}

	if updateData.Price != nil {
		variant.Price = *updateData.Price
	}

	if updateData.Images != nil {
		variant.Images = updateData.Images
	}

	if updateData.IsPopular != nil {
		variant.IsPopular = *updateData.IsPopular
	}

	if updateData.IsDefault != nil {
		variant.IsDefault = *updateData.IsDefault
	}

	// Apply AllowPurchase logic based on business rules:
	// - AllowPurchase is user-controlled, only apply if explicitly provided
	if updateData.AllowPurchase != nil {
		variant.AllowPurchase = *updateData.AllowPurchase
	}

	return variant
}

// CreateVariantOptionValue creates a VariantOptionValue entity
func CreateVariantOptionValue(
	variantID uint,
	optionID uint,
	optionValueID uint,
) entity.VariantOptionValue {
	return entity.VariantOptionValue{
		VariantID:     variantID,
		OptionID:      optionID,
		OptionValueID: optionValueID,
	}
}

// CreateVariantOptionValues creates multiple VariantOptionValue entities
func CreateVariantOptionValues(
	variantID uint,
	optionValueIDs map[uint]uint, // optionID -> optionValueID
) []entity.VariantOptionValue {
	variantOptionValues := make([]entity.VariantOptionValue, 0, len(optionValueIDs))

	for optionID, optionValueID := range optionValueIDs {
		variantOptionValues = append(variantOptionValues, entity.VariantOptionValue{
			VariantID:     variantID,
			OptionID:      optionID,
			OptionValueID: optionValueID,
		})
	}

	return variantOptionValues
}

/***********************************************
 *    Response Builders                         *
 ***********************************************/

// BuildVariantDetailResponse builds VariantDetailResponse from entities
func BuildVariantDetailResponse(
	variant *entity.ProductVariant,
	product *entity.Product,
	selectedOptions []model.VariantOptionResponse,
) *model.VariantDetailResponse {
	response := &model.VariantDetailResponse{
		ID:              variant.ID,
		ProductID:       variant.ProductID,
		SKU:             variant.SKU,
		Price:           variant.Price,
		AllowPurchase:   variant.AllowPurchase,
		Images:          helper.GetOrEmptySlice(variant.Images),
		IsDefault:       variant.IsDefault,
		IsPopular:       variant.IsPopular,
		SelectedOptions: selectedOptions,
		CreatedAt:       helper.FormatTimestamp(variant.CreatedAt),
		UpdatedAt:       helper.FormatTimestamp(variant.UpdatedAt),
	}

	// Add product basic info
	if product != nil {
		response.Product = model.ProductBasicInfo{
			ID:    product.ID,
			Name:  product.Name,
			Brand: product.Brand,
		}
	}

	return response
}

// BuildVariantResponse builds VariantResponse from entity
func BuildVariantResponse(
	variant *entity.ProductVariant,
	selectedOptions []model.VariantOptionResponse,
) *model.VariantResponse {
	return &model.VariantResponse{
		ID:              variant.ID,
		SKU:             variant.SKU,
		Price:           variant.Price,
		AllowPurchase:   variant.AllowPurchase,
		Images:          helper.GetOrEmptySlice(variant.Images),
		IsDefault:       variant.IsDefault,
		IsPopular:       variant.IsPopular,
		SelectedOptions: selectedOptions,
	}
}

// BuildVariantOptionResponses builds variant option responses from entities
func BuildVariantOptionResponses(
	variantOptionValues []entity.VariantOptionValue,
	productOptions []entity.ProductOption,
	optionValues []entity.ProductOptionValue,
) []model.VariantOptionResponse {
	optionResponses := []model.VariantOptionResponse{}

	// Create maps for quick lookup
	optionMap := make(map[uint]entity.ProductOption)
	for _, opt := range productOptions {
		optionMap[opt.ID] = opt
	}

	valueMap := make(map[uint]entity.ProductOptionValue)
	for _, val := range optionValues {
		valueMap[val.ID] = val
	}

	for _, vov := range variantOptionValues {
		option, optionExists := optionMap[vov.OptionID]
		value, valueExists := valueMap[vov.OptionValueID]

		if optionExists && valueExists {
			optionResponse := model.VariantOptionResponse{
				OptionID:          option.ID,
				OptionName:        option.Name,
				OptionDisplayName: helper.GetDisplayNameOrDefault(option.DisplayName, option.Name),
				ValueID:           value.ID,
				Value:             value.Value,
				ValueDisplayName:  helper.GetDisplayNameOrDefault(value.DisplayName, value.Value),
			}

			// Add color code if it exists
			if value.ColorCode != "" {
				optionResponse.ColorCode = value.ColorCode
			}

			optionResponses = append(optionResponses, optionResponse)
		}
	}

	return optionResponses
}

// BuildVariantDetailResponseFromMapper builds VariantDetailResponse from mapper.VariantWithOptions
func BuildVariantDetailResponseFromMapper(
	vwo *mapper.VariantWithOptions,
) *model.VariantDetailResponse {
	// Convert mapper.SelectedOptionValue to model.VariantOptionResponse
	selectedOptions := make([]model.VariantOptionResponse, 0, len(vwo.SelectedOptions))
	for _, selOpt := range vwo.SelectedOptions {
		selectedOptions = append(selectedOptions, model.VariantOptionResponse{
			OptionID:          selOpt.OptionID,
			OptionName:        selOpt.OptionName,
			OptionDisplayName: selOpt.OptionDisplayName,
			ValueID:           selOpt.ValueID,
			Value:             selOpt.Value,
			ValueDisplayName:  selOpt.ValueDisplayName,
			ColorCode:         selOpt.ColorCode,
		})
	}

	// Use existing BuildVariantDetailResponse to build the response
	return BuildVariantDetailResponse(&vwo.Variant, nil, selectedOptions)
}

// BuildVariantsDetailResponseFromMapper builds multiple VariantDetailResponse from mapper data
func BuildVariantsDetailResponseFromMapper(
	variantsWithOptions []mapper.VariantWithOptions,
) []model.VariantDetailResponse {
	result := make([]model.VariantDetailResponse, 0, len(variantsWithOptions))
	for i := range variantsWithOptions {
		result = append(result, *BuildVariantDetailResponseFromMapper(&variantsWithOptions[i]))
	}
	return result
}

// BuildVariantOptionResponsesFromAvailableOptions builds variant option responses from variant option values
// and available options response. This is optimized for the CreateVariant flow where we already have
// the GetAvailableOptionsResponse structure.
func BuildVariantOptionResponsesFromAvailableOptions(
	variantOptionValues []entity.VariantOptionValue,
	optionsResponse *model.GetAvailableOptionsResponse,
) []model.VariantOptionResponse {
	// Create lookup maps for O(1) access
	optionMap := make(map[uint]model.ProductOptionDetailResponse)
	valueMap := make(map[uint]model.OptionValueResponse)

	for _, opt := range optionsResponse.Options {
		optionMap[opt.OptionID] = opt
		for _, val := range opt.Values {
			valueMap[val.ValueID] = val
		}
	}

	// Build selected options responses
	selectedOptions := make([]model.VariantOptionResponse, 0, len(variantOptionValues))
	for _, vov := range variantOptionValues {
		opt, optExists := optionMap[vov.OptionID]
		val, valExists := valueMap[vov.OptionValueID]

		if optExists && valExists {
			selectedOptions = append(selectedOptions, model.VariantOptionResponse{
				OptionID:          opt.OptionID,
				OptionName:        opt.OptionName,
				OptionDisplayName: opt.OptionDisplayName,
				ValueID:           val.ValueID,
				Value:             val.Value,
				ValueDisplayName:  val.DisplayName,
				ColorCode:         val.ColorCode,
			})
		}
	}

	return selectedOptions
}
