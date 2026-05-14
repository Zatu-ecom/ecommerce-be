package singleton

import (
	"sync"
	"ecommerce-be/file/repository"
)

// RepositoryFactory manages all repository singleton instances
type RepositoryFactory struct {
	fileUploadRepo repository.FileUploadRepository
	configRepo     repository.ConfigRepository

	once sync.Once
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading)
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		f.fileUploadRepo = repository.NewFileRepository()
		f.configRepo = repository.NewConfigRepository()
	})
}

// GetFileUploadRepository returns the singleton file upload repository
func (f *RepositoryFactory) GetFileUploadRepository() repository.FileUploadRepository {
	f.initialize()
	return f.fileUploadRepo
}

// GetConfigRepository returns the singleton config repository
func (f *RepositoryFactory) GetConfigRepository() repository.ConfigRepository {
	f.initialize()
	return f.configRepo
}
