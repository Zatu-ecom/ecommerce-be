package promotion

import (
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/cron"
	"ecommerce-be/promotion/factory/singleton"
	routes "ecommerce-be/promotion/route"

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

// addModules registers all promotion-related modules
func addModules(c *common.Container) {
	c.RegisterModule(routes.NewPromotionScopeModule())
	c.RegisterModule(routes.NewPromotionModule())
}

// registerScheduler registers recurring background jobs
func registerScheduler() {
	// Register the sweep job to run on a 1-minute interval
	cron.RegisterIntervalJob(
		1*time.Minute,
		"promotion_status_sweep",
		singleton.GetInstance().GetPromotionCronService().SweepStatusTransitions,
	)
}
