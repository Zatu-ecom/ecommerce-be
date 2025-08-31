package userManagement

import (
	"ecommerce-be/common"
	"ecommerce-be/user_management/routes"

	"github.com/gin-gonic/gin"
)

/* NewContainer initializes dependencies dynamically */
func NewContainer(router *gin.Engine) *common.Container {

	/* Initialize Container */
	c := &common.Container{}

	/* Register all modules (Users, Auth, etc.) */
	addModules(c)

	/* Register routes for each module */
	for _, module := range c.Modules {
		module.RegisterRoutes(router)
	}

	return c
}

/* Register all modules (Users, Auth, etc.) */
func addModules(c *common.Container) {
	c.RegisterModule(routes.NewAddressModule())
	c.RegisterModule(routes.NewUserModule())
}
