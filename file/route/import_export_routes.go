package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/file/factory/singleton"
	"ecommerce-be/file/handler"

	"github.com/gin-gonic/gin"
)

// FileImportExportModule implements the Module interface for import/export routes.
type FileImportExportModule struct {
	fileHandler *handler.FileHandler
}

// NewFileImportExportModule creates a new instance of FileImportExportModule.
func NewFileImportExportModule() *FileImportExportModule {
	f := singleton.GetInstance()
	return &FileImportExportModule{
		fileHandler: f.GetFileHandler(),
	}
}

// RegisterRoutes registers all import/export-related routes.
func (m *FileImportExportModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	fileRoutes := router.Group(constants.APIBaseFile)
	{
		fileRoutes.POST("/imports", sellerAuth, m.fileHandler.CreateImportJob)
		fileRoutes.GET("/imports/:jobId", sellerAuth, m.fileHandler.GetImportJob)
		fileRoutes.POST("/exports", sellerAuth, m.fileHandler.CreateExportJob)
		fileRoutes.GET("/exports/:jobId", sellerAuth, m.fileHandler.GetExportJob)
	}
}
