package inventory

import (
	"ecommerce-be/common"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/inventory/factory/singleton"
	routes "ecommerce-be/inventory/route"
	"ecommerce-be/inventory/utils/constant"

	"github.com/gin-gonic/gin"
)

// NewContainer initializes dependencies dynamically
func NewContainer(router *gin.Engine) *common.Container {
	// Initialize Container
	c := &common.Container{}

	// Register all modules
	addModules(c)

	// Register schedulers
	registerScheduler()

	// Register routes for each module
	for _, module := range c.Modules {
		module.RegisterRoutes(router)
	}

	return c
}

// addModules registers all inventory-related modules
func addModules(c *common.Container) {
	c.RegisterModule(routes.NewLocationModule())
	c.RegisterModule(routes.NewInventoryModule())
	c.RegisterModule(routes.NewInventoryReservationModule())
	// TODO: Add other inventory modules here (stock transfer, etc.)
}

func registerScheduler() {
	f := singleton.GetInstance()
	scheduleInventoryReservationHandler := f.GetScheduleInventoryReservationHandler()
	
	scheduler.Register(
		constant.INVENTORYY_RESERVATION_EXPRIY_EVENT_COMMAND,
		scheduleInventoryReservationHandler.ExpireScheduleReservation,
	)
}
