package repositories

import (
	"errors"

	"ecommerce-be/common/log"
	"ecommerce-be/product/entity"
	productError "ecommerce-be/product/errors"
	"ecommerce-be/product/mapper"
	"ecommerce-be/product/model"
	productQuery "ecommerce-be/product/query"
	"ecommerce-be/product/utils/helper"

	"gorm.io/gorm"
)

// ProductRepository defines the interface for product-related database operations
type ProductRepository interface {
	Create(product *entity.Product) error
	Update(product *entity.Product) error
	FindByID(id uint) (*entity.Product, error)
	// FindBySKU removed - BaseSKU validation no longer required
	FindAll(filter model.GetProductsFilter, page, limit int) ([]entity.Product, int64, error)
	Search(
		query string,
		filters map[string]interface{},
		page, limit int,
	) ([]entity.Product, int64, error)
	Delete(id uint) error
	UpdateStock(id uint, inStock bool) error
	FindRelated(
		categoryID, excludeProductID uint,
		limit int,
		sellerID *uint,
	) ([]entity.Product, error)
	// New method for intelligent related products with scoring
	FindRelatedScored(
		productID uint,
		sellerID *uint,
		limit int,
		offset int,
		strategies string,
	) ([]mapper.RelatedProductScored, int64, error)
	FindPackageOptionByProductID(productID uint) ([]entity.PackageOption, error)
	CreatePackageOptions(option []entity.PackageOption) error
	UpdatePackageOptions(option []entity.PackageOption) error
	GetProductFilters(sellerID *uint) (
		[]mapper.BrandWithProductCount,
		[]mapper.CategoryWithProductCount,
		[]mapper.AttributeWithProductCount,
		*mapper.PriceRangeData,
		[]mapper.VariantOptionData,
		*mapper.StockStatusData,
		error,
	)

	// Bulk deletion methods for product cleanup
	DeletePackageOptionsByProductID(productID uint) error
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
	result := r.db.Preload("Category").
		Preload("Category.Parent").
		Where("id = ?", id).
		First(&product)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, productError.ErrProductNotFound
		}
		return nil, result.Error
	}
	return &product, nil
}

