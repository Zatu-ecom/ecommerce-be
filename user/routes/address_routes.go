package routes

import (
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	"ecommerce-be/user/handlers"
	"ecommerce-be/user/repositories"
	"ecommerce-be/user/service"

	"github.com/gin-gonic/gin"
)

type AddressModule struct {
	addressHandler *handlers.AddressHandler
}

func NewAddressModule() *AddressModule {
	addressRepo := repositories.NewAddressRepository(db.GetDB())
	addressService := service.NewAddressService(addressRepo)

	return &AddressModule{
		addressHandler: handlers.NewAddressHandler(addressService),
	}
}

// RegisterRoutes registers all user-related routes
func (m *AddressModule) RegisterRoutes(router *gin.Engine) {
	// Auth middleware for protected routes
	auth := middleware.CustomerAuth()

	// Address routes (protected)
	userRoutes := router.Group("/api/users")
	{
		userRoutes.GET("/addresses", auth, m.addressHandler.GetAddresses)
		userRoutes.POST("/addresses", auth, m.addressHandler.AddAddress)
		userRoutes.PUT("/addresses/:id", auth, m.addressHandler.UpdateAddress)
		userRoutes.DELETE("/addresses/:id", auth, m.addressHandler.DeleteAddress)
		userRoutes.PATCH("/addresses/:id/default", auth, m.addressHandler.SetDefaultAddress)
	}
}
