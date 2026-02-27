package singleton

import (
	"sync"

	"ecommerce-be/promotion/repository"
)

// RepositoryFactory manages all repository singleton instances
type RepositoryFactory struct {
	promotionRepository               repository.PromotionRepository
	promotionProductRepository        repository.PromotionProductScopeRepository
	promotionProductVariantRepository repository.PromotionProductVariantScopeRepository
	promotionCategoryRepository       repository.PromotionCategoryScopeRepository
	promotionCollectionRepository     repository.PromotionCollectionScopeRepository
	once                              sync.Once
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading)
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		f.promotionRepository = repository.NewPromotionRepository()
		f.promotionProductRepository = repository.NewPromotionProductScopeRepository()
		f.promotionProductVariantRepository = repository.NewPromotionProductVariantScopeRepository()
		f.promotionCategoryRepository = repository.NewPromotionCategoryScopeRepository()
		f.promotionCollectionRepository = repository.NewPromotionCollectionScopeRepository()
	})
}

func (f *RepositoryFactory) GetPromotionProductScopeRepository() repository.PromotionProductScopeRepository {
	f.initialize()
	return f.promotionProductRepository
}

func (f *RepositoryFactory) GetPromotionProductVariantScopeRepository() repository.PromotionProductVariantScopeRepository {
	f.initialize()
	return f.promotionProductVariantRepository
}

func (f *RepositoryFactory) GetPromotionCategoryScopeRepository() repository.PromotionCategoryScopeRepository {
	f.initialize()
	return f.promotionCategoryRepository
}

func (f *RepositoryFactory) GetPromotionCollectionScopeRepository() repository.PromotionCollectionScopeRepository {
	f.initialize()
	return f.promotionCollectionRepository
}

func (f *RepositoryFactory) GetPromotionRepository() repository.PromotionRepository {
	f.initialize()
	return f.promotionRepository
}
