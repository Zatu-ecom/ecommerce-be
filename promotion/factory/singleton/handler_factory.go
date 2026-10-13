package singleton

import (
	"sync"

	"ecommerce-be/promotion/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	promotionHandler              *handler.PromotionHandler
	promotionProductHandler       *handler.PromotionProductScopeHandler
	promotionVariantHandler       *handler.PromotionVariantScopeHandler
	promotionCategoryHandler      *handler.PromotionCategoryScopeHandler
	promotionCollectionHandler    *handler.PromotionCollectionScopeHandler
	saleHandler                   *handler.SaleHandler
	discountCodeHandler           *handler.DiscountCodeHandler
	discountCodeProductHandler    *handler.DiscountCodeProductScopeHandler
	discountCodeVariantHandler    *handler.DiscountCodeVariantScopeHandler
	discountCodeCategoryHandler   *handler.DiscountCodeCategoryScopeHandler
	discountCodeCollectionHandler *handler.DiscountCodeCollectionScopeHandler

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
		f.promotionHandler = handler.NewPromotionHandler(f.serviceFactory.GetPromotionService())
		f.promotionProductHandler = handler.NewPromotionProductScopeHandler(promotionProductService)
		f.promotionVariantHandler = handler.NewPromotionVariantScopeHandler(promotionVariantService)
		f.promotionCategoryHandler = handler.NewPromotionCategoryScopeHandler(
			promotionCategoryService,
		)
		f.promotionCollectionHandler = handler.NewPromotionCollectionScopeHandler(
			promotionCollectionService,
		)
		f.saleHandler = handler.NewSaleHandler(f.serviceFactory.GetSaleService())
		f.discountCodeHandler = handler.NewDiscountCodeHandler(
			f.serviceFactory.GetDiscountCodeService(),
		)
		f.discountCodeProductHandler = handler.NewDiscountCodeProductScopeHandler(
			f.serviceFactory.GetDiscountCodeProductScopeService(),
		)
		f.discountCodeVariantHandler = handler.NewDiscountCodeVariantScopeHandler(
			f.serviceFactory.GetDiscountCodeVariantScopeService(),
		)
		f.discountCodeCategoryHandler = handler.NewDiscountCodeCategoryScopeHandler(
			f.serviceFactory.GetDiscountCodeCategoryScopeService(),
		)
		f.discountCodeCollectionHandler = handler.NewDiscountCodeCollectionScopeHandler(
			f.serviceFactory.GetDiscountCodeCollectionScopeService(),
		)
	})
}

func (f *HandlerFactory) GetPromotionHandler() *handler.PromotionHandler {
	f.initialize()
	return f.promotionHandler
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

func (f *HandlerFactory) GetSaleHandler() *handler.SaleHandler {
	f.initialize()
	return f.saleHandler
}

func (f *HandlerFactory) GetDiscountCodeHandler() *handler.DiscountCodeHandler {
	f.initialize()
	return f.discountCodeHandler
}

func (f *HandlerFactory) GetDiscountCodeProductScopeHandler() *handler.DiscountCodeProductScopeHandler {
	f.initialize()
	return f.discountCodeProductHandler
}

func (f *HandlerFactory) GetDiscountCodeVariantScopeHandler() *handler.DiscountCodeVariantScopeHandler {
	f.initialize()
	return f.discountCodeVariantHandler
}

func (f *HandlerFactory) GetDiscountCodeCategoryScopeHandler() *handler.DiscountCodeCategoryScopeHandler {
	f.initialize()
	return f.discountCodeCategoryHandler
}

func (f *HandlerFactory) GetDiscountCodeCollectionScopeHandler() *handler.DiscountCodeCollectionScopeHandler {
	f.initialize()
	return f.discountCodeCollectionHandler
}
