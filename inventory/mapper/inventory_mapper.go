package mapper

// LocationInventorySummaryAggregate represents aggregated inventory statistics for a location
// Used by GetLocationInventorySummary repository method
type LocationInventorySummaryAggregate struct {
	VariantCount    uint
	TotalStock      uint
	TotalReserved   uint
	LowStockCount   uint
	OutOfStockCount uint
	VariantIDs      []uint
}
