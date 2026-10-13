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
	saleRepository                    repository.SaleRepository
	discountCodeRepository            repository.DiscountCodeRepository
	discountCodeUsageRepository       repository.DiscountCodeUsageRepository
	discountCodeProductRepository     repository.DiscountCodeProductScopeRepository
	discountCodeVariantRepository     repository.DiscountCodeVariantScopeRepository
	discountCodeCategoryRepository    repository.DiscountCodeCategoryScopeRepository
	discountCodeCollectionRepository  repository.DiscountCodeCollectionScopeRepository
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
		f.saleRepository = repository.NewSaleRepository()
		f.discountCodeRepository = repository.NewDiscountCodeRepository()
		f.discountCodeUsageRepository = repository.NewDiscountCodeUsageRepository()
		f.discountCodeProductRepository = repository.NewDiscountCodeProductScopeRepository()
		f.discountCodeVariantRepository = repository.NewDiscountCodeVariantScopeRepository()
		f.discountCodeCategoryRepository = repository.NewDiscountCodeCategoryScopeRepository()
		f.discountCodeCollectionRepository = repository.NewDiscountCodeCollectionScopeRepository()
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

func (f *RepositoryFactory) GetSaleRepository() repository.SaleRepository {
	f.initialize()
	return f.saleRepository
}

func (f *RepositoryFactory) GetDiscountCodeRepository() repository.DiscountCodeRepository {
	f.initialize()
	return f.discountCodeRepository
}

func (f *RepositoryFactory) GetDiscountCodeUsageRepository() repository.DiscountCodeUsageRepository {
	f.initialize()
	return f.discountCodeUsageRepository
}

func (f *RepositoryFactory) GetDiscountCodeProductScopeRepository() repository.DiscountCodeProductScopeRepository {
	f.initialize()
	return f.discountCodeProductRepository
}

func (f *RepositoryFactory) GetDiscountCodeVariantScopeRepository() repository.DiscountCodeVariantScopeRepository {
	f.initialize()
	return f.discountCodeVariantRepository
}

func (f *RepositoryFactory) GetDiscountCodeCategoryScopeRepository() repository.DiscountCodeCategoryScopeRepository {
	f.initialize()
	return f.discountCodeCategoryRepository
}

func (f *RepositoryFactory) GetDiscountCodeCollectionScopeRepository() repository.DiscountCodeCollectionScopeRepository {
	f.initialize()
	return f.discountCodeCollectionRepository
}
