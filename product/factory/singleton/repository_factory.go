package singleton

import (
	"sync"

	"ecommerce-be/product/repository"
)

// RepositoryFactory manages all repository singleton instances
// Note: DB is fetched dynamically via db.GetDB() to support test scenarios
// where database connections change between test runs
type RepositoryFactory struct {
	categoryRepo           repository.CategoryRepository
	attributeRepo          repository.AttributeDefinitionRepository
	productRepo            repository.ProductRepository
	variantRepo            repository.VariantRepository
	optionRepo             repository.ProductOptionRepository
	productAttrRepo        repository.ProductAttributeRepository
	wishlistRepo           repository.WishlistRepository
	wishlistItemRepo       repository.WishlistItemRepository
	collectionProductRepo  repository.CollectionProductRepository

	once sync.Once
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading)
// Uses db.GetDB() to fetch current database connection dynamically
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		f.categoryRepo = repository.NewCategoryRepository()
		f.attributeRepo = repository.NewAttributeDefinitionRepository()
		f.productRepo = repository.NewProductRepository()
		f.variantRepo = repository.NewVariantRepository()
		f.optionRepo = repository.NewProductOptionRepository()
		f.productAttrRepo = repository.NewProductAttributeRepository()
		f.wishlistRepo = repository.NewWishlistRepository()
		f.wishlistItemRepo = repository.NewWishlistItemRepository()
		f.collectionProductRepo = repository.NewCollectionProductRepository()
	})
}

// GetCategoryRepository returns the singleton category repository
func (f *RepositoryFactory) GetCategoryRepository() repository.CategoryRepository {
	f.initialize()
	return f.categoryRepo
}

// GetAttributeDefinitionRepository returns the singleton attribute repository
func (f *RepositoryFactory) GetAttributeDefinitionRepository() repository.AttributeDefinitionRepository {
	f.initialize()
	return f.attributeRepo
}

// GetProductRepository returns the singleton product repository
func (f *RepositoryFactory) GetProductRepository() repository.ProductRepository {
	f.initialize()
	return f.productRepo
}

// GetVariantRepository returns the singleton variant repository
func (f *RepositoryFactory) GetVariantRepository() repository.VariantRepository {
	f.initialize()
	return f.variantRepo
}

// GetProductOptionRepository returns the singleton product option repository
func (f *RepositoryFactory) GetProductOptionRepository() repository.ProductOptionRepository {
	f.initialize()
	return f.optionRepo
}

// GetProductAttributeRepository returns the singleton product attribute repository
func (f *RepositoryFactory) GetProductAttributeRepository() repository.ProductAttributeRepository {
	f.initialize()
	return f.productAttrRepo
}

// GetWishlistRepository returns the singleton wishlist repository
func (f *RepositoryFactory) GetWishlistRepository() repository.WishlistRepository {
	f.initialize()
	return f.wishlistRepo
}

// GetWishlistItemRepository returns the singleton wishlist item repository
func (f *RepositoryFactory) GetWishlistItemRepository() repository.WishlistItemRepository {
	f.initialize()
	return f.wishlistItemRepo
}

func (f *RepositoryFactory) GetCollectionProductRepository() repository.CollectionProductRepository {
	f.initialize()
	return f.collectionProductRepo
}
