package file

import (
	"ecommerce-be/common"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/file/factory/singleton"
	"ecommerce-be/file/route"
	"ecommerce-be/file/utils/constant"

	"github.com/gin-gonic/gin"
)

// NewContainer initializes file module dependencies and routes.
func NewContainer(router *gin.Engine) *common.Container {
	c := &common.Container{}

	addModules(c)

	// Register schedulers
	registerScheduler()

	for _, m := range c.Modules {
		m.RegisterRoutes(router)
	}

	return c
}

func registerScheduler() {
	f := singleton.GetInstance()
	scheduler.Register(
		constant.SchedulerCommandUploadExpiry,
		f.GetUploadExpiryHandler().Handle,
	)
}

// addModules registers all file-related modules to the container.
func addModules(c *common.Container) {
	c.RegisterModule(route.NewFileOperationModule())
	c.RegisterModule(route.NewFileStorageConfigModule())
	c.RegisterModule(route.NewFileImportExportModule())
}
