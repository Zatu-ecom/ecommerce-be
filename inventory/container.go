package inventory

import (
	"ecommerce-be/common"
	routes "ecommerce-be/inventory/route"

	"github.com/gin-gonic/gin"
)

// NewContainer initializes dependencies dynamically
func NewContainer(router *gin.Engine) *common.Container {
	// Initialize Container
	c := &common.Container{}

	// Register all modules
	addModules(c)

	// Register routes for each module
	for _, module := range c.Modules {
		module.RegisterRoutes(router)
	}

	return c
}

// addModules registers all inventory-related modules
func addModules(c *common.Container) {
	c.RegisterModule(routes.NewLocationModule())
	// TODO: Add other inventory modules here (inventory, stock transfer, etc.)
}
