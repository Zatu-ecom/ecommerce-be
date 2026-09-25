package factory

import (
	"time"

	commonModel "ecommerce-be/common/model"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	"ecommerce-be/product/utils/helper"
)

// ProductFactory handles creation and updates of product entities
// Stateless factory - all methods are pure functions

// CreateProductFromRequest creates a new Product entity from a creation request
func CreateProductFromRequest(
	req model.ProductCreateRequest,
	sellerID uint,
) *entity.Product {
	return &entity.Product{
		Name:             req.Name,
		CategoryID:       req.CategoryID,
		Brand:            req.Brand,
		BaseSKU:          req.BaseSKU,
		ShortDescription: req.ShortDescription,
		LongDescription:  req.LongDescription,
		Tags:             req.Tags,
		SellerID:         sellerID,
		BaseEntity:       helper.NewBaseEntity(),
	}
}

// CreateProductEntityFromUpdateRequest updates an existing Product entity with new data
// Uses pointer fields to distinguish between null (don't update) and empty (clear field)
func CreateProductEntityFromUpdateRequest(
	product *entity.Product,
	req model.ProductUpdateRequest,
) *entity.Product {
	// Update basic fields if provided (not nil)
	// If pointer is not nil, update even if value is empty string/zero
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Brand != nil {
		product.Brand = *req.Brand
	}
	if req.CategoryID != nil {
		product.CategoryID = *req.CategoryID
	}
	if req.ShortDescription != nil {
		product.ShortDescription = *req.ShortDescription
	}
	if req.LongDescription != nil {
		product.LongDescription = *req.LongDescription
	}
	if req.Tags != nil {
		product.Tags = *req.Tags
	}

	product.UpdatedAt = time.Now()
	return product
}

// CreateProductOptionsFromRequests creates ProductOption entities from requests
func CreateProductOptionsFromRequests(
	productID uint,
	optionReqs []model.ProductOptionCreateRequest,
) []*entity.ProductOption {
	options := make([]*entity.ProductOption, 0, len(optionReqs))

	for i, optionReq := range optionReqs {
		position := helper.GetPositionOrDefault(optionReq.Position, i+1)

		option := &entity.ProductOption{
			ProductID:   productID,
			Name:        optionReq.Name,
			DisplayName: optionReq.DisplayName,
			Position:    position,
		}
		options = append(options, option)
	}

	return options
}

// CreateProductAttributesFromRequests creates ProductAttribute entities from requests
func CreateProductAttributesFromRequests(
	productID uint,
	attributes []model.ProductAttributeRequest,
	attributeMap map[string]*entity.AttributeDefinition,
) []*entity.ProductAttribute {
	productAttributes := make([]*entity.ProductAttribute, 0, len(attributes))

	for _, attr := range attributes {
		attributeDefinition := attributeMap[attr.Key]

		productAttribute := &entity.ProductAttribute{
			ProductID:             productID,
			AttributeDefinitionID: attributeDefinition.ID,
			Value:                 attr.Value,
			SortOrder:             attr.SortOrder,
			AttributeDefinition:   attributeDefinition,
		}
		productAttributes = append(productAttributes, productAttribute)
	}

	return productAttributes
}

// CreateNewAttributeDefinition creates a new AttributeDefinition entity
func CreateNewAttributeDefinition(
	attr model.ProductAttributeRequest,
) *entity.AttributeDefinition {
	return &entity.AttributeDefinition{
		Key:           attr.Key,
		Name:          attr.Name,
		Unit:          attr.Unit,
		AllowedValues: []string{attr.Value},
	}
}

// UpdateAttributeDefinitionValues adds a new value to an existing AttributeDefinition
// Returns true if the value was added, false if it already existed
func UpdateAttributeDefinitionValues(
	attribute *entity.AttributeDefinition,
	newValue string,
) bool {
	// Check if value already exists using map for O(1) lookup
	valueMap := make(map[string]bool)
	for _, val := range attribute.AllowedValues {
		valueMap[val] = true
	}

	// Only add if value doesn't exist
	if !valueMap[newValue] {
		attribute.AllowedValues = append(attribute.AllowedValues, newValue)
		return true
	}
	return false
}

