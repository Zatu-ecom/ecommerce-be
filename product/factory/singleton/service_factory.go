package singleton

import (
	"sync"

	"ecommerce-be/product/service"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	categoryService         service.CategoryService
	attributeService        service.AttributeDefinitionService
	productService          service.ProductService
	productQueryService     service.ProductQueryService
	variantService          service.VariantService
	variantQueryService     service.VariantQueryService
	variantBulkService      service.VariantBulkService
	productAttributeService service.ProductAttributeService
	productOptionService    service.ProductOptionService
	optionValueService      service.ProductOptionValueService
	validatorService        service.ProductValidatorService

	once sync.Once
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{repoFactory: repoFactory}
}

// initialize creates all service instances (lazy loading)
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		// Get repositories
		categoryRepo := f.repoFactory.GetCategoryRepository()
		attributeRepo := f.repoFactory.GetAttributeDefinitionRepository()
		productRepo := f.repoFactory.GetProductRepository()
		variantRepo := f.repoFactory.GetVariantRepository()
		optionRepo := f.repoFactory.GetProductOptionRepository()
		productAttrRepo := f.repoFactory.GetProductAttributeRepository()

		// Initialize validator service first (used by other services)
		f.validatorService = service.NewProductValidatorService(productRepo)

		// Initialize product option service (used by variant services)
		f.productOptionService = service.NewProductOptionService(optionRepo, f.validatorService)
		f.optionValueService = service.NewProductOptionValueService(optionRepo, productRepo)

		// Initialize VariantQueryService (query operations only - no circular dependencies)
		f.variantQueryService = service.NewVariantQueryService(
			variantRepo,
			f.productOptionService,
			f.validatorService,
		)

		// Initialize VariantService with VariantQueryService dependency
		f.variantService = service.NewVariantService(
			variantRepo,
			f.productOptionService,
			f.validatorService,
			f.variantQueryService,
		)

		// Initialize VariantBulkService for bulk operations
		f.variantBulkService = service.NewVariantBulkService(
			variantRepo,
			f.productOptionService,
			f.validatorService,
		)

		f.categoryService = service.NewCategoryService(categoryRepo, productRepo, attributeRepo)
		f.attributeService = service.NewAttributeDefinitionService(attributeRepo)
		f.productAttributeService = service.NewProductAttributeService(
			productAttrRepo,
			productRepo,
			attributeRepo,
		)

		// Initialize ProductQueryService with VariantQueryService
		f.productQueryService = service.NewProductQueryService(
			productRepo,
			f.variantQueryService,
			f.categoryService,
			f.productAttributeService,
			f.productOptionService,
		)

		// Initialize ProductService with its dependencies
		f.productService = service.NewProductService(
			productRepo,
			categoryRepo,
			f.productQueryService,
			f.validatorService,
			f.variantService,
			f.variantBulkService,
			f.productOptionService,
			f.productAttributeService,
		)
	})
}

// GetCategoryService returns the singleton category service
func (f *ServiceFactory) GetCategoryService() service.CategoryService {
	f.initialize()
	return f.categoryService
}

// GetAttributeDefinitionService returns the singleton attribute service
func (f *ServiceFactory) GetAttributeDefinitionService() service.AttributeDefinitionService {
	f.initialize()
	return f.attributeService
}

// GetProductService returns the singleton product service
func (f *ServiceFactory) GetProductService() service.ProductService {
	f.initialize()
	return f.productService
}

// GetProductQueryService returns the singleton product query service
func (f *ServiceFactory) GetProductQueryService() service.ProductQueryService {
	f.initialize()
	return f.productQueryService
}

// GetVariantService returns the singleton variant service
func (f *ServiceFactory) GetVariantService() service.VariantService {
	f.initialize()
	return f.variantService
}

// GetVariantQueryService returns the singleton variant query service
func (f *ServiceFactory) GetVariantQueryService() service.VariantQueryService {
	f.initialize()
	return f.variantQueryService
}

// GetVariantBulkService returns the singleton variant bulk service
func (f *ServiceFactory) GetVariantBulkService() service.VariantBulkService {
	f.initialize()
	return f.variantBulkService
}

// GetProductAttributeService returns the singleton product attribute service
func (f *ServiceFactory) GetProductAttributeService() service.ProductAttributeService {
	f.initialize()
	return f.productAttributeService
}

// GetProductOptionService returns the singleton product option service
func (f *ServiceFactory) GetProductOptionService() service.ProductOptionService {
	f.initialize()
	return f.productOptionService
}

// GetProductOptionValueService returns the singleton product option value service
func (f *ServiceFactory) GetProductOptionValueService() service.ProductOptionValueService {
	f.initialize()
	return f.optionValueService
}

// GetProductValidatorService returns the singleton product validator service
func (f *ServiceFactory) GetProductValidatorService() service.ProductValidatorService {
	f.initialize()
	return f.validatorService
}