// FindAll finds all products with filtering and pagination
// Updated to work with variant-based pricing and stock
func (r *ProductRepositoryImpl) FindAll(
	filter model.GetProductsFilter,
	page, limit int,
) ([]entity.Product, int64, error) {
	var products []entity.Product
	var total int64

	query := r.db.Model(&entity.Product{})

	// Apply filters
	// Multi-tenant filter: seller_id (CRITICAL for data isolation)
	if filter.SellerID != nil {
		query = query.Where("seller_id = ?", *filter.SellerID)
	}
	if len(filter.CategoryIDs) > 0 {
		query = query.Where("category_id IN ?", filter.CategoryIDs)
	}
	if len(filter.Brands) > 0 {
		query = query.Where("brand IN ?", filter.Brands)
	}
	if len(filter.IDs) > 0 {
		query = query.Where("id IN ?", filter.IDs)
	}

	// Price filters - now based on variants
	if filter.MinPrice != nil {
		query = query.Where(productQuery.FILTER_PRICE_MIN_SUBQUERY, *filter.MinPrice)
	}
	if filter.MaxPrice != nil {
		query = query.Where(productQuery.FILTER_PRICE_MAX_SUBQUERY, *filter.MaxPrice)
	}

	// Stock filter - now based on variants
	if filter.InStock != nil {
		if *filter.InStock {
			query = query.Where(productQuery.FILTER_IN_STOCK_SUBQUERY)
		} else {
			query = query.Where(productQuery.FILTER_OUT_OF_STOCK_SUBQUERY)
		}
	}

	// Popularity filter - now based on variants
	if filter.IsPopular != nil {
		query = query.Where(productQuery.FILTER_IS_POPULAR_SUBQUERY, *filter.IsPopular)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and sorting
	offset := (page - 1) * limit
	sortBy := filter.SortBy
	sortOrder := filter.SortOrder

	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// Use eager loading to avoid N+1 queries
	query = query.Preload("Category").
		Preload("Category.Parent").
		Offset(offset).
		Limit(limit).
		Order(sortBy + " " + sortOrder)

	if err := query.Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Search searches products with query and filters
// Updated to work with variant-based pricing and stock
func (r *ProductRepositoryImpl) Search(
	query string,
	filters map[string]interface{},
	page, limit int,
) ([]entity.Product, int64, error) {
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
	// Multi-tenant filter: seller_id (CRITICAL for data isolation)
	if sellerID, exists := filters["sellerId"]; exists {
		dbQuery = dbQuery.Where("seller_id = ?", sellerID)
	}
	if categoryID, exists := filters["categoryId"]; exists {
		dbQuery = dbQuery.Where("category_id = ?", categoryID)
	}
	if brand, exists := filters["brand"]; exists {
		dbQuery = dbQuery.Where("brand = ?", brand)
	}

	// Price filters - now based on variants
	if minPrice, exists := filters["minPrice"]; exists {
		dbQuery = dbQuery.Where(productQuery.FILTER_PRICE_MIN_SUBQUERY, minPrice)
	}
	if maxPrice, exists := filters["maxPrice"]; exists {
		dbQuery = dbQuery.Where(productQuery.FILTER_PRICE_MAX_SUBQUERY, maxPrice)
	}

	// Stock filter - now based on variants
	if inStock, exists := filters["inStock"]; exists {
		if inStock.(bool) {
			dbQuery = dbQuery.Where(productQuery.FILTER_IN_STOCK_SUBQUERY)
		} else {
			dbQuery = dbQuery.Where(productQuery.FILTER_OUT_OF_STOCK_SUBQUERY)
		}
	}

	// Popularity filter - now based on variants
	if isPopular, exists := filters["isPopular"]; exists {
		dbQuery = dbQuery.Where(productQuery.FILTER_IS_POPULAR_SUBQUERY, isPopular)
	}

	// Count total
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and eager loading
	offset := (page - 1) * limit
	dbQuery = dbQuery.Preload("Category").
		Preload("Category.Parent").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC")

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
func (r *ProductRepositoryImpl) FindRelated(
	categoryID, excludeProductID uint,
	limit int,
	sellerID *uint,
) ([]entity.Product, error) {
	var products []entity.Product
	query := r.db.Preload("Category").
		Where("category_id = ? AND id != ?", categoryID, excludeProductID)

	// Apply seller filter if sellerID is provided (non-admin)
	if sellerID != nil {
		query = query.Where("seller_id = ?", *sellerID)
	}

	result := query.Order("created_at DESC").
		Limit(limit).
		Find(&products)

	if result.Error != nil {
		return nil, result.Error
	}
	return products, nil
}

// FindRelatedScored finds related products using stored procedure with intelligent scoring
func (r *ProductRepositoryImpl) FindRelatedScored(
	productID uint,
	sellerID *uint,
	limit int,
	offset int,
	strategies string,
) ([]mapper.RelatedProductScored, int64, error) {
	var results []mapper.RelatedProductScored

	// Convert sellerID to BIGINT or NULL
	var sellerParam interface{}
	if sellerID != nil {
		sellerParam = *sellerID
	} else {
		sellerParam = nil
	}

	// Call stored procedure for related products using query constants
	err := r.db.Raw(productQuery.FIND_RELATED_PRODUCTS_SCORED_QUERY, productID, sellerParam, limit, offset, strategies).
		Scan(&results).
		Error
	if err != nil {
		return nil, 0, err
	}

	// Get total count for pagination
	var totalCount int64
	err = r.db.Raw(productQuery.FIND_RELATED_PRODUCTS_COUNT_QUERY, productID, sellerParam, strategies).
		Scan(&totalCount).
		Error
	if err != nil {
		return nil, 0, err
	}

	return results, totalCount, nil
}

func (r *ProductRepositoryImpl) FindPackageOptionByProductID(
	productID uint,
) ([]entity.PackageOption, error) {
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

// GetProductFilters fetches all filter data in optimized queries including variant-based filters
// Multi-tenant: If sellerID is provided, filter by seller. If nil (admin), get all.
func (r *ProductRepositoryImpl) GetProductFilters(sellerID *uint) (
	[]mapper.BrandWithProductCount,
	[]mapper.CategoryWithProductCount,
	[]mapper.AttributeWithProductCount,
	*mapper.PriceRangeData,
	[]mapper.VariantOptionData,
	*mapper.StockStatusData,
	error,
) {
	var brands []mapper.BrandWithProductCount
	var categories []mapper.CategoryWithProductCount
	var attributes []mapper.AttributeWithProductCount
	var priceRange mapper.PriceRangeData
	var variantOptions []mapper.VariantOptionData
	var stockStatus mapper.StockStatusData

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Brands query with optional seller filter
		if sellerID != nil {
			// Multi-tenant: Filter by seller_id
			if err := tx.Raw(productQuery.FIND_BRANDS_WITH_PRODUCT_COUNT_BY_SELLER_QUERY, *sellerID).
				Scan(&brands).Error; err != nil {
				return err
			}
		} else {
			// Admin: Get all brands
			if err := tx.Raw(productQuery.FIND_BRANDS_WITH_PRODUCT_COUNT_QUERY).
				Scan(&brands).Error; err != nil {
				return err
			}
		}

		// Categories query with optional seller filter
		if sellerID != nil {
			// Multi-tenant: Global categories + seller-specific categories
			if err := tx.Raw(productQuery.FIND_CATEGORIES_WITH_PRODUCT_COUNT_BY_SELLER_QUERY, *sellerID, *sellerID).
				Scan(&categories).Error; err != nil {
				return err
			}
		} else {
			// Admin: Get all categories
			if err := tx.Raw(productQuery.FIND_CATEGORIES_WITH_PRODUCT_COUNT_QUERY).
				Scan(&categories).Error; err != nil {
				return err
			}
		}

		// Attributes query with optional seller filter
		if sellerID != nil {
			// Multi-tenant: Filter by seller's products
			if err := tx.Raw(productQuery.FIND_ATTRIBUTES_WITH_PRODUCT_COUNT_BY_SELLER_QUERY, *sellerID).
				Scan(&attributes).Error; err != nil {
				return err
			}
		} else {
			// Admin: Get all attributes
			if err := tx.Raw(productQuery.FIND_ATTRIBUTES_WITH_PRODUCT_COUNT_QUERY).
				Scan(&attributes).Error; err != nil {
				return err
			}
		}

		// Price range query from variants
		if sellerID != nil {
			if err := tx.Raw(productQuery.FIND_PRICE_RANGE_BY_SELLER_QUERY, *sellerID).
				Scan(&priceRange).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Raw(productQuery.FIND_PRICE_RANGE_QUERY).
				Scan(&priceRange).Error; err != nil {
				return err
			}
		}

		// Variant options query (Color, Size, etc.)
		if sellerID != nil {
			if err := tx.Raw(productQuery.FIND_VARIANT_OPTIONS_BY_SELLER_QUERY, *sellerID).
				Scan(&variantOptions).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Raw(productQuery.FIND_VARIANT_OPTIONS_QUERY).
				Scan(&variantOptions).Error; err != nil {
				return err
			}
		}

		// Stock status query
		if sellerID != nil {
			if err := tx.Raw(productQuery.FIND_STOCK_STATUS_BY_SELLER_QUERY, *sellerID).
				Scan(&stockStatus).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Raw(productQuery.FIND_STOCK_STATUS_QUERY).
				Scan(&stockStatus).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	logmap := map[string]interface{}{
		"categories":     len(categories),
		"attributes":     len(attributes),
		"brands":         len(brands),
		"priceRange":     priceRange,
		"variantOptions": len(variantOptions),
		"stockStatus":    stockStatus,
	}

	prettyLogmap, _ := helper.MapToPrettyJSON(logmap)

	// Log filter data (replaces fmt.Println debug statements)
	log.Debug("Product filters fetched" + prettyLogmap)

	return brands, categories, attributes, &priceRange, variantOptions, &stockStatus, nil
}

/***********************************************
 *    Bulk Deletion Methods for Product Cleanup
 ***********************************************/

// DeletePackageOptionsByProductID deletes all package options for a given product
func (r *ProductRepositoryImpl) DeletePackageOptionsByProductID(productID uint) error {
	return r.db.Where("product_id = ?", productID).Delete(&entity.PackageOption{}).Error
}