// CreatePackageOptionsFromRequests creates PackageOption entities from requests.
// Prices are converted from major units to cents via CurrencyInfo.
func CreatePackageOptionsFromRequests(
	productID uint,
	options []model.PackageOptionRequest,
	ccy commonModel.CurrencyInfo,
) ([]entity.PackageOption, error) {
	packageOptions := make([]entity.PackageOption, 0, len(options))

	for _, option := range options {
		priceCents, err := ccy.ToCents(option.Price)
		if err != nil {
			return nil, err
		}
		packageOption := entity.PackageOption{
			Name:        option.Name,
			Description: option.Description,
			PriceCents:  priceCents,
			Quantity:    option.Quantity,
			ProductID:   productID,
			BaseEntity:  helper.NewBaseEntity(),
		}
		packageOptions = append(packageOptions, packageOption)
	}

	return packageOptions, nil
}

// FlattenProductAttributes converts []*entity.ProductAttribute to []entity.ProductAttribute
func FlattenProductAttributes(
	attrs []*entity.ProductAttribute,
) []entity.ProductAttribute {
	result := make([]entity.ProductAttribute, 0, len(attrs))
	for _, attr := range attrs {
		result = append(result, *attr)
	}
	return result
}

/***********************************************
 *    Response Builders                         *
 ***********************************************/

// BuildPackageOptionResponse builds PackageOptionResponse from entity.
// Price is rendered as the shared Money contract in the seller's currency.
func BuildPackageOptionResponse(
	packageOption *entity.PackageOption,
	ccy commonModel.CurrencyInfo,
) *model.PackageOptionResponse {
	return &model.PackageOptionResponse{
		ID:          packageOption.ID,
		ProductID:   packageOption.ProductID,
		Name:        packageOption.Name,
		Description: packageOption.Description,
		Price:       commonModel.NewMoney(packageOption.PriceCents, ccy),
		Quantity:    packageOption.Quantity,
		CreatedAt:   helper.FormatTimestamp(packageOption.CreatedAt),
		UpdatedAt:   helper.FormatTimestamp(packageOption.UpdatedAt),
	}
}

// BuildPackageOptionResponses builds multiple PackageOptionResponse from entities
func BuildPackageOptionResponses(
	packageOptions []entity.PackageOption,
	ccy commonModel.CurrencyInfo,
) []model.PackageOptionResponse {
	responses := make([]model.PackageOptionResponse, 0, len(packageOptions))
	for _, option := range packageOptions {
		responses = append(responses, *BuildPackageOptionResponse(&option, ccy))
	}
	return responses
}

// BuildPackageOptionFromCreateRequest creates a PackageOption entity from a create request.
// Price is converted from major units to cents via CurrencyInfo.
func BuildPackageOptionFromCreateRequest(
	productID uint,
	req model.PackageOptionCreateRequest,
	ccy commonModel.CurrencyInfo,
) (*entity.PackageOption, error) {
	priceCents, err := ccy.ToCents(req.Price)
	if err != nil {
		return nil, err
	}
	return &entity.PackageOption{
		ProductID:   productID,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  priceCents,
		Quantity:    req.Quantity,
		BaseEntity:  helper.NewBaseEntity(),
	}, nil
}

// ApplyPackageOptionUpdate applies update request fields to a package option entity.
// When a new price is provided it is converted from major units to cents.
func ApplyPackageOptionUpdate(
	packageOption *entity.PackageOption,
	req model.PackageOptionUpdateRequest,
	ccy commonModel.CurrencyInfo,
) error {
	packageOption.Name = req.Name
	packageOption.Description = req.Description
	priceCents, err := ccy.ToCents(req.Price)
	if err != nil {
		return err
	}
	packageOption.PriceCents = priceCents
	packageOption.Quantity = req.Quantity
	return nil
}

// ApplyBulkPackageOptionUpdate applies bulk update item fields to a package option entity.
// When a new price is provided it is converted from major units to cents.
func ApplyBulkPackageOptionUpdate(
	packageOption *entity.PackageOption,
	item model.BulkUpdatePackageOptionItem,
	ccy commonModel.CurrencyInfo,
) error {
	packageOption.Name = item.Name
	packageOption.Description = item.Description
	priceCents, err := ccy.ToCents(item.Price)
	if err != nil {
		return err
	}
	packageOption.PriceCents = priceCents
	packageOption.Quantity = item.Quantity
	return nil
}

// BuildPackageOptionsListResponse builds the list response for package options
func BuildPackageOptionsListResponse(
	packageOptions []entity.PackageOption,
	ccy commonModel.CurrencyInfo,
) *model.PackageOptionsResponse {
	return &model.PackageOptionsResponse{
		PackageOptions: BuildPackageOptionResponses(packageOptions, ccy),
	}
}

// BuildCategoryFilter builds CategoryFilter from mapper data
func BuildCategoryFilter(
	category mapper.CategoryWithProductCount,
) model.CategoryFilter {
	return model.CategoryFilter{
		ID:           category.CategoryID,
		Name:         category.CategoryName,
		ProductCount: category.ProductCount,
	}
}

