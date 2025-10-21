package factory

import (
	"time"

	"ecommerce-be/product/entity"
	"ecommerce-be/product/model"
)

// VariantFactory handles the creation of product variant entities from requests
type VariantFactory struct{}

// NewVariantFactory creates a new instance of VariantFactory
func NewVariantFactory() *VariantFactory {
	return &VariantFactory{}
}

// CreateVariantFromRequest creates a ProductVariant entity from a create request
func (f *VariantFactory) CreateVariantFromRequest(
	productID uint,
	req *model.CreateVariantRequest,
) *entity.ProductVariant {
	// Set default values
	inStock := true
	if req.InStock != nil {
		inStock = *req.InStock
	}

	isPopular := false
	if req.IsPopular != nil {
		isPopular = *req.IsPopular
	}

	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	return &entity.ProductVariant{
		ProductID: productID,
		SKU:       req.SKU,
		Price:     req.Price,
		Stock:     req.Stock,
		Images:    req.Images,
		InStock:   inStock,
		IsPopular: isPopular,
		IsDefault: isDefault,
	}
}

// UpdateVariantEntity updates an existing ProductVariant entity from an update request
func (f *VariantFactory) UpdateVariantEntity(
	variant *entity.ProductVariant,
	req *model.UpdateVariantRequest,
) *entity.ProductVariant {
	if req.SKU != nil {
		variant.SKU = *req.SKU
	}

	if req.Price != nil {
		variant.Price = *req.Price
	}

	if req.Stock != nil {
		variant.Stock = *req.Stock
	}

	if req.Images != nil {
		variant.Images = req.Images
	}

	if req.InStock != nil {
		variant.InStock = *req.InStock
	}

	if req.IsPopular != nil {
		variant.IsPopular = *req.IsPopular
	}

	if req.IsDefault != nil {
		variant.IsDefault = *req.IsDefault
	}

	return variant
}

// ApplyStockOperation applies a stock operation to a variant
func (f *VariantFactory) ApplyStockOperation(
	variant *entity.ProductVariant,
	operation string,
	stock int,
) *entity.ProductVariant {
	switch operation {
	case "set":
		variant.Stock = stock
	case "add":
		variant.Stock += stock
	case "subtract":
		variant.Stock -= stock
	}

	// Update InStock status based on new stock value
	variant.InStock = variant.Stock > 0

	return variant
}

// BulkUpdateVariantEntity updates a variant entity from bulk update data
func (f *VariantFactory) BulkUpdateVariantEntity(
	variant *entity.ProductVariant,
	updateData *model.BulkUpdateVariantItem,
) *entity.ProductVariant {
	if updateData.SKU != nil {
		variant.SKU = *updateData.SKU
	}

	if updateData.Price != nil {
		variant.Price = *updateData.Price
	}

	if updateData.Stock != nil {
		variant.Stock = *updateData.Stock
	}

	if updateData.Images != nil {
		variant.Images = updateData.Images
	}

	if updateData.InStock != nil {
		variant.InStock = *updateData.InStock
	}

	if updateData.IsPopular != nil {
		variant.IsPopular = *updateData.IsPopular
	}

	if updateData.IsDefault != nil {
		variant.IsDefault = *updateData.IsDefault
	}

	return variant
}

// CreateVariantOptionValue creates a VariantOptionValue entity
func (f *VariantFactory) CreateVariantOptionValue(
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
func (f *VariantFactory) CreateVariantOptionValues(
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
func (f *VariantFactory) BuildVariantDetailResponse(
	variant *entity.ProductVariant,
	product *entity.Product,
	selectedOptions []model.VariantOptionResponse,
) *model.VariantDetailResponse {
	images := []string{}
	if variant.Images != nil {
		images = variant.Images
	}

	response := &model.VariantDetailResponse{
		ID:              variant.ID,
		ProductID:       variant.ProductID,
		SKU:             variant.SKU,
		Price:           variant.Price,
		Stock:           variant.Stock,
		InStock:         variant.InStock,
		Images:          images,
		IsDefault:       variant.IsDefault,
		IsPopular:       variant.IsPopular,
		SelectedOptions: selectedOptions,
		CreatedAt:       variant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       variant.UpdatedAt.Format(time.RFC3339),
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
func (f *VariantFactory) BuildVariantResponse(
	variant *entity.ProductVariant,
	selectedOptions []model.VariantOptionResponse,
) *model.VariantResponse {
	images := []string{}
	if variant.Images != nil {
		images = variant.Images
	}

	return &model.VariantResponse{
		ID:              variant.ID,
		SKU:             variant.SKU,
		Price:           variant.Price,
		Stock:           variant.Stock,
		InStock:         variant.InStock,
		Images:          images,
		IsDefault:       variant.IsDefault,
		IsPopular:       variant.IsPopular,
		SelectedOptions: selectedOptions,
	}
}

// BuildVariantOptionResponses builds variant option responses from entities
func (f *VariantFactory) BuildVariantOptionResponses(
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
				OptionDisplayName: f.getDisplayNameOrDefault(option.DisplayName, option.Name),
				ValueID:           value.ID,
				Value:             value.Value,
				ValueDisplayName:  f.getDisplayNameOrDefault(value.DisplayName, value.Value),
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

// getDisplayNameOrDefault returns display name if not empty, otherwise returns the name
func (f *VariantFactory) getDisplayNameOrDefault(displayName, name string) string {
	if displayName != "" {
		return displayName
	}
	return name
}
