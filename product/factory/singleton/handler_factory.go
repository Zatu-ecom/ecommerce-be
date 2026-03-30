package singleton

import (
	"sync"

	"ecommerce-be/product/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	categoryHandler         *handler.CategoryHandler
	attributeHandler        *handler.AttributeHandler
	productHandler          *handler.ProductHandler
	variantHandler          *handler.VariantHandler
	productAttributeHandler *handler.ProductAttributeHandler
	productOptionHandler    *handler.ProductOptionHandler
	optionValueHandler      *handler.ProductOptionValueHandler
	wishlistHandler         *handler.WishlistHandler
	wishlistItemHandler     *handler.WishlistItemHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		f.categoryHandler = handler.NewCategoryHandler(f.serviceFactory.GetCategoryService())
		f.attributeHandler = handler.NewAttributeHandler(
			f.serviceFactory.GetAttributeDefinitionService(),
		)
		f.productHandler = handler.NewProductHandler(
			f.serviceFactory.GetProductService(),
			f.serviceFactory.GetProductQueryService(),
		)
		f.variantHandler = handler.NewVariantHandler(
			f.serviceFactory.GetVariantService(),
			f.serviceFactory.GetVariantQueryService(),
			f.serviceFactory.GetVariantBulkService(),
		)
		f.productAttributeHandler = handler.NewProductAttributeHandler(
			f.serviceFactory.GetProductAttributeService(),
		)
		f.productOptionHandler = handler.NewProductOptionHandler(
			f.serviceFactory.GetProductOptionService(),
		)
		f.optionValueHandler = handler.NewProductOptionValueHandler(
			f.serviceFactory.GetProductOptionValueService(),
		)
		f.wishlistHandler = handler.NewWishlistHandler(
			f.serviceFactory.GetWishlistService(),
		)
		f.wishlistItemHandler = handler.NewWishlistItemHandler(
			f.serviceFactory.GetWishlistItemService(),
		)
	})
}

// GetCategoryHandler returns the singleton category handler
func (f *HandlerFactory) GetCategoryHandler() *handler.CategoryHandler {
	f.initialize()
	return f.categoryHandler
}

// GetAttributeHandler returns the singleton attribute handler
func (f *HandlerFactory) GetAttributeHandler() *handler.AttributeHandler {
	f.initialize()
	return f.attributeHandler
}

// GetProductHandler returns the singleton product handler
func (f *HandlerFactory) GetProductHandler() *handler.ProductHandler {
	f.initialize()
	return f.productHandler
}

// GetVariantHandler returns the singleton variant handler
func (f *HandlerFactory) GetVariantHandler() *handler.VariantHandler {
	f.initialize()
	return f.variantHandler
}

// GetProductAttributeHandler returns the singleton product attribute handler
func (f *HandlerFactory) GetProductAttributeHandler() *handler.ProductAttributeHandler {
	f.initialize()
	return f.productAttributeHandler
}

// GetProductOptionHandler returns the singleton product option handler
func (f *HandlerFactory) GetProductOptionHandler() *handler.ProductOptionHandler {
	f.initialize()
	return f.productOptionHandler
}

// GetProductOptionValueHandler returns the singleton product option value handler
func (f *HandlerFactory) GetProductOptionValueHandler() *handler.ProductOptionValueHandler {
	f.initialize()
	return f.optionValueHandler
}

// GetWishlistHandler returns the singleton wishlist handler
func (f *HandlerFactory) GetWishlistHandler() *handler.WishlistHandler {
	f.initialize()
	return f.wishlistHandler
}

// GetWishlistItemHandler returns the singleton wishlist item handler
func (f *HandlerFactory) GetWishlistItemHandler() *handler.WishlistItemHandler {
	f.initialize()
	return f.wishlistItemHandler
}
