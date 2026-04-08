package handler

import (
	"ecommerce-be/file/service"

	"github.com/gin-gonic/gin"
)

// FileHandler handles file upload, download, and asynchronous file job requests.
type FileHandler struct {
	fileService service.FileService
}

func NewFileHandler(fileService service.FileService) *FileHandler {
	return &FileHandler{
		fileService: fileService,
	}
}

// Bellow are the stub methods according to the API Design

// InitUpload handles POST /init-upload
func (h *FileHandler) InitUpload(c *gin.Context) {}

// CompleteUpload handles POST /complete-upload
func (h *FileHandler) CompleteUpload(c *gin.Context) {}

// GetFile handles GET /{fileId}
func (h *FileHandler) GetFile(c *gin.Context) {}

// GetDownloadURL handles GET /{fileId}/download-url
func (h *FileHandler) GetDownloadURL(c *gin.Context) {}

// DeleteFile handles DELETE /{fileId}
func (h *FileHandler) DeleteFile(c *gin.Context) {}

// RequestVariants handles POST /{fileId}/variants
func (h *FileHandler) RequestVariants(c *gin.Context) {}

// CreateImportJob handles POST /imports
func (h *FileHandler) CreateImportJob(c *gin.Context) {}

// GetImportJob handles GET /imports/{jobId}
func (h *FileHandler) GetImportJob(c *gin.Context) {}

// CreateExportJob handles POST /exports
func (h *FileHandler) CreateExportJob(c *gin.Context) {}

// GetExportJob handles GET /exports/{jobId}
func (h *FileHandler) GetExportJob(c *gin.Context) {}
