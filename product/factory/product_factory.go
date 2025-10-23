package factory

import (
	"time"

	"ecommerce-be/common/db"
	"ecommerce-be/product/entity"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
)

// ProductFactory handles creation and updates of product entities
type ProductFactory struct{}

// NewProductFactory creates a new product factory
func NewProductFactory() *ProductFactory {
	return &ProductFactory{}
}

// CreateProductFromRequest creates a new Product entity from a creation request
func (f *ProductFactory) CreateProductFromRequest(
	req model.ProductCreateRequest,
	sellerID uint,
) *entity.Product {
	now := time.Now()
	return &entity.Product{
		Name:             req.Name,
		CategoryID:       req.CategoryID,
		Brand:            req.Brand,
		BaseSKU:          req.BaseSKU,
		ShortDescription: req.ShortDescription,
		LongDescription:  req.LongDescription,
		Tags:             req.Tags,
		SellerID:         sellerID,
		BaseEntity: db.BaseEntity{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// UpdateProductEntity updates an existing Product entity with new data
func (f *ProductFactory) UpdateProductEntity(
	product *entity.Product,
	req model.ProductUpdateRequest,
) *entity.Product {
	// Update basic fields if provided
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Brand != "" {
		product.Brand = req.Brand
	}
	if req.CategoryID != 0 {
		product.CategoryID = req.CategoryID
	}
	if req.ShortDescription != "" {
		product.ShortDescription = req.ShortDescription
	}
	if req.LongDescription != "" {
		product.LongDescription = req.LongDescription
	}
	if len(req.Tags) > 0 {
		product.Tags = req.Tags
	}

	product.UpdatedAt = time.Now()
	return product
}

// CreateProductOptionsFromRequests creates ProductOption entities from requests
func (f *ProductFactory) CreateProductOptionsFromRequests(
	productID uint,
	optionReqs []model.ProductOptionCreateRequest,
) []*entity.ProductOption {
	options := make([]*entity.ProductOption, 0, len(optionReqs))
	
	for i, optionReq := range optionReqs {
		position := optionReq.Position
		if position == 0 {
			position = i + 1
		}

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

// CreateOptionValuesFromRequests creates ProductOptionValue entities from requests
func (f *ProductFactory) CreateOptionValuesFromRequests(
	optionID uint,
	valueReqs []model.ProductOptionValueRequest,
) []*entity.ProductOptionValue {
	values := make([]*entity.ProductOptionValue, 0, len(valueReqs))
	
	for j, valueReq := range valueReqs {
		position := valueReq.Position
		if position == 0 {
			position = j + 1
		}

		value := &entity.ProductOptionValue{
			OptionID:    optionID,
			Value:       valueReq.Value,
			DisplayName: valueReq.DisplayName,
			ColorCode:   valueReq.ColorCode,
			Position:    position,
		}
		values = append(values, value)
	}
	
	return values
}

// CreateProductAttributesFromRequests creates ProductAttribute entities from requests
func (f *ProductFactory) CreateProductAttributesFromRequests(
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
func (f *ProductFactory) CreateNewAttributeDefinition(
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
func (f *ProductFactory) UpdateAttributeDefinitionValues(
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

// CreatePackageOptionsFromRequests creates PackageOption entities from requests
func (f *ProductFactory) CreatePackageOptionsFromRequests(
	productID uint,
	options []model.PackageOptionRequest,
) []entity.PackageOption {
	packageOptions := make([]entity.PackageOption, 0, len(options))
	now := time.Now()
	
	for _, option := range options {
		packageOption := entity.PackageOption{
			Name:        option.Name,
			Description: option.Description,
			Price:       option.Price,
			Quantity:    option.Quantity,
			ProductID:   productID,
			BaseEntity: db.BaseEntity{
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		packageOptions = append(packageOptions, packageOption)
	}
	
	return packageOptions
}

// FlattenProductAttributes converts []*entity.ProductAttribute to []entity.ProductAttribute
func (f *ProductFactory) FlattenProductAttributes(attrs []*entity.ProductAttribute) []entity.ProductAttribute {
	result := make([]entity.ProductAttribute, 0, len(attrs))
	for _, attr := range attrs {
		result = append(result, *attr)
	}
	return result
}

/***********************************************
 *    Response Builders                         *
 ***********************************************/

// BuildPackageOptionResponse builds PackageOptionResponse from entity
func (f *ProductFactory) BuildPackageOptionResponse(
	packageOption *entity.PackageOption,
) *model.PackageOptionResponse {
	return &model.PackageOptionResponse{
		ID:          packageOption.ID,
		Name:        packageOption.Name,
		Description: packageOption.Description,
		Price:       packageOption.Price,
		Quantity:    packageOption.Quantity,
		CreatedAt:   packageOption.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   packageOption.UpdatedAt.Format(time.RFC3339),
	}
}

// BuildPackageOptionResponses builds multiple PackageOptionResponse from entities
func (f *ProductFactory) BuildPackageOptionResponses(
	packageOptions []entity.PackageOption,
) []model.PackageOptionResponse {
	responses := make([]model.PackageOptionResponse, 0, len(packageOptions))
	for _, option := range packageOptions {
		responses = append(responses, *f.BuildPackageOptionResponse(&option))
	}
	return responses
}

// BuildCategoryFilter builds CategoryFilter from mapper data
func (f *ProductFactory) BuildCategoryFilter(category mapper.CategoryWithProductCount) model.CategoryFilter {
	return model.CategoryFilter{
		ID:           category.CategoryID,
		Name:         category.CategoryName,
		ProductCount: category.ProductCount,
	}
}

// BuildBrandFilter builds BrandFilter from mapper data
func (f *ProductFactory) BuildBrandFilter(brand mapper.BrandWithProductCount) model.BrandFilter {
	return model.BrandFilter{
		Brand:        brand.Brand,
		ProductCount: brand.ProductCount,
	}
}

// BuildBrandFilters builds multiple BrandFilter from mapper data
func (f *ProductFactory) BuildBrandFilters(brands []mapper.BrandWithProductCount) []model.BrandFilter {
	filters := make([]model.BrandFilter, 0, len(brands))
	for _, brand := range brands {
		filters = append(filters, f.BuildBrandFilter(brand))
	}
	return filters
}

// BuildAttributeFilter builds AttributeFilter from mapper data
func (f *ProductFactory) BuildAttributeFilter(attribute mapper.AttributeWithProductCount) model.AttributeFilter {
	return model.AttributeFilter{
		Key:           attribute.Key,
		Name:          attribute.Name,
		AllowedValues: attribute.AllowedValues,
		ProductCount:  attribute.ProductCount,
	}
}

// BuildAttributeFilters builds multiple AttributeFilter from mapper data
func (f *ProductFactory) BuildAttributeFilters(attributes []mapper.AttributeWithProductCount) []model.AttributeFilter {
	filters := make([]model.AttributeFilter, 0, len(attributes))
	for _, attribute := range attributes {
		filters = append(filters, f.BuildAttributeFilter(attribute))
	}
	return filters
}

// BuildPriceRangeFilter builds PriceRangeFilter from mapper data
func (f *ProductFactory) BuildPriceRangeFilter(data *mapper.PriceRangeData) *model.PriceRangeFilter {
	if data == nil || data.ProductCount == 0 {
		return nil
	}
	return &model.PriceRangeFilter{
		Min:          data.MinPrice,
		Max:          data.MaxPrice,
		ProductCount: data.ProductCount,
	}
}

// BuildVariantTypeFilters builds VariantTypeFilter list from variant options data
func (f *ProductFactory) BuildVariantTypeFilters(variantOptions []mapper.VariantOptionData) []model.VariantTypeFilter {
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
		optionMap[vo.OptionID].Values = append(optionMap[vo.OptionID].Values, model.VariantOptionFilter{
			Value:        vo.OptionValue,
			DisplayName:  vo.ValueDisplayName,
			ColorCode:    vo.ColorCode,
			ProductCount: vo.ProductCount,
		})

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
func (f *ProductFactory) BuildStockStatusFilter(data *mapper.StockStatusData) *model.StockStatusFilter {
	if data == nil || data.TotalProducts == 0 {
		return nil
	}
	return &model.StockStatusFilter{
		InStock:       data.InStock,
		OutOfStock:    data.OutOfStock,
		TotalProducts: data.TotalProducts,
	}
}

// BuildCategoryHierarchyInfo builds CategoryHierarchyInfo from category and parent
func (f *ProductFactory) BuildCategoryHierarchyInfo(
	category *entity.Category,
	parentCategory *entity.Category,
) *model.CategoryHierarchyInfo {
	var parentInfo *model.CategoryInfo
	if parentCategory != nil {
		parentInfo = &model.CategoryInfo{
			ID:   parentCategory.ID,
			Name: parentCategory.Name,
		}
	}

	return &model.CategoryHierarchyInfo{
		ID:     category.ID,
		Name:   category.Name,
		Parent: parentInfo,
	}
}

// BuildProductAttributesResponse builds ProductAttributeResponse from ProductAttribute entities
func (f *ProductFactory) BuildProductAttributesResponse(
	productAttributes []entity.ProductAttribute,
) []model.ProductAttributeResponse {
	responses := make([]model.ProductAttributeResponse, 0, len(productAttributes))
	for _, attr := range productAttributes {
		responses = append(responses, model.ProductAttributeResponse{
			ID:        attr.ID,
			Key:       attr.AttributeDefinition.Key,
			Value:     attr.Value,
			Name:      attr.AttributeDefinition.Name,
			Unit:      attr.AttributeDefinition.Unit,
			SortOrder: attr.SortOrder,
		})
	}
	return responses
}