// BuildBrandFilter builds BrandFilter from mapper data
func BuildBrandFilter(brand mapper.BrandWithProductCount) model.BrandFilter {
	return model.BrandFilter{
		Brand:        brand.Brand,
		ProductCount: brand.ProductCount,
	}
}

// BuildBrandFilters builds multiple BrandFilter from mapper data
func BuildBrandFilters(
	brands []mapper.BrandWithProductCount,
) []model.BrandFilter {
	filters := make([]model.BrandFilter, 0, len(brands))
	for _, brand := range brands {
		filters = append(filters, BuildBrandFilter(brand))
	}
	return filters
}

// BuildAttributeFilter builds AttributeFilter from mapper data
func BuildAttributeFilter(
	attribute mapper.AttributeWithProductCount,
) model.AttributeFilter {
	return model.AttributeFilter{
		Key:           attribute.Key,
		Name:          attribute.Name,
		AllowedValues: attribute.AllowedValues,
		ProductCount:  attribute.ProductCount,
	}
}

// BuildAttributeFilters builds multiple AttributeFilter from mapper data
func BuildAttributeFilters(
	attributes []mapper.AttributeWithProductCount,
) []model.AttributeFilter {
	filters := make([]model.AttributeFilter, 0, len(attributes))
	for _, attribute := range attributes {
		filters = append(filters, BuildAttributeFilter(attribute))
	}
	return filters
}

// BuildPriceRangeFilter builds PriceRangeFilter from mapper data.
// PriceRangeData holds cents; the filter display uses major units in the
// seller's currency, so cents are converted via CurrencyInfo.
func BuildPriceRangeFilter(
	data *mapper.PriceRangeData,
	ccy commonModel.CurrencyInfo,
) *model.PriceRangeFilter {
	if data == nil || data.ProductCount == 0 {
		return nil
	}
	return &model.PriceRangeFilter{
		Min:          commonModel.FromCents(data.MinPriceCents, ccy),
		Max:          commonModel.FromCents(data.MaxPriceCents, ccy),
		ProductCount: data.ProductCount,
	}
}

// BuildVariantTypeFilters builds VariantTypeFilter list from variant options data
func BuildVariantTypeFilters(
	variantOptions []mapper.VariantOptionData,
) []model.VariantTypeFilter {
	// Group by option ID to create VariantTypeFilter
	optionMap := make(map[uint]*model.VariantTypeFilter)
	optionOrder := []uint{} // Preserve order

	for _, vo := range variantOptions {
		if _, exists := optionMap[vo.OptionID]; !exists {
			// First time seeing this option
			optionMap[vo.OptionID] = &model.VariantTypeFilter{
				Name:         vo.OptionName,
				DisplayName:  vo.OptionDisplayName,
				Values:       []model.VariantOptionFilter{},
				ProductCount: 0,
			}
			optionOrder = append(optionOrder, vo.OptionID)
		}

		// Add value to this option
		optionMap[vo.OptionID].Values = append(
			optionMap[vo.OptionID].Values,
			model.VariantOptionFilter{
				Value:        vo.OptionValue,
				DisplayName:  vo.ValueDisplayName,
				ColorCode:    vo.ColorCode,
				ProductCount: vo.ProductCount,
			},
		)

		// Update product count for this option type
		optionMap[vo.OptionID].ProductCount += vo.ProductCount
	}

	// Convert map to slice maintaining order
	result := make([]model.VariantTypeFilter, 0, len(optionMap))
	for _, optionID := range optionOrder {
		result = append(result, *optionMap[optionID])
	}

	return result
}

// BuildStockStatusFilter builds StockStatusFilter from mapper data
func BuildStockStatusFilter(
	data *mapper.StockStatusData,
) *model.StockStatusFilter {
	if data == nil || data.TotalProducts == 0 {
		return nil
	}
	return &model.StockStatusFilter{
		InStock:       data.InStock,
		OutOfStock:    data.OutOfStock,
		TotalProducts: data.TotalProducts,
	}
}

