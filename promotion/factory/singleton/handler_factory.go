package singleton

import (
	"sync"

	"ecommerce-be/promotion/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	promotionProductHandler    *handler.PromotionProductScopeHandler
	promotionVariantHandler    *handler.PromotionVariantScopeHandler
	promotionCategoryHandler   *handler.PromotionCategoryScopeHandler
	promotionCollectionHandler *handler.PromotionCollectionScopeHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{
		serviceFactory: serviceFactory,
	}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		// Get services
		promotionProductService := f.serviceFactory.GetPromotionProductScopeService()
		promotionVariantService := f.serviceFactory.GetPromotionVariantScopeService()
		promotionCategoryService := f.serviceFactory.GetPromotionCategoryScopeService()
		promotionCollectionService := f.serviceFactory.GetPromotionCollectionScopeService()

		// Initialize handlers
		f.promotionProductHandler = handler.NewPromotionProductScopeHandler(promotionProductService)
		f.promotionVariantHandler = handler.NewPromotionVariantScopeHandler(promotionVariantService)
		f.promotionCategoryHandler = handler.NewPromotionCategoryScopeHandler(
			promotionCategoryService,
		)
		f.promotionCollectionHandler = handler.NewPromotionCollectionScopeHandler(
			promotionCollectionService,
		)
	})
}

func (f *HandlerFactory) GetPromotionProductScopeHandler() *handler.PromotionProductScopeHandler {
	f.initialize()
	return f.promotionProductHandler
}

func (f *HandlerFactory) GetPromotionVariantScopeHandler() *handler.PromotionVariantScopeHandler {
	f.initialize()
	return f.promotionVariantHandler
}

func (f *HandlerFactory) GetPromotionCategoryScopeHandler() *handler.PromotionCategoryScopeHandler {
	f.initialize()
	return f.promotionCategoryHandler
}

func (f *HandlerFactory) GetPromotionCollectionScopeHandler() *handler.PromotionCollectionScopeHandler {
	f.initialize()
	return f.promotionCollectionHandler
}
