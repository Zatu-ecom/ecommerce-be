package payment

import (
	"ecommerce-be/common"

	"github.com/gin-gonic/gin"
)

/* NewContainer initializes dependencies dynamically */
func NewContainer(router *gin.Engine) *common.Container {
	/* Initialize Container */
	c := &common.Container{}

	/* Register all modules (Categories, Products, Attributes, etc.) */
	addModules(c)

	/* Register routes for each module */
	for _, module := range c.Modules {
		module.RegisterRoutes(router)
	}

	return c
}

/* Register all modules (Categories, Products, Attributes, etc.) */
// TODO: we have to implement payment service and this the start point for that
func addModules(c *common.Container) {
}
