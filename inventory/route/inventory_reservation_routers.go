package routes

import (
	"ecommerce-be/common/constants"
	"ecommerce-be/common/middleware"
	"ecommerce-be/inventory/factory/singleton"
	"ecommerce-be/inventory/handler"

	"github.com/gin-gonic/gin"
)

type InventoryReservationModule struct {
	inventoryReservationHandler *handler.InventoryReservationHandler
}

func NewInventoryReservationModule() *InventoryReservationModule {
	f := singleton.GetInstance()

	return &InventoryReservationModule{
		inventoryReservationHandler: f.GetInventoryReservationHandler(),
	}
}

func (m *InventoryReservationModule) RegisterRoutes(router *gin.Engine) {
	sellerAuth := middleware.SellerAuth()

	reservationGroup := router.Group(constants.APIBaseInventory + "/reservation")
	{
		// Create a new inventory reservation
		reservationGroup.POST("", sellerAuth, m.inventoryReservationHandler.CreateReservation)

		// Update reservation status (CANCELLED or COMPLETED) by reference ID
		reservationGroup.PUT(
			"/status",
			sellerAuth,
			m.inventoryReservationHandler.UpdateReservationStatus,
		)
	}
}
