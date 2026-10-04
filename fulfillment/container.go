package fulfillment

import (
	"context"
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/cron"
	"ecommerce-be/fulfillment/factory/singleton"
	"ecommerce-be/fulfillment/route"

	"github.com/gin-gonic/gin"
)

/* NewContainer initializes dependencies dynamically */
func NewContainer(router *gin.Engine) *common.Container {
	/* Initialize Container */
	c := &common.Container{}

	/* Register all modules (courier catalog, shipments, tracking, webhook). */
	addModules(c)

	/* Register schedulers */
	registerScheduler()

	/* Register routes for each module */
	for _, module := range c.Modules {
		module.RegisterRoutes(router)
	}

	return c
}

/*
Register all fulfillment sub-modules.

	Route modules land with the user-story phases:
	- US1 (T026): provider catalog + configure routes
	- US2 (T031): shipment draft + plan routes
	- US3 (T039): book/pickup/cancel/label routes
	- US4 (T045-T046): webhook + tracking routes
	- US5 (T050): NDR/return routes
*/
func addModules(c *common.Container) {
	c.RegisterModule(route.NewProviderModule())
	c.RegisterModule(route.NewShipmentModule())
	c.RegisterModule(route.NewTrackingModule())
	c.RegisterModule(route.NewWebhookModule())
}

/* registerScheduler registers recurring background jobs */
func registerScheduler() {
	// Finish interrupted bookings every 2 minutes. The closure resolves
	// the service at run time (lazy singletons). cron.Init() runs in
	// main.go before containers; in tests the scheduler is nil and
	// registration is a logged no-op (same as payment).
	_ = cron.RegisterIntervalJob(2*time.Minute, "fulfillment.recover_drafts", func() {
		_, _ = singleton.GetInstance().GetServiceFactory().GetRecoverService().
			RecoverDrafts(context.Background())
	})
	_ = cron.RegisterIntervalJob(5*time.Minute, "fulfillment.reconcile_pending", func() {
		_, _ = singleton.GetInstance().GetServiceFactory().GetReconcileService().
			ReconcilePending(context.Background())
	})
	_ = cron.RegisterIntervalJob(15*time.Minute, "fulfillment.ndr_sweep", func() {
		_, _ = singleton.GetInstance().GetServiceFactory().GetReconcileService().
			SweepNDR(context.Background())
	})
}
