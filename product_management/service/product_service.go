package service

import (
	"errors"
	"math"
	"time"

	commonEntity "ecommerce-be/common/entity"
	"ecommerce-be/product_management/entity"
	"ecommerce-be/product_management/model"
	"ecommerce-be/product_management/repositories"
	"ecommerce-be/product_management/utils"
)

// ProductService defines the interface for product-related business logic
type ProductService interface {
	CreateProduct(req model.ProductCreateRequest) (*model.ProductResponse, error)
	UpdateProduct(id uint, req model.ProductUpdateRequest) (*model.ProductResponse, error)
	DeleteProduct(id uint) error
	GetAllProducts(page, limit int, filters map[string]interface{}) (*model.ProductsResponse, error)
	GetProductByID(id uint) (*model.ProductDetailResponse, error)
	UpdateProductStock(id uint, req model.ProductStockUpdateRequest) error
	SearchProducts(query string, filters map[string]interface{}, page, limit int) (*model.SearchResponse, error)
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

// CreateProduct creates a new product
func (s *ProductServiceImpl) CreateProduct(req model.ProductCreateRequest) (*model.ProductResponse, error) {
	// Check if product with same SKU already exists
	existingProduct, err := s.productRepo.FindBySKU(req.SKU)
	if err != nil {
		return nil, err
	}
	if existingProduct != nil {
		return nil, errors.New(utils.PRODUCT_EXISTS_MSG)
	}

	// Validate category exists
	category, err := s.categoryRepo.FindByID(req.CategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, errors.New(utils.PRODUCT_CATEGORY_INVALID_MSG)
	}

	// Validate attributes based on category configuration
	if err := s.validateProductAttributes(req.CategoryID, req.Attributes); err != nil {
		return nil, err
	}

	// Set default currency if not provided
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	// Create product entity
	product := &entity.Product{
		Name:             req.Name,
		CategoryID:       req.CategoryID,
		Brand:            req.Brand,
		SKU:              req.SKU,
		Price:            req.Price,
		Currency:         currency,
		ShortDescription: req.ShortDescription,
		LongDescription:  req.LongDescription,
		Images:           req.Images,
		InStock:          true,
		IsPopular:        req.IsPopular,
		IsActive:         true,
		Discount:         req.Discount,
		Tags:             req.Tags,
		BaseEntity: commonEntity.BaseEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Save product to database
	if err := s.productRepo.Create(product); err != nil {
		return nil, err
	}

	// Build response using converter
	productResponse := utils.ConvertProductToResponse(product)
	// Ensure category info is set
	productResponse.Category = model.CategoryInfo{ID: category.ID, Name: category.Name}
	// Add attributes to response
	for _, attr := range req.Attributes {
		productResponse.Attributes[attr.Key] = attr.Value
	}

	return productResponse, nil
}

// UpdateProduct updates an existing product
func (s *ProductServiceImpl) UpdateProduct(id uint, req model.ProductUpdateRequest) (*model.ProductResponse, error) {
	// Find existing product
	product, err := s.productRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Validate category if provided
	if req.CategoryID != 0 {
		category, err := s.categoryRepo.FindByID(req.CategoryID)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, errors.New(utils.PRODUCT_CATEGORY_INVALID_MSG)
		}
		product.CategoryID = req.CategoryID
	}

	// Update product fields
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

	product.UpdatedAt = time.Now()

	// Save updated product
	if err := s.productRepo.Update(product); err != nil {
		return nil, err
	}

	// Get category info for response
	category, err := s.categoryRepo.FindByID(product.CategoryID)
	if err != nil {
		return nil, err
	}

	// Build response using converter
	productResponse := utils.ConvertProductToResponse(product)
	productResponse.Category = model.CategoryInfo{ID: category.ID, Name: category.Name}
	return productResponse, nil
}

// DeleteProduct soft deletes a product
func (s *ProductServiceImpl) DeleteProduct(id uint) error {
	return s.productRepo.SoftDelete(id)
}

// GetAllProducts gets all products with pagination and filters
func (s *ProductServiceImpl) GetAllProducts(page, limit int, filters map[string]interface{}) (*model.ProductsResponse, error) {
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
		pr := utils.ConvertProductToResponse(&product)
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

// GetProductByID gets a product by ID with detailed information
func (s *ProductServiceImpl) GetProductByID(id uint) (*model.ProductDetailResponse, error) {
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

	// Create detailed response using converter
	productDetailResponse := utils.ConvertProductToDetailResponse(product, *categoryInfo)
	return productDetailResponse, nil
}

// UpdateProductStock updates the stock status of a product
func (s *ProductServiceImpl) UpdateProductStock(id uint, req model.ProductStockUpdateRequest) error {
	return s.productRepo.UpdateStock(id, req.InStock)
}

// SearchProducts searches for products with the given query and filters
func (s *ProductServiceImpl) SearchProducts(query string, filters map[string]interface{}, page, limit int) (*model.SearchResponse, error) {
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

// GetProductFilters gets available filters for product search
func (s *ProductServiceImpl) GetProductFilters(categoryID *uint) (*model.ProductFilters, error) {
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

// GetRelatedProducts gets products related to a specific product
func (s *ProductServiceImpl) GetRelatedProducts(productID uint, limit int) (*model.RelatedProductsResponse, error) {
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

// validateProductAttributes validates product attributes based on category configuration
func (s *ProductServiceImpl) validateProductAttributes(categoryID uint, attributes []model.ProductAttributeRequest) error {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Get category attribute configuration
	// 2. Validate required attributes are present
	// 3. Validate attribute values against allowed values
	// 4. Validate data types
	return nil
}
