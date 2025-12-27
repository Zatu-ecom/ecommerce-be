package routes

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/user/factory/singleton"
	"ecommerce-be/user/handler"

	"github.com/gin-gonic/gin"
)

type AddressModule struct {
	addressHandler *handler.AddressHandler
}

func NewAddressModule() *AddressModule {
	f := singleton.GetInstance()

	return &AddressModule{
		addressHandler: f.GetAddressHandler(),
	}
}

// RegisterRoutes registers all user-related routes
func (m *AddressModule) RegisterRoutes(router *gin.Engine) {
	// Auth middleware for protected routes
	auth := middleware.CustomerAuth()

	// Address routes (protected) - /api/user/addresses/*
	addressRoutes := router.Group(constants.APIBaseUser + "/addresses")
	{
		addressRoutes.GET("", auth, m.addressHandler.GetAddresses)
		addressRoutes.GET("/:id", auth, m.addressHandler.GetAddressByID)
		addressRoutes.POST("", auth, m.addressHandler.AddAddress)
		addressRoutes.PUT("/:id", auth, m.addressHandler.UpdateAddress)
		addressRoutes.DELETE("/:id", auth, m.addressHandler.DeleteAddress)
		addressRoutes.PATCH("/:id/default", auth, m.addressHandler.SetDefaultAddress)
	}
}
