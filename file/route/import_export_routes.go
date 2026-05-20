package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/file/handler"

	"github.com/gin-gonic/gin"
)

// FileImportExportModule implements the Module interface for import/export routes.
type FileImportExportModule struct {
	exportImportHandler *handler.ExportImportHandler
}

// NewFileImportExportModule creates a new instance of FileImportExportModule.
func NewFileImportExportModule() *FileImportExportModule {
	return &FileImportExportModule{
		exportImportHandler: handler.NewExportImportHandler(),
	}
}

// RegisterRoutes registers all import/export-related routes.
func (m *FileImportExportModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	fileRoutes := router.Group(constants.APIBaseFile)
	{
		fileRoutes.POST("/imports", sellerAuth, m.exportImportHandler.CreateImportJob)
		fileRoutes.GET("/imports/:jobId", sellerAuth, m.exportImportHandler.GetImportJob)
		fileRoutes.POST("/exports", sellerAuth, m.exportImportHandler.CreateExportJob)
		fileRoutes.GET("/exports/:jobId", sellerAuth, m.exportImportHandler.GetExportJob)
	}
}
