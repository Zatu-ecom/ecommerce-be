package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/file/factory/singleton"
	"ecommerce-be/file/handler"

	"github.com/gin-gonic/gin"
)

// FileStorageConfigModule implements the Module interface for storage config routes.
type FileStorageConfigModule struct {
	configHandler *handler.ConfigHandler
}

// NewFileStorageConfigModule creates a new instance of FileStorageConfigModule.
func NewFileStorageConfigModule() *FileStorageConfigModule {
	f := singleton.GetInstance()
	return &FileStorageConfigModule{
		configHandler: f.GetConfigHandler(),
	}
}

// RegisterRoutes registers all storage config-related routes.
func (m *FileStorageConfigModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	fileRoutes := router.Group(constants.APIBaseFile)
	{
		fileRoutes.GET("/storage/providers", sellerAuth, m.configHandler.GetProviders)
		fileRoutes.POST("/storage-config/test", sellerAuth, m.configHandler.TestConfig)
		fileRoutes.POST("/storage-config", sellerAuth, m.configHandler.SaveConfig)
		fileRoutes.GET("/storage-config", sellerAuth, m.configHandler.ListConfigs)
		fileRoutes.POST("/storage-config/:id/activate", sellerAuth, m.configHandler.ActivateConfig)
		// NOTE: GET /storage-config/active is superseded by GET /storage-config (listing).
		// See research Decision 8.
	}
}
