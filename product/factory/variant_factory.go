package factory

import (
	commonModel "ecommerce-be/common/model"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// VariantFactory handles the creation of product variant entities from requests
// Stateless factory - all methods are pure functions

// CreateVariantFromRequest creates a ProductVariant entity from a create request.
// The price is interpreted as major units in the seller's currency and converted
// to cents via CurrencyInfo (strict precision validation inside ToCents).
func CreateVariantFromRequest(
	productID uint,
	req *model.CreateVariantRequest,
	ccy commonModel.CurrencyInfo,
) (*entity.ProductVariant, error) {
	priceCents, err := ccy.ToCents(req.Price)
	if err != nil {
		return nil, err
	}
	return &entity.ProductVariant{
		ProductID:     productID,
		SKU:           req.SKU,
		PriceCents:    priceCents,
		AllowPurchase: helper.GetBoolOrDefault(req.AllowPurchase, true),
		IsPopular:     helper.GetBoolOrDefault(req.IsPopular, false),
		IsDefault:     helper.GetBoolOrDefault(req.IsDefault, false),
	}, nil
}

// UpdateVariantEntity updates an existing ProductVariant entity from an update request.
// When a new price is provided it is converted from major units to cents.
func UpdateVariantEntity(
	variant *entity.ProductVariant,
	req *model.UpdateVariantRequest,
	ccy commonModel.CurrencyInfo,
) error {
	if req.SKU != nil {
		variant.SKU = *req.SKU
	}

	if req.Price != nil {
		priceCents, err := ccy.ToCents(*req.Price)
		if err != nil {
			return err
		}
		variant.PriceCents = priceCents
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

	return nil
}

// BulkUpdateVariantEntity updates a variant entity from bulk update data.
// When a new price is provided it is converted from major units to cents.
func BulkUpdateVariantEntity(
	variant *entity.ProductVariant,
	updateData *model.BulkUpdateVariantItem,
	ccy commonModel.CurrencyInfo,
) error {
	if updateData.SKU != nil {
		variant.SKU = *updateData.SKU
	}

	if updateData.Price != nil {
		priceCents, err := ccy.ToCents(*updateData.Price)
		if err != nil {
			return err
		}
		variant.PriceCents = priceCents
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

	return nil
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

// BuildVariantDetailResponse builds VariantDetailResponse from entities.
// Media is always initialised as a non-nil slice; callers should overwrite it
// after resolving file URLs via VariantMediaService.GetMediaForVariants.
// Price is rendered as the shared Money contract in the seller's currency.
func BuildVariantDetailResponse(
	variant *entity.ProductVariant,
	product *entity.Product,
	selectedOptions []model.VariantOptionResponse,
	ccy commonModel.CurrencyInfo,
) *model.VariantDetailResponse {
	response := &model.VariantDetailResponse{
		ID:              variant.ID,
		ProductID:       variant.ProductID,
		SKU:             variant.SKU,
		Price:           commonModel.NewMoney(variant.PriceCents, ccy),
		Currency:        ccy,
		AllowPurchase:   variant.AllowPurchase,
		IsDefault:       variant.IsDefault,
		IsPopular:       variant.IsPopular,
		SelectedOptions: selectedOptions,
		Media:           []model.VariantMediaResponse{},
		CreatedAt:       helper.FormatTimestamp(variant.CreatedAt),
		UpdatedAt:       helper.FormatTimestamp(variant.UpdatedAt),
	}

	// Add product basic info
	if product != nil {
		response.Product = model.ProductBasicInfo{
			ID:         product.ID,
			Name:       product.Name,
			Brand:      product.Brand,
			CategoryID: product.CategoryID,
		}
	} else if variant.Product != nil {
		response.Product = model.ProductBasicInfo{
			ID:         variant.Product.ID,
			Name:       variant.Product.Name,
			Brand:      variant.Product.Brand,
			CategoryID: variant.Product.CategoryID,
		}
	}

	return response
}

// BuildVariantResponse builds VariantResponse from entity.
// Media is always initialised as a non-nil empty slice.
// Price is rendered as the shared Money contract in the seller's currency.
func BuildVariantResponse(
	variant *entity.ProductVariant,
	selectedOptions []model.VariantOptionResponse,
	ccy commonModel.CurrencyInfo,
) *model.VariantResponse {
	return &model.VariantResponse{
		ID:              variant.ID,
		SKU:             variant.SKU,
		Price:           commonModel.NewMoney(variant.PriceCents, ccy),
		Currency:        ccy,
		AllowPurchase:   variant.AllowPurchase,
		IsDefault:       variant.IsDefault,
		IsPopular:       variant.IsPopular,
		SelectedOptions: selectedOptions,
		Media:           []model.VariantMediaResponse{},
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
	ccy commonModel.CurrencyInfo,
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
	return BuildVariantDetailResponse(&vwo.Variant, nil, selectedOptions, ccy)
}

// BuildVariantsDetailResponseFromMapper builds multiple VariantDetailResponse from mapper data
func BuildVariantsDetailResponseFromMapper(
	variantsWithOptions []mapper.VariantWithOptions,
	ccy commonModel.CurrencyInfo,
) []model.VariantDetailResponse {
	result := make([]model.VariantDetailResponse, 0, len(variantsWithOptions))
	for i := range variantsWithOptions {
		result = append(result, *BuildVariantDetailResponseFromMapper(&variantsWithOptions[i], ccy))
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

// BuildVariantOptionResponsesFromMapper converts mapper.SelectedOptionValue to model.VariantOptionResponse
// This is used when we have data from mapper structures (e.g., from list queries)
func BuildVariantOptionResponsesFromMapper(
	selectedOptions []mapper.SelectedOptionValue,
) []model.VariantOptionResponse {
	responses := make([]model.VariantOptionResponse, 0, len(selectedOptions))

	for _, opt := range selectedOptions {
		responses = append(responses, model.VariantOptionResponse{
			OptionID:          opt.OptionID,
			OptionName:        opt.OptionName,
			OptionDisplayName: opt.OptionDisplayName,
			ValueID:           opt.ValueID,
			Value:             opt.Value,
			ValueDisplayName:  opt.ValueDisplayName,
			ColorCode:         opt.ColorCode,
		})
	}

	return responses
}
