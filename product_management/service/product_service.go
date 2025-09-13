package service

import (
	"errors"
	"math"
	"time"

	"ecommerce-be/common"
	commonEntity "ecommerce-be/common/entity"
	"ecommerce-be/product_management/entity"
	"ecommerce-be/product_management/model"
	"ecommerce-be/product_management/repositories"
	"ecommerce-be/product_management/utils"

	"gorm.io/gorm"
)

// ProductService defines the interface for product-related business logic
type ProductService interface {
	CreateProduct(req model.ProductCreateRequest) (*model.ProductResponse, error)
	UpdateProduct(id uint, req model.ProductUpdateRequest) (*model.ProductResponse, error)
	DeleteProduct(id uint) error
	GetAllProducts(page, limit int, filters map[string]interface{}) (*model.ProductsResponse, error)
	GetProductByID(id uint) (*model.ProductResponse, error)
	UpdateProductStock(id uint, req model.ProductStockUpdateRequest) error
	SearchProducts(
		query string,
		filters map[string]interface{},
		page, limit int,
	) (*model.SearchResponse, error)
	GetProductFilters(categoryID *uint) (*model.ProductFilters, error)
	GetRelatedProducts(productID uint, limit int) (*model.RelatedProductsResponse, error)
}

// ProductServiceImpl implements the ProductService interface
type ProductServiceImpl struct {
	productRepo   repositories.ProductRepository
	categoryRepo  repositories.CategoryRepository
	attributeRepo repositories.AttributeDefinitionRepository
}

// NewProductService creates a new instance of ProductService
func NewProductService(
	productRepo repositories.ProductRepository,
	categoryRepo repositories.CategoryRepository,
	attributeRepo repositories.AttributeDefinitionRepository,
) ProductService {
	return &ProductServiceImpl{
		productRepo:   productRepo,
		categoryRepo:  categoryRepo,
		attributeRepo: attributeRepo,
	}
}

/***********************************************
 *    CreateProduct creates a new product      *
 ***********************************************/
