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
	c.RegisterModule(routes.NewSaleModule())
	c.RegisterModule(routes.NewPromotionModule())
	c.RegisterModule(routes.NewDiscountCodeScopeModule())
	c.RegisterModule(routes.NewDiscountCodeModule())
}

// registerScheduler registers recurring background jobs
//
// Multi-pod audit (012 T065): SweepStatusTransitions issues four single
// conditional UPDATEs (status + auto_* + time window in WHERE). Concurrent
// pods serialize per row in Postgres; the first transition wins and the
// second matches zero rows, so the sweep is idempotent by construction —
// unlike payment reconcile, which does per-row follow-up work and therefore
// needs FOR UPDATE SKIP LOCKED. No distributed lock required here.
//
// Stale-PENDING reconciler decision (012 T065): not applicable — neither
// promotions (scheduled/active/ended) nor discount codes (is_active) have a
// PENDING state, so there is nothing to age out. If a pending state is ever
// added, it needs a reconciler with the same conditional-UPDATE shape.
func registerScheduler() {
	// Register the sweep job to run on a 1-minute interval
	cron.RegisterIntervalJob(
		1*time.Minute,
		"promotion_status_sweep",
		singleton.GetInstance().GetPromotionCronService().SweepStatusTransitions,
	)
}
