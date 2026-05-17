package route

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/file/factory/singleton"
	"ecommerce-be/file/handler"

	"github.com/gin-gonic/gin"
)

// FileOperationModule implements the Module interface for file operation routes.
type FileOperationModule struct {
	fileHandler   *handler.FileHandler
	uploadHandler *handler.FileUploadHandler
}

// NewFileOperationModule creates a new instance of FileOperationModule.
func NewFileOperationModule() *FileOperationModule {
	f := singleton.GetInstance()
	return &FileOperationModule{
		fileHandler:   f.GetFileHandler(),
		uploadHandler: f.GetFileUploadHandler(),
	}
}

// RegisterRoutes registers all file operation-related routes.
func (m *FileOperationModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()
	// adminAuth := middleware.AdminAuth()

	// Seller endpoints for generic file operations
	fileRoutes := router.Group(constants.APIBaseFile)
	{
		fileRoutes.GET("", sellerAuth, m.fileHandler.GetAllFiles)
		fileRoutes.GET("/:fileId", sellerAuth, m.fileHandler.GetFile)
		fileRoutes.GET("/:fileId/download-url", sellerAuth, m.fileHandler.GetDownloadURL)
		fileRoutes.DELETE("/:fileId", sellerAuth, m.fileHandler.DeleteFile)
		fileRoutes.POST("/:fileId/variants", sellerAuth, m.fileHandler.RequestVariants)

		fileRoutes.POST("/init-upload", sellerAuth, m.uploadHandler.InitUpload)
		fileRoutes.POST("/complete-upload", sellerAuth, m.uploadHandler.CompleteUpload)
	}
}
