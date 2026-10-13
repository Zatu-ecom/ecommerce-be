package fulfillment_test

import (
	"net/http"

	"github.com/stretchr/testify/require"
)

// ─── Foundation: fulfillment tables exist ────────────────────────────────────
// Scenario: migrations 032/033 ran. Validates the eight fulfillment tables
// plus the physical-spec catalog table are present.
func (s *FulfillmentSuite) TestFoundationTablesExist() {
	tables := []string{
		"courier_provider", "courier_provider_field", "courier_provider_config",
		"fulfillment_shipment", "fulfillment_shipment_item", "fulfillment_shipment_event",
		"fulfillment_webhook_log", "fulfillment_ndr", "physical_spec_unit",
	}
	for _, table := range tables {
		s.Run(table, func() {
			require.True(s.T(),
				s.container.DB.Migrator().HasTable(table),
				"expected table %s from migrations 032/033", table)
		})
	}
}

// ─── Foundation: courier catalog seeded ──────────────────────────────────────
// Scenario: core seed 006 ran. Validates the Shiprocket catalog row and its
// three credential-field rows exist.
func (s *FulfillmentSuite) TestFoundationCourierCatalogSeeded() {
	var providerCount int64
	require.NoError(s.T(), s.container.DB.Table("courier_provider").
		Where("code = ? AND is_active = ?", "shiprocket", true).
		Count(&providerCount).Error)
	require.Equal(s.T(), int64(1), providerCount, "shiprocket catalog row")

	var fieldCount int64
	require.NoError(s.T(), s.container.DB.Table("courier_provider_field").
		Where("provider_code = ?", "shiprocket").
		Count(&fieldCount).Error)
	require.GreaterOrEqual(s.T(), fieldCount, int64(2), "credential field rows")
}

// ─── Foundation: physical spec catalog seeded ────────────────────────────────
// Scenario: core seed 005 ran. Validates 4 families × 2 units with exactly
// one active base row per family.
func (s *FulfillmentSuite) TestFoundationPhysicalSpecsSeeded() {
	type family struct {
		Parameter string
		Units     int64
		Bases     int64
	}
	var rows []family
	require.NoError(s.T(), s.container.DB.Table("physical_spec_unit").
		Select("parameter, COUNT(*) AS units, SUM(CASE WHEN is_base THEN 1 ELSE 0 END) AS bases").
		Where("is_active = ?", true).
		Group("parameter").
		Scan(&rows).Error)
	require.Len(s.T(), rows, 4, "weight/length/breadth/height families")
	for _, row := range rows {
		require.Equal(s.T(), int64(2), row.Units, "two units for %s", row.Parameter)
		require.Equal(s.T(), int64(1), row.Bases, "one base for %s", row.Parameter)
	}
}

// ─── Foundation: unknown fulfillment route 404s ──────────────────────────────
// Scenario: the module container is registered but exposes no routes yet
// (routes land with the story phases). Validates boot + router wiring.
func (s *FulfillmentSuite) TestFoundationUnknownRoute404() {
	w := s.sellerClient.Get(s.T(), "/api/fulfillment/no-such-route")
	require.Equal(s.T(), http.StatusNotFound, w.Code)
}
