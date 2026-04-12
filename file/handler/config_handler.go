package handler

import (
	"net/http"

	"ecommerce-be/common"
	"ecommerce-be/common/auth"
	"ecommerce-be/common/constants"
	commonError "ecommerce-be/common/error"
	baseHandler "ecommerce-be/common/handler"
	"ecommerce-be/file/entity"
	fileError "ecommerce-be/file/error"
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

// resolveAuthContext extracts userID and roleName from the gin context.
// Returns false and writes the error response if either is missing.
func (h *ConfigHandler) resolveAuthContext(c *gin.Context) (userID uint, roleName string, ok bool) {
	userID, ok = auth.GetUserIDFromContext(c)
	if !ok {
		h.HandleError(c, commonError.UnauthorizedError, constant.FILE_AUTH_REQUIRED_MSG)
		return userID, roleName, ok
	}
	_, roleName, ok = auth.GetUserRoleFromContext(c)
	if !ok {
		h.HandleError(c, commonError.ErrRoleDataMissing, constant.FILE_ROLE_DATA_MISSING_MSG)
	}
	return userID, roleName, ok
}

// GetProviders handles GET /storage/providers
func (h *ConfigHandler) GetProviders(c *gin.Context) {
	providers, err := h.configService.GetProviders(c)
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

	userID, roleName, ok := h.resolveAuthContext(c)
	if !ok {
		return
	}

	res, err := h.configService.SaveConfig(c, userID, roleName, req)
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
	configID, err := h.ParseUintParam(c, "id")
	if err != nil {
		h.HandleError(c, err, constant.FILE_CONFIG_INVALID_ID_MSG)
		return
	}

	userID, roleName, ok := h.resolveAuthContext(c)
	if !ok {
		return
	}

	res, err := h.configService.ActivateConfig(c, userID, roleName, configID)
	if err != nil {
		h.HandleError(c, err, constant.FILE_CONFIG_ACTIVATION_ERR_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.FILE_CONFIG_ACTIVATED_MSG, res)
}

// ListConfigs handles GET /storage-config
func (h *ConfigHandler) ListConfigs(c *gin.Context) {
	if c.Query(constant.FILE_LIST_SELLER_ID_FIELD) != "" {
		common.ErrorWithValidation(
			c,
			http.StatusBadRequest,
			constant.FILE_LIST_VALIDATION_ERR_MSG,
			[]common.ValidationError{{
				Field:   constant.FILE_LIST_SELLER_ID_FIELD,
				Message: constant.FILE_LIST_SELLER_ID_ERR_MSG,
			}},
			constant.FILE_LIST_VALIDATION_ERR_CODE,
		)
		return
	}

	var params model.ListStorageConfigQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		h.HandleValidationError(c, err)
		return
	}

	userID, roleName, ok := h.resolveAuthContext(c)
	if !ok {
		return
	}

	filter := params.ToFilter()
	filter.SetDefaults()

	switch roleName {
	case constants.SELLER_ROLE_NAME:
		filter.OwnerType = entity.OwnerTypeSeller
		filter.OwnerID = &userID
	case constants.ADMIN_ROLE_NAME:
		filter.OwnerType = entity.OwnerTypePlatform
	default:
		h.HandleError(c, fileError.ErrInvalidRole, constant.FILE_CONFIG_LIST_ERR_MSG)
		return
	}

	res, err := h.configService.ListConfigs(c, filter)
	if err != nil {
		h.HandleError(c, err, constant.FILE_CONFIG_LIST_ERR_MSG)
		return
	}

	h.Success(c, http.StatusOK, constant.FILE_CONFIG_LISTED_MSG, res)
}
