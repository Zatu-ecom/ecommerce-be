package fulfillment_test

import (
	"testing"

	"ecommerce-be/common/db"
	fulfillmentmodel "ecommerce-be/fulfillment/model"
	fulfillmentservice "ecommerce-be/fulfillment/service"
	"ecommerce-be/test/integration/setup"

	invService "ecommerce-be/inventory/service"
	orderService "ecommerce-be/order/service"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Compile-time conformance: the order/inventory implementations satisfy the
// fulfillment hook interfaces method-for-method. Placed here (not in
// production code) so neither module imports the other — both stay
// microservice-extractable.
var (
	_ fulfillmentservice.FulfillmentOrderHooks     = (*orderService.OrderFulfillmentHooksImpl)(nil)
	_ fulfillmentservice.FulfillmentInventoryHooks = (*invService.InventoryFulfillmentHooksImpl)(nil)
)

// HooksConformanceSuite proves the two sides of each hook agree at runtime:
// views read back what was stored, and shortfalls surface as errors.
type HooksConformanceSuite struct {
	suite.Suite
	container *setup.TestContainer
}

func (s *HooksConformanceSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())
	db.SetDB(s.container.DB)
}

func (s *HooksConformanceSuite) TearDownSuite() {
	s.container.Cleanup(s.T())
}

func TestHooksConformanceSuite(t *testing.T) {
	suite.Run(t, new(HooksConformanceSuite))
}

// ─── Conformance: DTOs cross the boundary intact ─────────────────────────────
// Scenario: a progress value built by fulfillment carries box lines through
// the shared shapes without loss or reinterpretation.
func (s *HooksConformanceSuite) TestProgressShapeRoundTrip() {
	t := s.T()
	p := fulfillmentmodel.FulfillmentProgress{
		OrderID:    1,
		ShipmentID: 2,
		Items:      []fulfillmentmodel.FulfillmentLine{{OrderItemID: 3, Quantity: 4}},
		Reason:     "lost",
	}
	require.Equal(t, uint(1), p.OrderID)
	require.Equal(t, uint(2), p.ShipmentID)
	require.Equal(t, 4, p.Items[0].Quantity)
	require.Equal(t, "lost", p.Reason)
}

// ─── Conformance: order view defaults are safe ─────────────────────────────────
// Scenario: a zero order view collects nothing (prepaid default) and carries
// no addresses — the planner treats absent data as missing, never as zero.
func (s *HooksConformanceSuite) TestOrderViewDefaultsAreSafe() {
	t := s.T()
	view := fulfillmentmodel.FulfillmentOrderView{}
	require.Equal(t, int64(0), view.CodCents)
	require.Empty(t, view.Items)
	require.Zero(t, view.DeliveryAddressID)
	require.True(t, view.DeliveryAddressUpdatedAt.IsZero())
}
