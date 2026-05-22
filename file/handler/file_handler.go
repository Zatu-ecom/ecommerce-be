package handler

import (
	"net/http"

	baseHandler "ecommerce-be/common/handler"
	"ecommerce-be/common/log"
	"ecommerce-be/file/model"
	"ecommerce-be/file/service"
	"ecommerce-be/file/utils"
	"ecommerce-be/file/utils/constant"

	"github.com/gin-gonic/gin"
)

// FileHandler handles file upload, download, and asynchronous file job requests.
type FileHandler struct {
	*baseHandler.BaseHandler
	fileReadService   service.FileReadService
	fileDeleteService service.FileDeleteService
}

func NewFileHandler(
	fileReadService service.FileReadService,
	fileDeleteService service.FileDeleteService,
) *FileHandler {
	return &FileHandler{
		BaseHandler:       baseHandler.NewBaseHandler(),
		fileReadService:   fileReadService,
		fileDeleteService: fileDeleteService,
	}
}

// Bellow are the stub methods according to the API Design

// GetAllFiles handles GET /api/file
func (h *FileHandler) GetAllFiles(c *gin.Context) {
	principal, appErr := utils.ExtractPrincipal(c)
	if appErr != nil {
		h.HandleError(c, appErr, constant.FILE_NOT_FOUND_MSG)
		return
	}

	var params model.GetFilesParam
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	res, err := h.fileReadService.GetAllFiles(
		c,
		principal,
		params.ToFilter(),
	)
	if err != nil {
		h.HandleError(c, err, constant.FILE_NOT_FOUND_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.FILE_LIST_SUCCESS_MSG, res)
}

// GetFile handles GET /{fileId}
func (h *FileHandler) GetFile(c *gin.Context) {
	principal, appErr := utils.ExtractPrincipal(c)
	if appErr != nil {
		h.HandleError(c, appErr, constant.FILE_NOT_FOUND_MSG)
		return
	}

	var query model.GetFileQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	res, err := h.fileReadService.GetFile(
		c,
		principal,
		c.Param("fileId"),
		query,
	)
	if err != nil {
		h.HandleError(c, err, constant.FILE_NOT_FOUND_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.FILE_GET_SUCCESS_MSG, res)
}

// GetDownloadURL handles GET /{fileId}/download-url
func (h *FileHandler) GetDownloadURL(c *gin.Context) {
	principal, appErr := utils.ExtractPrincipal(c)
	if appErr != nil {
		h.HandleError(c, appErr, constant.FILE_DOWNLOAD_URL_SUCCESS_MSG)
		return
	}

	var query model.DownloadURLQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	res, err := h.fileReadService.GetDownloadURL(
		c,
		principal,
		c.Param("fileId"),
		query,
	)
	if err != nil {
		h.HandleError(c, err, constant.FILE_DOWNLOAD_URL_SUCCESS_MSG)
		return
	}

	log.InfoWithContext(
		c,
		"file download url generated"+
			" action=getDownloadURL"+
			" fileId="+res.FileID,
	)

	h.Success(c, http.StatusOK, constant.FILE_DOWNLOAD_URL_SUCCESS_MSG, res)
}

// DeleteFile handles DELETE /{fileId}
func (h *FileHandler) DeleteFile(c *gin.Context) {
	principal, appErr := utils.ExtractPrincipal(c)
	if appErr != nil {
		h.HandleError(c, appErr, constant.FILE_DELETE_SUCCESS_MSG)
		return
	}

	res, err := h.fileDeleteService.DeleteFile(
		c,
		principal,
		c.Param("fileId"),
	)
	if err != nil {
		h.HandleError(c, err, constant.FILE_DELETE_SUCCESS_MSG)
		return
	}

	log.InfoWithContext(
		c,
		"file deleted"+
			" action=deleteFile"+
			" fileId="+res.FileID,
	)

	h.Success(c, http.StatusOK, constant.FILE_DELETE_SUCCESS_MSG, res)
}

// RequestVariants handles POST /{fileId}/variants
func (h *FileHandler) RequestVariants(c *gin.Context) {}
