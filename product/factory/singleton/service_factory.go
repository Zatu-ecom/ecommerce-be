package singleton

import (
	"sync"

	fileSingleton "ecommerce-be/file/factory/singleton"
	"ecommerce-be/product/service"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	categoryService          service.CategoryService
	attributeService         service.AttributeDefinitionService
	productService           service.ProductService
	productQueryService      service.ProductQueryService
	variantService           service.VariantService
	variantQueryService      service.VariantQueryService
	variantBulkService       service.VariantBulkService
	productAttributeService  service.ProductAttributeService
	productOptionService     service.ProductOptionService
	optionValueService       service.ProductOptionValueService
	validatorService         service.ProductValidatorService
	wishlistService          service.WishlistService
	wishlistItemService      service.WishlistItemService
	collectionProductService service.CollectionProductService
	productMediaService      service.ProductMediaService
	variantMediaService      service.VariantMediaService

	once sync.Once
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(repoFactory *RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{repoFactory: repoFactory}
}

// initialize creates all service instances (lazy loading)
func (f *ServiceFactory) initialize() {
	f.once.Do(func() {
		// Get repositories
		categoryRepo := f.repoFactory.GetCategoryRepository()
		attributeRepo := f.repoFactory.GetAttributeDefinitionRepository()
		productRepo := f.repoFactory.GetProductRepository()
		variantRepo := f.repoFactory.GetVariantRepository()
		optionRepo := f.repoFactory.GetProductOptionRepository()
		productAttrRepo := f.repoFactory.GetProductAttributeRepository()

		// Initialize validator service first (used by other services)
		f.validatorService = service.NewProductValidatorService(productRepo)

		// Initialize product option service (used by variant services)
		f.productOptionService = service.NewProductOptionService(optionRepo, f.validatorService)
		f.optionValueService = service.NewProductOptionValueService(
			optionRepo,
			productRepo,
			f.validatorService,
		)

		// Initialize WishlistItemService (needed by VariantQueryService)
		f.wishlistItemService = service.NewWishlistItemService(
			f.repoFactory.GetWishlistItemRepository(),
			f.repoFactory.GetWishlistRepository(),
		)

		// Initialize ProductFileGateway early — both VariantMediaService and
		// ProductMediaService depend on it, and VariantQueryService must be
		// initialized before VariantService.
		fileFact := fileSingleton.GetInstance()
		fileGateway := service.NewProductFileGateway(
			fileFact.GetFileReadService(),
			fileFact.GetFileDeleteService(),
		)

		// Initialize VariantMediaService BEFORE VariantQueryService so it can be
		// injected into the query service for embedding media in variant responses.
		f.variantMediaService = service.NewVariantMediaService(
			f.repoFactory.GetVariantMediaRepository(),
			variantRepo,
			productRepo,
			fileGateway,
		)

		// Initialize VariantQueryService with VariantMediaService dependency
		f.variantQueryService = service.NewVariantQueryService(
			variantRepo,
			f.wishlistItemService,
			f.productOptionService,
			f.validatorService,
			f.variantMediaService,
		)

		// Initialize VariantService with VariantQueryService dependency
		f.variantService = service.NewVariantService(
			variantRepo,
			f.productOptionService,
			f.validatorService,
			f.variantQueryService,
		)

		// Initialize VariantBulkService for bulk operations
		f.variantBulkService = service.NewVariantBulkService(
			variantRepo,
			f.productOptionService,
			f.validatorService,
		)

		f.categoryService = service.NewCategoryService(categoryRepo, productRepo, attributeRepo)
		f.attributeService = service.NewAttributeDefinitionService(attributeRepo)
		f.productAttributeService = service.NewProductAttributeService(
			productAttrRepo,
			productRepo,
			attributeRepo,
			f.validatorService,
		)

		// Initialize CollectionProductService
		f.collectionProductService = service.NewCollectionProductService(
			f.repoFactory.GetCollectionProductRepository(),
		)

		// Initialize ProductMediaService BEFORE ProductQueryService so it can be
		// injected into the query service for embedding media in product responses.
		f.productMediaService = service.NewProductMediaService(
			f.repoFactory.GetProductMediaRepository(),
			productRepo,
			fileGateway,
		)

		// Initialize ProductQueryService with VariantQueryService and media service
		f.productQueryService = service.NewProductQueryService(
			productRepo,
			f.variantQueryService,
			f.categoryService,
			f.productAttributeService,
			f.productOptionService,
			f.productMediaService,
		)

		// Initialize WishlistService (needs ProductQueryService for product details)
		f.wishlistService = service.NewWishlistService(
			f.repoFactory.GetWishlistRepository(),
			f.repoFactory.GetWishlistItemRepository(),
			f.productQueryService,
		)

		// Initialize ProductService with its dependencies
		f.productService = service.NewProductService(
			productRepo,
			categoryRepo,
			f.productQueryService,
			f.validatorService,
			f.variantService,
			f.variantBulkService,
			f.productOptionService,
			f.productAttributeService,
		)
	})
}

// GetCategoryService returns the singleton category service
func (f *ServiceFactory) GetCategoryService() service.CategoryService {
	f.initialize()
	return f.categoryService
}

// GetAttributeDefinitionService returns the singleton attribute service
func (f *ServiceFactory) GetAttributeDefinitionService() service.AttributeDefinitionService {
	f.initialize()
	return f.attributeService
}

// GetProductService returns the singleton product service
func (f *ServiceFactory) GetProductService() service.ProductService {
	f.initialize()
	return f.productService
}

// GetProductQueryService returns the singleton product query service
func (f *ServiceFactory) GetProductQueryService() service.ProductQueryService {
	f.initialize()
	return f.productQueryService
}

// GetVariantService returns the singleton variant service
func (f *ServiceFactory) GetVariantService() service.VariantService {
	f.initialize()
	return f.variantService
}

// GetVariantQueryService returns the singleton variant query service
func (f *ServiceFactory) GetVariantQueryService() service.VariantQueryService {
	f.initialize()
	return f.variantQueryService
}

// GetVariantBulkService returns the singleton variant bulk service
func (f *ServiceFactory) GetVariantBulkService() service.VariantBulkService {
	f.initialize()
	return f.variantBulkService
}

// GetProductAttributeService returns the singleton product attribute service
func (f *ServiceFactory) GetProductAttributeService() service.ProductAttributeService {
	f.initialize()
	return f.productAttributeService
}

// GetProductOptionService returns the singleton product option service
func (f *ServiceFactory) GetProductOptionService() service.ProductOptionService {
	f.initialize()
	return f.productOptionService
}

// GetProductOptionValueService returns the singleton product option value service
func (f *ServiceFactory) GetProductOptionValueService() service.ProductOptionValueService {
	f.initialize()
	return f.optionValueService
}

// GetProductValidatorService returns the singleton product validator service
func (f *ServiceFactory) GetProductValidatorService() service.ProductValidatorService {
	f.initialize()
	return f.validatorService
}

// GetWishlistService returns the singleton wishlist service
func (f *ServiceFactory) GetWishlistService() service.WishlistService {
	f.initialize()
	return f.wishlistService
}

// GetWishlistItemService returns the singleton wishlist item service
func (f *ServiceFactory) GetWishlistItemService() service.WishlistItemService {
	f.initialize()
	return f.wishlistItemService
}

func (f *ServiceFactory) GetCollectionProductService() service.CollectionProductService {
	f.initialize()
	return f.collectionProductService
}

// GetProductMediaService returns the singleton product-media service.
func (f *ServiceFactory) GetProductMediaService() service.ProductMediaService {
	f.initialize()
	return f.productMediaService
}

// GetVariantMediaService returns the singleton variant-media service.
func (f *ServiceFactory) GetVariantMediaService() service.VariantMediaService {
	f.initialize()
	return f.variantMediaService
}
