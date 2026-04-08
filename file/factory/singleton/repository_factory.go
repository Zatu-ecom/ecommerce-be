package singleton

import (
	"sync"
	"ecommerce-be/file/repository"
)

// RepositoryFactory manages all repository singleton instances
type RepositoryFactory struct {
	fileRepo   repository.FileRepository
	configRepo repository.ConfigRepository

	once sync.Once
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

// initialize creates all repository instances (lazy loading)
func (f *RepositoryFactory) initialize() {
	f.once.Do(func() {
		f.fileRepo = repository.NewFileRepository()
		f.configRepo = repository.NewConfigRepository()
	})
}

// GetFileRepository returns the singleton file repository
func (f *RepositoryFactory) GetFileRepository() repository.FileRepository {
	f.initialize()
	return f.fileRepo
}

// GetConfigRepository returns the singleton config repository
func (f *RepositoryFactory) GetConfigRepository() repository.ConfigRepository {
	f.initialize()
	return f.configRepo
}
