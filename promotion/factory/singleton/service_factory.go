package singleton

import (
	"sync"

	"ecommerce-be/promotion/service"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	promotionProductService    service.PromotionProductScopeService
	promotionVariantService    service.PromotionVariantScopeService
	promotionCategoryService   service.PromotionCategoryScopeService
	promotionCollectionService service.PromotionCollectionScopeService

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
		f.promotionProductService = service.NewPromotionProductScopeService(promotionProductRepo)
		f.promotionVariantService = service.NewPromotionVariantScopeService(promotionVariantRepo)
		f.promotionCategoryService = service.NewPromotionCategoryScopeService(promotionCategoryRepo)
		f.promotionCollectionService = service.NewPromotionCollectionScopeService(
			promotionCollectionRepo,
		)
	})
}

func (f *ServiceFactory) GetPromotionProductScopeService() service.PromotionProductScopeService {
	f.initialize()
	return f.promotionProductService
}

func (f *ServiceFactory) GetPromotionVariantScopeService() service.PromotionVariantScopeService {
	f.initialize()
	return f.promotionVariantService
}

func (f *ServiceFactory) GetPromotionCategoryScopeService() service.PromotionCategoryScopeService {
	f.initialize()
	return f.promotionCategoryService
}

func (f *ServiceFactory) GetPromotionCollectionScopeService() service.PromotionCollectionScopeService {
	f.initialize()
	return f.promotionCollectionService
}
