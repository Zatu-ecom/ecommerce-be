package repositories

import (
	"errors"

	"ecommerce-be/product_management/entity"
	"ecommerce-be/product_management/utils"

	"gorm.io/gorm"
)

// ProductRepository defines the interface for product-related database operations
type ProductRepository interface {
	Create(product *entity.Product) error
	Update(product *entity.Product) error
	FindByID(id uint) (*entity.Product, error)
	FindBySKU(sku string) (*entity.Product, error)
	FindAll(filters map[string]interface{}, page, limit int) ([]entity.Product, int64, error)
	Search(query string, filters map[string]interface{}, page, limit int) ([]entity.Product, int64, error)
	Delete(id uint) error
	UpdateStock(id uint, inStock bool) error
	FindRelated(categoryID, excludeProductID uint, limit int) ([]entity.Product, error)
	FindPackageOptionByProductID(productID uint) ([]entity.PackageOption, error)
	CreatePackageOptions(option []entity.PackageOption) error
	UpdatePackageOptions(option []entity.PackageOption) error
}

// ProductRepositoryImpl implements the ProductRepository interface
type ProductRepositoryImpl struct {
	db *gorm.DB
}

// NewProductRepository creates a new instance of ProductRepository
func NewProductRepository(db *gorm.DB) ProductRepository {
	return &ProductRepositoryImpl{db: db}
}

// Create creates a new product
func (r *ProductRepositoryImpl) Create(product *entity.Product) error {
	return r.db.Create(product).Error
}

// Update updates an existing product
func (r *ProductRepositoryImpl) Update(product *entity.Product) error {
	return r.db.Save(product).Error
}

// FindByID finds a product by ID with eager loading
func (r *ProductRepositoryImpl) FindByID(id uint) (*entity.Product, error) {
	var product entity.Product
	result := r.db.Preload("Category").Preload("Category.Parent").Where("id = ?", id).First(&product)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New(utils.PRODUCT_NOT_FOUND_MSG)
		}
		return nil, result.Error
	}
	return &product, nil
}

// FindBySKU finds a product by SKU
func (r *ProductRepositoryImpl) FindBySKU(sku string) (*entity.Product, error) {
	var product entity.Product
	result := r.db.Where("sku = ?", sku).First(&product)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Not found, but not an error
		}
		return nil, result.Error
	}
	return &product, nil
}

// FindAll finds all products with filtering and pagination
func (r *ProductRepositoryImpl) FindAll(filters map[string]interface{}, page, limit int) ([]entity.Product, int64, error) {
	var products []entity.Product
	var total int64

	query := r.db.Model(&entity.Product{})

	// Apply filters
	if categoryID, exists := filters["categoryId"]; exists {
		query = query.Where("category_id = ?", categoryID)
	}
	if brand, exists := filters["brand"]; exists {
		query = query.Where("brand = ?", brand)
	}
	if minPrice, exists := filters["minPrice"]; exists {
		query = query.Where("price >= ?", minPrice)
	}
	if maxPrice, exists := filters["maxPrice"]; exists {
		query = query.Where("price <= ?", maxPrice)
	}
	if inStock, exists := filters["inStock"]; exists {
		query = query.Where("in_stock = ?", inStock)
	}
	if isPopular, exists := filters["isPopular"]; exists {
		query = query.Where("is_popular = ?", isPopular)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and sorting
	offset := (page - 1) * limit
	sortBy := "created_at"
	sortOrder := "desc"

	if sort, exists := filters["sortBy"]; exists {
		sortBy = sort.(string)
	}
	if order, exists := filters["sortOrder"]; exists {
		sortOrder = order.(string)
	}

	// Use eager loading to avoid N+1 queries
	query = query.Preload("Category").Preload("Category.Parent").Offset(offset).Limit(limit).Order(sortBy + " " + sortOrder)

	if err := query.Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Search searches products with query and filters
func (r *ProductRepositoryImpl) Search(query string, filters map[string]interface{}, page, limit int) ([]entity.Product, int64, error) {
	var products []entity.Product
	var total int64

	dbQuery := r.db.Model(&entity.Product{})

	// Apply search query
	if query != "" {
		dbQuery = dbQuery.Where(
			`name ILIKE ? OR short_description ILIKE ? OR EXISTS (
				SELECT 1
				FROM unnest(tags) AS tag
				WHERE tag ILIKE ?
			)`, "%"+query+"%", "%"+query+"%", "%"+query+"%")
	}

	// Apply filters
	if categoryID, exists := filters["categoryId"]; exists {
		dbQuery = dbQuery.Where("category_id = ?", categoryID)
	}
	if brand, exists := filters["brand"]; exists {
		dbQuery = dbQuery.Where("brand = ?", brand)
	}
	if minPrice, exists := filters["minPrice"]; exists {
		dbQuery = dbQuery.Where("price >= ?", minPrice)
	}
	if maxPrice, exists := filters["maxPrice"]; exists {
		dbQuery = dbQuery.Where("price <= ?", maxPrice)
	}
	
	// Count total
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and eager loading
	offset := (page - 1) * limit
	dbQuery = dbQuery.Preload("Category").Preload("Category.Parent").Offset(offset).Limit(limit).Order("created_at DESC")

	if err := dbQuery.Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// SoftDelete soft deletes a product
func (r *ProductRepositoryImpl) Delete(id uint) error {
	return r.db.Model(&entity.Product{}).Delete("id = ?", id).Error
}

// UpdateStock updates product stock status
func (r *ProductRepositoryImpl) UpdateStock(id uint, inStock bool) error {
	return r.db.Model(&entity.Product{}).Where("id = ?", id).Update("in_stock", inStock).Error
}

// FindRelated finds related products in the same category
func (r *ProductRepositoryImpl) FindRelated(categoryID, excludeProductID uint, limit int) ([]entity.Product, error) {
	var products []entity.Product
	result := r.db.Preload("Category").
		Where("category_id = ? AND id != ?", categoryID, excludeProductID).
		Order("created_at DESC").
		Limit(limit).
		Find(&products)

	if result.Error != nil {
		return nil, result.Error
	}
	return products, nil
}

func (r *ProductRepositoryImpl) FindPackageOptionByProductID(productID uint) ([]entity.PackageOption, error) {
	var packageOptions []entity.PackageOption
	result := r.db.Where("product_id = ?", productID).Find(&packageOptions)
	if result.Error != nil {
		return nil, result.Error
	}
	return packageOptions, nil
}

func (r *ProductRepositoryImpl) CreatePackageOptions(options []entity.PackageOption) error {
	return r.db.Create(options).Error
}

func (r *ProductRepositoryImpl) UpdatePackageOptions(options []entity.PackageOption) error {
	return r.db.Save(options).Error
}
