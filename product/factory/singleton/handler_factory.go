package singleton

import (
	"ecommerce-be/product/handlers"
	"sync"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	categoryHandler         *handlers.CategoryHandler
	attributeHandler        *handlers.AttributeHandler
	productHandler          *handlers.ProductHandler
	variantHandler          *handlers.VariantHandler
	productAttributeHandler *handlers.ProductAttributeHandler
	productOptionHandler    *handlers.ProductOptionHandler
	optionValueHandler      *handlers.ProductOptionValueHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		f.categoryHandler = handlers.NewCategoryHandler(f.serviceFactory.GetCategoryService())
		f.attributeHandler = handlers.NewAttributeHandler(f.serviceFactory.GetAttributeDefinitionService())
		f.productHandler = handlers.NewProductHandler(
			f.serviceFactory.GetProductService(),
			f.serviceFactory.GetProductQueryService(),
		)
		f.variantHandler = handlers.NewVariantHandler(f.serviceFactory.GetVariantService())
		f.productAttributeHandler = handlers.NewProductAttributeHandler(f.serviceFactory.GetProductAttributeService())
		f.productOptionHandler = handlers.NewProductOptionHandler(f.serviceFactory.GetProductOptionService())
		f.optionValueHandler = handlers.NewProductOptionValueHandler(f.serviceFactory.GetProductOptionValueService())
	})
}

// GetCategoryHandler returns the singleton category handler
func (f *HandlerFactory) GetCategoryHandler() *handlers.CategoryHandler {
	f.initialize()
	return f.categoryHandler
}

// GetAttributeHandler returns the singleton attribute handler
func (f *HandlerFactory) GetAttributeHandler() *handlers.AttributeHandler {
	f.initialize()
	return f.attributeHandler
}

// GetProductHandler returns the singleton product handler
func (f *HandlerFactory) GetProductHandler() *handlers.ProductHandler {
	f.initialize()
	return f.productHandler
}

// GetVariantHandler returns the singleton variant handler
func (f *HandlerFactory) GetVariantHandler() *handlers.VariantHandler {
	f.initialize()
	return f.variantHandler
}

// GetProductAttributeHandler returns the singleton product attribute handler
func (f *HandlerFactory) GetProductAttributeHandler() *handlers.ProductAttributeHandler {
	f.initialize()
	return f.productAttributeHandler
}

// GetProductOptionHandler returns the singleton product option handler
func (f *HandlerFactory) GetProductOptionHandler() *handlers.ProductOptionHandler {
	f.initialize()
	return f.productOptionHandler
}

// GetProductOptionValueHandler returns the singleton product option value handler
func (f *HandlerFactory) GetProductOptionValueHandler() *handlers.ProductOptionValueHandler {
	f.initialize()
	return f.optionValueHandler
}
