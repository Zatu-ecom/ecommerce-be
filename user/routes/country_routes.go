package routes

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/user/factory/singleton"
	"ecommerce-be/user/handler"

	"github.com/gin-gonic/gin"
)

// CountryModule handles country-related routes
type CountryModule struct {
	countryHandler         *handler.CountryHandler
	countryCurrencyHandler *handler.CountryCurrencyHandler
}

// NewCountryModule creates a new instance of CountryModule
func NewCountryModule() *CountryModule {
	f := singleton.GetInstance()
	return &CountryModule{
		countryHandler:         f.GetCountryHandler(),
		countryCurrencyHandler: f.GetCountryCurrencyHandler(),
	}
}

// RegisterRoutes registers all country-related routes
func (m *CountryModule) RegisterRoutes(router *gin.Engine) {
	// ========================================
	// PUBLIC ROUTES - No authentication required
	// ========================================

	// GET /api/user/country - List active countries (with currencies)
	// Query params: ?region=Asia&page=1&limit=20
	// Response: CountryListWithCurrenciesResponse
	publicRoutes := router.Group(constants.APIBaseUser + "/country")
	{
		publicRoutes.GET("", m.countryHandler.ListActiveCountries)

		// GET /api/user/country/:id - Get country by ID (with currencies)
		// Response: CountryDetailResponse
		publicRoutes.GET("/:id", m.countryHandler.GetCountryByID)
	}

	// ========================================
	// ADMIN ROUTES - Admin authentication required
	// ========================================

	adminAuth := middleware.AdminAuth()
	adminRoutes := router.Group(constants.APIBaseUser + "/admin/country")
	adminRoutes.Use(adminAuth)
	{
		// GET /api/user/admin/country - List all countries (including inactive)
		// Query params: ?region=Asia&isActive=false&page=1&limit=20
		// Response: CountryListResponse
		adminRoutes.GET("", m.countryHandler.ListAllCountries)

		// GET /api/user/admin/country/:id - Get country by ID (admin view)
		// Response: CountryDetailResponse
		adminRoutes.GET("/:id", m.countryHandler.GetCountryByIDAdmin)

		// POST /api/user/admin/country - Create new country
		// Request: CountryCreateRequest
		// Response: CountryResponse
		adminRoutes.POST("", m.countryHandler.CreateCountry)

		// PUT /api/user/admin/country/:id - Update country (including deactivation)
		// Request: CountryUpdateRequest
		// Response: CountryResponse
		adminRoutes.PUT("/:id", m.countryHandler.UpdateCountry)

		// DELETE /api/user/admin/country/:id - Hard delete country
		// Response: { "message": "Country deleted successfully" }
		adminRoutes.DELETE("/:id", m.countryHandler.DeleteCountry)

		// ========================================
		// COUNTRY-CURRENCY MAPPING ROUTES
		// ========================================

		// GET /api/user/admin/country/:countryId/currency - List currencies for a country
		// Response: CountryCurrencyListResponse
		adminRoutes.GET("/:countryId/currency", m.countryCurrencyHandler.ListCountryCurrencies)

		// POST /api/user/admin/country/:countryId/currency - Add currency to country
		// Request: CountryCurrencyCreateRequest
		// Response: CountryCurrencySimpleResponse
		adminRoutes.POST("/:countryId/currency", m.countryCurrencyHandler.AddCurrencyToCountry)

		// POST /api/user/admin/country/:countryId/currency/bulk - Add multiple currencies to country
		// Request: CountryCurrencyBulkRequest
		// Response: []CountryCurrencySimpleResponse
		adminRoutes.POST("/:countryId/currency/bulk", m.countryCurrencyHandler.BulkAddCurrenciesToCountry)

		// PUT /api/user/admin/country/:countryId/currency/:currencyId - Update mapping (set primary)
		// Request: CountryCurrencyUpdateRequest
		// Response: CountryCurrencySimpleResponse
		adminRoutes.PUT("/:countryId/currency/:currencyId", m.countryCurrencyHandler.UpdateCountryCurrency)

		// DELETE /api/user/admin/country/:countryId/currency/:currencyId - Remove currency from country
		// Response: { "message": "Currency removed from country" }
		adminRoutes.DELETE("/:countryId/currency/:currencyId", m.countryCurrencyHandler.RemoveCurrencyFromCountry)
	}
}
