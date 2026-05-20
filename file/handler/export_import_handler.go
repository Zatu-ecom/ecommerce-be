package handler

import (
	baseHandler "ecommerce-be/common/handler"

	"github.com/gin-gonic/gin"
)

type ExportImportHandler struct {
	*baseHandler.BaseHandler
}

func NewExportImportHandler() *ExportImportHandler {
	return &ExportImportHandler{
		BaseHandler: baseHandler.NewBaseHandler(),
	}
}

// CreateImportJob handles POST /imports
func (h *ExportImportHandler) CreateImportJob(c *gin.Context) {}

// GetImportJob handles GET /imports/{jobId}
func (h *ExportImportHandler) GetImportJob(c *gin.Context) {}

// CreateExportJob handles POST /exports
func (h *ExportImportHandler) CreateExportJob(c *gin.Context) {}

// GetExportJob handles GET /exports/{jobId}
func (h *ExportImportHandler) GetExportJob(c *gin.Context) {}
