package singleton

import (
	"sync"
	"ecommerce-be/file/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	fileHandler   *handler.FileHandler
	configHandler *handler.ConfigHandler

	once sync.Once
}

// NewHandlerFactory creates a new handler factory
func NewHandlerFactory(serviceFactory *ServiceFactory) *HandlerFactory {
	return &HandlerFactory{serviceFactory: serviceFactory}
}

// initialize creates all handler instances (lazy loading)
func (f *HandlerFactory) initialize() {
	f.once.Do(func() {
		// Get services
		fileService := f.serviceFactory.GetFileService()
		configService := f.serviceFactory.GetConfigService()

		// Initialize handlers
		f.fileHandler = handler.NewFileHandler(fileService)
		f.configHandler = handler.NewConfigHandler(configService)
	})
}

// GetFileHandler returns the singleton file handler
func (f *HandlerFactory) GetFileHandler() *handler.FileHandler {
	f.initialize()
	return f.fileHandler
}

// GetConfigHandler returns the singleton config handler
func (f *HandlerFactory) GetConfigHandler() *handler.ConfigHandler {
	f.initialize()
	return f.configHandler
}
