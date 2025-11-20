package singleton

import (
	"ecommerce-be/common/db"
	"ecommerce-be/product/repositories"
	"sync"
)

// RepositoryFactory manages all repository singleton instances
// Note: DB is fetched dynamically via db.GetDB() to support test scenarios
// where database connections change between test runs
type RepositoryFactory struct {
	categoryRepo    repositories.CategoryRepository
	attributeRepo   repositories.AttributeDefinitionRepository
	productRepo     repositories.ProductRepository
	variantRepo     repositories.VariantRepository
	optionRepo      repositories.ProductOptionRepository
	productAttrRepo repositories.ProductAttributeRepository

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
		currentDB := db.GetDB()
		f.categoryRepo = repositories.NewCategoryRepository(currentDB)
		f.attributeRepo = repositories.NewAttributeDefinitionRepository(currentDB)
		f.productRepo = repositories.NewProductRepository(currentDB)
		f.variantRepo = repositories.NewVariantRepository(currentDB)
		f.optionRepo = repositories.NewProductOptionRepository(currentDB)
		f.productAttrRepo = repositories.NewProductAttributeRepository(currentDB)
	})
}

// GetCategoryRepository returns the singleton category repository
func (f *RepositoryFactory) GetCategoryRepository() repositories.CategoryRepository {
	f.initialize()
	return f.categoryRepo
}

// GetAttributeDefinitionRepository returns the singleton attribute repository
func (f *RepositoryFactory) GetAttributeDefinitionRepository() repositories.AttributeDefinitionRepository {
	f.initialize()
	return f.attributeRepo
}

// GetProductRepository returns the singleton product repository
func (f *RepositoryFactory) GetProductRepository() repositories.ProductRepository {
	f.initialize()
	return f.productRepo
}

// GetVariantRepository returns the singleton variant repository
func (f *RepositoryFactory) GetVariantRepository() repositories.VariantRepository {
	f.initialize()
	return f.variantRepo
}

// GetProductOptionRepository returns the singleton product option repository
func (f *RepositoryFactory) GetProductOptionRepository() repositories.ProductOptionRepository {
	f.initialize()
	return f.optionRepo
}

// GetProductAttributeRepository returns the singleton product attribute repository
func (f *RepositoryFactory) GetProductAttributeRepository() repositories.ProductAttributeRepository {
	f.initialize()
	return f.productAttrRepo
}
