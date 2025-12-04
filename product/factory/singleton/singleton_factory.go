package singleton

import (
	"sync"

	"ecommerce-be/product/handlers"
	"ecommerce-be/product/repositories"
	"ecommerce-be/product/service"
)

// SingletonFactory is the main facade for accessing all factories
// Delegates to specialized factories for repositories, services, and handlers
type SingletonFactory struct {
	repoFactory    *RepositoryFactory
	serviceFactory *ServiceFactory
	handlerFactory *HandlerFactory
}

var (
	instance *SingletonFactory
	once     sync.Once
)

// GetInstance returns the singleton instance of SingletonFactory
// DB connection is fetched dynamically from db.GetDB() when repositories are accessed
func GetInstance() *SingletonFactory {
	once.Do(func() {
		repoFactory := NewRepositoryFactory()
		serviceFactory := NewServiceFactory(repoFactory)
		handlerFactory := NewHandlerFactory(serviceFactory)

		instance = &SingletonFactory{
			repoFactory:    repoFactory,
			serviceFactory: serviceFactory,
			handlerFactory: handlerFactory,
		}
	})
	return instance
}

// ResetInstance resets the singleton instance
// This should ONLY be used in tests to ensure clean state between test runs
func ResetInstance() {
	once = sync.Once{}
	instance = nil
}

// ===============================
// Repository Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetCategoryRepository() repositories.CategoryRepository {
	return f.repoFactory.GetCategoryRepository()
}

func (f *SingletonFactory) GetAttributeDefinitionRepository() repositories.AttributeDefinitionRepository {
	return f.repoFactory.GetAttributeDefinitionRepository()
}

func (f *SingletonFactory) GetProductRepository() repositories.ProductRepository {
	return f.repoFactory.GetProductRepository()
}

func (f *SingletonFactory) GetVariantRepository() repositories.VariantRepository {
	return f.repoFactory.GetVariantRepository()
}

func (f *SingletonFactory) GetProductOptionRepository() repositories.ProductOptionRepository {
	return f.repoFactory.GetProductOptionRepository()
}

func (f *SingletonFactory) GetProductAttributeRepository() repositories.ProductAttributeRepository {
	return f.repoFactory.GetProductAttributeRepository()
}

// ===============================
// Service Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetCategoryService() service.CategoryService {
	return f.serviceFactory.GetCategoryService()
}

func (f *SingletonFactory) GetAttributeDefinitionService() service.AttributeDefinitionService {
	return f.serviceFactory.GetAttributeDefinitionService()
}

func (f *SingletonFactory) GetProductService() service.ProductService {
	return f.serviceFactory.GetProductService()
}

func (f *SingletonFactory) GetProductQueryService() service.ProductQueryService {
	return f.serviceFactory.GetProductQueryService()
}

func (f *SingletonFactory) GetVariantService() service.VariantService {
	return f.serviceFactory.GetVariantService()
}

func (f *SingletonFactory) GetVariantQueryService() service.VariantQueryService {
	return f.serviceFactory.GetVariantQueryService()
}

func (f *SingletonFactory) GetProductAttributeService() service.ProductAttributeService {
	return f.serviceFactory.GetProductAttributeService()
}

func (f *SingletonFactory) GetProductOptionService() service.ProductOptionService {
	return f.serviceFactory.GetProductOptionService()
}

func (f *SingletonFactory) GetProductOptionValueService() service.ProductOptionValueService {
	return f.serviceFactory.GetProductOptionValueService()
}

func (f *SingletonFactory) GetProductValidatorService() service.ProductValidatorService {
	return f.serviceFactory.GetProductValidatorService()
}

// ===============================
// Handler Getters (Delegates)
// ===============================

func (f *SingletonFactory) GetCategoryHandler() *handlers.CategoryHandler {
	return f.handlerFactory.GetCategoryHandler()
}

func (f *SingletonFactory) GetAttributeHandler() *handlers.AttributeHandler {
	return f.handlerFactory.GetAttributeHandler()
}

func (f *SingletonFactory) GetProductHandler() *handlers.ProductHandler {
	return f.handlerFactory.GetProductHandler()
}

func (f *SingletonFactory) GetVariantHandler() *handlers.VariantHandler {
	return f.handlerFactory.GetVariantHandler()
}

func (f *SingletonFactory) GetProductAttributeHandler() *handlers.ProductAttributeHandler {
	return f.handlerFactory.GetProductAttributeHandler()
}

func (f *SingletonFactory) GetProductOptionHandler() *handlers.ProductOptionHandler {
	return f.handlerFactory.GetProductOptionHandler()
}

func (f *SingletonFactory) GetProductOptionValueHandler() *handlers.ProductOptionValueHandler {
	return f.handlerFactory.GetProductOptionValueHandler()
}
