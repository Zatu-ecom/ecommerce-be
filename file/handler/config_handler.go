package handler

import (
	"ecommerce-be/file/model"
	"ecommerce-be/file/service"

	"github.com/gin-gonic/gin"
)

// ConfigHandler handles storage configuration APIs for the file module.
type ConfigHandler struct {
	configService service.ConfigService
}

func NewConfigHandler(configService service.ConfigService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

// GetProviders handles GET /storage/providers
func (h *ConfigHandler) GetProviders(c *gin.Context) {
	providers, err := h.configService.GetProviders(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "data": providers})
}

// SaveConfig handles POST /storage-config
func (h *ConfigHandler) SaveConfig(c *gin.Context) {
	var req model.SaveConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Retrieve user details from context (usually set by middleware)
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	roleVal, exists := c.Get("userRole")
	role := "SELLER" // Default or parse from context
	if exists {
		role = roleVal.(string)
	}

	res, err := h.configService.SaveConfig(c.Request.Context(), userID, role, req)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": res})
}

// TestConfig handles POST /storage-config/test
func (h *ConfigHandler) TestConfig(c *gin.Context) {
	c.JSON(200, gin.H{"success": true, "message": "Not implemented yet"})
}

// ActivateConfig handles POST /storage-config/{id}/activate
func (h *ConfigHandler) ActivateConfig(c *gin.Context) {
	c.JSON(200, gin.H{"success": true, "message": "Not implemented yet"})
}

// GetActiveConfig handles GET /storage-config/active
func (h *ConfigHandler) GetActiveConfig(c *gin.Context) {
	c.JSON(200, gin.H{"success": true, "message": "Not implemented yet"})
}
