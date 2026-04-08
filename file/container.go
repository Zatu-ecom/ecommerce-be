package file

import (
	"ecommerce-be/common"
	"ecommerce-be/file/route"

	"github.com/gin-gonic/gin"
)

// NewContainer initializes file module dependencies and routes.
func NewContainer(router *gin.Engine) *common.Container {
	c := &common.Container{}

	addModules(c)

	for _, m := range c.Modules {
		m.RegisterRoutes(router)
	}

	return c
}

// addModules registers all file-related modules to the container.
func addModules(c *common.Container) {
	c.RegisterModule(route.NewFileOperationModule())
	c.RegisterModule(route.NewFileStorageConfigModule())
	c.RegisterModule(route.NewFileImportExportModule())
}
