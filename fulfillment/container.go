package fulfillment

import (
	"context"
	"fmt"
	"time"

	"ecommerce-be/common"
	"ecommerce-be/common/cron"
	logger "ecommerce-be/common/log"
	"ecommerce-be/fulfillment/factory/singleton"
	fulfillmentservice "ecommerce-be/fulfillment/service"
	"ecommerce-be/fulfillment/route"
	orderSingleton "ecommerce-be/order/factory/singleton"

	"github.com/gin-gonic/gin"
)

/* NewContainer initializes dependencies dynamically */
func NewContainer(router *gin.Engine) *common.Container {
	/* Initialize Container */
	c := &common.Container{}

	/* Register all modules (courier catalog, shipments, tracking, webhook). */
	addModules(c)

	/* Cross-module wiring: order confirmation triggers fulfillment
	   auto-planning. Owned here (not main) so the general entry point
	   stays free of domain wiring; safe when fulfillment is disabled. */
	wirePlanner()

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

// wirePlanner hands the fulfillment planner to the order service so order
// confirmation auto-plans drafts. Both singletons are lazy. Depends on a
// narrow interface (not the concrete OrderServiceImpl) so decorators and
// test doubles don't silently disable auto-planning; a mismatch logs loudly
// instead of no-op'ing.
func wirePlanner() {
	planner := singleton.GetInstance().GetServiceFactory().GetShipmentPlanner()
	type plannerSetter interface {
		SetShipmentPlanner(planner fulfillmentservice.ShipmentPlanner)
	}
	if setter, ok := orderSingleton.GetInstance().GetOrderService().(plannerSetter); ok {
		setter.SetShipmentPlanner(planner)
		return
	}
	logger.ErrorWithContext(context.Background(),
		"fulfillment planner wiring skipped: order service does not expose SetShipmentPlanner",
		fmt.Errorf("type %T lacks SetShipmentPlanner", orderSingleton.GetInstance().GetOrderService()))
}
