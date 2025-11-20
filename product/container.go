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
// TODO: We havve to use cache for most of this APIs because this sevice is very frequently
// use service by users so it is very important to use cache for this service and create this sevice by AI so
// as per my observation AI did not cache the data properly

// TODO: create reviews and ratings for product
func addModules(c *common.Container) {
	c.RegisterModule(routes.NewCategoryModule())
	c.RegisterModule(routes.NewAttributeModule())
	c.RegisterModule(routes.NewProductModule())
	c.RegisterModule(routes.NewProductAttributeModule())
	c.RegisterModule(routes.NewProductOptionModule())
	c.RegisterModule(routes.NewVariantModule())
}
