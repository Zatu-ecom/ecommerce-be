package product

import (
	"ecommerce-be/common"
	"ecommerce-be/product/routes"

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
func addModules(c *common.Container) {
	c.RegisterModule(routes.NewCategoryModule())
	c.RegisterModule(routes.NewAttributeModule())
	c.RegisterModule(routes.NewProductModule())
}