// BuildProductResponse builds a ProductResponse from product entity and variant aggregation
// Used for list views (GetAllProducts, GetRelatedProducts, etc.)
func BuildProductResponse(
	product *entity.Product,
	variantAgg *mapper.VariantAggregation,
	ccy commonModel.CurrencyInfo,
) model.ProductResponse {
	// Build category hierarchy using existing helper method
	categoryInfo := BuildCategoryHierarchyInfo(product.Category, product.Category.Parent)

	// Build base product response
	productResp := model.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         *categoryInfo,
		Brand:            product.Brand,
		SKU:              product.BaseSKU,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Tags:             product.Tags,
		SellerID:         product.SellerID,
		CreatedAt:        helper.FormatTimestamp(product.CreatedAt),
		UpdatedAt:        helper.FormatTimestamp(product.UpdatedAt),
	}

	ApplyCommerceFieldsFromAggregation(&productResp, variantAgg, ccy)

	return productResp
}

// ApplyCommerceFieldsFromAggregation sets listing commerce fields from variant aggregation.
// Prices are cents in the aggregation; Money is rendered via the seller's currency.
func ApplyCommerceFieldsFromAggregation(
	productResp *model.ProductResponse,
	variantAgg *mapper.VariantAggregation,
	ccy commonModel.CurrencyInfo,
) {
	if productResp == nil || variantAgg == nil {
		return
	}

	productResp.HasVariants = variantAgg.HasVariants
	productResp.Price = commonModel.NewMoney(variantAgg.DefaultPriceCents, ccy)
	productResp.Currency = ccy
	productResp.AllowPurchase = variantAgg.AllowPurchase
	productResp.IsPopular = variantAgg.IsPopular
	productResp.IsWishlisted = variantAgg.IsWishlisted
	productResp.VariantPreview = nil

	if variantAgg.DefaultPriceCents > 0 || variantAgg.HasVariants {
		productResp.PriceRange = &model.PriceRange{
			Min: commonModel.NewMoney(variantAgg.MinPriceCents, ccy),
			Max: commonModel.NewMoney(variantAgg.MaxPriceCents, ccy),
		}
	}

	if variantAgg.HasVariants && variantAgg.TotalVariants > 0 {
		variantPreview := &model.VariantPreview{
			TotalVariants: variantAgg.TotalVariants,
			Options:       []model.OptionPreview{},
		}

		for _, optionName := range variantAgg.OptionNames {
			optionValues := variantAgg.OptionValues[optionName]
			variantPreview.Options = append(variantPreview.Options, model.OptionPreview{
				Name:            optionName,
				DisplayName:     optionName,
				AvailableValues: optionValues,
			})
		}

		productResp.VariantPreview = variantPreview
	}
}

// BuildRelatedProductItemScored builds a RelatedProductItemScored from RelatedProductScored mapper
// Used for related products API with scoring information
func BuildRelatedProductItemScored(
	scoredResult *mapper.RelatedProductScored,
) model.RelatedProductItemScored {
	// Build category hierarchy info
	var parentInfo *model.CategoryInfo
	if scoredResult.ParentCategoryID != nil && scoredResult.ParentCategoryName != nil {
		parentInfo = &model.CategoryInfo{
			ID:   *scoredResult.ParentCategoryID,
			Name: *scoredResult.ParentCategoryName,
		}
	}

	categoryInfo := model.CategoryHierarchyInfo{
		ID:     scoredResult.CategoryID,
		Name:   scoredResult.CategoryName,
		Parent: parentInfo,
	}

	// Build base product response (commerce fields applied via ApplyCommerceFieldsFromAggregation)
	productResponse := model.ProductResponse{
		ID:               scoredResult.ProductID,
		Name:             scoredResult.ProductName,
		CategoryID:       scoredResult.CategoryID,
		Category:         categoryInfo,
		Brand:            scoredResult.Brand,
		SKU:              scoredResult.SKU,
		ShortDescription: scoredResult.ShortDescription,
		LongDescription:  scoredResult.LongDescription,
		Tags:             scoredResult.Tags,
		SellerID:         scoredResult.SellerID,
	}

	// Build scored item with relation metadata
	return model.RelatedProductItemScored{
		ProductResponse: productResponse,
		RelationReason:  scoredResult.RelationReason,
		Score:           scoredResult.FinalScore,
		StrategyUsed:    scoredResult.StrategyUsed,
	}
}

func BuildRelatedProductsScoredResponse(
	relatedItems []model.RelatedProductItemScored,
	strategiesUsed []string,
	avgScore float64,
	pagination model.PaginationResponse,
	totalStrategies int,
) *model.RelatedProductsScoredResponse {
	return &model.RelatedProductsScoredResponse{
		RelatedProducts: relatedItems,
		Pagination:      pagination,
		Meta: model.RelatedProductsMeta{
			StrategiesUsed:  strategiesUsed,
			AvgScore:        avgScore,
			TotalStrategies: totalStrategies,
		},
	}
}
