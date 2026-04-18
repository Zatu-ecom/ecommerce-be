package singleton

import (
	"context"
	"sync"

	"ecommerce-be/file/entity"
	"ecommerce-be/file/service"
	"ecommerce-be/file/service/blob_adapter"
)

// ServiceFactory manages all service singleton instances
type ServiceFactory struct {
	repoFactory *RepositoryFactory

	fileService   service.FileService
	configService service.ConfigService

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
		fileRepo := f.repoFactory.GetFileRepository()
		configRepo := f.repoFactory.GetConfigRepository()

		// Initialize services
		f.fileService = service.NewFileService(fileRepo)
		f.configService = service.NewConfigService(configRepo)
	})
}

// GetFileService returns the singleton file service
func (f *ServiceFactory) GetFileService() service.FileService {
	f.initialize()
	return f.fileService
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
func (f *ServiceFactory) GetBlobAdapter(ctx context.Context, cfg entity.StorageConfig) (blob_adapter.BlobAdapter, error) {
	return blob_adapter.NewAdapterFromConfig(ctx, cfg)
}
