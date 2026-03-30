package singleton

import (
	"sync"

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

		// Initialize services
		f.promotionProductService = service.NewPromotionProductScopeServiceImpl(
			promotionProductRepo,
		)
		f.promotionVariantService = service.NewPromotionVariantScopeServiceImpl(
			promotionVariantRepo,
		)
		f.promotionCategoryService = service.NewPromotionCategoryScopeServiceImpl(
			promotionCategoryRepo,
		)

		// Initialize promotion service with all dependencies
		promotionRepo := f.repoFactory.GetPromotionRepository()
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
			f.promotionProductService,
			f.promotionCategoryService,
			f.promotionCollectionService,
			collectionProductService,
			promotionScopeEligibilityServiceFactory,
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