func (s *ProductServiceImpl) CreateProduct(
	req model.ProductCreateRequest,
) (*model.ProductResponse, error) {
	var product *entity.Product
	var attributes []*entity.ProductAttribute
	var packageOptions []entity.PackageOption

	err := common.Atomic(func(tx *gorm.DB) error {
		// Validate request
		if err := s.validateProductCreateRequest(req); err != nil {
			return err
		}

		product = utils.ConvertProductCreateRequestToEntity(req)
		if err := s.productRepo.Create(product); err != nil {
			return err
		}

		attrs, err := s.createProductAttributes(product.ID, req.Attributes)
		if err != nil {
			return err
		}
		attributes = attrs

		opts, err := s.createPackageOption(product.ID, req.PackageOptions)
		if err != nil {
			return err
		}
		packageOptions = opts

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.buildProductCreateResponse(product, req.CategoryID, attributes, packageOptions), nil
}

// Helper to validate product creation request
func (s *ProductServiceImpl) validateProductCreateRequest(req model.ProductCreateRequest) error {
	existingProduct, err := s.productRepo.FindBySKU(req.SKU)
	if err != nil {
		return err
	}
	if existingProduct != nil {
		return errors.New(utils.PRODUCT_EXISTS_MSG)
	}
	category, err := s.categoryRepo.FindByID(req.CategoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return errors.New(utils.PRODUCT_CATEGORY_INVALID_MSG)
	}
	return nil
}

// Helper to build product response
func (s *ProductServiceImpl) buildProductCreateResponse(
	product *entity.Product,
	categoryID uint,
	attributes []*entity.ProductAttribute,
	packageOptions []entity.PackageOption,
) *model.ProductResponse {
	category, _ := s.categoryRepo.FindByID(categoryID)
	categoryInfo := model.CategoryHierarchyInfo{ID: category.ID, Name: category.Name}
	attributeResponses := utils.ConvertProductAttributesEntityToResponse(
		flattenAttributes(attributes),
	)
	packageOptionResponses := utils.ConvertPackageOptionsEntityToResponse(packageOptions)
	return utils.ConvertProductResponse(
		product,
		categoryInfo,
		attributeResponses,
		packageOptionResponses,
	)
}

// Helper to flatten []*entity.ProductAttribute to []entity.ProductAttribute
func flattenAttributes(attrs []*entity.ProductAttribute) []entity.ProductAttribute {
	var result []entity.ProductAttribute
	for _, attr := range attrs {
		result = append(result, *attr)
	}
	return result
}

// createProductAttributes creates product attributes for a given product
func (s *ProductServiceImpl) createProductAttributes(
	productID uint,
	attributes []model.ProductAttributeRequest,
) ([]*entity.ProductAttribute, error) {
	// Extract unique keys and fetch existing attributes
	keys := s.extractUniqueKeys(attributes)
	attributeMap, err := s.attributeRepo.FindByKeys(keys)
	if err != nil {
		return nil, err
	}

	// Process attributes and prepare bulk operations
	operations := s.processAttributesForBulkOperations(productID, attributes, attributeMap)

	// Execute all bulk operations
	if err = s.executeBulkOperations(operations); err != nil {
		return nil, err
	}

	return operations.productAttributesToCreate, nil
}

// extractUniqueKeys extracts unique keys from attribute requests
func (s *ProductServiceImpl) extractUniqueKeys(
	attributes []model.ProductAttributeRequest,
) []string {
	keys := make([]string, 0, len(attributes))
	keySet := make(map[string]bool)
	for _, attr := range attributes {
		if !keySet[attr.Key] {
			keys = append(keys, attr.Key)
			keySet[attr.Key] = true
		}
	}
	return keys
}

// BulkOperations holds all bulk operations to be executed
type BulkOperations struct {
	attributesToUpdate        []*entity.AttributeDefinition
	attributesToCreate        []*entity.AttributeDefinition
	productAttributesToCreate []*entity.ProductAttribute
}

// processAttributesForBulkOperations processes attributes and prepares bulk operations
func (s *ProductServiceImpl) processAttributesForBulkOperations(
	productID uint,
	attributes []model.ProductAttributeRequest,
	attributeMap map[string]*entity.AttributeDefinition,
) *BulkOperations {
	operations := &BulkOperations{
		attributesToUpdate:        make([]*entity.AttributeDefinition, 0),
		attributesToCreate:        make([]*entity.AttributeDefinition, 0),
		productAttributesToCreate: make([]*entity.ProductAttribute, 0),
	}

	for _, attr := range attributes {
		attribute, exists := attributeMap[attr.Key]

		if exists {
			s.processExistingAttribute(attribute, attr.Value, operations)
		} else {
			attribute = s.createNewAttributeDefinition(attr, operations)
			attributeMap[attr.Key] = attribute
		}

		// Prepare product attribute for bulk creation
		productAttribute := &entity.ProductAttribute{
			ProductID:             productID,
			AttributeDefinitionID: attribute.ID,
			Value:                 attr.Value,
			SortOrder:             attr.SortOrder,
			AttributeDefinition:   attribute,
		}
		operations.productAttributesToCreate = append(
			operations.productAttributesToCreate,
			productAttribute,
		)
	}

	return operations
}

// processExistingAttribute processes an existing attribute definition
func (s *ProductServiceImpl) processExistingAttribute(
	attribute *entity.AttributeDefinition,
	value string,
	operations *BulkOperations,
) {
	// Check if value already exists using map for O(1) lookup
	valueMap := make(map[string]bool)
	for _, val := range attribute.AllowedValues {
		valueMap[val] = true
	}

	// Only add if value doesn't exist
	if !valueMap[value] {
		attribute.AllowedValues = append(attribute.AllowedValues, value)
		operations.attributesToUpdate = append(operations.attributesToUpdate, attribute)
	}
}

// createNewAttributeDefinition creates a new attribute definition
func (s *ProductServiceImpl) createNewAttributeDefinition(
	attr model.ProductAttributeRequest,
	operations *BulkOperations,
) *entity.AttributeDefinition {
	attribute := &entity.AttributeDefinition{
		Key:           attr.Key,
		Name:          attr.Name,
		Unit:          attr.Unit,
		AllowedValues: []string{attr.Value},
	}
	operations.attributesToCreate = append(operations.attributesToCreate, attribute)
	return attribute
}

// executeBulkOperations executes all bulk database operations
func (s *ProductServiceImpl) executeBulkOperations(operations *BulkOperations) error {
	// Bulk create new attribute definitions
	if len(operations.attributesToCreate) > 0 {
		if err := s.attributeRepo.CreateBulk(operations.attributesToCreate); err != nil {
			return err
		}
	}

	// Bulk update modified attributes
	if len(operations.attributesToUpdate) > 0 {
		if err := s.attributeRepo.UpdateBulk(operations.attributesToUpdate); err != nil {
			return err
		}
	}

	// Bulk create product attributes
	if len(operations.productAttributesToCreate) > 0 {
		if err := s.attributeRepo.CreateProductAttributesBulk(operations.productAttributesToCreate); err != nil {
			return err
		}
	}

	return nil
}

func (s *ProductServiceImpl) createPackageOption(
	parentID uint,
	options []model.PackageOptionRequest,
) ([]entity.PackageOption, error) {
	var packageOptions []entity.PackageOption
	for _, option := range options {
		packageOption := entity.PackageOption{
			Name:        option.Name,
			Description: option.Description,
			Price:       option.Price,
			Quantity:    option.Quantity,
			ProductID:   parentID,
			BaseEntity: commonEntity.BaseEntity{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		packageOptions = append(packageOptions, packageOption)
	}

	return packageOptions, s.productRepo.CreatePackageOptions(packageOptions)
}

/********************************************************
 *		UpdateProduct updates an existing product 		*
 ********************************************************/
func (s *ProductServiceImpl) UpdateProduct(
	id uint,
	req model.ProductUpdateRequest,
) (*model.ProductResponse, error) {
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Validate and update category
	if err := s.updateProductCategory(product, req.CategoryID); err != nil {
		return nil, err
	}

	// Update fields
	s.updateProductFields(product, req)
	product.UpdatedAt = time.Now()

	// Save updated product
	if err := s.productRepo.Update(product); err != nil {
		return nil, err
	}

	// Build response
	return s.buildProductResponse(product), nil
}

// Helper to validate and update category
func (s *ProductServiceImpl) updateProductCategory(product *entity.Product, categoryID uint) error {
	if categoryID != 0 {
		category, err := s.categoryRepo.FindByID(categoryID)
		if err != nil {
			return err
		}
		if category == nil {
			return errors.New(utils.PRODUCT_CATEGORY_INVALID_MSG)
		}
		product.CategoryID = categoryID
	}
	return nil
}

// Helper to update product fields
func (s *ProductServiceImpl) updateProductFields(
	product *entity.Product,
	req model.ProductUpdateRequest,
) {
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Brand != "" {
		product.Brand = req.Brand
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Currency != "" {
		product.Currency = req.Currency
	}
	if req.ShortDescription != "" {
		product.ShortDescription = req.ShortDescription
	}
	if req.LongDescription != "" {
		product.LongDescription = req.LongDescription
	}
	if len(req.Images) > 0 {
		product.Images = req.Images
	}
	product.IsPopular = req.IsPopular
	if req.Discount >= 0 {
		product.Discount = req.Discount
	}
	if len(req.Tags) > 0 {
		product.Tags = req.Tags
	}
}

// Helper to build response
func (s *ProductServiceImpl) buildProductResponse(product *entity.Product) *model.ProductResponse {
	category, _ := s.categoryRepo.FindByID(product.CategoryID)
	categoryInfo := model.CategoryHierarchyInfo{ID: category.ID, Name: category.Name}
	return utils.ConvertProductResponse(product, categoryInfo, nil, nil)
}

/**********************************************************
*                     Deletes a product                   *
***********************************************************/
func (s *ProductServiceImpl) DeleteProduct(id uint) error {
	return s.productRepo.Delete(id)
}

/*************************************************************************
*       GetAllProducts gets all products with pagination and filters     *
**************************************************************************/
func (s *ProductServiceImpl) GetAllProducts(
	page,
	limit int,
	filters map[string]interface{},
) (*model.ProductsResponse, error) {
	// Set default values
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	products, total, err := s.productRepo.FindAll(filters, page, limit)
	if err != nil {
		return nil, err
	}

	// Convert to response models
	var productsResponse []model.ProductResponse
	for _, product := range products {
		CategoryHierarchyInfo := model.CategoryHierarchyInfo{
			ID:   product.Category.ID,
			Name: product.Category.Name,
		}
		if product.Category.Parent != nil {
			CategoryHierarchyInfo.Parent = &model.CategoryInfo{
				ID:   product.Category.Parent.ID,
				Name: product.Category.Parent.Name,
			}
		}
		pr := utils.ConvertProductResponse(&product, CategoryHierarchyInfo, nil, nil)
		productsResponse = append(productsResponse, *pr)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	hasNext := page < totalPages
	hasPrev := page > 1

	pagination := model.PaginationResponse{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   int(total),
		ItemsPerPage: limit,
		HasNext:      hasNext,
		HasPrev:      hasPrev,
	}

	return &model.ProductsResponse{
		Products:   productsResponse,
		Pagination: pagination,
	}, nil
}

/*****************************************************************************
*        GetProductByID gets a product by ID with detailed information       *
******************************************************************************/
func (s *ProductServiceImpl) GetProductByID(id uint) (*model.ProductResponse, error) {
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Get category with parent info
	category, err := s.categoryRepo.FindByID(product.CategoryID)
	if err != nil {
		return nil, err
	}

	var parentCategory *entity.Category
	if category.ParentID != nil && *category.ParentID != 0 {
		if pc, err := s.categoryRepo.FindByID(*category.ParentID); err == nil {
			parentCategory = pc
		}
	}
	categoryInfo := utils.ConvertCategoryToHierarchyInfo(category, parentCategory)

	// Get product attributes
	productAttributes, err := s.attributeRepo.FindProductAttributeByProductID(id)
	var attribute []model.ProductAttributeResponse
	if err == nil {
		attribute = utils.ConvertProductAttributesEntityToResponse(productAttributes)
	}

	packageOptions, err := s.productRepo.FindPackageOptionByProductID(id)
	var packageOptionResponses []model.PackageOptionResponse
	if err == nil {
		packageOptionResponses = utils.ConvertPackageOptionsEntityToResponse(packageOptions)
	}

	// Create detailed response using converter
	productDetailResponse := utils.ConvertProductResponse(
		product,
		*categoryInfo,
		attribute,
		packageOptionResponses)
	return productDetailResponse, nil
}

/**********************************************************************
*      UpdateProductStock updates the stock status of a product       *
***********************************************************************/
func (s *ProductServiceImpl) UpdateProductStock(
	id uint,
	req model.ProductStockUpdateRequest,
) error {
	return s.productRepo.UpdateStock(id, req.InStock)
}

/**********************************************************************************
*     SearchProducts searches for products with the given query and filters       *
***********************************************************************************/
func (s *ProductServiceImpl) SearchProducts(
	query string,
	filters map[string]interface{},
	page, limit int,
) (*model.SearchResponse, error) {
	// Set default values
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	products, total, err := s.productRepo.Search(query, filters, page, limit)
	if err != nil {
		return nil, err
	}

	// Convert to search results
	var searchResults []model.SearchResult
	for _, product := range products {
		result := utils.ConvertProductToSearchResult(&product)
		searchResults = append(searchResults, *result)
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	pagination := model.PaginationResponse{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   int(total),
		ItemsPerPage: limit,
		HasNext:      page < totalPages,
		HasPrev:      page > 1,
	}

	return &model.SearchResponse{
		Query:      query,
		Results:    searchResults,
		Pagination: pagination,
		SearchTime: "0.05s", // Placeholder
	}, nil
}

/****************************************************************************
*        GetProductFilters gets available filters for product search		*
*****************************************************************************/
// TODO: Implement actual filter fetching logic
func (s *ProductServiceImpl) GetProductFilters(
	categoryID *uint,
) (*model.ProductFilters, error) {
	// This is a placeholder implementation
	// In a real implementation, you would query the database for actual filter data
	filters := &model.ProductFilters{
		Categories:  []model.CategoryFilter{},
		Brands:      []model.BrandFilter{},
		PriceRanges: []model.PriceRangeFilter{},
		Attributes:  []model.AttributeFilter{},
	}

	return filters, nil
}

/*****************************************************************************
*      GetRelatedProducts gets products related to a specific product        *
******************************************************************************/
func (s *ProductServiceImpl) GetRelatedProducts(
	productID uint,
	limit int,
) (*model.RelatedProductsResponse, error) {
	// Get the product to find its category
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}

	// Set default limit
	if limit < 1 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	// Find related products in the same category
	relatedProducts, err := s.productRepo.FindRelated(product.CategoryID, productID, limit)
	if err != nil {
		return nil, err
	}

	// Convert to response models
	var relatedProductsResponse []model.RelatedProductResponse
	for _, relatedProduct := range relatedProducts {
		r := utils.ConvertProductToRelatedProduct(&relatedProduct)
		relatedProductsResponse = append(relatedProductsResponse, *r)
	}

	return &model.RelatedProductsResponse{
		RelatedProducts: relatedProductsResponse,
	}, nil
}
