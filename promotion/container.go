package promotion

import (
	"ecommerce-be/common"
	routes "ecommerce-be/promotion/route"

	"github.com/gin-gonic/gin"
)

// NewContainer initializes dependencies dynamically
func NewContainer(router *gin.Engine) *common.Container {
	// Initialize Container
	c := &common.Container{}

	// Register all modules
	c.RegisterModule(routes.NewPromotionScopeModule())

	// Register routes for each module
	for _, module := range c.Modules {
		module.RegisterRoutes(router)
	}

	return c
}
