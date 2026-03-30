package routes

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/user/factory/singleton"
	"ecommerce-be/user/handler"

	"github.com/gin-gonic/gin"
)

// CurrencyModule handles currency-related routes
type CurrencyModule struct {
	currencyHandler *handler.CurrencyHandler
}

// NewCurrencyModule creates a new instance of CurrencyModule
func NewCurrencyModule() *CurrencyModule {
	f := singleton.GetInstance()
	return &CurrencyModule{
		currencyHandler: f.GetCurrencyHandler(),
	}
}

// RegisterRoutes registers all currency-related routes
func (m *CurrencyModule) RegisterRoutes(router *gin.Engine) {
	// ========================================
	// PUBLIC ROUTES - No authentication required
	// ========================================

	// GET /api/user/currency - List active currencies
	// Query params: ?page=1&limit=20
	// Response: CurrencyListResponse
	publicRoutes := router.Group(constants.APIBaseUser + "/currency")
	{
		publicRoutes.GET("") // TODO: m.currencyHandler.ListActiveCurrencies

		// GET /api/user/currency/:id - Get currency by ID
		// Response: CurrencyResponse
		publicRoutes.GET("/:id") // TODO: m.currencyHandler.GetCurrencyByID
	}

	// ========================================
	// ADMIN ROUTES - Admin authentication required
	// ========================================

	adminAuth := middleware.AdminAuth()
	adminRoutes := router.Group(constants.APIBaseUser + "/admin/currency")
	adminRoutes.Use(adminAuth)
	{
		// GET /api/user/admin/currency - List all currencies (including inactive)
		// Query params: ?isActive=false&page=1&limit=20
		// Response: CurrencyListResponse
		adminRoutes.GET("", m.currencyHandler.ListAllCurrencies)

		// GET /api/user/admin/currency/:id - Get currency by ID (admin view)
		// Response: CurrencyDetailResponse
		adminRoutes.GET("/:id", m.currencyHandler.GetCurrencyByIDAdmin)

		// POST /api/user/admin/currency - Create new currency
		// Request: CurrencyCreateRequest
		// Response: CurrencyResponse
		adminRoutes.POST("", m.currencyHandler.CreateCurrency)

		// PUT /api/user/admin/currency/:id - Update currency (including deactivation)
		// Request: CurrencyUpdateRequest
		// Response: CurrencyResponse
		adminRoutes.PUT("/:id", m.currencyHandler.UpdateCurrency)

		// DELETE /api/user/admin/currency/:id - Hard delete currency
		// Response: { "message": "Currency deleted successfully" }
		adminRoutes.DELETE("/:id", m.currencyHandler.DeleteCurrency)
	}
}
