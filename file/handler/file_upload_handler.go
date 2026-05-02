package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"ecommerce-be/common"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	baseHandler "ecommerce-be/common/handler"
	"ecommerce-be/common/log"
	"ecommerce-be/file/model"
	"ecommerce-be/file/service"
	"ecommerce-be/file/utils"
	"ecommerce-be/file/utils/constant"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// FileUploadHandler handles init-upload and complete-upload flow.
type FileUploadHandler struct {
	*baseHandler.BaseHandler
	uploadService service.FileUploadService
}

// NewFileUploadHandler creates a new instance of FileUploadHandler.
func NewFileUploadHandler(uploadService service.FileUploadService) *FileUploadHandler {
	return &FileUploadHandler{
		BaseHandler:   baseHandler.NewBaseHandler(),
		uploadService: uploadService,
	}
}

// InitUpload handles POST /init-upload
func (h *FileUploadHandler) InitUpload(c *gin.Context) {
	correlationID := c.GetHeader(constants.CORRELATION_ID_HEADER)
	if correlationID == "" {
		h.HandleError(c, commonError.ErrCorrelationIDMissing, "X-Correlation-ID header is missing")
		return
	}

	principal, err := utils.ExtractPrincipal(c)
	if err != nil {
		h.HandleError(c, err, constant.FILE_UPLOAD_INTERNAL_MSG)
		return
	}

	var req model.InitUploadRequest
	// bindJSON rejects unknown fields natively when DisallowUnknownFields is implemented.
	// We'll use gin's ShouldBindBodyWith or similar, but the strict way is:
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decErr := decoder.Decode(&req); decErr != nil {
		h.HandleValidationError(c, decErr)
		return
	}

	// Read idempotency key from header if present
	idemKey := c.GetHeader("Idempotency-Key")
	var idempotencyKey *string
	if idemKey != "" {
		if !utils.ValidateIdempotencyKey(idemKey) {
			common.ErrorWithValidation(
				c,
				http.StatusBadRequest,
				constants.VALIDATION_FAILED_MSG,
				[]common.ValidationError{
					{
						Field:   "Idempotency-Key",
						Message: "Idempotency-Key must be 8..128 characters and contain only A-Z, a-z, 0-9, '.', '_', '~', or '-'",
					},
				},
				constants.VALIDATION_ERROR_CODE,
			)
			return
		}
		idempotencyKey = &idemKey
	}

	// Actually validate standard gin tags as Decode skips validator tags
	if valErr := binding.Validator.ValidateStruct(req); valErr != nil {
		h.HandleValidationError(c, valErr)
		return
	}

	log.DebugWithContext(
		c,
		"file upload init request received"+
			" action=initUpload"+
			" actorRole="+principal.Role+
			" purpose="+string(req.Purpose)+
			" mimeType="+req.MimeType,
	)

	ctx := h.withPrincipalContext(c, principal)
	res, svcErr := h.uploadService.InitUpload(ctx, principal, req, idempotencyKey)
	if svcErr != nil {
		h.HandleError(c, svcErr, constant.FILE_UPLOAD_INTERNAL_MSG)
		return
	}

	log.InfoWithContext(
		c,
		"file upload initialised"+
			" action=initUpload"+
			" actorRole="+principal.Role+
			" fileId="+res.FileID+
			" objectKey="+res.ObjectKey,
	)

	status := http.StatusCreated
	if res.Replayed {
		status = http.StatusOK
	}
	h.Success(c, status, "Upload initialised", res)
}

// CompleteUpload handles POST /complete-upload
func (h *FileUploadHandler) CompleteUpload(c *gin.Context) {
	correlationID := c.GetHeader(constants.CORRELATION_ID_HEADER)
	if correlationID == "" {
		h.HandleError(c, commonError.ErrCorrelationIDMissing, "X-Correlation-ID header is missing")
		return
	}

	principal, err := utils.ExtractPrincipal(c)
	if err != nil {
		h.HandleError(c, err, constant.FILE_UPLOAD_INTERNAL_MSG)
		return
	}

	var req model.CompleteUploadRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if decErr := decoder.Decode(&req); decErr != nil {
		h.HandleValidationError(c, decErr)
		return
	}

	if valErr := binding.Validator.ValidateStruct(req); valErr != nil {
		h.HandleValidationError(c, valErr)
		return
	}

	log.DebugWithContext(
		c,
		"file upload complete request received"+
			" action=completeUpload"+
			" actorRole="+principal.Role+
			" fileId="+req.FileID,
	)

	ctx := h.withPrincipalContext(c, principal)
	res, svcErr := h.uploadService.CompleteUpload(ctx, principal, req)
	if svcErr != nil {
		h.HandleError(c, svcErr, constant.FILE_UPLOAD_INTERNAL_MSG)
		return
	}

	log.InfoWithContext(
		c,
		"file upload completed"+
			" action=completeUpload"+
			" actorRole="+principal.Role+
			" fileId="+res.FileID+
			" status="+res.Status,
	)

	h.Success(c, http.StatusOK, "Upload completed", res)
}

func (h *FileUploadHandler) withPrincipalContext(
	c *gin.Context,
	principal utils.Principal,
) context.Context {
	ctx := c.Request.Context()
	ctx = context.WithValue(
		ctx,
		constants.CORRELATION_ID_KEY,
		c.GetHeader(constants.CORRELATION_ID_HEADER),
	)
	ctx = context.WithValue(ctx, constants.USER_ID_KEY, uint(principal.UserID))
	ctx = context.WithValue(ctx, constants.ROLE_NAME_KEY, principal.Role)
	if principal.SellerID != nil {
		ctx = context.WithValue(ctx, constants.SELLER_ID_KEY, uint(*principal.SellerID))
	}
	return ctx
}
