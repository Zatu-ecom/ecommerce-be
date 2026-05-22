package singleton

import (
	"sync"

	"ecommerce-be/common/cache"
	msgFactory "ecommerce-be/common/messaging/factory"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/file/service"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	configService service.ConfigService

	fileReadService   service.FileReadService
	fileDeleteService service.FileDeleteService
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
		f.fileReadService = service.NewFileReadService(fileUploadRepo, configRepo)

		// Create infrastructure dependencies for upload
		redisClient, _ := cache.GetRedisClient()
		sched := scheduler.New(redisClient)

		f.uploadExpiryScheduler = service.NewUploadExpiryScheduler(sched)
		f.fileDeleteService = service.NewFileDeleteService(
			fileUploadRepo,
			configRepo,
			f.uploadExpiryScheduler,
		)

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

// GetFileReadService returns the singleton file read service.
func (f *ServiceFactory) GetFileReadService() service.FileReadService {
	f.initialize()
	return f.fileReadService
}

// GetFileDeleteService returns the singleton file delete service.
func (f *ServiceFactory) GetFileDeleteService() service.FileDeleteService {
	f.initialize()
	return f.fileDeleteService
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
