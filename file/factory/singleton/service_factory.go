package singleton

import (
	"context"
	"sync"

	"ecommerce-be/common/cache"
	msgFactory "ecommerce-be/common/messaging/factory"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/file/entity"
	"ecommerce-be/file/service"
	"ecommerce-be/file/service/blobAdapter"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	configService service.ConfigService

	fileUploadService     service.FileUploadService
	uploadExpiryScheduler service.UploadExpiryScheduler
	uploadExpiryHandler   *service.UploadExpiryHandler
	variantPublisher      service.VariantPublisher

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
		configRepo := f.repoFactory.GetConfigRepository()
		fileUploadRepo := f.repoFactory.GetFileUploadRepository()

		// Initialize services
		f.configService = service.NewConfigService(configRepo)

		// Create infrastructure dependencies for upload
		redisClient, _ := cache.GetRedisClient()
		sched := scheduler.New(redisClient)

		f.uploadExpiryScheduler = service.NewUploadExpiryScheduler(sched)

		mf, err := msgFactory.New("")
		if err == nil {
			if pub, err := mf.Publisher(); err == nil {
				f.variantPublisher = service.NewVariantPublisher(pub)
			}
		}

		f.fileUploadService = service.NewFileUploadService(
			fileUploadRepo,
			configRepo,
			f.uploadExpiryScheduler,
			f.variantPublisher,
			redisClient,
		)

		f.uploadExpiryHandler = service.NewUploadExpiryHandler(
			fileUploadRepo,
			configRepo,
		)
	})
}

// GetFileUploadService returns the singleton file upload service
func (f *ServiceFactory) GetFileUploadService() service.FileUploadService {
	f.initialize()
	return f.fileUploadService
}

// GetUploadExpiryScheduler returns the singleton upload expiry scheduler
func (f *ServiceFactory) GetUploadExpiryScheduler() service.UploadExpiryScheduler {
	f.initialize()
	return f.uploadExpiryScheduler
}

// GetUploadExpiryHandler returns the singleton upload expiry handler
func (f *ServiceFactory) GetUploadExpiryHandler() *service.UploadExpiryHandler {
	f.initialize()
	return f.uploadExpiryHandler
}

// GetVariantPublisher returns the singleton variant publisher
func (f *ServiceFactory) GetVariantPublisher() service.VariantPublisher {
	f.initialize()
	return f.variantPublisher
}

// GetConfigService returns the singleton config service
func (f *ServiceFactory) GetConfigService() service.ConfigService {
	f.initialize()
	return f.configService
}

// GetBlobAdapter constructs a BlobAdapter from the supplied StorageConfig.
// cfg.Provider must be preloaded (GORM Preload or equivalent) so that
// cfg.Provider.AdapterType is populated before this method is called.
// Returns a categorised error from file/error on decryption or init failure.
func (f *ServiceFactory) GetBlobAdapter(
	ctx context.Context,
	cfg entity.StorageConfig,
) (blobAdapter.BlobAdapter, error) {
	return blobAdapter.NewAdapterFromConfig(ctx, cfg)
}
