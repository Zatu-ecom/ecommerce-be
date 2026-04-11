package handler

import (
	"net/http"

	"ecommerce-be/common"
	"ecommerce-be/common/auth"
	commonError "ecommerce-be/common/error"
	baseHandler "ecommerce-be/common/handler"
	"ecommerce-be/file/model"
	"ecommerce-be/file/service"
	"ecommerce-be/file/utils/constant"

	"github.com/gin-gonic/gin"
)

// ConfigHandler handles storage configuration APIs for the file module.
type ConfigHandler struct {
	*baseHandler.BaseHandler
	configService service.ConfigService
}

func NewConfigHandler(configService service.ConfigService) *ConfigHandler {
	return &ConfigHandler{
		BaseHandler:   baseHandler.NewBaseHandler(),
		configService: configService,
	}
}

// GetProviders handles GET /storage/providers
func (h *ConfigHandler) GetProviders(c *gin.Context) {
	providers, err := h.configService.GetProviders(c.Request.Context())
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_FETCH_PROVIDERS_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.FILE_PROVIDERS_FETCHED_MSG, providers)
}

// SaveConfig handles POST /storage-config
func (h *ConfigHandler) SaveConfig(c *gin.Context) {
	var req model.SaveConfigRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		h.HandleError(c, commonError.UnauthorizedError, constant.FILE_AUTH_REQUIRED_MSG)
		return
	}

	_, roleName, exists := auth.GetUserRoleFromContext(c)
	if !exists {
		h.HandleError(c, commonError.ErrRoleDataMissing, constant.FILE_ROLE_DATA_MISSING_MSG)
		return
	}

	res, err := h.configService.SaveConfig(c.Request.Context(), userID, roleName, req)
	if err != nil {
		h.HandleError(c, err, constant.FAILED_TO_SAVE_CONFIG_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.FILE_CONFIG_SAVED_MSG, res)
}

// TestConfig handles POST /storage-config/test
func (h *ConfigHandler) TestConfig(c *gin.Context) {
	common.ErrorWithCode(
		c,
		http.StatusNotImplemented,
		constant.FILE_CONFIG_NOT_IMPLEMENTED_MSG,
		constant.FILE_NOT_IMPLEMENTED_CODE,
	)
}

// ActivateConfig handles POST /storage-config/{id}/activate
func (h *ConfigHandler) ActivateConfig(c *gin.Context) {
	common.ErrorWithCode(
		c,
		http.StatusNotImplemented,
		constant.FILE_ACTIVATE_NOT_IMPLEMENTED_MSG,
		constant.FILE_NOT_IMPLEMENTED_CODE,
	)
}

// GetActiveConfig handles GET /storage-config/active
func (h *ConfigHandler) GetActiveConfig(c *gin.Context) {
	common.ErrorWithCode(
		c,
		http.StatusNotImplemented,
		constant.FILE_ACTIVE_NOT_IMPLEMENTED_MSG,
		constant.FILE_NOT_IMPLEMENTED_CODE,
	)
}
