package singleton

import (
	"sync"

	fileSingleton "ecommerce-be/file/factory/singleton"
	fileGateway "ecommerce-be/file/gateway"
	productSingleton "ecommerce-be/product/factory/singleton"
	"ecommerce-be/promotion/service"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	promotionService           service.PromotionService
	promotionProductService    *service.PromotionProductScopeServiceImpl
	promotionVariantService    *service.PromotionVariantScopeServiceImpl
	promotionCategoryService   *service.PromotionCategoryScopeServiceImpl
	promotionCollectionService *service.PromotionCollectionScopeServiceImpl
	promotionCronService       service.PromotionCronService
	saleService                service.SaleService

	once sync.Once
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{
		repoFactory: repoFactory,
	}
}

// initialize creates all service instances (lazy loading)
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		// Get repositories
		promotionProductRepo := f.repoFactory.GetPromotionProductScopeRepository()
		promotionVariantRepo := f.repoFactory.GetPromotionProductVariantScopeRepository()
		promotionCategoryRepo := f.repoFactory.GetPromotionCategoryScopeRepository()
		promotionCollectionRepo := f.repoFactory.GetPromotionCollectionScopeRepository()
		promotionRepo := f.repoFactory.GetPromotionRepository()

		// Initialize services
		f.promotionProductService = service.NewPromotionProductScopeServiceImpl(
			promotionProductRepo,
			promotionRepo,
			productSingleton.GetInstance().GetProductRepository(),
			productSingleton.GetInstance().GetProductMediaService(),
		)
		f.promotionVariantService = service.NewPromotionVariantScopeServiceImpl(
			promotionVariantRepo,
		)
		f.promotionCategoryService = service.NewPromotionCategoryScopeServiceImpl(
			promotionCategoryRepo,
		)

		// Initialize promotion service with all dependencies
		collectionProductService := productSingleton.GetInstance().GetCollectionProductService()

		f.promotionCollectionService = service.NewPromotionCollectionScopeServiceImpl(
			promotionCollectionRepo,
			collectionProductService,
		)

		promotionScopeEligibilityServiceFactory := service.NewPromotionScopeEligibilityServiceFactory(
			f.promotionProductService,
			f.promotionCategoryService,
			f.promotionCollectionService,
			f.promotionVariantService,
		)

		f.promotionService = service.NewPromotionService(
			promotionRepo,
			f.repoFactory.GetSaleRepository(),
			f.promotionProductService,
			f.promotionCategoryService,
			f.promotionCollectionService,
			collectionProductService,
			promotionScopeEligibilityServiceFactory,
		)

		f.saleService = service.NewSaleService(
			f.repoFactory.GetSaleRepository(),
			fileGateway.NewDisplayGateway(fileSingleton.GetInstance().GetFileReadService()),
		)

		f.promotionCronService = service.NewPromotionCronService(promotionRepo)
	})
}

func (f *ServiceFactory) GetPromotionProductScopeService() *service.PromotionProductScopeServiceImpl {
	f.initialize()
	return f.promotionProductService
}

func (f *ServiceFactory) GetPromotionVariantScopeService() *service.PromotionVariantScopeServiceImpl {
	f.initialize()
	return f.promotionVariantService
}

func (f *ServiceFactory) GetPromotionCategoryScopeService() *service.PromotionCategoryScopeServiceImpl {
	f.initialize()
	return f.promotionCategoryService
}

func (f *ServiceFactory) GetPromotionCollectionScopeService() *service.PromotionCollectionScopeServiceImpl {
	f.initialize()
	return f.promotionCollectionService
}

func (f *ServiceFactory) GetPromotionService() service.PromotionService {
	f.initialize()
	return f.promotionService
}

func (f *ServiceFactory) GetPromotionCronService() service.PromotionCronService {
	f.initialize()
	return f.promotionCronService
}

func (f *ServiceFactory) GetSaleService() service.SaleService {
	f.initialize()
	return f.saleService
}
