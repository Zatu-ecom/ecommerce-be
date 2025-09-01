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

	// Create response
	productResponse := &model.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         model.CategoryInfo{ID: category.ID, Name: category.Name},
		Brand:            product.Brand,
		SKU:              product.SKU,
		Price:            product.Price,
		Currency:         product.Currency,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Images:           product.Images,
		InStock:          product.InStock,
		IsPopular:        product.IsPopular,
		IsActive:         product.IsActive,
		Discount:         product.Discount,
		Tags:             product.Tags,
		Attributes:       make(map[string]string),
		PackageOptions:   []model.PackageOptionResponse{},
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}

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

	// Create response
	productResponse := &model.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         model.CategoryInfo{ID: category.ID, Name: category.Name},
		Brand:            product.Brand,
		SKU:              product.SKU,
		Price:            product.Price,
		Currency:         product.Currency,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Images:           product.Images,
		InStock:          product.InStock,
		IsPopular:        product.IsPopular,
		IsActive:         product.IsActive,
		Discount:         product.Discount,
		Tags:             product.Tags,
		Attributes:       make(map[string]string),
		PackageOptions:   []model.PackageOptionResponse{},
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}

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
		productResponse := s.convertProductToResponse(product)
		productsResponse = append(productsResponse, productResponse)
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

	var categoryInfo model.CategoryHierarchyInfo
	if category.ParentID != 0 {
		parentCategory, err := s.categoryRepo.FindByID(category.ParentID)
		if err == nil && parentCategory != nil {
			categoryInfo = model.CategoryHierarchyInfo{
				ID:     category.ID,
				Name:   category.Name,
				Parent: &model.CategoryInfo{ID: parentCategory.ID, Name: parentCategory.Name},
			}
		}
	} else {
		categoryInfo = model.CategoryHierarchyInfo{
			ID:     category.ID,
			Name:   category.Name,
			Parent: nil,
		}
	}

	// Create detailed response
	productDetailResponse := &model.ProductDetailResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         categoryInfo,
		Brand:            product.Brand,
		SKU:              product.SKU,
		Price:            product.Price,
		Currency:         product.Currency,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Images:           product.Images,
		InStock:          product.InStock,
		IsPopular:        product.IsPopular,
		IsActive:         product.IsActive,
		Discount:         product.Discount,
		Tags:             product.Tags,
		Attributes:       []model.ProductAttributeResponse{},
		PackageOptions:   []model.PackageOptionResponse{},
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}

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
		searchResult := model.SearchResult{
			ID:               product.ID,
			Name:             product.Name,
			Price:            product.Price,
			ShortDescription: product.ShortDescription,
			Images:           product.Images,
			RelevanceScore:   0.8,                             // Placeholder - implement actual relevance scoring
			MatchedFields:    []string{"name", "description"}, // Placeholder
		}
		searchResults = append(searchResults, searchResult)
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
		relatedProductResponse := model.RelatedProductResponse{
			ID:               relatedProduct.ID,
			Name:             relatedProduct.Name,
			Price:            relatedProduct.Price,
			ShortDescription: relatedProduct.ShortDescription,
			Images:           relatedProduct.Images,
			RelationReason:   "Same category",
		}
		relatedProductsResponse = append(relatedProductsResponse, relatedProductResponse)
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

// convertProductToResponse converts a product entity to response model
func (s *ProductServiceImpl) convertProductToResponse(product entity.Product) model.ProductResponse {
	// Get category info
	var categoryInfo model.CategoryInfo
	if product.Category.ID != 0 {
		categoryInfo = model.CategoryInfo{
			ID:   product.Category.ID,
			Name: product.Category.Name,
		}
	}

	return model.ProductResponse{
		ID:               product.ID,
		Name:             product.Name,
		CategoryID:       product.CategoryID,
		Category:         categoryInfo,
		Brand:            product.Brand,
		SKU:              product.SKU,
		Price:            product.Price,
		Currency:         product.Currency,
		ShortDescription: product.ShortDescription,
		LongDescription:  product.LongDescription,
		Images:           product.Images,
		InStock:          product.InStock,
		IsPopular:        product.IsPopular,
		IsActive:         product.IsActive,
		Discount:         product.Discount,
		Tags:             product.Tags,
		Attributes:       make(map[string]string),
		PackageOptions:   []model.PackageOptionResponse{},
		CreatedAt:        product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        product.UpdatedAt.Format(time.RFC3339),
	}
}
