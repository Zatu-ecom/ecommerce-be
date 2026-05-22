package singleton

import (
	"sync"
	"ecommerce-be/file/handler"
)

// HandlerFactory manages all handler singleton instances
type HandlerFactory struct {
	serviceFactory *ServiceFactory

	fileHandler       *handler.FileHandler
	configHandler     *handler.ConfigHandler
	fileUploadHandler *handler.FileUploadHandler

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
		configService := f.serviceFactory.GetConfigService()
		fileReadService := f.serviceFactory.GetFileReadService()
		fileDeleteService := f.serviceFactory.GetFileDeleteService()
		fileUploadService := f.serviceFactory.GetFileUploadService()

		// Initialize handlers
		f.fileHandler = handler.NewFileHandler(fileReadService, fileDeleteService)
		f.configHandler = handler.NewConfigHandler(configService)
		f.fileUploadHandler = handler.NewFileUploadHandler(fileUploadService)
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

// GetFileUploadHandler returns the singleton file upload handler
func (f *HandlerFactory) GetFileUploadHandler() *handler.FileUploadHandler {
	f.initialize()
	return f.fileUploadHandler
}
