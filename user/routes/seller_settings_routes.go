package routes

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"

	"github.com/gin-gonic/gin"
)

// SellerSettingsModule handles seller settings routes
type SellerSettingsModule struct {
	// TODO: Add handlers when implemented
	// sellerSettingsHandler *handler.SellerSettingsHandler
}

// NewSellerSettingsModule creates a new instance of SellerSettingsModule
func NewSellerSettingsModule() *SellerSettingsModule {
	// TODO: Get handlers from singleton factory
	// f := singleton.GetInstance()
	return &SellerSettingsModule{
		// sellerSettingsHandler: f.GetSellerSettingsHandler(),
	}
}

// RegisterRoutes registers all seller settings routes
func (m *SellerSettingsModule) RegisterRoutes(router *gin.Engine) {
	// ========================================
	// SELLER ROUTES - Seller authentication required
	// ========================================

	sellerAuth := middleware.SellerAuth()
	sellerRoutes := router.Group(constants.APIBaseUser + "/seller/settings")
	sellerRoutes.Use(sellerAuth)
	{
		// GET /api/user/seller/settings - Get current seller's settings
		// Response: SellerSettingsDetailResponse
		// Note: Returns 404 if settings don't exist yet (seller needs to complete onboarding)
		sellerRoutes.GET("") // TODO: m.sellerSettingsHandler.GetSellerSettings

		// POST /api/user/seller/settings - Create seller settings (onboarding)
		// Request: SellerSettingsCreateRequest
		// Response: SellerSettingsDetailResponse
		// Note: Can only be called once per seller
		sellerRoutes.POST("") // TODO: m.sellerSettingsHandler.CreateSellerSettings

		// PUT /api/user/seller/settings - Update seller settings
		// Request: SellerSettingsUpdateRequest
		// Response: SellerSettingsDetailResponse
		sellerRoutes.PUT("") // TODO: m.sellerSettingsHandler.UpdateSellerSettings
	}

	// ========================================
	// ADMIN ROUTES - Admin can view/manage any seller's settings
	// ========================================

	adminAuth := middleware.AdminAuth()
	adminRoutes := router.Group(constants.APIBaseUser + "/admin/seller-setting")
	adminRoutes.Use(adminAuth)
	{
		// GET /api/user/admin/seller-setting - List all seller settings
		// Query params: ?page=1&limit=20
		// Response: { "settings": []SellerSettingsDetailResponse, "pagination": {...} }
		adminRoutes.GET("") // TODO: m.sellerSettingsHandler.ListAllSellerSettings

		// GET /api/user/admin/seller-setting/:sellerId - Get specific seller's settings
		// Response: SellerSettingsDetailResponse
		adminRoutes.GET("/:sellerId") // TODO: m.sellerSettingsHandler.GetSellerSettingsBySellerID
	}
}
